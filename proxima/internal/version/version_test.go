package version_test

import (
	"testing"

	"proxima/proxima/internal/version"
)

func TestVersionConstant(t *testing.T) {
	if version.Version != "0.1.0" {
		t.Fatalf("version.Version = %q, want %q", version.Version, "0.1.0")
	}
}
