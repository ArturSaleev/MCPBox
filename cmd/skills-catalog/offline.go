// SPDX-License-Identifier: Apache-2.0
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/skills"
	"github.com/PuerkitoBio/goquery"
)

var archiveCache string
var resolveWithGit = true

type parserReport struct {
	GeneratedAt  time.Time         `json:"generated_at"`
	Discovery    string            `json:"discovery"`
	Pages        int               `json:"pages"`
	Repositories int               `json:"repositories"`
	Sources      []string          `json:"sources"`
	Successful   int               `json:"successful_repositories"`
	Skills       int               `json:"skills"`
	Retained     int               `json:"retained_from_previous_catalog"`
	Failures     map[string]string `json:"failures"`
}

func repositoriesFromHTMLDirectory(dir string, origin string) ([]string, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	base, err := urlParseOrigin(origin)
	if err != nil {
		return nil, 0, err
	}
	seen := map[string]bool{}
	pages := 0
	for _, entry := range entries {
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".html") || entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, pages, err
		}
		if !info.Mode().IsRegular() || info.Size() > 8<<20 {
			return nil, pages, errors.New("HTML export must be a regular file up to 8 MiB")
		}
		f, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, pages, err
		}
		doc, err := goquery.NewDocumentFromReader(io.LimitReader(f, 8<<20))
		f.Close()
		if err != nil {
			return nil, pages, err
		}
		if strings.Contains(strings.ToLower(doc.Find("title").Text()), "just a moment") {
			return nil, pages, fmt.Errorf("%s contains a browser verification page", entry.Name())
		}
		repos, _ := parseDirectoryLinks(doc, base)
		for _, repo := range repos {
			seen[repo] = true
		}
		pages++
	}
	result := []string{}
	for repo := range seen {
		result = append(result, repo)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return nil, pages, errors.New("no source repositories in HTML exports")
	}
	return result, pages, nil
}
func resolveGitCommit(ctx context.Context, repository string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", repository, "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.WaitDelay = 2 * time.Second
	raw, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve repository HEAD: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 || len(fields[0]) != 40 {
		return "", errors.New("invalid git HEAD response")
	}
	if _, err = hex.DecodeString(fields[0]); err != nil {
		return "", errors.New("invalid commit hash")
	}
	return strings.ToLower(fields[0]), nil
}
func cachePath(target string) string {
	sum := sha256.Sum256([]byte(target))
	return filepath.Join(archiveCache, hex.EncodeToString(sum[:])+".zip")
}
func readArchiveCache(target string, max int64) ([]byte, bool) {
	if archiveCache == "" {
		return nil, false
	}
	p := cachePath(target)
	f, err := os.Open(p)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > max {
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(raw)) > max {
		return nil, false
	}
	if _, err = zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err != nil {
		return nil, false
	}
	return raw, true
}
func saveArchiveCache(target string, raw []byte) error {
	if archiveCache == "" {
		return nil
	}
	if err := os.MkdirAll(archiveCache, 0700); err != nil {
		return err
	}
	return atomicWrite(cachePath(target), raw)
}
func atomicWrite(target string, raw []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".skills-catalog-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), target)
}
func writeReport(target string, report parserReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(target, append(raw, '\n'))
}
func retainedEntries(target string, failed map[string]string) ([]skills.Entry, error) {
	raw, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var previous skills.Catalog
	if err = json.Unmarshal(raw, &previous); err != nil {
		return nil, err
	}
	if err = skills.ValidateCatalog(previous); err != nil {
		return nil, err
	}
	result := []skills.Entry{}
	for _, entry := range previous.Items {
		if _, failedRepo := failed[entry.Source.Repository]; failedRepo {
			result = append(result, entry)
		}
	}
	return result, nil
}

func urlParseOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return nil, errors.New("source requires an HTTPS URL")
	}
	return u, nil
}
