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
	if len(args) > 0 && args[0] == "run" && f.workDir != "" {
		if err := os.WriteFile(filepath.Join(f.workDir, "image.qcow2"), []byte("fake-qcow2"), 0o644); err != nil {
			return err
		}
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
	outDir := t.TempDir()
	output := filepath.Join(outDir, "alpine-ready.qcow2")
	setupPath := filepath.Join(workDir, "setup.sh")
	var sawSetupBeforeDocker bool

	docker := &fakeDocker{
		workDir: workDir,
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
		OutputQcow2:  output,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
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
	outDir := t.TempDir()
	output := filepath.Join(outDir, "out.qcow2")
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatal(err)
	}

	docker := &fakeDocker{workDir: workDir}
	b := &diskbuild.Builder{Docker: docker}
	_, err = b.Build(context.Background(), config.Config{
		WorkDir:      workDir,
		RootPassword: "test-root-pass",
		DiskSize:     "8G",
		OutputQcow2:  output,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
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
	if !containsArg(runArgs, "/dev:/dev") {
		t.Fatalf("missing /dev mount: %v", runArgs)
	}
	if !containsArg(runArgs, "/lib/modules:/lib/modules:ro") {
		t.Fatalf("missing /lib/modules mount: %v", runArgs)
	}
}

func TestBuildMovesQcow2ToOutput(t *testing.T) {
	workDir := t.TempDir()
	outDir := t.TempDir()
	output := filepath.Join(outDir, "subdir", "alpine-ready.qcow2")

	docker := &fakeDocker{workDir: workDir}
	b := &diskbuild.Builder{Docker: docker}
	res, err := b.Build(context.Background(), config.Config{
		WorkDir:      workDir,
		RootPassword: "test-root-pass",
		DiskSize:     "8G",
		OutputQcow2:  output,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "image.qcow2")); !os.IsNotExist(err) {
		t.Fatalf("work image should be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "setup.sh")); !os.IsNotExist(err) {
		t.Fatalf("setup.sh should be removed, err=%v", err)
	}
	if res.Bytes != int64(len("fake-qcow2")) {
		t.Fatalf("Bytes = %d, want %d", res.Bytes, len("fake-qcow2"))
	}
	absOut, _ := filepath.Abs(output)
	if res.Qcow2Path != absOut {
		t.Fatalf("Qcow2Path = %q, want %q", res.Qcow2Path, absOut)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-qcow2" {
		t.Fatalf("output content = %q", data)
	}
}

func TestBuildCleansWorkDirOnFailure(t *testing.T) {
	workDir := t.TempDir()
	outDir := t.TempDir()
	output := filepath.Join(outDir, "alpine-ready.qcow2")
	setupPath := filepath.Join(workDir, "setup.sh")
	imagePath := filepath.Join(workDir, "image.qcow2")

	docker := &fakeDocker{
		onRun: func(args []string) error {
			if len(args) > 0 && args[0] == "run" {
				if err := os.WriteFile(imagePath, []byte("partial-image"), 0o644); err != nil {
					return err
				}
				return errors.New("container failed")
			}
			return nil
		},
	}

	b := &diskbuild.Builder{Docker: docker}
	_, err := b.Build(context.Background(), config.Config{
		WorkDir:      workDir,
		RootPassword: "test-root-pass",
		DiskSize:     "8G",
		OutputQcow2:  output,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "local disk build failed: ") {
		t.Fatalf("error = %q", err.Error())
	}
	if _, err := os.Stat(setupPath); !os.IsNotExist(err) {
		t.Fatalf("setup.sh should be cleaned, err=%v", err)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("image.qcow2 should be cleaned, err=%v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output should not remain from failed build, err=%v", err)
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
