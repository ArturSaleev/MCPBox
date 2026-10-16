// Package cli implements project-scoped, administrator-approved process execution.
// It is an allowlist, not an OS sandbox: admins must trust the registered binaries
// and operations (interpreters and tools can themselves execute arbitrary code).
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

var ErrDenied = errors.New("CLI access denied")
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func Validate(item *models.CLI) error {
	item.Name = strings.TrimSpace(item.Name)
	item.Slug = strings.TrimSpace(item.Slug)
	item.Binary = strings.TrimSpace(item.Binary)
	if item.Name == "" || !slugPattern.MatchString(item.Slug) || item.Binary == "" || strings.ContainsRune(item.Binary, 0) {
		return errors.New("name, slug (lowercase letters, numbers, - or _) and binary are required")
	}
	if !filepath.IsAbs(item.Binary) && (strings.ContainsAny(item.Binary, "/\\ \t\r\n")) {
		return errors.New("binary must be a PATH name or an absolute path")
	}
	switch strings.TrimSuffix(strings.ToLower(filepath.Base(item.Binary)), ".exe") {
	case "sh", "bash", "zsh", "dash", "fish", "ksh", "csh", "tcsh", "cmd", "powershell", "pwsh":
		return errors.New("shell binaries are not allowed")
	}
	if len(item.Operations) == 0 || len(item.Operations) > 200 {
		return errors.New("register between 1 and 200 operations")
	}
	seen := map[string]bool{}
	for i := range item.Operations {
		op := &item.Operations[i]
		op.Command = strings.Join(strings.Fields(op.Command), " ")
		if op.Command == "" || strings.ContainsRune(op.Command, 0) || seen[op.Command] {
			return errors.New("operations must be nonempty and unique")
		}
		seen[op.Command] = true
		if len(op.ArgumentSets) > 100 {
			return errors.New("too many argument sets")
		}
		for _, args := range op.ArgumentSets {
			if len(args) > 100 {
				return errors.New("too many arguments")
			}
			for _, arg := range args {
				if len(arg) > 8192 || strings.ContainsRune(arg, 0) {
					return errors.New("invalid argument")
				}
			}
		}
	}
	if item.TimeoutSeconds == 0 {
		item.TimeoutSeconds = 30
	}
	if item.MaxOutputBytes == 0 {
		item.MaxOutputBytes = 1024 * 1024
	}
	if item.TimeoutSeconds < 1 || item.TimeoutSeconds > 300 {
		return errors.New("timeout_seconds must be between 1 and 300")
	}
	if item.MaxOutputBytes < 1024 || item.MaxOutputBytes > 4*1024*1024 {
		return errors.New("max_output_bytes must be between 1024 and 4194304")
	}
	return nil
}

type Store interface {
	GetProject(context.Context, uint) (*models.Project, error)
	ListProjectCLIs(context.Context, uint, bool) ([]models.CLI, error)
}
type Request struct {
	CLI     string   `json:"cli"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
type Result struct {
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	TimedOut  bool   `json:"timed_out"`
	Truncated bool   `json:"truncated"`
}

func Describe(ctx context.Context, store Store, projectID uint, slug string) (*models.CLI, error) {
	items, err := store.ListProjectCLIs(ctx, projectID, true)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Slug == slug {
			return &item, nil
		}
	}
	return nil, ErrDenied
}

func Execute(ctx context.Context, store Store, projectID uint, req Request) (*Result, error) {
	project, err := store.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsPaused {
		return nil, fmt.Errorf("%w: project is paused", ErrDenied)
	}
	item, err := Describe(ctx, store, projectID, req.CLI)
	if err != nil {
		return nil, err
	}
	if err := Validate(item); err != nil {
		return nil, fmt.Errorf("%w: invalid registry configuration", ErrDenied)
	}
	var argv []string
	allowed := false
	for _, op := range item.Operations {
		if op.Command != req.Command {
			continue
		}
		allowed = len(req.Args) == 0
		for _, args := range op.ArgumentSets {
			if slices.Equal(args, req.Args) {
				allowed = true
			}
		}
		argv = append(strings.Fields(op.Command), req.Args...)
		break
	}
	if !allowed {
		return nil, fmt.Errorf("%w: operation or arguments are not allowed", ErrDenied)
	}
	// Never accept caller-supplied cwd or environment. PATH resolution happens on
	// the server, before applying the project's working directory.
	binary, err := exec.LookPath(item.Binary)
	if err != nil {
		return nil, err
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(project.RootPath) == "" {
		return nil, errors.New("project root_path is required for CLI execution")
	}
	root, err := filepath.Abs(project.RootPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.New("project root_path must be an existing directory")
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(item.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, binary, argv...)
	cmd.Dir = root
	// Do not expose MCPBox's API keys and other server secrets to child processes.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "LANG=C.UTF-8", "TERM=dumb", "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "PAGER=cat"}
	if v := os.Getenv("SystemRoot"); v != "" {
		cmd.Env = append(cmd.Env, "SystemRoot="+v)
	}
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	budget := &outputBudget{remaining: item.MaxOutputBytes}
	stdout, stderr := &limitedOutput{budget: budget}, &limitedOutput{budget: budget}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	result := &Result{Stdout: stdout.buf.String(), Stderr: stderr.buf.String(), ExitCode: 0, TimedOut: errors.Is(timeoutCtx.Err(), context.DeadlineExceeded), Truncated: budget.truncated}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else if cmd.ProcessState != nil {
			result.ExitCode = -1
		} else {
			return nil, err
		}
	}
	return result, nil
}

type outputBudget struct {
	sync.Mutex
	remaining int
	truncated bool
}
type limitedOutput struct {
	budget *outputBudget
	buf    bytes.Buffer
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	w.budget.Lock()
	defer w.budget.Unlock()
	n := len(p)
	keep := min(n, w.budget.remaining)
	_, _ = w.buf.Write(p[:keep])
	w.budget.remaining -= keep
	if keep < n {
		w.budget.truncated = true
	}
	// Continue draining pipes after the cap instead of blocking the child.
	return n, nil
}
