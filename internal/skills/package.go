// SPDX-License-Identifier: Apache-2.0
package skills

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func validPath(p string) bool {
	return p != "" && p != "." && fs.ValidPath(p) && !strings.ContainsAny(p, "\\:\x00") && !strings.HasPrefix(p, "/")
}

// listFiles rejects symlinks and special files instead of following them.
func listFiles(root string, limit int64) ([]File, error) {
	files := []File{}
	var total int64
	nodes := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		nodes++
		if nodes > MaxFiles*4 {
			return errors.New("too many filesystem entries")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlinks are not supported")
		}
		if d.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("special files are not supported")
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !validPath(rel) {
			return errors.New("invalid file path")
		}
		total += info.Size()
		if info.Size() > MaxFileBytes || total > limit || len(files) >= MaxFiles {
			return errors.New("file quota exceeded")
		}
		files = append(files, File{rel, info.Size()})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, err
}

// openRegular uses os.Root so paths cannot escape even if a job replaces directories.
func openRegular(root, rel string) (*os.File, error) {
	if !validPath(rel) {
		return nil, errors.New("invalid file path")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	// Reject all symlinks, including those targeting files inside the root.
	parts := strings.Split(rel, "/")
	for i := range parts {
		info, e := r.Lstat(strings.Join(parts[:i+1], "/"))
		if e != nil {
			return nil, e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlinks are not supported")
		}
	}
	f, err := r.Open(rel)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
		f.Close()
		return nil, errors.New("not a supported regular file")
	}
	return f, nil
}

func inspectPackage(root string) (string, string, string, []File, []string, []string, error) {
	files, err := listFiles(root, MaxPackageBytes)
	if err != nil {
		return "", "", "", nil, nil, nil, err
	}
	f, err := openRegular(root, "SKILL.md")
	if err != nil {
		return "", "", "", nil, nil, nil, errors.New("package requires SKILL.md")
	}
	data, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	f.Close()
	if err != nil || len(data) > 1<<20 {
		return "", "", "", nil, nil, nil, errors.New("SKILL.md exceeds 1 MiB")
	}
	normalized := strings.ReplaceAll(strings.TrimPrefix(string(data), "\ufeff"), "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", "", "", nil, nil, nil, errors.New("SKILL.md requires YAML frontmatter")
	}
	end := strings.Index(normalized[4:], "\n---")
	if end < 0 {
		return "", "", "", nil, nil, nil, errors.New("unclosed frontmatter")
	}
	var meta struct {
		Name          string `yaml:"name"`
		Description   string `yaml:"description"`
		Compatibility string `yaml:"compatibility"`
	}
	if err = yaml.Unmarshal([]byte(normalized[4:4+end]), &meta); err != nil {
		return "", "", "", nil, nil, nil, err
	}
	if strings.TrimSpace(meta.Name) == "" || strings.TrimSpace(meta.Description) == "" {
		return "", "", "", nil, nil, nil, errors.New("name and description are required")
	}
	h := sha256.New()
	findings := []string{}
	requirements := []string{}
	if meta.Compatibility != "" {
		requirements = append(requirements, meta.Compatibility)
	}
	for _, file := range files {
		f, err := openRegular(root, file.Path)
		if err != nil {
			return "", "", "", nil, nil, nil, err
		}
		fmt.Fprintf(h, "%d:%s:%d:", len(file.Path), file.Path, file.Size)
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", "", "", nil, nil, nil, err
		}
		ext := strings.ToLower(path.Ext(file.Path))
		switch ext {
		case ".py":
			findings = append(findings, "Python script: "+file.Path)
		case ".js", ".mjs", ".cjs":
			findings = append(findings, "Node.js script: "+file.Path)
		case ".sh":
			findings = append(findings, "Shell script: "+file.Path)
		}
		switch path.Base(file.Path) {
		case "requirements.txt", "pyproject.toml", "package.json", "package-lock.json", "uv.lock":
			requirements = append(requirements, "Dependency manifest: "+file.Path)
		}
	}
	findings = append(findings, "Static inspection does not establish safety. Package code has not been executed.")
	return meta.Name, meta.Description, hex.EncodeToString(h.Sum(nil)), files, requirements, findings, nil
}

// extractPackage extracts a directory from a GitHub ZIP or a portable ZIP upload.
// The repository archive prefix is stripped only for repository downloads.
func extractPackage(z *zip.Reader, dest, subdir string, repository bool) error {
	if subdir != "" && !validPath(subdir) {
		return errors.New("invalid source path")
	}
	if len(z.File) > 100000 {
		return errors.New("too many archive entries")
	}
	var total int64
	count := 0
	for _, item := range z.File {
		name := strings.TrimSuffix(item.Name, "/")
		if !validPath(name) {
			return errors.New("unsafe archive path")
		}
		if repository {
			_, rest, ok := strings.Cut(name, "/")
			if !ok {
				continue
			}
			name = rest
		}
		if subdir != "" {
			if !strings.HasPrefix(name, subdir+"/") {
				continue
			}
			name = strings.TrimPrefix(name, subdir+"/")
		}
		if !validPath(name) {
			return errors.New("unsafe package path")
		}
		if item.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink in archive")
		}
		if item.FileInfo().IsDir() {
			continue
		}
		if !item.Mode().IsRegular() {
			return errors.New("special file in archive")
		}
		total += int64(item.UncompressedSize64)
		count++
		if item.UncompressedSize64 > uint64(MaxFileBytes) || total > MaxPackageBytes || count > MaxFiles {
			return errors.New("package quota exceeded")
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		in, err := item.Open()
		if err != nil {
			out.Close()
			return err
		}
		n, copyErr := io.Copy(out, io.LimitReader(in, MaxFileBytes+1))
		in.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n > MaxFileBytes {
			return errors.New("file quota exceeded")
		}
	}
	return nil
}
