package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestProjectWorkingDirectoryPolicy(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "repo")
	child := filepath.Join(allowed, "child")
	outside := filepath.Join(base, "repo-other")
	for _, dir := range []string{child, outside} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	canonicalChild, _ := filepath.EvalSymlinks(child)
	canonicalOutside, _ := filepath.EvalSymlinks(outside)
	p := &models.Project{CLIWorkdirMode: "restricted", CLIAllowedDirectories: []string{allowed}, RootPath: outside}
	for _, dir := range []string{"", allowed, child} {
		got, cleanup, err := workingDirectory(p, dir)
		cleanup()
		if err != nil {
			t.Fatalf("allowed %q: %v", dir, err)
		}
		want, _ := filepath.EvalSymlinks(allowed)
		if dir == child {
			want = canonicalChild
		}
		if got != want {
			t.Fatalf("cwd: %q want %q", got, want)
		}
	}
	file := filepath.Join(allowed, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{outside, filepath.Join(allowed, "..", "repo-other"), "relative", filepath.Join(allowed, "missing"), file} {
		if _, _, err := workingDirectory(p, dir); !errors.Is(err, ErrDenied) {
			t.Fatalf("must reject %q: %v", dir, err)
		}
	}
	link := filepath.Join(allowed, "escape")
	if err := os.Symlink(outside, link); err == nil {
		if _, _, err := workingDirectory(p, link); !errors.Is(err, ErrDenied) {
			t.Fatalf("symlink escaped: %v", err)
		}
	}
	p.CLIWorkdirMode = "all"
	if got, _, err := workingDirectory(p, outside); err != nil || got != canonicalOutside {
		t.Fatalf("all folders: %q %v", got, err)
	}
	p.CLIWorkdirMode, p.CLIAllowedDirectories = "restricted", nil
	got, cleanup, err := workingDirectory(p, "")
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Fatal("temporary directory not removed")
	}
	if _, _, err := workingDirectory(p, outside); !errors.Is(err, ErrDenied) {
		t.Fatal("empty list allowed a chosen cwd")
	}
	p.CLIWorkdirMode = ""
	if got, _, err := workingDirectory(p, ""); err != nil || got != canonicalOutside {
		t.Fatalf("legacy root: %q %v", got, err)
	}
	policy := DirectoryPolicy{Mode: "restricted", AllowedDirectories: []string{allowed, allowed}}
	if err := ValidateDirectoryPolicy(&policy); err != nil || len(policy.AllowedDirectories) != 1 {
		t.Fatalf("validation: %+v %v", policy, err)
	}
	for _, policy := range []DirectoryPolicy{{Mode: "invalid"}, {Mode: "restricted", AllowedDirectories: []string{"relative"}}, {Mode: "restricted", AllowedDirectories: []string{file}}} {
		if err := ValidateDirectoryPolicy(&policy); err == nil {
			t.Fatalf("accepted invalid policy: %+v", policy)
		}
	}
}

func TestGitFetchPullInSelectedProjectFolder(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	ctx := context.Background()
	base := t.TempDir()
	remote, first, second := filepath.Join(base, "remote.git"), filepath.Join(base, "first"), filepath.Join(base, "second")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(git, args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, output)
		}
	}
	run("init", "--bare", remote)
	run("clone", remote, first)
	run("-C", first, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	run("-C", first, "push", "origin", "HEAD")
	run("clone", remote, second)
	run("-C", first, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "next")
	run("-C", first, "push", "origin", "HEAD")
	store, err := storage.NewStore(filepath.Join(base, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	p := &models.Project{Name: "Git", RootPath: base, CLIWorkdirMode: "restricted", CLIAllowedDirectories: []string{second}}
	if err := store.CreateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	entry := &models.CLI{Name: "Git", Slug: "git", Binary: git, IsEnabled: true, Operations: []models.CLIOperation{{Command: "fetch"}, {Command: "pull", ArgumentSets: [][]string{{"--ff-only"}}}, {Command: "log", ArgumentSets: [][]string{{"-1", "--format=%s"}}}}}
	if err := Validate(entry); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := store.SetProjectCLI(ctx, p.ID, entry.ID, true); err != nil {
		t.Fatal(err)
	}
	for _, req := range []Request{{CLI: "git", Command: "fetch", WorkingDirectory: second}, {CLI: "git", Command: "pull", Args: []string{"--ff-only"}, WorkingDirectory: second}, {CLI: "git", Command: "log", Args: []string{"-1", "--format=%s"}}} {
		result, err := Execute(ctx, store, p.ID, req)
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("execute %+v: %+v %v", req, result, err)
		}
		if req.Command == "log" && strings.TrimSpace(result.Stdout) != "next" {
			t.Fatalf("wrong repository: %+v", result)
		}
	}
	if _, err := Execute(ctx, store, p.ID, Request{CLI: "git", Command: "fetch", WorkingDirectory: first}); !errors.Is(err, ErrDenied) {
		t.Fatalf("foreign repo: %v", err)
	}
	if _, err := Execute(ctx, store, p.ID, Request{CLI: "git", Command: "pull", Args: []string{"--rebase"}, WorkingDirectory: second}); !errors.Is(err, ErrDenied) {
		t.Fatalf("argument allowlist bypassed: %v", err)
	}
}
