package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

const CatalogURL = "https://mcpbox.sh/cli.json"
const maxCatalogBytes = 1024 * 1024

//go:embed cli.json
var bundledCatalog []byte

type CatalogEntry struct {
	Slug           string                `json:"slug"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	Binary         string                `json:"binary"`
	Platforms      []string              `json:"platforms"`
	Operations     []models.CLIOperation `json:"operations"`
	TimeoutSeconds int                   `json:"timeout_seconds"`
	MaxOutputBytes int                   `json:"max_output_bytes"`
}

func (e CatalogEntry) CLI() models.CLI {
	return models.CLI{Slug: e.Slug, Name: e.Name, Description: e.Description, Binary: e.Binary, IsEnabled: true, Operations: e.Operations, TimeoutSeconds: e.TimeoutSeconds, MaxOutputBytes: e.MaxOutputBytes}
}

type Catalog struct {
	SchemaVersion int            `json:"schema_version"`
	Version       string         `json:"version"`
	Items         []CatalogEntry `json:"items"`
}
type CatalogStore interface {
	ListCLIs(context.Context) ([]models.CLI, error)
	CLICatalogDismissals(context.Context) (map[string]bool, error)
	ImportCatalogCLI(context.Context, *models.CLI) (bool, error)
	SaveCLI(context.Context, *models.CLI) error
}
type CatalogService struct {
	Store    CatalogStore
	URL      string
	Client   *http.Client
	OS       string
	LookPath func(string) (string, error)
}
type CatalogItemStatus struct {
	Entry      CatalogEntry `json:"entry"`
	Status     string       `json:"status"`
	BinaryPath string       `json:"binary_path,omitempty"`
	Current    *models.CLI  `json:"current,omitempty"`
	Revision   string       `json:"revision"`
}
type CatalogStatus struct {
	URL     string              `json:"url"`
	Source  string              `json:"source"`
	Version string              `json:"version"`
	Warning string              `json:"warning,omitempty"`
	Items   []CatalogItemStatus `json:"items"`
	Added   []string            `json:"added"`
}

func DecodeCatalog(data []byte) (Catalog, error) {
	var c Catalog
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, errors.New("catalog has trailing JSON")
	}
	if c.SchemaVersion != 1 || strings.TrimSpace(c.Version) == "" || len(c.Items) > 200 {
		return c, errors.New("unsupported or invalid CLI catalog")
	}
	seen := map[string]bool{}
	for i, e := range c.Items {
		item := e.CLI()
		if err := Validate(&item); err != nil {
			return c, fmt.Errorf("catalog %s: %w", e.Slug, err)
		}
		if seen[item.Slug] {
			return c, errors.New("duplicate CLI ID")
		}
		seen[item.Slug] = true
		if len(e.Platforms) == 0 {
			return c, errors.New("platforms are required")
		}
		for _, os := range e.Platforms {
			if os != "linux" && os != "darwin" && os != "windows" {
				return c, errors.New("unsupported platform")
			}
		}
		// Remote catalogs use portable PATH names only; never machine-specific paths.
		if strings.ContainsAny(item.Binary, "/\\") {
			return c, errors.New("catalog binary must be a PATH name")
		}
		e.Slug, e.Name, e.Binary, e.Operations, e.TimeoutSeconds, e.MaxOutputBytes = item.Slug, item.Name, item.Binary, item.Operations, item.TimeoutSeconds, item.MaxOutputBytes
		c.Items[i] = e
	}
	return c, nil
}
func (s CatalogService) load(ctx context.Context) (Catalog, string, string) {
	url := s.URL
	if url == "" {
		url = CatalogURL
	}
	c, err := s.fetch(ctx, url)
	if err == nil {
		return c, "remote", ""
	}
	fallback, _ := DecodeCatalog(bundledCatalog)
	return fallback, "bundled", err.Error()
}
func (s CatalogService) fetch(ctx context.Context, url string) (Catalog, error) {
	var empty Catalog
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return empty, err
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("CLI catalog HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxCatalogBytes+1))
	if err != nil {
		return empty, err
	}
	if len(data) > maxCatalogBytes {
		return empty, errors.New("CLI catalog exceeds 1 MiB")
	}
	return DecodeCatalog(data)
}
func revision(e CatalogEntry) string {
	data, _ := json.Marshal(e)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func (s CatalogService) Inspect(ctx context.Context) (*CatalogStatus, error) {
	c, source, warning := s.load(ctx)
	current, err := s.Store.ListCLIs(ctx)
	if err != nil {
		return nil, err
	}
	dismissed, err := s.Store.CLICatalogDismissals(ctx)
	if err != nil {
		return nil, err
	}
	bySlug := map[string]models.CLI{}
	for _, item := range current {
		bySlug[item.Slug] = item
	}
	osName := s.OS
	if osName == "" {
		osName = runtime.GOOS
	}
	lookup := s.LookPath
	if lookup == nil {
		lookup = exec.LookPath
	}
	url := s.URL
	if url == "" {
		url = CatalogURL
	}
	result := &CatalogStatus{URL: url, Source: source, Warning: warning, Version: c.Version, Items: make([]CatalogItemStatus, 0, len(c.Items)), Added: []string{}}
	for _, e := range c.Items {
		item := CatalogItemStatus{Entry: e, Revision: revision(e), Status: "available"}
		if existing, ok := bySlug[e.Slug]; ok {
			item.Current = &existing
		}
		switch {
		case !slices.Contains(e.Platforms, osName):
			item.Status = "unsupported_os"
		default:
			path, err := lookup(e.Binary)
			if err != nil {
				item.Status = "binary_missing"
			} else {
				item.BinaryPath = path
				if item.Current != nil {
					proposed := e.CLI()
					existing := *item.Current
					existing.ID = 0
					existing.CreatedAt = proposed.CreatedAt
					existing.UpdatedAt = proposed.UpdatedAt
					existing.IsEnabled = true
					a, _ := json.Marshal(existing)
					b, _ := json.Marshal(proposed)
					if bytes.Equal(a, b) {
						item.Status = "installed"
					} else {
						item.Status = "update_available"
					}
				} else if dismissed[e.Slug] {
					item.Status = "dismissed"
				}
			}
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}
func (s CatalogService) Sync(ctx context.Context) (*CatalogStatus, error) {
	result, err := s.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	for i, item := range result.Items {
		if item.Status != "available" {
			continue
		}
		entry := item.Entry.CLI()
		added, err := s.Store.ImportCatalogCLI(ctx, &entry)
		if err != nil {
			return nil, err
		}
		if added {
			result.Added = append(result.Added, entry.Slug)
			result.Items[i].Status = "installed"
			result.Items[i].Current = &entry
		}
	}
	return result, nil
}
func (s CatalogService) Apply(ctx context.Context, slug, expectedRevision string) error {
	result, err := s.Inspect(ctx)
	if err != nil {
		return err
	}
	for _, item := range result.Items {
		if item.Entry.Slug != slug {
			continue
		}
		if item.Revision != expectedRevision {
			return errors.New("catalog changed; refresh and review again")
		}
		if item.Status == "binary_missing" || item.Status == "unsupported_os" {
			return errors.New("CLI binary is unavailable on this server")
		}
		entry := item.Entry.CLI()
		if item.Current != nil {
			entry.ID = item.Current.ID
			entry.IsEnabled = item.Current.IsEnabled
		}
		return s.Store.SaveCLI(ctx, &entry)
	}
	return errors.New("CLI catalog entry not found")
}
