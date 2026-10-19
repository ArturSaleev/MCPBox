package main

import (
	"encoding/json"
	"github.com/ArturSaleev/MCPBox/internal/skills"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestDiscoverRepositoryAndCatalogPaginationLinks(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<a href="https://github.com/owner/repo/tree/HEAD/skills/one">GitHub</a><a href="/ru/agent-skills/category/research?page=2">Next</a><a href="/ru/agent-skills/owner/name">Skill</a><a href="https://evil.test/ru/agent-skills/x">External</a><a href="/ru/servers">MCP</a>`))
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := url.Parse("https://mcpservers.org/ru/agent-skills")
	repos, links := parseDirectoryLinks(doc, origin)
	if len(repos) != 1 || repos[0] != "https://github.com/owner/repo" {
		t.Fatalf("repos=%v", repos)
	}
	if len(links) != 2 {
		t.Fatalf("pagination/discovery=%v", links)
	}
}

func TestSavedHTMLDirectoryAndAtomicOutput(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"one.html":      `<html><title>Skill</title><code>npx skills add https://github.com/owner/repo --skill one</code></html>`,
		"two.html":      `<a href="https://github.com/owner/repo/tree/HEAD/skills/two">GitHub 12</a><a href="https://github.com/second/skills">GitHub</a>`,
		"manifest.json": `{"pages":2}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repos, pages, err := repositoriesFromHTMLDirectory(dir, "https://mcpservers.org/ru/agent-skills")
	if err != nil || pages != 2 || len(repos) != 2 {
		t.Fatalf("repos=%v pages=%d err=%v", repos, pages, err)
	}
	out := filepath.Join(dir, "public", "skills.json")
	if err = atomicWrite(out, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err = atomicWrite(out, []byte("new")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil || string(raw) != "new" {
		t.Fatal("atomic replacement failed", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatal("catalog must be readable by web server", err)
	}
}
func TestFailedRepositoriesKeepTheirPreviousEntries(t *testing.T) {
	previous := skills.Catalog{SchemaVersion: 1, Items: []skills.Entry{
		{ID: "a", Name: "one", Source: skills.Source{Repository: "https://github.com/one/skills"}},
		{ID: "b", Name: "two", Source: skills.Source{Repository: "https://github.com/two/skills"}},
	}}
	raw, _ := json.Marshal(previous)
	out := filepath.Join(t.TempDir(), "skills.json")
	if err := os.WriteFile(out, raw, 0600); err != nil {
		t.Fatal(err)
	}
	retained, err := retainedEntries(out, map[string]string{"https://github.com/one/skills": "unavailable"})
	if err != nil || len(retained) != 1 || retained[0].ID != "a" {
		t.Fatalf("retained=%v err=%v", retained, err)
	}
}
func TestSavedChallengePageIsRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "challenge.html"), []byte(`<title>Just a moment...</title>`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repositoriesFromHTMLDirectory(dir, "https://mcpservers.org/ru/agent-skills"); err == nil {
		t.Fatal("challenge page treated as catalog")
	}
}
func TestPaginationOnRootAndInstallCommandSource(t *testing.T) {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(`<code>npx skills add https://github.com/real/skills --skill one</code><a href="https://github.com/unrelated/repo">GitHub</a><a href="/ru/agent-skills?page=2">Next</a>`))
	base, _ := url.Parse("https://mcpservers.org/ru/agent-skills")
	repos, links := parseDirectoryLinks(doc, base)
	if len(repos) != 1 || repos[0] != "https://github.com/real/skills" || len(links) != 1 {
		t.Fatalf("repos=%v links=%v", repos, links)
	}
}
