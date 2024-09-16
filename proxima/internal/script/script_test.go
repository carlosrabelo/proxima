package script_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/script"
)

func TestRenderRequiresPassword(t *testing.T) {
	_, err := script.Render(script.Params{})
	if err == nil {
		t.Fatal("expected error for empty root password")
	}
	if err.Error() != "root_password is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "root_password is required")
	}

	_, err = script.Render(script.Params{RootPassword: "test-root-pass"})
	if err != nil {
		t.Fatalf("Render with password: %v", err)
	}
}

func TestRenderMatchesGolden(t *testing.T) {
	got, err := script.Render(script.Params{RootPassword: "test-root-pass"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "setup_golden.sh"))
	if err != nil {
		t.Fatal(err)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("Render output does not match golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderContainsPackagesAndServices(t *testing.T) {
	got, err := script.Render(script.Params{RootPassword: "test-root-pass"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, needle := range []string{
		"#!/bin/sh",
		"set -e",
		"apk update",
		"apk add --no-cache docker docker-compose qemu-guest-agent openssh",
		"rc-update add docker default",
		"rc-update add qemu-guest-agent default",
		"rc-update add sshd default",
		"chpasswd",
		"PermitRootLogin yes",
		"dl-cdn.alpinelinux.org/alpine",
		"/main",
		"/community",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("script missing %q", needle)
		}
	}
}

func TestRenderIncludesAuthorizedKey(t *testing.T) {
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyMaterialForTest carlos@host"
	got, err := script.Render(script.Params{
		RootPassword:     "test-root-pass",
		SSHAuthorizedKey: key + "\n",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, needle := range []string{
		"mkdir -p /root/.ssh",
		"chmod 700 /root/.ssh",
		"chmod 600 /root/.ssh/authorized_keys",
		"printf '%s\\n' '" + key + "' > /root/.ssh/authorized_keys",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("script missing %q\n%s", needle, got)
		}
	}
}

func TestReadPublicKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_rsa.pub")
	key := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7 test@host\n"
	if err := os.WriteFile(path, []byte(key), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := script.ReadPublicKeyFile(path)
	if err != nil {
		t.Fatalf("ReadPublicKeyFile: %v", err)
	}
	want := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7 test@host"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteFileMode0755(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "setup.sh")
	if err := script.WriteFile(path, script.Params{RootPassword: "test-root-pass"}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o755 {
		t.Fatalf("mode = %o, want 0755", mode)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := script.Render(script.Params{RootPassword: "test-root-pass"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatal("WriteFile content does not match Render output")
	}
}
