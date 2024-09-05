package proxmox_test

import (
	"context"
	"testing"

	"proxima/proxima/internal/config"
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

func TestSSHRequiresKeyForTokenAuth(t *testing.T) {
	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost: "127.0.0.1",
		ProxmoxPort: 8006,
		Node:        "pve",
		Auth: config.Auth{
			TokenID:     "root@pam!proxima",
			TokenSecret: "secret",
		},
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.SSH(context.Background())
	if err == nil {
		t.Fatal("expected SSH credentials error")
	}
	want := "ssh credentials required: set ssh_private_key_path when using token authentication"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
