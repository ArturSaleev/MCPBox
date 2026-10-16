package prohttp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectKnowledgeBaseArchiveFormat(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"docs.zip":    "zip",
		"docs.tar":    "tar",
		"docs.tgz":    "tgz",
		"docs.tar.gz": "tgz",
		"docs.txt":    "",
	}

	for input, expected := range cases {
		if actual := detectKnowledgeBaseArchiveFormat(input); actual != expected {
			t.Fatalf("detectKnowledgeBaseArchiveFormat(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSafeArchiveDestinationRejectsTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := safeArchiveDestination(root, "../secret.txt"); err == nil {
		t.Fatal("expected traversal entry to be rejected")
	}
}

func TestMaybeExpandKnowledgeBaseArchiveZip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "bundle.zip")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}

	writer := zip.NewWriter(file)
	entry, err := writer.Create("docs/readme.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte("hello")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	extractedRoot, files, err := maybeExpandKnowledgeBaseArchive(archivePath, root)
	if err != nil {
		t.Fatalf("maybeExpandKnowledgeBaseArchive(zip): %v", err)
	}
	if extractedRoot == "" {
		t.Fatal("expected extracted root")
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 extracted file, got %d", len(files))
	}
	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected extracted content: %q", string(content))
	}
}

func TestMaybeExpandKnowledgeBaseArchiveTarGz(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "bundle.tar.gz")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	content := []byte("hello tar")
	header := &tar.Header{
		Name: "nested/info.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close tar.gz file: %v", err)
	}

	extractedRoot, files, err := maybeExpandKnowledgeBaseArchive(archivePath, root)
	if err != nil {
		t.Fatalf("maybeExpandKnowledgeBaseArchive(tar.gz): %v", err)
	}
	if extractedRoot == "" {
		t.Fatal("expected extracted root")
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 extracted file, got %d", len(files))
	}
	contentRead, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(contentRead) != string(content) {
		t.Fatalf("unexpected extracted content: %q", string(contentRead))
	}
}
