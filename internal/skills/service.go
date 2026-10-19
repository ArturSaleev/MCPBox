// SPDX-License-Identifier: Apache-2.0
package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed catalog_seed.json
var bundledCatalog []byte

type Service struct {
	db     *gorm.DB
	root   string
	mu     sync.Mutex // coordinates package deletion, bindings, profile edits and job claims
	client *http.Client
}

func New(db *gorm.DB, root string) *Service {
	return &Service{db: db, root: root, client: &http.Client{Timeout: 90 * time.Second}}
}
func (s *Service) Root() string { return s.root }

func (s *Service) Catalog(ctx context.Context) (CatalogCache, error) {
	var c CatalogCache
	err := s.db.WithContext(ctx).First(&c, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var seed Catalog
		if err := json.Unmarshal(bundledCatalog, &seed); err != nil {
			return CatalogCache{}, err
		}
		return CatalogCache{URL: DefaultCatalogURL, Content: seed}, nil
	}
	return c, err
}
func ValidateCatalog(c Catalog) error {
	if c.SchemaVersion != 1 {
		return errors.New("unsupported Skills catalog schema_version")
	}
	if len(c.Items) > 50000 {
		return errors.New("too many catalog entries")
	}
	seen := map[string]bool{}
	for _, e := range c.Items {
		if e.ID == "" || len(e.ID) > 512 || seen[e.ID] || e.Name == "" {
			return errors.New("missing or duplicate catalog id/name")
		}
		seen[e.ID] = true
		if _, _, err := RepositoryParts(e.Source.Repository); err != nil {
			return fmt.Errorf("%s: %w", e.ID, err)
		}
		if e.Source.Path != "" && !validPath(e.Source.Path) {
			return fmt.Errorf("%s: invalid source path", e.ID)
		}
	}
	return nil
}
func (s *Service) ImportCatalog(ctx context.Context, raw []byte, source string) (CatalogCache, error) {
	var c Catalog
	if len(raw) > 16<<20 {
		return CatalogCache{}, errors.New("catalog exceeds 16 MiB")
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return CatalogCache{}, err
	}
	if err := ValidateCatalog(c); err != nil {
		return CatalogCache{}, err
	}
	cache := CatalogCache{ID: 1, URL: source, Content: c, UpdatedAt: time.Now().UTC()}
	err := s.db.WithContext(ctx).Save(&cache).Error
	return cache, err
}
func (s *Service) SyncCatalog(ctx context.Context, source string) (CatalogCache, error) {
	if source == "" {
		source = DefaultCatalogURL
	}
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return CatalogCache{}, errors.New("catalog requires an HTTPS URL")
	}
	raw, err := s.fetch(ctx, source, 16<<20)
	if err != nil {
		return CatalogCache{}, err
	}
	return s.ImportCatalog(ctx, raw, source)
}
func (s *Service) fetch(ctx context.Context, target string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MCPBox-Skills/1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("download exceeds size limit")
	}
	return data, nil
}

var repoPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var commitPattern = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

func RepositoryParts(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", errors.New("source must be a public https://github.com/owner/repository URL")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return "", "", errors.New("source must identify a repository; use source.path for a subdirectory")
	}
	owner, repo := parts[0], strings.TrimSuffix(parts[1], ".git")
	if !repoPart.MatchString(owner) || !repoPart.MatchString(repo) || owner == "." || owner == ".." || repo == "." || repo == ".." {
		return "", "", errors.New("invalid repository")
	}
	return owner, repo, nil
}
func (s *Service) Install(ctx context.Context, entry Entry) (*Skill, error) {
	owner, repo, err := RepositoryParts(entry.Source.Repository)
	if err != nil {
		return nil, err
	}
	if entry.Source.Path != "" && !validPath(entry.Source.Path) {
		return nil, errors.New("invalid package path")
	}
	commit := entry.Source.Commit
	if commit == "" {
		ref := entry.Source.Ref
		if ref == "" {
			ref = "HEAD"
		}
		data, e := s.fetch(ctx, "https://api.github.com/repos/"+owner+"/"+repo+"/commits/"+url.PathEscape(ref), 1<<20)
		if e != nil {
			return nil, e
		}
		var v struct {
			SHA string `json:"sha"`
		}
		if e = json.Unmarshal(data, &v); e != nil {
			return nil, e
		}
		commit = v.SHA
	}
	if !commitPattern.MatchString(commit) {
		return nil, errors.New("invalid commit SHA")
	}
	entry.Source.Commit = strings.ToLower(commit)
	raw, err := s.fetch(ctx, "https://codeload.github.com/"+owner+"/"+repo+"/zip/"+entry.Source.Commit, MaxPackageBytes)
	if err != nil {
		return nil, err
	}
	return s.installZIP(ctx, raw, entry, true)
}
func (s *Service) ImportZIP(ctx context.Context, raw []byte, prefix string) (*Skill, error) {
	return s.installZIP(ctx, raw, Entry{Source: Source{Path: prefix}}, false)
}
func (s *Service) installZIP(ctx context.Context, raw []byte, e Entry, repository bool) (*Skill, error) {
	if int64(len(raw)) > MaxPackageBytes {
		return nil, errors.New("archive exceeds size limit")
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(s.root, "packages"), 0700); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(s.root, ".import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	if err = extractPackage(z, stage, e.Source.Path, repository); err != nil {
		return nil, err
	}
	if repository {
		// Preserve repository-level notices even when installing a nested Skill.
		for _, file := range z.File {
			_, rel, ok := strings.Cut(file.Name, "/")
			if !ok {
				continue
			}
			switch strings.ToLower(rel) {
			case "license", "license.txt", "license.md", "copying", "notice", "notice.md":
			default:
				continue
			}
			if file.UncompressedSize64 > 1<<20 || !file.Mode().IsRegular() {
				return nil, errors.New("invalid repository license file")
			}
			r, err := file.Open()
			if err != nil {
				return nil, err
			}
			notice, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
			r.Close()
			if err != nil {
				return nil, err
			}
			if len(notice) > 1<<20 {
				return nil, errors.New("repository notice too large")
			}
			noticeDir := filepath.Join(stage, ".mcpbox-origin")
			if err = os.MkdirAll(noticeDir, 0700); err != nil {
				return nil, err
			}
			f, err := os.OpenFile(filepath.Join(noticeDir, filepath.Base(rel)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			_, err = f.Write(notice)
			closeErr := f.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	}

	name, desc, hash, files, requirements, findings, err := inspectPackage(stage)
	if err != nil {
		return nil, err
	}
	identity := sha256.Sum256([]byte(e.Source.Repository + "\x00" + e.Source.Path + "\x00" + e.Source.Commit + "\x00" + hash))
	item := Skill{ID: hex.EncodeToString(identity[:]), CatalogID: e.ID, Name: name, Description: desc, Author: e.Author, License: e.License, Source: e.Source, Hash: hash, Files: files, Requirements: append(e.Requirements, requirements...), Findings: findings, Profile: Profile{Scripts: []string{}, TimeoutSeconds: 60}, CreatedAt: time.Now().UTC()}
	// Make the package readable by the unprivileged container user, never executable.
	for _, file := range files {
		if err = os.Chmod(filepath.Join(stage, filepath.FromSlash(file.Path)), 0444); err != nil {
			return nil, err
		}
	}
	if err = filepath.WalkDir(stage, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(p, 0755)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing Skill
	err = s.db.WithContext(ctx).First(&existing, "id = ?", item.ID).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	dest := s.packageRoot(item.ID)
	if _, err = os.Stat(dest); err == nil {
		return nil, errors.New("package files exist without an index entry; inspect before importing")
	}
	if err = os.Rename(stage, dest); err != nil {
		return nil, err
	}
	if err = s.db.WithContext(ctx).Create(&item).Error; err != nil {
		os.RemoveAll(dest)
		return nil, err
	}
	return &item, nil
}
func validID(id string) bool {
	return len(id) == 64 && regexp.MustCompile(`^[a-f0-9]+$`).MatchString(id)
}
func (s *Service) packageRoot(id string) string { return filepath.Join(s.root, "packages", id) }
func (s *Service) List(ctx context.Context, project uint) ([]Skill, error) {
	items := []Skill{}
	q := s.db.WithContext(ctx).Model(&Skill{})
	if project != 0 {
		q = q.Joins("JOIN project_skills ON project_skills.skill_id = skills.id").Where("project_skills.project_id = ?", project)
	}
	if err := q.Order("skills.name, skills.created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.db.WithContext(ctx).Model(&Binding{}).Where("skill_id = ?", items[i].ID).Count(&items[i].ProjectCount).Error; err != nil {
			return nil, err
		}
	}
	return items, nil
}
func (s *Service) Get(ctx context.Context, id string) (*Skill, error) {
	if !validID(id) {
		return nil, errors.New("invalid skill ID")
	}
	var item Skill
	err := s.db.WithContext(ctx).First(&item, "id = ?", id).Error
	return &item, err
}
func (s *Service) Bound(ctx context.Context, project uint, id string) (*Skill, error) {
	if err := s.checkProject(ctx, project); err != nil {
		return nil, err
	}
	var b Binding
	if err := s.db.WithContext(ctx).First(&b, "project_id = ? AND skill_id = ?", project, id).Error; err != nil {
		return nil, errors.New("skill is not connected to this project")
	}
	return s.Get(ctx, id)
}
func (s *Service) checkProject(ctx context.Context, id uint) error {
	var p models.Project
	if id == 0 {
		return errors.New("project is required")
	}
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return err
	}
	if p.IsPaused {
		return errors.New("project is paused")
	}
	return nil
}
func (s *Service) Bind(ctx context.Context, project uint, id string, connect bool) error {
	_, err := s.SetBinding(ctx, project, id, connect)
	return err
}

// SetBinding reports whether the binding changed, so callers can invalidate
// a client's tool catalog without notifying on idempotent requests.
func (s *Service) SetBinding(ctx context.Context, project uint, id string, connect bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkProject(ctx, project); err != nil {
		return false, err
	}
	if _, err := s.Get(ctx, id); err != nil {
		return false, err
	}
	b := Binding{ProjectID: project, SkillID: id}
	if !connect {
		// NewJob and Run use the same mutex: a job cannot be created or
		// started between this check and removing its project binding.
		var count int64
		if err := s.db.WithContext(ctx).Model(&Job{}).
			Where("project_id = ? AND skill_id = ?", project, id).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return false, errors.New("delete this skill's jobs in this project before disconnecting it; wait for running jobs to finish")
		}
		result := s.db.WithContext(ctx).Delete(&b)
		return result.RowsAffected > 0, result.Error
	}
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&b)
	return result.RowsAffected > 0, result.Error
}
func (s *Service) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&Binding{}).Where("skill_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("disconnect skill from all projects before deleting it")
	}
	if err := s.db.WithContext(ctx).Model(&Job{}).Where("skill_id = ? AND status IN ?", id, []string{"running", "preparing"}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("skill has a running job")
	}
	// Rename before deleting the row, allowing rollback on a database error.
	old := s.packageRoot(id)
	trash := filepath.Join(s.root, ".deleted-"+uuid.NewString())
	if err := os.Rename(old, trash); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&Skill{}, "id = ?", id).Error; err != nil {
		_ = os.Rename(trash, old)
		return err
	}
	return os.RemoveAll(trash)
}
func (s *Service) ReadPackageFile(ctx context.Context, id, rel string) ([]byte, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	f, err := openRegular(s.packageRoot(id), rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
}
