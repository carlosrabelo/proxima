package diskbuild

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"proxima/proxima/internal/config"
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

	var stderr bytes.Buffer
	if err := b.Docker.Run(ctx, []string{"version"}, io.Discard, &stderr); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return Result{}, fmt.Errorf("docker is not available: %s", detail)
	}

	return Result{}, fmt.Errorf("local disk build failed: not implemented")
}
