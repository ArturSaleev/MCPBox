package app

import (
	"path/filepath"
	"testing"
)

func TestNormalizeOptionsAnchorsDefaultStoreNextToExecutable(t *testing.T) {
	options := normalizeOptions(Options{})
	want := filepath.Join(executableDir(), "mcpbox.db")
	if options.StoreDSN != want {
		t.Fatalf("StoreDSN = %q, want %q", options.StoreDSN, want)
	}
}

func TestNormalizeOptionsAnchorsRelativeStoreNextToExecutable(t *testing.T) {
	options := normalizeOptions(Options{StoreDSN: filepath.Join("data", "custom.db")})
	want := filepath.Join(executableDir(), "data", "custom.db")
	if options.StoreDSN != want {
		t.Fatalf("StoreDSN = %q, want %q", options.StoreDSN, want)
	}
}

func TestNormalizeOptionsPreservesAbsoluteStore(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.db")
	options := normalizeOptions(Options{StoreDSN: want})
	if options.StoreDSN != want {
		t.Fatalf("StoreDSN = %q, want %q", options.StoreDSN, want)
	}
}
