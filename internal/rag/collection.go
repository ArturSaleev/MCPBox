package rag

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

const (
	defaultSearchLimit = 10
	targetChunkLines   = 80
	chunkLineOverlap   = 12
	maxChunkChars      = 2000
	charChunkOverlap   = 250
)

var supportedExtensions = map[string]struct{}{
	".go":  {},
	".js":  {},
	".md":  {},
	".php": {},
	".txt": {},
}

// Chunk is the smallest retrievable unit in a collection.
// Each chunk points back to its source file and stores the searchable content.
type Chunk struct {
	ID       string `json:"id"`
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// Collection represents one independent knowledge base with its own Bleve index.
// Different collections can safely point to different index paths on disk.
type Collection struct {
	ID        string
	Name      string
	IndexPath string
	index     bleve.Index
}

// NewCollection opens an existing Bleve index or creates a new one if it does not exist yet.
func NewCollection(id, name, indexPath string) (*Collection, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	indexPath = strings.TrimSpace(indexPath)

	if id == "" {
		return nil, errors.New("collection id is required")
	}
	if name == "" {
		return nil, errors.New("collection name is required")
	}
	if indexPath == "" {
		return nil, errors.New("index path is required")
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, fmt.Errorf("create index parent directory: %w", err)
	}

	index, err := openOrCreateIndex(indexPath)
	if err != nil {
		return nil, err
	}

	return &Collection{
		ID:        id,
		Name:      name,
		IndexPath: indexPath,
		index:     index,
	}, nil
}

// Close releases the underlying Bleve index handle.
func (c *Collection) Close() error {
	if c == nil || c.index == nil {
		return nil
	}
	err := c.index.Close()
	c.index = nil
	return err
}

// IndexFolder walks through the provided directory and indexes supported text/code files.
// Files are split into overlapping chunks to preserve surrounding context for search results.
func (c *Collection) IndexFolder(dirPath string) error {
	if c == nil || c.index == nil {
		return errors.New("collection index is not initialized")
	}

	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return errors.New("directory path is required")
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	batch := c.index.NewBatch()

	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}
		if !isSupportedFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		chunks := chunkText(string(content))
		for i, chunkContent := range chunks {
			chunk := Chunk{
				ID:       chunkID(path, i),
				FilePath: filepath.Clean(path),
				Content:  chunkContent,
			}
			if err := batch.Index(chunk.ID, chunk); err != nil {
				return fmt.Errorf("index chunk %s: %w", chunk.ID, err)
			}
			if batch.Size() >= 100 {
				if err := c.index.Batch(batch); err != nil {
					return fmt.Errorf("flush index batch: %w", err)
				}
				batch = c.index.NewBatch()
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if batch.Size() == 0 {
		return nil
	}
	if err := c.index.Batch(batch); err != nil {
		return fmt.Errorf("final index batch flush: %w", err)
	}

	return nil
}

// Search executes a keyword query within this collection and returns the most relevant chunks.
func (c *Collection) Search(query string, limit int) ([]Chunk, error) {
	if c == nil || c.index == nil {
		return nil, errors.New("collection index is not initialized")
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	matchQuery := bleve.NewMatchQuery(query)
	matchQuery.SetField("content")

	request := bleve.NewSearchRequestOptions(matchQuery, limit, 0, false)
	request.Fields = []string{"id", "file_path", "content"}

	result, err := c.index.Search(request)
	if err != nil {
		return nil, fmt.Errorf("search collection: %w", err)
	}

	chunks := make([]Chunk, 0, len(result.Hits))
	for _, hit := range result.Hits {
		chunks = append(chunks, Chunk{
			ID:       stringField(hit.Fields, "id", hit.ID),
			FilePath: stringField(hit.Fields, "file_path", ""),
			Content:  stringField(hit.Fields, "content", ""),
		})
	}

	return chunks, nil
}

func openOrCreateIndex(indexPath string) (bleve.Index, error) {
	index, err := bleve.Open(indexPath)
	if err == nil {
		return index, nil
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) && !strings.Contains(strings.ToLower(err.Error()), "cannot open index") {
		return nil, fmt.Errorf("open bleve index: %w", err)
	}

	index, err = bleve.New(indexPath, newIndexMapping())
	if err != nil {
		return nil, fmt.Errorf("create bleve index: %w", err)
	}
	return index, nil
}

func newIndexMapping() *mapping.IndexMappingImpl {
	indexMapping := bleve.NewIndexMapping()

	docMapping := bleve.NewDocumentMapping()

	idField := bleve.NewTextFieldMapping()
	idField.Store = true
	idField.Index = false

	filePathField := bleve.NewTextFieldMapping()
	filePathField.Store = true

	contentField := bleve.NewTextFieldMapping()
	contentField.Store = true

	docMapping.AddFieldMappingsAt("id", idField)
	docMapping.AddFieldMappingsAt("file_path", filePathField)
	docMapping.AddFieldMappingsAt("content", contentField)

	indexMapping.DefaultMapping = docMapping
	indexMapping.TypeField = "type"

	return indexMapping
}

func isSupportedFile(path string) bool {
	_, ok := supportedExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func chunkID(filePath string, index int) string {
	return fmt.Sprintf("%s#%06d", filepath.Clean(filePath), index)
}

func chunkText(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	chunks := make([]string, 0)

	for start := 0; start < len(lines); {
		end := start
		charCount := 0

		for end < len(lines) && end-start < targetChunkLines {
			nextLine := lines[end]
			additional := len(nextLine)
			if end > start {
				additional++
			}
			if end > start && charCount+additional > maxChunkChars {
				break
			}
			charCount += additional
			end++
		}

		if end == start {
			end++
		}

		chunk := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		if chunk != "" {
			chunks = append(chunks, splitOversizedChunk(chunk)...)
		}

		if end >= len(lines) {
			break
		}

		nextStart := end - chunkLineOverlap
		if nextStart <= start {
			nextStart = start + 1
		}
		start = nextStart
	}

	return chunks
}

func splitOversizedChunk(chunk string) []string {
	if len(chunk) <= maxChunkChars {
		return []string{chunk}
	}

	result := make([]string, 0)
	for start := 0; start < len(chunk); {
		end := min(start+maxChunkChars, len(chunk))
		part := strings.TrimSpace(chunk[start:end])
		if part != "" {
			result = append(result, part)
		}
		if end >= len(chunk) {
			break
		}

		nextStart := end - charChunkOverlap
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return result
}

func stringField(fields map[string]any, key, fallback string) string {
	value, ok := fields[key]
	if !ok {
		return fallback
	}
	str, ok := value.(string)
	if !ok {
		return fallback
	}
	return str
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
