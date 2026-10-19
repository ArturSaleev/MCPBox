// SPDX-License-Identifier: Apache-2.0
// skills-catalog builds a static catalog. It never installs or executes Skills.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/skills"
	"github.com/PuerkitoBio/goquery"
	"gopkg.in/yaml.v3"
)

var client = &http.Client{Timeout: 60 * time.Second}

func get(ctx context.Context, target string, max int64) ([]byte, error) {
	isArchive := strings.HasPrefix(target, "https://codeload.github.com/")
	if isArchive {
		if raw, ok := readArchiveCache(target, max); ok {
			return raw, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MCPBox-Skills-Catalog/1.0 (+https://mcpbox.sh)")
	if strings.HasPrefix(target, "https://api.github.com/") && os.Getenv("GITHUB_TOKEN") != "" {
		req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", target, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("response exceeds limit")
	}
	if isArchive {
		if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
			return nil, errors.New("source returned an invalid ZIP archive")
		}
		if err := saveArchiveCache(target, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}
func discover(ctx context.Context, source string, maxPages int) ([]string, error) {
	origin, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	if origin.Scheme != "https" {
		return nil, errors.New("source requires HTTPS")
	}
	queue := []string{source}
	seen := map[string]bool{}
	repos := map[string]bool{}
	for len(queue) > 0 && len(seen) < maxPages {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		data, err := get(ctx, current, 8<<20)
		if err != nil {
			return nil, err
		}
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(doc.Find("title").Text()), "just a moment") {
			return nil, errors.New("source requires browser verification; use an approved HTML export or --repositories; existing catalog was not changed")
		}
		discovered, links := parseDirectoryLinks(doc, origin)
		for _, repo := range discovered {
			repos[repo] = true
		}
		for _, link := range links {
			if !seen[link] {
				queue = append(queue, link)
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	if len(queue) > 0 {
		return nil, errors.New("max-pages reached; increase limit to avoid publishing an incomplete catalog")
	}
	result := []string{}
	for repo := range repos {
		result = append(result, repo)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return nil, errors.New("no GitHub Skill repositories found; existing catalog was not changed")
	}
	return result, nil
}
func parseDirectoryLinks(doc *goquery.Document, origin *url.URL) ([]string, []string) {
	repos, links := []string{}, []string{}
	installRepo := regexp.MustCompile(`npx\s+skills\s+add\s+(https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)`).FindStringSubmatch(doc.Find("code, pre").Text())
	if len(installRepo) > 1 {
		repos = append(repos, strings.TrimSuffix(installRepo[1], ".git"))
	}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		u = origin.ResolveReference(u)
		u.Fragment = ""
		if u.Host == "github.com" && len(installRepo) == 0 && strings.HasPrefix(strings.TrimSpace(s.Text()), "GitHub") {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) >= 2 {
				repo := "https://github.com/" + parts[0] + "/" + parts[1]
				if _, _, err := skills.RepositoryParts(repo); err == nil {
					repos = append(repos, repo)
				}
			}
		}
		if u.Host == origin.Host && (strings.HasPrefix(u.Path, strings.TrimRight(origin.Path, "/")+"/") || (u.Path == strings.TrimRight(origin.Path, "/") && u.RawQuery != "")) {
			links = append(links, u.String())
		}
	})
	return repos, links
}
func repositoryEntries(ctx context.Context, repository string) ([]skills.Entry, error) {
	owner, repo, err := skills.RepositoryParts(repository)
	if err != nil {
		return nil, err
	}
	var licenseSPDX, commitSHA string
	if resolveWithGit {
		commitSHA, err = resolveGitCommit(ctx, repository)
		if err != nil {
			return nil, err
		}
	} else {
		infoData, err := get(ctx, "https://api.github.com/repos/"+owner+"/"+repo, 1<<20)
		if err != nil {
			return nil, err
		}
		var info struct {
			License *struct {
				SPDX string `json:"spdx_id"`
			} `json:"license"`
		}
		if err = json.Unmarshal(infoData, &info); err != nil {
			return nil, err
		}
		if info.License != nil && info.License.SPDX != "NOASSERTION" {
			licenseSPDX = info.License.SPDX
		}
		commitData, err := get(ctx, "https://api.github.com/repos/"+owner+"/"+repo+"/commits/HEAD", 1<<20)
		if err != nil {
			return nil, err
		}
		var commit struct {
			SHA string `json:"sha"`
		}
		if err = json.Unmarshal(commitData, &commit); err != nil {
			return nil, err
		}
		commitSHA = commit.SHA
		if len(commitSHA) != 40 {
			return nil, errors.New("source returned invalid commit")
		}
	}
	data, err := get(ctx, "https://codeload.github.com/"+owner+"/"+repo+"/zip/"+commitSHA, skills.MaxPackageBytes)
	if err != nil {
		return nil, err
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	entries := []skills.Entry{}
	for _, file := range z.File {
		if path.Base(file.Name) != "SKILL.md" || file.UncompressedSize64 > 1<<20 {
			continue
		}
		_, relative, ok := strings.Cut(file.Name, "/")
		if !ok {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(reader, 1<<20))
		reader.Close()
		if err != nil {
			return nil, err
		}
		normalized := strings.ReplaceAll(strings.TrimPrefix(string(raw), "\ufeff"), "\r\n", "\n")
		if !strings.HasPrefix(normalized, "---\n") {
			continue
		}
		end := strings.Index(normalized[4:], "\n---")
		if end < 0 {
			continue
		}
		var meta struct {
			Name          string `yaml:"name"`
			Description   string `yaml:"description"`
			License       string `yaml:"license"`
			Compatibility string `yaml:"compatibility"`
		}
		if err = yaml.Unmarshal([]byte(normalized[4:4+end]), &meta); err != nil || meta.Name == "" || meta.Description == "" {
			continue
		}
		subdir := path.Dir(relative)
		if subdir == "." {
			subdir = ""
		}
		license := meta.License
		if license == "" {
			license = licenseSPDX
		}
		entry := skills.Entry{ID: strings.ToLower(owner+"/"+repo) + "/" + subdir, Name: meta.Name, Description: meta.Description, Author: owner, License: license, Source: skills.Source{Repository: repository, Path: subdir, Commit: commitSHA}, URL: repository + "/tree/" + commitSHA + "/" + subdir, RequirementsSource: "unknown"}
		if meta.Compatibility != "" {
			entry.Requirements = []string{meta.Compatibility}
			entry.RequirementsSource = "author"
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid Skill packages in %s", repository)
	}
	return entries, nil
}
func run() error {
	source := flag.String("source", "https://mcpservers.org/ru/agent-skills", "Directory discovery URL")
	repositories := flag.String("repositories", "", "Comma-separated GitHub repositories")
	repositoriesFile := flag.String("repositories-file", "", "Text file with one GitHub repository URL per line")
	htmlFile := flag.String("html", "", "Single saved HTML page")
	htmlDir := flag.String("html-dir", "", "Directory of saved HTML pages; reads all .html files")
	output := flag.String("out", "skills.json", "Output JSON, replaced atomically after validation")
	reportPath := flag.String("report", "", "Build report path (default: output + .report.json)")
	maxPages := flag.Int("max-pages", 10000, "Maximum directory pages; fail if discovery is incomplete")
	cache := flag.String("cache-dir", "", "Cache immutable GitHub archives for repeat builds")
	resolver := flag.String("resolve-with", "git", "Resolve commits using git or api (GITHUB_TOKEN optional)")
	keepGoing := flag.Bool("keep-going", false, "Continue after repository failures; retain previous entries for failed repositories")
	flag.Parse()
	if *resolver != "git" && *resolver != "api" {
		return errors.New("resolve-with must be git or api")
	}
	resolveWithGit = *resolver == "git"
	archiveCache = *cache
	if *maxPages < 1 {
		return errors.New("max-pages must be positive")
	}
	modes := 0
	for _, v := range []string{*repositories, *repositoriesFile, *htmlFile, *htmlDir} {
		if v != "" {
			modes++
		}
	}
	if modes > 1 {
		return errors.New("choose one of repositories, repositories-file, html, html-dir")
	}
	if *reportPath == "" {
		*reportPath = *output + ".report.json"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	report := parserReport{GeneratedAt: time.Now().UTC(), Failures: map[string]string{}}
	repos := []string{}
	switch {
	case *repositoriesFile != "":
		report.Discovery = "repository-list"
		raw, err := os.ReadFile(*repositoriesFile)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				repos = append(repos, line)
			}
		}
		if len(repos) == 0 {
			return errors.New("repository list is empty")
		}
	case *repositories != "":
		report.Discovery = "explicit-repositories"
		for _, r := range strings.Split(*repositories, ",") {
			repos = append(repos, strings.TrimSpace(r))
		}
	case *htmlDir != "":
		report.Discovery = "saved-html-pages"
		var err error
		repos, report.Pages, err = repositoriesFromHTMLDirectory(*htmlDir, *source)
		if err != nil {
			return err
		}
	case *htmlFile != "":
		report.Discovery = "saved-html-page"
		report.Pages = 1
		raw, err := os.ReadFile(*htmlFile)
		if err != nil {
			return err
		}
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		origin, err := urlParseOrigin(*source)
		if err != nil {
			return err
		}
		repos, _ = parseDirectoryLinks(doc, origin)
		if len(repos) == 0 {
			return errors.New("no GitHub repositories in HTML export")
		}
	default:
		report.Discovery = "live-directory"
		var err error
		repos, err = discover(ctx, *source, *maxPages)
		if err != nil {
			return err
		}
	}
	// Validate all sources before starting network activity.
	unique := []string{}
	seen := map[string]bool{}
	for _, repo := range repos {
		if _, _, err := skills.RepositoryParts(repo); err != nil {
			return err
		}
		if !seen[repo] {
			seen[repo] = true
			unique = append(unique, repo)
		}
	}
	sort.Strings(unique)
	report.Repositories = len(unique)
	report.Sources = unique
	catalog := skills.Catalog{SchemaVersion: 1, GeneratedAt: report.GeneratedAt, Items: []skills.Entry{}}
	for index, repo := range unique {
		fmt.Fprintf(os.Stderr, "[%d/%d] Indexing %s\n", index+1, len(unique), repo)
		repoCtx, repoCancel := context.WithTimeout(ctx, 3*time.Minute)
		entries, err := repositoryEntries(repoCtx, repo)
		repoCancel()
		if err != nil {
			report.Failures[repo] = err.Error()
			fmt.Fprintf(os.Stderr, "WARNING: %s: %v\n", repo, err)
			if !*keepGoing {
				_ = writeReport(*reportPath, report)
				return fmt.Errorf("catalog was not replaced: %w", err)
			}
			continue
		}
		report.Successful++
		catalog.Items = append(catalog.Items, entries...)
	}
	if len(report.Failures) > 0 && *keepGoing {
		retained, err := retainedEntries(*output, report.Failures)
		if err != nil {
			return fmt.Errorf("cannot preserve previous entries: %w", err)
		}
		report.Retained = len(retained)
		catalog.Items = append(catalog.Items, retained...)
	}
	if len(catalog.Items) == 0 {
		_ = writeReport(*reportPath, report)
		return errors.New("no Skills indexed; previous catalog was not changed")
	}
	sort.Slice(catalog.Items, func(i, j int) bool { return catalog.Items[i].ID < catalog.Items[j].ID })
	if err := skills.ValidateCatalog(catalog); err != nil {
		return err
	}
	report.Skills = len(catalog.Items)
	raw, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	if err = writeReport(*reportPath, report); err != nil {
		return err
	}
	if err = atomicWrite(*output, append(raw, '\n')); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Generated %d Skills from %d repositories (%d failed, %d entries retained) -> %s\n", report.Skills, report.Successful, len(report.Failures), report.Retained, *output)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
