package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
)

func TestVersionCommandOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got, want := stdout.String(), "proxima 0.1.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRootHelpExitZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "proxima") {
		t.Fatalf("help output missing binary name proxima: %q", out)
	}
	if !strings.Contains(out, "version") {
		t.Fatalf("help output missing version subcommand: %q", out)
	}
}
