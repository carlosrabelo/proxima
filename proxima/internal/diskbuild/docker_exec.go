package diskbuild

import (
	"context"
	"io"
	"os/exec"
)

// ExecDocker runs the host docker CLI.
type ExecDocker struct{}

// Run executes `docker <args...>`.
func (ExecDocker) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
