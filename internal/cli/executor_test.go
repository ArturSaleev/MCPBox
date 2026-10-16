package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestCLIHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "cli-test" || i+1 >= len(os.Args) {
			continue
		}
		switch os.Args[i+1] {
		case "echo":
			fmt.Print(strings.Join(os.Args[i+2:], "|"))
			fmt.Fprint(os.Stderr, "stderr")
		case "fail":
			fmt.Fprint(os.Stderr, "failed")
			os.Exit(7)
		case "output":
			fmt.Print(strings.Repeat("x", 8192))
			fmt.Fprint(os.Stderr, strings.Repeat("y", 8192))
		case "sleep":
			time.Sleep(10 * time.Second)
		case "environment":
			cwd, _ := os.Getwd()
			fmt.Print(cwd + "|" + os.Getenv("CLI_TEST_SECRET"))
		}
		os.Exit(0)
	}
}
func TestExecutionPermissionsAndLimits(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewStore(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	root := t.TempDir()
	project := &models.Project{Name: "CLI", RootPath: root}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	binary, _ := os.Executable()
	prefix := "-test.run=TestCLIHelper -- cli-test "
	entry := &models.CLI{Name: "Test", Slug: "test", Binary: binary, IsEnabled: true, TimeoutSeconds: 3, MaxOutputBytes: 1024, Operations: []models.CLIOperation{
		{Command: prefix + "echo", ArgumentSets: [][]string{{"literal; $(touch nope)", "-c"}}},
		{Command: prefix + "fail"}, {Command: prefix + "output"}, {Command: prefix + "sleep"}, {Command: prefix + "environment"},
	}}
	if err := Validate(entry); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	run := func(op string, args ...string) (*Result, error) {
		return Execute(ctx, store, project.ID, Request{CLI: "test", Command: prefix + op, Args: args})
	}
	if _, err := run("echo"); !errors.Is(err, ErrDenied) {
		t.Fatalf("unlinked: %v", err)
	}
	if err := store.SetProjectCLI(ctx, project.ID, entry.ID, true); err != nil {
		t.Fatal(err)
	}
	for _, req := range []Request{{CLI: "unknown", Command: prefix + "echo"}, {CLI: "test", Command: prefix + "echo extra"}, {CLI: "test", Command: prefix + "echo", Args: []string{"-c", "bad"}}, {CLI: "test", Command: "echo"}} {
		if _, err := Execute(ctx, store, project.ID, req); !errors.Is(err, ErrDenied) {
			t.Fatalf("should deny %#v: %v", req, err)
		}
	}
	result, err := run("echo", "literal; $(touch nope)", "-c")
	if err != nil || result.Stdout != "literal; $(touch nope)|-c" || result.Stderr != "stderr" || result.ExitCode != 0 {
		t.Fatalf("literal argv: %+v %v", result, err)
	}
	result, err = run("fail")
	if err != nil || result.ExitCode != 7 || result.Stderr != "failed" {
		t.Fatalf("exit: %+v %v", result, err)
	}
	result, err = run("output")
	if err != nil || !result.Truncated || len(result.Stdout)+len(result.Stderr) != 1024 {
		t.Fatalf("output cap: %+v %v", result, err)
	}
	entry.TimeoutSeconds = 1
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	result, err = run("sleep")
	if err != nil || !result.TimedOut || result.ExitCode == 0 || time.Since(start) > 4*time.Second {
		t.Fatalf("timeout: %+v %v", result, err)
	}
	entry.TimeoutSeconds = 3
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLI_TEST_SECRET", "do-not-inherit")
	result, err = run("environment")
	// macOS may resolve /var to /private/var in the child working directory.
	if err != nil || !strings.HasSuffix(result.Stdout, "|") || strings.Contains(result.Stdout, "do-not-inherit") {
		t.Fatalf("environment: %+v %v", result, err)
	}
	gotRoot := strings.TrimSuffix(result.Stdout, "|")
	a, _ := os.Stat(gotRoot)
	b, _ := os.Stat(root)
	if a == nil || b == nil || !os.SameFile(a, b) {
		t.Fatalf("cwd: %q != %q", gotRoot, root)
	}

	if err := store.SetProjectCLI(ctx, project.ID, entry.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := run("echo"); !errors.Is(err, ErrDenied) {
		t.Fatalf("revoked: %v", err)
	}
	if err := store.SetProjectCLI(ctx, project.ID, entry.ID, true); err != nil {
		t.Fatal(err)
	}
	entry.IsEnabled = false
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := run("echo"); !errors.Is(err, ErrDenied) {
		t.Fatalf("disabled: %v", err)
	}
	entry.IsEnabled = true
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().Model(project).Update("is_paused", true).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := run("echo"); !errors.Is(err, ErrDenied) {
		t.Fatalf("paused: %v", err)
	}
}
func TestValidateRegistry(t *testing.T) {
	for _, binary := range []string{"sh", "/bin/bash", "zsh", "cmd.exe", "powershell", "git status", "./git"} {
		entry := models.CLI{Name: "Bad", Slug: "bad", Binary: binary, Operations: []models.CLIOperation{{Command: "-c"}}}
		if Validate(&entry) == nil {
			t.Errorf("accepted %q", binary)
		}
	}
	for _, binary := range []string{"git", "opencode", "/tmp/a folder/git"} {
		entry := models.CLI{Name: "Good", Slug: "good", Binary: binary, Operations: []models.CLIOperation{{Command: "mcp list"}}}
		if err := Validate(&entry); err != nil {
			t.Errorf("%q: %v", binary, err)
		}
	}
}
