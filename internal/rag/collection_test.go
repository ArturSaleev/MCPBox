package rag

import (
	"archive/zip"
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

func TestCollectionSkipsSystemFoldersAndVirtualEnvs(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "indexes", "knowledge.bleve")
	projectDir := filepath.Join(rootDir, "project")

	mustMkdirAll(t, filepath.Join(projectDir, "src"))
	mustMkdirAll(t, filepath.Join(projectDir, "node_modules", "pkg"))
	mustMkdirAll(t, filepath.Join(projectDir, "vendor", "lib"))
	mustMkdirAll(t, filepath.Join(projectDir, "myenv", "bin"))

	mustWriteFile(t, filepath.Join(projectDir, "src", "billing.go"), `package billing

func invoiceRetry() {
	println("invoice retry token")
}
`)
	mustWriteFile(t, filepath.Join(projectDir, "node_modules", "pkg", "skip.js"), `payment gateway hidden token`)
	mustWriteFile(t, filepath.Join(projectDir, "vendor", "lib", "skip.php"), `payment gateway vendor token`)
	mustWriteFile(t, filepath.Join(projectDir, "myenv", "pyvenv.cfg"), `home = /usr/bin/python3`)
	mustWriteFile(t, filepath.Join(projectDir, "myenv", "lib.py"), `payment gateway myenv token`)

	collection, err := NewCollection("crm_gym", "CRM Gym", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	if err := collection.IndexFolder(projectDir); err != nil {
		t.Fatalf("IndexFolder() error = %v", err)
	}

	results, err := collection.Search("invoice retry token", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search() returned no results")
	}
	for _, result := range results {
		if strings.Contains(result.FilePath, "node_modules") || strings.Contains(result.FilePath, "vendor") || strings.Contains(result.FilePath, "myenv") {
			t.Fatalf("system or virtualenv path was indexed: %q", result.FilePath)
		}
	}
}

func TestCollectionIndexesDeepNestedFiles(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "indexes", "nested.bleve")
	projectDir := filepath.Join(rootDir, "project")
	nestedDir := filepath.Join(projectDir, "src", "billing", "handlers")
	mustMkdirAll(t, nestedDir)

	mustWriteFile(t, filepath.Join(nestedDir, "retry.go"), `package handlers

func RetryGatewayPayment() {
	println("deep nested gateway retry")
}
`)

	collection, err := NewCollection("nested", "Nested", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	if err := collection.IndexFolder(projectDir); err != nil {
		t.Fatalf("IndexFolder() error = %v", err)
	}

	results, err := collection.Search("deep nested gateway retry", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search() returned no results")
	}
	if !strings.Contains(results[0].FilePath, filepath.Join("src", "billing", "handlers", "retry.go")) {
		t.Fatalf("top result path = %q, want nested file path", results[0].FilePath)
	}
}

func TestCollectionSkipsFilesWithoutExtractableText(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "indexes", "skip-empty.bleve")
	projectDir := filepath.Join(rootDir, "project")
	mustMkdirAll(t, filepath.Join(projectDir, "docs"))

	mustWriteFile(t, filepath.Join(projectDir, "docs", "Guidelines.md"), "")
	mustWriteFile(t, filepath.Join(projectDir, "docs", "billing.md"), `Payment rules for billing retry`)

	collection, err := NewCollection("skip-empty", "Skip Empty", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	if err := collection.IndexFolder(projectDir); err != nil {
		t.Fatalf("IndexFolder() error = %v", err)
	}

	assertSearchContainsFile(t, collection, "billing retry", "billing.md")
	results, err := collection.Search("Guidelines", 5)
	if err != nil {
		t.Fatalf("Search(Guidelines) error = %v", err)
	}
	for _, result := range results {
		if strings.HasSuffix(result.FilePath, "Guidelines.md") {
			t.Fatalf("empty file was indexed: %q", result.FilePath)
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

func TestCollectionIndexesOfficeAndTabularDocuments(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "indexes", "documents.bleve")
	docsDir := filepath.Join(rootDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(docsDir, "members.csv"), []byte("name,plan\nIvan,Premium\n"), 0o644); err != nil {
		t.Fatalf("write csv error = %v", err)
	}
	if err := writeZipFile(filepath.Join(docsDir, "report.xlsx"), map[string]string{
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Plans" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8"?>
<sst>
  <si><t>Plan</t></si>
  <si><t>Premium</t></si>
  <si><t>Owner</t></si>
  <si><t>Ivan</t></si>
</sst>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<worksheet>
  <sheetData>
    <row r="1"><c t="s"><v>0</v></c><c t="s"><v>1</v></c></row>
    <row r="2"><c t="s"><v>2</v></c><c t="s"><v>3</v></c></row>
  </sheetData>
</worksheet>`,
	}); err != nil {
		t.Fatalf("write xlsx error = %v", err)
	}
	if err := writeZipFile(filepath.Join(docsDir, "contract.docx"), map[string]string{
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Membership contract for premium plan</w:t></w:r></w:p>
  </w:body>
</w:document>`,
	}); err != nil {
		t.Fatalf("write docx error = %v", err)
	}
	if err := writeZipFile(filepath.Join(docsDir, "pitch.pptx"), map[string]string{
		"ppt/slides/slide1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Sales deck premium onboarding</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`,
	}); err != nil {
		t.Fatalf("write pptx error = %v", err)
	}

	collection, err := NewCollection("documents", "Documents", indexPath)
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}
	defer func() { _ = collection.Close() }()

	if err := collection.IndexFolder(docsDir); err != nil {
		t.Fatalf("IndexFolder() error = %v", err)
	}

	assertSearchContainsFile(t, collection, "Ivan Premium", "members.csv")
	assertSearchContainsFile(t, collection, "Plan Premium", "report.xlsx")
	assertSearchSection(t, collection, "Plan Premium", "Sheet: Plans")
	assertSearchContainsFile(t, collection, "membership contract premium", "contract.docx")
	assertSearchContainsFile(t, collection, "sales deck onboarding", "pitch.pptx")
	assertSearchSection(t, collection, "sales deck onboarding", "Slide: 1")
}

func assertSearchContainsFile(t *testing.T, collection *Collection, query, fileName string) {
	t.Helper()

	results, err := collection.Search(query, 5)
	if err != nil {
		t.Fatalf("Search(%q) error = %v", query, err)
	}
	if len(results) == 0 {
		t.Fatalf("Search(%q) returned no results", query)
	}

	for _, result := range results {
		if strings.HasSuffix(result.FilePath, fileName) {
			return
		}
	}

	t.Fatalf("Search(%q) did not return %s; results = %#v", query, fileName, results)
}

func assertSearchSection(t *testing.T, collection *Collection, query, wantSection string) {
	t.Helper()

	results, err := collection.Search(query, 5)
	if err != nil {
		t.Fatalf("Search(%q) error = %v", query, err)
	}
	if len(results) == 0 {
		t.Fatalf("Search(%q) returned no results", query)
	}
	for _, result := range results {
		if result.Section == wantSection {
			return
		}
	}
	t.Fatalf("Search(%q) missing section %q; results = %#v", query, wantSection, results)
}

func writeZipFile(path string, files map[string]string) error {
	handle, err := os.Create(path)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(handle)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			_ = handle.Close()
			return err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			_ = writer.Close()
			_ = handle.Close()
			return err
		}
	}

	if err := writer.Close(); err != nil {
		_ = handle.Close()
		return err
	}
	return handle.Close()
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
