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
