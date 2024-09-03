package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
)

func TestConfigCheckPrintsRedactedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxima.yaml")
	content := `
proxmox_host: "192.168.1.10"
node: "pve"
user: "root@pam"
password: "super-secret"
vmid: 200
name: "alpine-docker"
memory: 2048
cores: 2
disk_size: "8G"
root_password: "root-secret"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"config", "check", "-c", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "super-secret") || strings.Contains(out, "root-secret") {
		t.Fatalf("secrets leaked in output: %q", out)
	}
	if !strings.Contains(out, "password: '***'") && !strings.Contains(out, "password: ***") {
		t.Fatalf("missing redacted password: %q", out)
	}
	if !strings.Contains(out, "root_password: '***'") && !strings.Contains(out, "root_password: ***") {
		t.Fatalf("missing redacted root_password: %q", out)
	}
	if !strings.Contains(out, "proxmox_host:") || !strings.Contains(out, "alpine-docker") {
		t.Fatalf("unexpected yaml: %q", out)
	}
}

func TestConfigCheckInvalidExitsOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := `
proxmox_host: ""
node: "pve"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"config", "check", "-c", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "Error: ") {
		t.Fatalf("stderr = %q, want Error: prefix", stderr.String())
	}
	if !strings.Contains(stderr.String(), "proxmox_host is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
