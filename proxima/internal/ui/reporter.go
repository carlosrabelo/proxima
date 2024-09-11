package ui

import (
	"fmt"
	"io"
)

// Reporter emits stage status lines.
type Reporter interface {
	StageStart(stage string)
	StageDone(stage string, detail string)
	StageFail(stage string, err error)
}

// TextReporter writes ==> stage lines to Out (typically stderr).
type TextReporter struct {
	Out io.Writer
}

// StageStart prints the starting line for a stage.
func (r *TextReporter) StageStart(stage string) {
	fmt.Fprintf(r.Out, "==> %s: starting\n", stage)
}

// StageDone prints the done line for a stage.
func (r *TextReporter) StageDone(stage string, detail string) {
	fmt.Fprintf(r.Out, "==> %s: done (%s)\n", stage, detail)
}

// StageFail prints the failed line for a stage.
func (r *TextReporter) StageFail(stage string, err error) {
	fmt.Fprintf(r.Out, "==> %s: failed: %v\n", stage, err)
}
