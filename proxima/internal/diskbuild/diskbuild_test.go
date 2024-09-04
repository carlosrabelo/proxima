package diskbuild_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
)

type fakeDocker struct {
	err     error
	calls   [][]string
	onRun   func(args []string) error
	workDir string
}

func (f *fakeDocker) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cp := append([]string(nil), args...)
	f.calls = append(f.calls, cp)
	if f.onRun != nil {
		if err := f.onRun(args); err != nil {
			return err
		}
	}
	if f.err != nil {
		if stderr != nil {
			_, _ = io.WriteString(stderr, "Cannot connect to the Docker daemon")
		}
		return f.err
	}
	return nil
}

func TestBuildDockerUnavailable(t *testing.T) {
	b := &diskbuild.Builder{
		Docker: &fakeDocker{err: errors.New("exit status 1")},
	}
	_, err := b.Build(context.Background(), config.Config{
		WorkDir:      t.TempDir(),
		RootPassword: "test-root-pass",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "docker is not available: ") {
		t.Fatalf("error = %q, want prefix %q", msg, "docker is not available: ")
	}
	if !strings.Contains(msg, "Cannot connect to the Docker daemon") {
		t.Fatalf("error = %q, want docker detail", msg)
	}
}

func TestBuildWritesSetupScript(t *testing.T) {
	workDir := t.TempDir()
	setupPath := filepath.Join(workDir, "setup.sh")
	var sawSetupBeforeDocker bool

	docker := &fakeDocker{
		onRun: func(args []string) error {
			if _, err := os.Stat(setupPath); err != nil {
				t.Fatalf("setup.sh missing before docker %v: %v", args, err)
			}
			data, err := os.ReadFile(setupPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "apk add") {
				t.Fatalf("setup.sh content unexpected: %q", data)
			}
			sawSetupBeforeDocker = true
			return nil
		},
	}

	b := &diskbuild.Builder{Docker: docker}
	_, err := b.Build(context.Background(), config.Config{
		WorkDir:      workDir,
		RootPassword: "test-root-pass",
		DiskSize:     "8G",
	})
	if err == nil {
		t.Fatal("expected not-implemented error")
	}
	if !strings.Contains(err.Error(), "local disk build failed:") {
		t.Fatalf("error = %q", err.Error())
	}
	if !sawSetupBeforeDocker {
		t.Fatal("docker was not invoked after setup.sh write")
	}
	if len(docker.calls) == 0 || docker.calls[0][0] != "version" {
		t.Fatalf("expected docker version first, got %v", docker.calls)
	}
}

func TestBuildInvokesPrivilegedAlpine(t *testing.T) {
	workDir := t.TempDir()
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatal(err)
	}

	docker := &fakeDocker{}
	b := &diskbuild.Builder{Docker: docker}
	_, err = b.Build(context.Background(), config.Config{
		WorkDir:      workDir,
		RootPassword: "test-root-pass",
		DiskSize:     "8G",
	})
	if err == nil {
		t.Fatal("expected error after docker run")
	}

	var runArgs []string
	for _, call := range docker.calls {
		if len(call) > 0 && call[0] == "run" {
			runArgs = call
			break
		}
	}
	if runArgs == nil {
		t.Fatalf("docker run not invoked; calls=%v", docker.calls)
	}

	joined := strings.Join(runArgs, " ")
	for _, want := range []string{"--rm", "--privileged", "alpine:latest", absWork + ":/work", "-w", "/work"} {
		if !containsArg(runArgs, want) && !strings.Contains(joined, want) {
			t.Fatalf("run args missing %q: %v", want, runArgs)
		}
	}
	if !containsArg(runArgs, "--privileged") {
		t.Fatalf("missing --privileged: %v", runArgs)
	}
	if !containsArg(runArgs, "alpine:latest") {
		t.Fatalf("missing alpine:latest: %v", runArgs)
	}
	mount := absWork + ":/work"
	if !containsArg(runArgs, mount) {
		t.Fatalf("missing mount %q: %v", mount, runArgs)
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
