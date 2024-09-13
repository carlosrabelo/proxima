package proxmox

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHSession is a remote shell session on the Proxmox node.
type SSHSession interface {
	CombinedOutput(ctx context.Context, command string) ([]byte, error)
	Close() error
}

// SSHUploader is an SSHSession that can also copy files (SCP).
type SSHUploader interface {
	SSHSession
	Upload(ctx context.Context, localPath, remotePath string) error
}

// DeriveSSHUser returns the SSH username from a Proxmox identity
// (password User or TokenID), using the substring before '@'.
func DeriveSSHUser(identity string) string {
	if i := strings.IndexByte(identity, '@'); i >= 0 {
		return identity[:i]
	}
	return identity
}

// SSH opens an SSH session to the Proxmox host.
func (c *Client) SSH(ctx context.Context) (SSHSession, error) {
	if c.sshOpen != nil {
		return c.sshOpen(ctx)
	}

	hasToken := c.auth.TokenID != "" || c.auth.TokenSecret != ""
	hasPasswordAuth := c.auth.User != "" || c.auth.Password != ""
	if c.sshPrivateKeyPath == "" && hasToken && !hasPasswordAuth {
		return nil, fmt.Errorf("ssh credentials required: set ssh_private_key_path when using token authentication")
	}

	authMethods, err := c.sshAuthMethods()
	if err != nil {
		return nil, err
	}
	hostKeyCb, err := c.sshHostKeyCallback()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            c.sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCb,
		Timeout:         30 * time.Second,
	}

	addr := net.JoinHostPort(c.sshHost, fmt.Sprintf("%d", c.sshPort))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake: %w", err)
	}
	_ = conn.SetDeadline(time.Time{}) // clear dial deadline for long uploads

	return &liveSSH{client: ssh.NewClient(sshConn, chans, reqs)}, nil
}

func (c *Client) sshAuthMethods() ([]ssh.AuthMethod, error) {
	if c.sshPrivateKeyPath != "" {
		key, err := os.ReadFile(c.sshPrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("ssh private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("ssh private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if c.sshPassword != "" {
		return []ssh.AuthMethod{ssh.Password(c.sshPassword)}, nil
	}
	return nil, fmt.Errorf("ssh credentials required: set ssh_private_key_path when using token authentication")
}

func (c *Client) sshHostKeyCallback() (ssh.HostKeyCallback, error) {
	if c.insecureTLS {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec // intentional when InsecureTLS
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("ssh host key verification unavailable: %v", err)
	}
	path := filepath.Join(home, ".ssh", "known_hosts")
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("ssh host key verification unavailable: %v", err)
	}
	return cb, nil
}

// SetSSHOpenForTest replaces the SSH dialer (tests only).
func (c *Client) SetSSHOpenForTest(fn func(ctx context.Context) (SSHSession, error)) {
	c.sshOpen = fn
}

type liveSSH struct {
	client *ssh.Client
}

func (s *liveSSH) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := session.CombinedOutput(command)
		ch <- result{out: out, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return nil, ctx.Err()
	case r := <-ch:
		return r.out, r.err
	}
}

func (s *liveSSH) Upload(ctx context.Context, localPath, remotePath string) error {
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	info, err := local.Stat()
	if err != nil {
		return err
	}

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	remoteDir := filepath.ToSlash(filepath.Dir(remotePath))
	remoteBase := filepath.Base(remotePath)
	cmd := "scp -t -- " + shellSingleQuote(remoteDir)

	if err := session.Start(cmd); err != nil {
		return err
	}

	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		defer stdin.Close()

		if err := scpReadAck(stdout); err != nil {
			done <- result{err}
			return
		}
		header := fmt.Sprintf("C0644 %d %s\n", info.Size(), remoteBase)
		if _, err := io.WriteString(stdin, header); err != nil {
			done <- result{err}
			return
		}
		if err := scpReadAck(stdout); err != nil {
			done <- result{err}
			return
		}
		if _, err := io.Copy(stdin, local); err != nil {
			done <- result{err}
			return
		}
		if _, err := stdin.Write([]byte{0}); err != nil {
			done <- result{err}
			return
		}
		if err := scpReadAck(stdout); err != nil {
			done <- result{err}
			return
		}
		done <- result{nil}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return ctx.Err()
	case r := <-done:
		waitErr := session.Wait()
		if r.err != nil {
			return r.err
		}
		return waitErr
	}
}

func (s *liveSSH) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func scpReadAck(r io.Reader) error {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("scp ack: %w", err)
	}
	switch buf[0] {
	case 0:
		return nil
	case 1, 2:
		var msg strings.Builder
		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				break
			}
			if buf[0] == '\n' {
				break
			}
			msg.WriteByte(buf[0])
		}
		return fmt.Errorf("scp: %s", msg.String())
	default:
		return fmt.Errorf("scp: unexpected ack %d", buf[0])
	}
}
