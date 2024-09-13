package proxmox_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func TestSSHPasswordCombinedOutput(t *testing.T) {
	host, port := sshListen(t, &sshServer{
		user:     "root",
		password: "lab-pass",
		exec: func(cmd string, stdout io.Writer) error {
			if cmd == "echo ok" {
				_, _ = io.WriteString(stdout, "ok\n")
				return nil
			}
			return fmt.Errorf("unexpected: %s", cmd)
		},
	})

	sess, err := newSSHClient(t, host, port, "root", "lab-pass", "").SSH(context.Background())
	if err != nil {
		t.Fatalf("SSH: %v", err)
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(context.Background(), "echo ok")
	if err != nil {
		t.Fatalf("CombinedOutput: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "ok" {
		t.Fatalf("output = %q", got)
	}
}

func TestSSHUploadSCP(t *testing.T) {
	var mu sync.Mutex
	files := map[string][]byte{}

	host, port := sshListen(t, &sshServer{
		user:     "root",
		password: "lab-pass",
		scpSink: func(dir, name string, data []byte) error {
			mu.Lock()
			defer mu.Unlock()
			files[filepath.ToSlash(filepath.Join(dir, name))] = append([]byte(nil), data...)
			return nil
		},
	})

	sess, err := newSSHClient(t, host, port, "root", "lab-pass", "").SSH(context.Background())
	if err != nil {
		t.Fatalf("SSH: %v", err)
	}
	defer sess.Close()

	up, ok := sess.(proxmox.SSHUploader)
	if !ok {
		t.Fatal("session should implement SSHUploader")
	}

	dir := t.TempDir()
	local := filepath.Join(dir, "disk.qcow2")
	payload := []byte("proxima-qcow2-bytes")
	if err := os.WriteFile(local, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	remote := "/var/tmp/proxima/101-disk.qcow2"
	if err := up.Upload(context.Background(), local, remote); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	mu.Lock()
	got := files[remote]
	mu.Unlock()
	if !bytes.Equal(got, payload) {
		t.Fatalf("remote = %q, want %q", got, payload)
	}
}

func TestSSHKeyAuth(t *testing.T) {
	keyPath, pub := writeTempSSHKey(t)
	host, port := sshListen(t, &sshServer{
		user:      "root",
		publicKey: pub,
		exec: func(cmd string, stdout io.Writer) error {
			_, _ = io.WriteString(stdout, "keyed\n")
			return nil
		},
	})

	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost:       host,
		ProxmoxPort:       8006,
		SSHPort:           port,
		Node:              "pve",
		Auth:              config.Auth{TokenID: "root@pam!proxima", TokenSecret: "secret"},
		SSHUser:           "root",
		SSHPrivateKeyPath: keyPath,
		InsecureTLS:       true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	sess, err := client.SSH(context.Background())
	if err != nil {
		t.Fatalf("SSH: %v", err)
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(context.Background(), "true")
	if err != nil {
		t.Fatalf("CombinedOutput: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "keyed" {
		t.Fatalf("output = %q", got)
	}
}

func TestSSHHostKeyVerificationUnavailable(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "missing-home"))
	keyPath, _ := writeTempSSHKey(t)
	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost:       "127.0.0.1",
		ProxmoxPort:       8006,
		Node:              "pve",
		Auth:              config.Auth{TokenID: "root@pam!proxima", TokenSecret: "secret"},
		SSHPrivateKeyPath: keyPath,
		InsecureTLS:       false,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.SSH(context.Background())
	if err == nil {
		t.Fatal("expected known_hosts error")
	}
	if !strings.HasPrefix(err.Error(), "ssh host key verification unavailable:") {
		t.Fatalf("error = %q", err.Error())
	}
}

// newSSHClient builds a client that dials SSH with password without calling the API ticket endpoint.
// Token+password together skips ticket auth in NewClient while still storing the SSH password.
func newSSHClient(t *testing.T, host string, port int, user, password, keyPath string) *proxmox.Client {
	t.Helper()
	cfg := config.Config{
		ProxmoxHost: host,
		ProxmoxPort: 8006,
		SSHPort:     port,
		Node:        "pve",
		Auth: config.Auth{
			User:        user + "@pam",
			Password:    password,
			TokenID:     user + "@pam!proxima",
			TokenSecret: "secret",
		},
		SSHUser:           user,
		SSHPrivateKeyPath: keyPath,
		InsecureTLS:       true,
	}
	client, err := proxmox.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

type sshServer struct {
	user      string
	password  string
	publicKey ssh.PublicKey
	exec      func(cmd string, stdout io.Writer) error
	scpSink   func(dir, name string, data []byte) error
}

func sshListen(t *testing.T, srv *sshServer) (host string, port int) {
	t.Helper()
	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &ssh.ServerConfig{}
	if srv.password != "" {
		cfg.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == srv.user && string(pass) == srv.password {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		}
	}
	if srv.publicKey != nil {
		cfg.PublicKeyCallback = func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() == srv.user && bytes.Equal(key.Marshal(), srv.publicKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		}
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			tcp, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSSHConn(tcp, cfg, srv)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

func handleSSHConn(tcp net.Conn, cfg *ssh.ServerConfig, srv *sshServer) {
	conn, chans, reqs, err := ssh.NewServerConn(tcp, cfg)
	if err != nil {
		_ = tcp.Close()
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "unknown")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			continue
		}
		go handleSSHSession(ch, requests, srv)
	}
}

func handleSSHSession(ch ssh.Channel, requests <-chan *ssh.Request, srv *sshServer) {
	defer ch.Close()
	for req := range requests {
		if req.Type != "exec" {
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
			continue
		}
		var payload struct {
			Command string
		}
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			_ = req.Reply(false, nil)
			return
		}
		_ = req.Reply(true, nil)

		cmd := payload.Command
		var status uint32
		switch {
		case strings.HasPrefix(cmd, "scp -t ") && srv.scpSink != nil:
			if err := handleScpSink(ch, cmd, srv.scpSink); err != nil {
				status = 1
				_, _ = io.WriteString(ch.Stderr(), err.Error()+"\n")
			}
		case srv.exec != nil:
			if err := srv.exec(cmd, ch); err != nil {
				status = 1
				_, _ = io.WriteString(ch.Stderr(), err.Error()+"\n")
			}
		default:
			status = 1
		}
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(&struct{ Status uint32 }{status}))
		return
	}
}

func handleScpSink(ch ssh.Channel, cmd string, sink func(dir, name string, data []byte) error) error {
	rest := strings.TrimSpace(strings.TrimPrefix(cmd, "scp -t "))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "--"))
	dir := strings.Trim(rest, "'")

	if _, err := ch.Write([]byte{0}); err != nil {
		return err
	}

	header, err := readLine(ch)
	if err != nil {
		return err
	}
	var mode string
	var size int
	var name string
	if _, err := fmt.Sscanf(header, "C%s %d %s", &mode, &size, &name); err != nil {
		_, _ = ch.Write([]byte{2})
		return fmt.Errorf("bad header %q", header)
	}
	if _, err := ch.Write([]byte{0}); err != nil {
		return err
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(ch, data); err != nil {
		return err
	}
	ack := make([]byte, 1)
	if _, err := io.ReadFull(ch, ack); err != nil {
		return err
	}
	if err := sink(dir, name, data); err != nil {
		_, _ = ch.Write([]byte{2})
		return err
	}
	_, err = ch.Write([]byte{0})
	return err
}

func readLine(r io.Reader) (string, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, tmp); err != nil {
			return "", err
		}
		if tmp[0] == '\n' {
			return buf.String(), nil
		}
		buf.WriteByte(tmp[0])
	}
}

func writeTempSSHKey(t *testing.T) (path string, pub ssh.PublicKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	path = filepath.Join(t.TempDir(), "id_rsa")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path, signer.PublicKey()
}
