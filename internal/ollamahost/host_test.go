package ollamahost

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHeaderUsesMCPBoxBrand(t *testing.T) {
	header := renderHeader()

	if !strings.Contains(header, "Ollama Chat") {
		t.Fatalf("header does not identify Ollama chat: %q", header)
	}
	for _, fragment := range []string{" __  __  ____ ____  ____", "|_|  |_|\\____|_|"} {
		if !strings.Contains(header, fragment) {
			t.Fatalf("header does not contain MCPBox logo fragment %q: %q", fragment, header)
		}
	}
}

func TestBufferIsNotInteractiveTerminal(t *testing.T) {
	if isInteractiveTerminal(&bytes.Buffer{}) {
		t.Fatal("buffer must not be treated as an interactive terminal")
	}
}
