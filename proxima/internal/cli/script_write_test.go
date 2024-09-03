package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
)

func TestScriptWriteSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "setup.sh")

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs(
		[]string{"script", "write", "--root-password", "test-root-pass", "-o", path},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if got, want := stdout.String(), "Wrote "+path+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
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
	body := string(data)
	for _, needle := range []string{"apk add", "docker", "qemu-guest-agent", "openssh", "rc-update add docker", "rc-update add qemu-guest-agent", "rc-update add sshd"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("script missing %q", needle)
		}
	}
}

func TestScriptWriteMissingPassword(t *testing.T) {
	t.Setenv("PROXIMA_ROOT_PASSWORD", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "setup.sh")

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"script", "write", "-o", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if got, want := stderr.String(), "Error: root_password is required\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
