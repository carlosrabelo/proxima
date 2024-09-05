package proxmox

import (
	"context"
	"fmt"
	"strings"
)

// SSHSession is a remote shell session on the Proxmox node.
type SSHSession interface {
	CombinedOutput(ctx context.Context, command string) ([]byte, error)
	Close() error
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
	_ = ctx
	hasToken := c.auth.TokenID != "" || c.auth.TokenSecret != ""
	hasPasswordAuth := c.auth.User != "" || c.auth.Password != ""
	if c.sshPrivateKeyPath == "" && hasToken && !hasPasswordAuth {
		return nil, fmt.Errorf("ssh credentials required: set ssh_private_key_path when using token authentication")
	}
	return nil, fmt.Errorf("ssh session not implemented")
}
