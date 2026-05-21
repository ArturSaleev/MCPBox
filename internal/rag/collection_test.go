package rag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCollectionCreatesAndReopensIndex(t *testing.T) {
	t.Parallel()

	indexPath := filepath.Join(t.TempDir(), "indexes", "kb.bleve")

	collection, err := NewCollection("crm_gym", "CRM Gym", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() create error = %v", err)
	}
	if collection.IndexPath != indexPath {
		t.Fatalf("collection.IndexPath = %q, want %q", collection.IndexPath, indexPath)
	}
	if err := collection.Close(); err != nil {
		t.Fatalf("collection.Close() error = %v", err)
	}

	reopened, err := NewCollection("crm_gym", "CRM Gym", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() reopen error = %v", err)
	}
	defer func() { _ = reopened.Close() }()
}

func TestCollectionIndexFolderAndSearch(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "indexes", "knowledge.bleve")

	projectDir := filepath.Join(rootDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	goFile := filepath.Join(projectDir, "main.go")
	goContent := `package main

// payment gateway retries failed invoices
func retryInvoice() {
	println("retry payment gateway")
}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0o644); err != nil {
		t.Fatalf("os.WriteFile(main.go) error = %v", err)
	}

	mdFile := filepath.Join(projectDir, "docs", "notes.md")
	mdContent := `# Search Notes

The billing pipeline sends payment events to the gateway queue.
`
	if err := os.WriteFile(mdFile, []byte(mdContent), 0o644); err != nil {
		t.Fatalf("os.WriteFile(notes.md) error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "image.png"), []byte("not indexed"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(image.png) error = %v", err)
	}

	collection, err := NewCollection("crm_gym", "CRM Gym", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	if err := collection.IndexFolder(projectDir); err != nil {
		t.Fatalf("IndexFolder() error = %v", err)
	}

	results, err := collection.Search("payment gateway", 5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search() returned no results")
	}

	if results[0].FilePath == "" {
		t.Fatal("Search() result FilePath is empty")
	}
	if !strings.Contains(results[0].Content, "payment") {
		t.Fatalf("Search() top result content = %q, want content containing payment", results[0].Content)
	}

	for _, result := range results {
		if strings.HasSuffix(result.FilePath, ".png") {
			t.Fatalf("unsupported file was indexed: %q", result.FilePath)
		}
	}
}

func TestChunkTextSplitsLongFilesWithOverlap(t *testing.T) {
	t.Parallel()

	lines := make([]string, 0, 220)
	for i := 0; i < 220; i++ {
		lines = append(lines, fmt.Sprintf("line %03d shared-token", i))
	}

	chunks := chunkText(strings.Join(lines, "\n"))
	if len(chunks) < 3 {
		t.Fatalf("len(chunks) = %d, want at least 3", len(chunks))
	}

	if !strings.Contains(chunks[0], "line 079") && !strings.Contains(chunks[0], "line 078") {
		t.Fatalf("first chunk missing expected tail lines: %q", chunks[0])
	}
	if !strings.Contains(chunks[1], "line 068") {
		t.Fatalf("second chunk does not appear to overlap previous chunk: %q", chunks[1])
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	t.Parallel()

	collection, err := NewCollection("docs", "Docs", filepath.Join(t.TempDir(), "docs.bleve"))
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	_, err = collection.Search("   ", 5)
	if err == nil {
		t.Fatal("Search() error = nil, want validation error")
	}
}
