package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
	"proxima/proxima/internal/config"
	"proxima/proxima/internal/upload"
)

func TestUploadCommandMissingFileExitThree(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ok.yaml")
	missing := filepath.Join(dir, "missing.qcow2")
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
output: "` + missing + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.SetUploadFactoryForTest(func(cfg config.Config) (*upload.Uploader, error) {
		return &upload.Uploader{}, nil
	})
	t.Cleanup(cli.ResetUploadFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"upload", "-c", cfgPath}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: disk upload failed: local file not found:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("stderr missing path %q: %q", missing, stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> Upload: starting") {
		t.Fatalf("missing start stage line: %q", stderr.String())
	}
}
