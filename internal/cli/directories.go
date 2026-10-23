package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

// DirectoryPolicy controls cwd selection, not the program's filesystem access.
type DirectoryPolicy struct {
	Mode               string   `json:"mode"`
	AllowedDirectories []string `json:"allowed_directories"`
	DefaultDirectory   string   `json:"default_directory"`
}

func ProjectDirectoryPolicy(project *models.Project) DirectoryPolicy {
	mode := project.CLIWorkdirMode
	dirs := append([]string{}, project.CLIAllowedDirectories...)
	if mode == "" {
		mode = "restricted"
		if strings.TrimSpace(project.RootPath) != "" {
			dirs = []string{project.RootPath}
		}
	}
	defaultDirectory := project.RootPath
	if strings.TrimSpace(defaultDirectory) == "" {
		defaultDirectory = ""
	}
	if project.CLIWorkdirMode == "restricted" {
		defaultDirectory = ""
		if len(dirs) > 0 {
			defaultDirectory = dirs[0]
		}
	}
	return DirectoryPolicy{Mode: mode, AllowedDirectories: dirs, DefaultDirectory: defaultDirectory}
}

func canonicalDirectory(dir string) (string, error) {
	if !filepath.IsAbs(dir) || strings.ContainsRune(dir, 0) {
		return "", fmt.Errorf("directory must be an absolute existing path")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(dir))
	if err != nil {
		return "", fmt.Errorf("directory must exist and be accessible")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory must exist and be a folder")
	}
	return resolved, nil
}

func ValidateDirectoryPolicy(policy *DirectoryPolicy) error {
	if policy.Mode != "restricted" && policy.Mode != "all" {
		return fmt.Errorf("directory mode must be restricted or all")
	}
	if len(policy.AllowedDirectories) > 100 {
		return fmt.Errorf("at most 100 directories are allowed")
	}
	if policy.Mode == "all" {
		policy.AllowedDirectories = []string{}
		return nil
	}
	dirs := []string{}
	seen := map[string]bool{}
	for _, dir := range policy.AllowedDirectories {
		if len(dir) > 4096 {
			return fmt.Errorf("directory path is too long")
		}
		dir, err := canonicalDirectory(strings.TrimSpace(dir))
		if err != nil {
			return err
		}
		if !seen[dir] {
			dirs = append(dirs, dir)
			seen[dir] = true
		}
	}
	policy.AllowedDirectories = dirs
	return nil
}

func workingDirectory(project *models.Project, requested string) (string, func(), error) {
	noop := func() {}
	policy := ProjectDirectoryPolicy(project)
	if policy.Mode != "restricted" && policy.Mode != "all" {
		return "", noop, fmt.Errorf("%w: invalid directory policy", ErrDenied)
	}
	chosen := requested
	if chosen == "" {
		chosen = policy.DefaultDirectory
		if chosen == "" && policy.Mode == "restricted" && len(policy.AllowedDirectories) > 0 {
			chosen = policy.AllowedDirectories[0]
		}
	}
	if chosen == "" {
		dir, err := os.MkdirTemp("", "mcpbox-cli-*")
		return dir, func() {
			if dir != "" {
				_ = os.RemoveAll(dir)
			}
		}, err
	}
	// Keep existing relative root_path configurations working, but requests must be absolute.
	if requested == "" && !filepath.IsAbs(chosen) {
		chosen, _ = filepath.Abs(chosen)
	}
	resolved, err := canonicalDirectory(chosen)
	if err != nil {
		return "", noop, fmt.Errorf("%w: working directory must be an absolute existing directory", ErrDenied)
	}
	if policy.Mode == "all" {
		return resolved, noop, nil
	}
	for _, root := range policy.AllowedDirectories {
		if project.CLIWorkdirMode == "" && !filepath.IsAbs(root) {
			root, _ = filepath.Abs(root)
		}
		root, err := canonicalDirectory(root)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(root, resolved)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			return resolved, noop, nil
		}
	}
	return "", noop, fmt.Errorf("%w: working directory is outside the project's allowed folders", ErrDenied)
}
