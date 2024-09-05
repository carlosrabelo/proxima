package proxmox

import "strings"

// DeriveSSHUser returns the SSH username from a Proxmox identity
// (password User or TokenID), using the substring before '@'.
func DeriveSSHUser(identity string) string {
	if i := strings.IndexByte(identity, '@'); i >= 0 {
		return identity[:i]
	}
	return identity
}
