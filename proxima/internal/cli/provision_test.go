package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func writeProvisionCfg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ok.yaml")
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
	return cfgPath
}

type provisionFakeSSH struct {
	failImport  bool
	failDestroy bool
	commands    []string
}

func (f *provisionFakeSSH) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	_ = ctx
	f.commands = append(f.commands, command)
	if strings.HasPrefix(command, "qm status ") {
		return nil, fmt.Errorf("missing")
	}
	if f.failImport && strings.HasPrefix(command, "qm importdisk ") {
		return nil, fmt.Errorf("import failed")
	}
	if f.failDestroy && strings.HasPrefix(command, "qm destroy ") {
		return nil, fmt.Errorf("destroy failed")
	}
	if strings.HasPrefix(command, "qm config ") {
		return []byte("unused0: local-lvm:vm-200-disk-0\n"), nil
	}
	return nil, nil
}

func (f *provisionFakeSSH) Close() error { return nil }

func TestProvisionCommandExitThreeOnFailure(t *testing.T) {
	cfgPath := writeProvisionCfg(t)
	fake := &provisionFakeSSH{failImport: true}
	cli.SetProvisionClientFactoryForTest(func(cfg config.Config) (*proxmox.Client, error) {
		c := &proxmox.Client{}
		c.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
			return fake, nil
		})
		return c, nil
	})
	t.Cleanup(cli.ResetProvisionClientFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"provision", "-c", cfgPath}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> Provision: starting") {
		t.Fatalf("missing start line: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: vm provision failed:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProvisionCommandExitFourOnCleanupFailure(t *testing.T) {
	cfgPath := writeProvisionCfg(t)
	fake := &provisionFakeSSH{failImport: true, failDestroy: true}
	cli.SetProvisionClientFactoryForTest(func(cfg config.Config) (*proxmox.Client, error) {
		c := &proxmox.Client{}
		c.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
			return fake, nil
		})
		return c, nil
	})
	t.Cleanup(cli.ResetProvisionClientFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"provision", "-c", cfgPath}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("exit code = %d, want 4; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: vm provision failed and cleanup failed:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProvisionCommandSuccess(t *testing.T) {
	cfgPath := writeProvisionCfg(t)
	fake := &provisionFakeSSH{}
	cli.SetProvisionClientFactoryForTest(func(cfg config.Config) (*proxmox.Client, error) {
		c := &proxmox.Client{}
		c.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
			return fake, nil
		})
		return c, nil
	})
	t.Cleanup(cli.ResetProvisionClientFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"provision", "-c", cfgPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> Provision: done (vmid=200 name=alpine-docker)") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
