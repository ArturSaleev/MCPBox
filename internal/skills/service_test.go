package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.Project{}, &Skill{}, &Binding{}, &Job{}, &CatalogCache{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	return New(db, filepath.Join(root, "skill_store"))
}
func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for p, data := range files {
		f, err := z.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(f, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func installTestSkill(t *testing.T, s *Service) *Skill {
	t.Helper()
	v, err := s.ImportZIP(context.Background(), zipBytes(t, map[string]string{"SKILL.md": "---\nname: transform\ndescription: Transform an input file\ncompatibility: Python 3\n---\nRead input.csv and write output.txt.\n", "scripts/transform.py": "import pathlib\npathlib.Path('output.txt').write_text(pathlib.Path('input.csv').read_text().upper())\n"}), "")
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func project(t *testing.T, s *Service, name string) uint {
	t.Helper()
	p := models.Project{Name: name, Token: name}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	return p.ID
}
func TestPackageInstallBindingsAndFileIsolation(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	item := installTestSkill(t, s)
	same := installTestSkill(t, s)
	if same.ID != item.ID {
		t.Fatal("identical package did not deduplicate")
	}
	if len(item.Files) != 2 || len(item.Requirements) == 0 {
		t.Fatal("package resources/requirements lost")
	}
	a, b := project(t, s, "a"), project(t, s, "b")
	if err := s.Bind(ctx, a, item.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bound(ctx, b, item.ID); err == nil {
		t.Fatal("cross-project skill access")
	}
	if err := s.Delete(ctx, item.ID); err == nil {
		t.Fatal("deleted connected package")
	}
	job, err := s.NewJob(ctx, a, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Upload(ctx, a, job.ID, "input.csv", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Upload(ctx, a, job.ID, "input.csv", strings.NewReader("overwrite")); err == nil {
		t.Fatal("silently overwrote input")
	}
	if _, err = s.Upload(ctx, a, job.ID, "../escape", strings.NewReader("x")); err == nil {
		t.Fatal("upload traversal")
	}
	if _, err = s.GetJob(ctx, b, job.ID); err == nil {
		t.Fatal("cross-project job access")
	}
	if _, err = s.ReadJobFile(ctx, b, job.ID, "input.csv"); err == nil {
		t.Fatal("cross-project download")
	}
	data, err := s.ReadJobFile(ctx, a, job.ID, "input.csv")
	if err != nil || string(data) != "hello" {
		t.Fatalf("read input: %q %v", data, err)
	}
	if _, err = s.ReadPackageFile(ctx, item.ID, "../test.db"); err == nil {
		t.Fatal("package traversal")
	}
	if err = s.db.Model(&models.Project{}).Where("id = ?", a).Update("is_paused", true).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = s.NewJob(ctx, a, item.ID); err == nil {
		t.Fatal("paused project created a job")
	}
}
func TestArchiveRejectsTraversalSymlinksAndDuplicatePaths(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "C:/windows", "a\\b"} {
		t.Run(name, func(t *testing.T) {
			s := testService(t)
			if _, err := s.ImportZIP(context.Background(), zipBytes(t, map[string]string{name: "bad"}), ""); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	s := testService(t)
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	h := &zip.FileHeader{Name: "link"}
	h.SetMode(os.ModeSymlink | 0777)
	f, _ := w.CreateHeader(h)
	io.WriteString(f, "/etc/passwd")
	w.Close()
	if _, err := s.ImportZIP(context.Background(), data.Bytes(), ""); err == nil {
		t.Fatal("symlink archive accepted")
	}
	data.Reset()
	w = zip.NewWriter(&data)
	for i := 0; i < 2; i++ {
		f, _ = w.Create("same")
		io.WriteString(f, "x")
	}
	w.Close()
	if _, err := s.ImportZIP(context.Background(), data.Bytes(), ""); err == nil {
		t.Fatal("duplicate archive paths accepted")
	}
}
func TestCatalogFailureRetainsPreviousSnapshot(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := Catalog{SchemaVersion: 1, Items: []Entry{{ID: "one", Name: "one", Source: Source{Repository: "https://github.com/example/skills", Path: "skills/one"}}}}
	raw, _ := json.Marshal(c)
	if _, err := s.ImportCatalog(ctx, raw, "local"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"schema_version":2}`, `{"schema_version":1,"items":[{"id":"x","name":"x","source":{"repository":"https://evil.test/repo"}}]}`} {
		if _, err := s.ImportCatalog(ctx, []byte(bad), "bad"); err == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
	cached, err := s.Catalog(ctx)
	if err != nil || len(cached.Content.Items) != 1 || cached.URL != "local" {
		t.Fatalf("lost catalog: %+v %v", cached, err)
	}
	for _, repo := range []string{"https://github.com.evil.test/o/r", "https://github.com/o/../r", "https://github.com/o/r/tree/main", "http://github.com/o/r", "https://user@github.com/o/r"} {
		if _, _, err = RepositoryParts(repo); err == nil {
			t.Fatalf("accepted %s", repo)
		}
	}
}
func fakeDocker(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "argv")
	t.Setenv("FAKE_DOCKER_LOG", log)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := `#!/bin/sh
printf '%s\n' "$@" >> "$FAKE_DOCKER_LOG"
if [ "$1" = image ]; then printf "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"; exit 0; fi
if [ "$1" = run ]; then
 work=''
 for arg do
  case "$arg" in type=bind,src=*,dst=/work) work="${arg#type=bind,src=}"; work="${work%,dst=/work}";; esac
 done
 if [ "$FAKE_DOCKER_MODE" = slow ]; then sleep 5; exit 0; fi
 if [ "$FAKE_DOCKER_MODE" = quota ]; then ln -s /etc/passwd "$work/escape"; sleep 5; exit 0; fi
 if [ -n "$work" ]; then printf 'TRANSFORMED' > "$work/output.txt"; fi
 printf 'script completed'
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return log
}
func TestRuntimeProfileAndServerExecution(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	item := installTestSkill(t, s)
	id := project(t, s, "a")
	if err := s.Bind(ctx, id, item.ID, true); err != nil {
		t.Fatal(err)
	}
	log := fakeDocker(t)
	profile := Profile{Enabled: true, Runtime: "python", Image: "python:3.12-slim", Scripts: []string{"scripts/transform.py"}, TimeoutSeconds: 5, PreparedImage: "forged"}
	saved, err := s.SaveProfile(ctx, item.ID, profile)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Profile.PreparedImage != "" {
		t.Fatal("client forged prepared image")
	}
	persisted, err := s.Get(ctx, item.ID)
	if err != nil || !persisted.Profile.Enabled || persisted.Profile.Runtime != "python" {
		t.Fatalf("profile persistence: %+v %v", persisted, err)
	}
	job, err := s.NewJob(ctx, id, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(ctx, id, job.ID, "scripts/transform.py", nil); err == nil {
		t.Fatal("ran unprepared environment")
	}
	if _, err = s.Prepare(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Upload(ctx, id, job.ID, "input.csv", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(ctx, id, job.ID, "SKILL.md", nil); err == nil {
		t.Fatal("ran a non-allowed file")
	}
	result, err := s.Run(ctx, id, job.ID, "scripts/transform.py", []string{"input.csv", "; touch /escape"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.Stdout != "script completed" {
		t.Fatalf("run: %+v", result)
	}
	if _, err = s.Run(ctx, id, job.ID, "scripts/transform.py", nil); err == nil {
		t.Fatal("ran the same job twice")
	}
	data, err := s.ReadJobFile(ctx, id, job.ID, "output.txt")
	if err != nil || string(data) != "TRANSFORMED" {
		t.Fatalf("output: %q %v", data, err)
	}
	argv, _ := os.ReadFile(log)
	for _, flag := range []string{"--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--network\nnone", "--entrypoint\npython", "dst=/package,readonly", "rm\n--force", "; touch /escape"} {
		if !strings.Contains(string(argv), flag) {
			t.Fatalf("missing Docker isolation/argv option %s: %s", flag, argv)
		}
	}
	if err = os.Chmod(filepath.Join(s.packageRoot(item.ID), "scripts/transform.py"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.packageRoot(item.ID), "scripts/transform.py"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := s.NewJob(ctx, id, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(ctx, id, next.ID, "scripts/transform.py", nil); err == nil {
		t.Fatal("executed modified package")
	}
}
func TestRuntimeTimeoutAndUnsafeOutputCleanup(t *testing.T) {
	for _, mode := range []string{"slow", "quota"} {
		t.Run(mode, func(t *testing.T) {
			s := testService(t)
			ctx := context.Background()
			item := installTestSkill(t, s)
			id := project(t, s, "a")
			if err := s.Bind(ctx, id, item.ID, true); err != nil {
				t.Fatal(err)
			}
			log := fakeDocker(t)
			t.Setenv("FAKE_DOCKER_MODE", mode)
			if _, err := s.SaveProfile(ctx, item.ID, Profile{Enabled: true, Runtime: "python", Image: "python:3.12-slim", Scripts: []string{"scripts/transform.py"}, TimeoutSeconds: 1}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Prepare(ctx, item.ID); err != nil {
				t.Fatal(err)
			}
			job, err := s.NewJob(ctx, id, item.ID)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			result, err := s.Run(ctx, id, job.ID, "scripts/transform.py", nil)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != "failed" || time.Since(start) > 5*time.Second {
				t.Fatalf("timeout/quota not enforced: %+v", result)
			}
			if mode == "slow" && !result.TimedOut {
				t.Fatal("timeout not reported")
			}
			argv, _ := os.ReadFile(log)
			if !strings.Contains(string(argv), "rm\n--force") {
				t.Fatal("container was not removed")
			}
			if mode == "quota" {
				if _, err = s.ReadJobFile(ctx, id, job.ID, "escape"); err == nil {
					t.Fatal("exposed symlink output")
				}
			}
		})
	}
}
func TestRealDockerInputOutput(t *testing.T) {
	if os.Getenv("MCPBOX_TEST_DOCKER") != "1" {
		t.Skip("set MCPBOX_TEST_DOCKER=1 for Docker integration")
	}
	s := testService(t)
	ctx := context.Background()
	item := installTestSkill(t, s)
	p := project(t, s, "docker")
	if err := s.Bind(ctx, p, item.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveProfile(ctx, item.ID, Profile{Enabled: true, Runtime: "python", Image: "python:3.12-slim", Scripts: []string{"scripts/transform.py"}, TimeoutSeconds: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prepare(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	job, err := s.NewJob(ctx, p, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Upload(ctx, p, job.ID, "input.csv", strings.NewReader("hello docker")); err != nil {
		t.Fatal(err)
	}
	result, err := s.Run(ctx, p, job.ID, "scripts/transform.py", nil)
	if err != nil || result.Status != "completed" {
		t.Fatalf("Docker execution: %+v %v", result, err)
	}
	data, err := s.ReadJobFile(ctx, p, job.ID, "output.txt")
	if err != nil || string(data) != "HELLO DOCKER" {
		t.Fatalf("Docker result %q %v", data, err)
	}
}

func TestRealDockerNodeDependencies(t *testing.T) {
	if os.Getenv("MCPBOX_TEST_DOCKER") != "1" {
		t.Skip("set MCPBOX_TEST_DOCKER=1 for Docker integration")
	}
	s := testService(t)
	ctx := context.Background()
	archive := zipBytes(t, map[string]string{
		"SKILL.md":                            "---\nname: node-transform\ndescription: Node dependency execution test\n---\nTransform uploaded input.\n",
		"package.json":                        `{"name":"node-test","version":"1.0.0","type":"module","dependencies":{"local-transform":"file:vendor/local-transform"}}`,
		"package-lock.json":                   `{"name":"node-test","version":"1.0.0","lockfileVersion":3,"requires":true,"packages":{"":{"name":"node-test","version":"1.0.0","dependencies":{"local-transform":"file:vendor/local-transform"}},"node_modules/local-transform":{"resolved":"vendor/local-transform","link":true},"vendor/local-transform":{"name":"local-transform","version":"1.0.0"}}}`,
		"vendor/local-transform/package.json": `{"name":"local-transform","version":"1.0.0","type":"module","exports":"./index.js"}`,
		"vendor/local-transform/index.js":     `export default value => value.toUpperCase();`,
		"scripts/transform.mjs":               "import transform from 'local-transform';\nimport fs from 'node:fs';\nfs.writeFileSync('output.txt',transform(fs.readFileSync('input.txt','utf8')));\n",
	})
	item, err := s.ImportZIP(ctx, archive, "")
	if err != nil {
		t.Fatal(err)
	}
	p := project(t, s, "node")
	if err = s.Bind(ctx, p, item.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveProfile(ctx, item.ID, Profile{Enabled: true, Runtime: "node", Image: "node:22-bookworm-slim", Scripts: []string{"scripts/transform.mjs"}, DependencyFile: "package-lock.json", TimeoutSeconds: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Prepare(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	job, err := s.NewJob(ctx, p, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Upload(ctx, p, job.ID, "input.txt", strings.NewReader("node dependencies")); err != nil {
		t.Fatal(err)
	}
	result, err := s.Run(ctx, p, job.ID, "scripts/transform.mjs", nil)
	if err != nil || result.Status != "completed" {
		t.Fatalf("Node execution: %+v %v", result, err)
	}
	data, err := s.ReadJobFile(ctx, p, job.ID, "output.txt")
	if err != nil || string(data) != "NODE DEPENDENCIES" {
		t.Fatalf("Node result %q %v", data, err)
	}
}

func TestDisconnectRequiresRemovingProjectSkillJobs(t *testing.T) {
	for _, status := range []string{"created", "running", "completed", "failed"} {
		t.Run(status, func(t *testing.T) {
			s := testService(t)
			ctx := context.Background()
			item := installTestSkill(t, s)
			a, b := project(t, s, "a"), project(t, s, "b")
			for _, p := range []uint{a, b} {
				if err := s.Bind(ctx, p, item.ID, true); err != nil {
					t.Fatal(err)
				}
			}
			job, err := s.NewJob(ctx, a, item.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Upload(ctx, a, job.ID, "input.txt", strings.NewReader("keep me")); err != nil {
				t.Fatal(err)
			}
			if err := s.db.Model(&Job{}).Where("id = ?", job.ID).Update("status", status).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.Bind(ctx, a, item.ID, false); err == nil {
				t.Fatal("disconnected a skill with retained jobs")
			}
			if _, err := s.Bound(ctx, a, item.ID); err != nil {
				t.Fatal("lost binding on blocked disconnect", err)
			}
			if data, err := os.ReadFile(filepath.Join(s.jobRoot(a, job.ID), "input.txt")); err != nil || string(data) != "keep me" {
				t.Fatal("disconnect damaged job files", err)
			}
			// Jobs in one project must not prevent disconnection in another.
			if err := s.Bind(ctx, b, item.ID, false); err != nil {
				t.Fatal("other project affected", err)
			}
			if err := s.db.Model(&Job{}).Where("id = ?", job.ID).Update("status", "completed").Error; err != nil {
				t.Fatal(err)
			}
			if err := s.DeleteJob(ctx, a, job.ID); err != nil {
				t.Fatal(err)
			}
			if err := s.Bind(ctx, a, item.ID, false); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Bound(ctx, a, item.ID); err == nil {
				t.Fatal("skill still connected")
			}
			if _, err := s.Get(ctx, item.ID); err != nil {
				t.Fatal("downloaded package lost", err)
			}
		})
	}
}
