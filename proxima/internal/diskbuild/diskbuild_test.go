package diskbuild_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/diskbuild"
)

type fakeDocker struct {
	err error
}

func (f *fakeDocker) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
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
	_, err := b.Build(context.Background(), config.Config{})
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
