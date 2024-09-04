package diskbuild

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/script"
)

// DockerRunner runs docker CLI invocations.
type DockerRunner interface {
	Run(ctx context.Context, args []string, stdout, stderr io.Writer) error
}

// Builder builds a local qcow2 via an ephemeral Alpine container.
type Builder struct {
	Docker DockerRunner
	Clock  interface{ Now() time.Time }
}

// Result is the successful local disk build outcome.
type Result struct {
	Qcow2Path string
	Bytes     int64
}

// Build produces a local qcow2 for cfg.
func (b *Builder) Build(ctx context.Context, cfg config.Config) (Result, error) {
	if b.Docker == nil {
		return Result{}, fmt.Errorf("docker is not available: docker runner is nil")
	}

	workDir := cfg.WorkDir
	if workDir == "" {
		workDir = ".proxima-work"
	}
	setupPath := filepath.Join(workDir, "setup.sh")
	imagePath := filepath.Join(workDir, "image.qcow2")
	output := cfg.OutputQcow2
	if output == "" {
		output = "alpine-ready.qcow2"
	}
	outputTouched := false

	cleanup := func() {
		_ = os.Remove(setupPath)
		_ = os.Remove(imagePath)
		if outputTouched {
			_ = os.Remove(output)
		}
	}
	fail := func(format string, args ...any) (Result, error) {
		cleanup()
		return Result{}, fmt.Errorf(format, args...)
	}

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("local disk build failed: %w", err)
	}
	if err := os.Chmod(workDir, 0o755); err != nil {
		return fail("local disk build failed: %w", err)
	}

	if err := script.WriteFile(setupPath, script.Params{RootPassword: cfg.RootPassword}); err != nil {
		return fail("local disk build failed: %w", err)
	}

	var stderr bytes.Buffer
	if err := b.Docker.Run(ctx, []string{"version"}, io.Discard, &stderr); err != nil {
		cleanup()
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return Result{}, fmt.Errorf("docker is not available: %s", detail)
	}

	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return fail("local disk build failed: %w", err)
	}

	diskSize := cfg.DiskSize
	if diskSize == "" {
		diskSize = "4G"
	}
	bootstrap := containerBootstrap(diskSize)

	runArgs := []string{
		"run", "--rm", "--privileged",
		"-v", absWork + ":/work",
		"-w", "/work",
		"alpine:latest",
		"/bin/sh", "-c", bootstrap,
	}
	var runErr bytes.Buffer
	if err := b.Docker.Run(ctx, runArgs, io.Discard, &runErr); err != nil {
		detail := strings.TrimSpace(runErr.String())
		if detail == "" {
			detail = err.Error()
		}
		return fail("local disk build failed: %s", detail)
	}

	if _, err := os.Stat(imagePath); err != nil {
		return fail("local disk build failed: image.qcow2 missing after container exit")
	}

	if dir := filepath.Dir(output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail("local disk build failed: %w", err)
		}
	}

	outputTouched = true
	if err := os.Rename(imagePath, output); err != nil {
		if copyErr := copyFile(imagePath, output); copyErr != nil {
			return fail("local disk build failed: %w", err)
		}
		_ = os.Remove(imagePath)
	}

	_ = os.Remove(setupPath)
	_ = os.Remove(imagePath)

	info, err := os.Stat(output)
	if err != nil {
		return fail("local disk build failed: %w", err)
	}
	absOut, err := filepath.Abs(output)
	if err != nil {
		absOut = output
	}
	return Result{Qcow2Path: absOut, Bytes: info.Size()}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// containerBootstrap installs alpine-make-vm-image and builds /work/image.qcow2.
// Uses virtio/serial defaults suitable for Proxmox (tool defaults).
func containerBootstrap(diskSize string) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("apk add --no-cache bash git wget qemu-img\n")
	b.WriteString("if [ ! -d /work/alpine-make-vm-image ]; then\n")
	b.WriteString("  git clone --depth 1 https://github.com/alpinelinux/alpine-make-vm-image.git /work/alpine-make-vm-image\n")
	b.WriteString("fi\n")
	b.WriteString("/work/alpine-make-vm-image/alpine-make-vm-image \\\n")
	b.WriteString("  --image-format qcow2 \\\n")
	b.WriteString(fmt.Sprintf("  --image-size %s \\\n", diskSize))
	b.WriteString("  --script-chroot /work/setup.sh \\\n")
	b.WriteString("  /work/image.qcow2\n")
	return b.String()
}
