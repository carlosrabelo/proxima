package script_test

import (
	"testing"

	"proxima/proxima/internal/script"
)

func TestRenderRequiresPassword(t *testing.T) {
	_, err := script.Render(script.Params{})
	if err == nil {
		t.Fatal("expected error for empty root password")
	}
	if err.Error() != "root_password is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "root_password is required")
	}

	_, err = script.Render(script.Params{RootPassword: "test-root-pass"})
	if err != nil {
		t.Fatalf("Render with password: %v", err)
	}
}
