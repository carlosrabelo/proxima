package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
	"proxima/proxima/internal/diskbuild"
)

type buildFakeDocker struct {
	failRun bool
	workDir string
}

func (f *buildFakeDocker) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if f.failRun && len(args) > 0 && args[0] == "run" {
		return errors.New("boom")
	}
	if len(args) > 0 && args[0] == "run" && f.workDir != "" {
		return os.WriteFile(filepath.Join(f.workDir, "image.qcow2"), []byte("img"), 0o644)
	}
	return nil
}

func TestBuildCommandConfigErrorExitOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("proxmox_host: \"\"\nnode: pve\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"build", "-c", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "Error: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "proxmox_host is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBuildCommandBuildErrorExitTwo(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ok.yaml")
	workDir := filepath.Join(dir, "work")
	output := filepath.Join(dir, "out.qcow2")
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
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.SetBuildDockerFactoryForTest(func() diskbuild.DockerRunner {
		return &buildFakeDocker{failRun: true, workDir: workDir}
	})
	t.Cleanup(cli.ResetBuildDockerFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs(
		[]string{"build", "-c", cfgPath, "--work-dir", workDir, "--output", output},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> BuildLocal: starting") {
		t.Fatalf("missing starting line: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: local disk build failed:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
