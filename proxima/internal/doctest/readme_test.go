package doctest_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// proxima/internal/doctest → repo root is three levels up from this file's dir...
	// file is .../proxima/internal/doctest/readme_test.go
	dir := filepath.Dir(file)
	return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
}

func TestREADMEMentionsCreate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"proxima create", "BuildLocal", "cfg/proxima.example.yaml"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}
}

func TestREADMEMentionsStartAndIP(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"proxima start", "proxima ip", "WaitIP"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}
}

func TestREADMEPTMentionsCreate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "README-PT.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"proxima create", "BuildLocal", "cfg/proxima.example.yaml"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README-PT.md missing %q", want)
		}
	}
}

func TestREADMEPTMentionsStartAndIP(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "README-PT.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"proxima start", "proxima ip", "WaitIP"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README-PT.md missing %q", want)
		}
	}
}
