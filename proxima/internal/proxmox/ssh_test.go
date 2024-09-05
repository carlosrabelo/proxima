package proxmox_test

import (
	"testing"

	"proxima/proxima/internal/proxmox"
)

func TestSSHUserDerivation(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"root@pam", "root"},
		{"root@pam!proxima", "root"},
		{"admin@pve", "admin"},
		{"plain", "plain"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := proxmox.DeriveSSHUser(tc.in); got != tc.want {
			t.Fatalf("DeriveSSHUser(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
