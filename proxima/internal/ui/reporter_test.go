package ui_test

import (
	"bytes"
	"errors"
	"testing"

	"proxima/proxima/internal/ui"
)

func TestStageLineFormats(t *testing.T) {
	var buf bytes.Buffer
	r := &ui.TextReporter{Out: &buf}

	r.StageStart("Validate")
	r.StageDone("Validate", "ok")
	r.StageFail("BuildLocal", errors.New("boom"))

	want := "==> Validate: starting\n==> Validate: done (ok)\n==> BuildLocal: failed: boom\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}
