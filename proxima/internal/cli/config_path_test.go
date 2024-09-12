package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
)

func TestConfigCheckUsesDefaultVmsProximaYAML(t *testing.T) {
	dir := t.TempDir()
	vmsDir := filepath.Join(dir, "vms")
	if err := os.MkdirAll(vmsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `
proxmox_host: "192.168.1.10"
node: "pve"
user: "root@pam"
password: "secret"
vmid: 200
name: "alpine-docker"
memory: 2048
cores: 2
disk_size: "8G"
root_password: "change-me"
`
	if err := os.WriteFile(filepath.Join(vmsDir, "Proxima.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"config", "check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "proxmox_host:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestConfigCheckMissingManifestExitOne(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"config", "check"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "config manifesto not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
