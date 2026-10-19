// SPDX-License-Identifier: Apache-2.0
package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var imagePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./_:@-]{0,255}$`)

func validateProfile(p *Profile, item *Skill) error {
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 60
	}
	if p.TimeoutSeconds < 1 || p.TimeoutSeconds > 300 {
		return errors.New("timeout_seconds must be between 1 and 300")
	}
	switch p.Runtime {
	case "python", "node", "bash":
	case "":
		if p.Enabled {
			return errors.New("select a runtime")
		}
	default:
		return errors.New("unsupported runtime")
	}
	if p.Runtime != "" && !imagePattern.MatchString(p.Image) {
		return errors.New("a Docker image is required")
	}
	available := map[string]bool{}
	for _, f := range item.Files {
		available[f.Path] = true
	}
	for _, script := range p.Scripts {
		if !validPath(script) || !available[script] {
			return errors.New("allowed script is not a package file")
		}
		ext := strings.ToLower(filepath.Ext(script))
		if (p.Runtime == "python" && ext != ".py") || (p.Runtime == "node" && ext != ".js" && ext != ".mjs" && ext != ".cjs") || (p.Runtime == "bash" && ext != ".sh") {
			return errors.New("script does not match runtime")
		}
	}
	if p.Enabled && len(p.Scripts) == 0 {
		return errors.New("select at least one allowed script")
	}
	if p.DependencyFile != "" {
		if !available[p.DependencyFile] || !validPath(p.DependencyFile) {
			return errors.New("dependency file is not in package")
		}
		if p.Runtime == "python" && !strings.HasSuffix(p.DependencyFile, ".txt") {
			return errors.New("Python dependency file must be a requirements .txt file")
		}
		if p.Runtime == "node" && p.DependencyFile != "package-lock.json" {
			return errors.New("Node dependencies require root package-lock.json")
		}
		if p.Runtime == "bash" {
			return errors.New("use a prebuilt image for shell dependencies")
		}
	}
	return nil
}
func (s *Service) SaveProfile(ctx context.Context, id string, p Profile) (*Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = validateProfile(&p, item); err != nil {
		return nil, err
	}
	// A model/client cannot mark an environment as prepared through this API.
	p.PreparedImage = ""
	p.PreparedAt = ""
	item.Profile = p
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Model(item).Update("profile", string(encoded)).Error
	return item, err
}

type boundedOutput struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
	max       int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := b.max - len(b.data)
	if remaining > 0 {
		b.data = append(b.data, p[:min(remaining, n)]...)
	}
	if n > remaining {
		b.truncated = true
	}
	return n, nil
}
func (b *boundedOutput) String() string { return string(b.data) }

func (s *Service) Prepare(ctx context.Context, id string) (*Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	p := item.Profile
	if err = validateProfile(&p, item); err != nil {
		return nil, err
	}
	if p.Runtime == "" {
		return nil, errors.New("configure runtime first")
	}
	if err = s.verifyPackage(item); err != nil {
		return nil, err
	}
	if _, err = exec.LookPath("docker"); err != nil {
		return nil, errors.New("Docker is required on the MCPBox server")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	args := []string{"pull", p.Image}
	prepared := p.Image
	if p.DependencyFile != "" {
		fingerprintProfile := p
		fingerprintProfile.PreparedImage = ""
		fingerprintProfile.PreparedAt = ""
		config, _ := json.Marshal(fingerprintProfile)
		digest := sha256.Sum256(append([]byte(item.Hash), config...))
		prepared = "mcpbox-skill:" + hex.EncodeToString(digest[:16])
		var build string
		if p.Runtime == "python" {
			encoded, _ := json.Marshal([]string{"python", "-m", "pip", "install", "--no-cache-dir", "-r", "/skill/" + p.DependencyFile})
			build = "RUN " + string(encoded) + "\n"
		} else {
			build = "WORKDIR /skill\nRUN [\"npm\", \"ci\", \"--ignore-scripts\", \"--no-audit\", \"--no-fund\"]\n"
		}
		file, err := os.CreateTemp(s.root, ".Dockerfile-*")
		if err != nil {
			return nil, err
		}
		defer os.Remove(file.Name())
		_, err = fmt.Fprintf(file, "FROM %s\nUSER root\nCOPY . /skill/\n%s", p.Image, build)
		file.Close()
		if err != nil {
			return nil, err
		}
		args = []string{"build", "--pull", "--tag", prepared, "--file", file.Name(), s.packageRoot(id)}
	}
	output := &boundedOutput{max: 64 << 10}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.WaitDelay = 2 * time.Second
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("environment preparation failed: %w: %s", err, output.String())
	}
	imageOutput, inspectErr := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{.Id}}", prepared).Output()
	if inspectErr != nil {
		return nil, fmt.Errorf("cannot inspect prepared image: %w", inspectErr)
	}
	immutableImage := strings.TrimSpace(string(imageOutput))
	if !regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(immutableImage) {
		return nil, errors.New("Docker returned an invalid image ID")
	}
	p.PreparedImage = immutableImage
	p.PreparedAt = time.Now().UTC().Format(time.RFC3339)
	item.Profile = p
	encoded, marshalErr := json.Marshal(p)
	if marshalErr != nil {
		return nil, marshalErr
	}
	if err = s.db.WithContext(context.WithoutCancel(ctx)).Model(item).Update("profile", string(encoded)).Error; err != nil {
		return nil, err
	}
	return item, nil
}
func (s *Service) verifyPackage(item *Skill) error {
	_, _, hash, _, _, _, err := inspectPackage(s.packageRoot(item.ID))
	if err != nil {
		return err
	}
	if hash != item.Hash {
		return errors.New("package changed since installation; reimport before executing")
	}
	return nil
}
func (s *Service) NewJob(ctx context.Context, project uint, skill string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Bound(ctx, project, skill); err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&Job{}).Where("project_id = ?", project).Count(&count).Error; err != nil {
		return nil, err
	}
	if count >= 100 {
		return nil, errors.New("project job limit reached; delete old jobs")
	}
	job := Job{ID: uuid.NewString(), ProjectID: project, SkillID: skill, Status: "created", Files: []File{}}
	if err := os.MkdirAll(s.jobRoot(project, job.ID), 0700); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		os.RemoveAll(s.jobRoot(project, job.ID))
		return nil, err
	}
	return &job, nil
}
func validJobID(id string) bool { _, err := uuid.Parse(id); return err == nil && len(id) == 36 }
func (s *Service) jobRoot(project uint, id string) string {
	return filepath.Join(s.root, "jobs", strconv.FormatUint(uint64(project), 10), id)
}
func (s *Service) GetJob(ctx context.Context, project uint, id string) (*Job, error) {
	if !validJobID(id) {
		return nil, errors.New("invalid job ID")
	}
	if err := s.checkProject(ctx, project); err != nil {
		return nil, err
	}
	var job Job
	if err := s.db.WithContext(ctx).First(&job, "project_id = ? AND id = ?", project, id).Error; err != nil {
		return nil, err
	}
	// Running containers can mutate their output tree; files are exposed only after exit.
	job.Files = []File{}
	if job.Status != "running" {
		files, err := listFiles(s.jobRoot(project, id), MaxJobBytes)
		if err != nil {
			job.Error = err.Error()
		} else {
			job.Files = files
		}
	}
	return &job, nil
}
func (s *Service) ListJobs(ctx context.Context, project uint) ([]Job, error) {
	if err := s.checkProject(ctx, project); err != nil {
		return nil, err
	}
	items := []Job{}
	err := s.db.WithContext(ctx).Omit("stdout", "stderr").Where("project_id = ?", project).Order("created_at DESC").Limit(100).Find(&items).Error
	return items, err
}
func (s *Service) Upload(ctx context.Context, project uint, id, name string, r io.Reader) (*Job, error) {
	if !validPath(name) || strings.Contains(name, "/") {
		return nil, errors.New("upload requires a plain filename")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.GetJob(ctx, project, id)
	if err != nil {
		return nil, err
	}
	if job.Status != "created" {
		return nil, errors.New("uploads are accepted only before the first run")
	}
	files, err := listFiles(s.jobRoot(project, id), MaxJobBytes)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, f := range files {
		total += f.Size
	}
	if len(files) >= MaxFiles {
		return nil, errors.New("too many files")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxFileBytes || total+int64(len(data)) > MaxJobBytes {
		return nil, errors.New("upload quota exceeded")
	}
	file, err := os.OpenFile(filepath.Join(s.jobRoot(project, id), name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return s.GetJob(ctx, project, id)
}
func (s *Service) ReadJobFile(ctx context.Context, project uint, id, name string) ([]byte, error) {
	job, err := s.GetJob(ctx, project, id)
	if err != nil {
		return nil, err
	}
	if job.Status == "running" {
		return nil, errors.New("wait for the running job to finish")
	}
	f, err := openRegular(s.jobRoot(project, id), name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
}
func (s *Service) DeleteJob(ctx context.Context, project uint, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.GetJob(ctx, project, id)
	if err != nil {
		return err
	}
	if job.Status == "running" {
		return errors.New("job is running")
	}
	if err = os.RemoveAll(s.jobRoot(project, id)); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Delete(job).Error
}
func (s *Service) Run(ctx context.Context, project uint, id, script string, args []string) (*Job, error) {
	s.mu.Lock()
	job, err := s.GetJob(ctx, project, id)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	item, err := s.Bound(ctx, project, job.SkillID)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	p := item.Profile
	if job.Status != "created" {
		s.mu.Unlock()
		return nil, errors.New("create a new job for each execution")
	}
	if !p.Enabled || p.PreparedImage == "" {
		s.mu.Unlock()
		return nil, errors.New("administrator must enable and prepare this skill runtime")
	}
	allowed := false
	for _, v := range p.Scripts {
		if script == v {
			allowed = true
		}
	}
	if !allowed {
		s.mu.Unlock()
		return nil, errors.New("script is not allowed by the execution profile")
	}
	if len(args) > 64 {
		s.mu.Unlock()
		return nil, errors.New("too many script arguments")
	}
	for _, a := range args {
		if len(a) > 8192 || strings.ContainsRune(a, 0) {
			s.mu.Unlock()
			return nil, errors.New("invalid script argument")
		}
	}
	if err = s.verifyPackage(item); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	job.Status = "running"
	job.Script = script
	claim := s.db.WithContext(ctx).Model(job).Where("status = ?", "created").Updates(map[string]any{"status": "running", "script": script})
	if claim.Error != nil || claim.RowsAffected != 1 {
		s.mu.Unlock()
		if claim.Error != nil {
			return nil, claim.Error
		}
		return nil, errors.New("job was already claimed")
	}
	s.mu.Unlock()
	// Persist the terminal state even if a client disconnects or the timeout fires.
	finish := func(runErr error) (*Job, error) {
		if runErr != nil {
			job.Error = runErr.Error()
			job.Status = "failed"
		} else if job.ExitCode != 0 {
			job.Status = "failed"
		} else {
			job.Status = "completed"
		}
		files, e := listFiles(s.jobRoot(project, id), MaxJobBytes)
		job.Files = files
		if e != nil {
			job.Error = e.Error()
			job.Status = "failed"
		}
		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		e = s.db.WithContext(saveCtx).Model(job).Select("status", "stdout", "stderr", "exit_code", "timed_out", "truncated", "error", "updated_at").Updates(job).Error
		if e != nil {
			return nil, e
		}
		return job, nil
	}
	// These are server-side paths; arguments are passed as argv, never a shell command.
	work, err := filepath.Abs(s.jobRoot(project, id))
	if err != nil {
		return finish(err)
	}
	pkg, err := filepath.Abs(s.packageRoot(item.ID))
	if err != nil {
		return finish(err)
	}
	if strings.ContainsAny(work+pkg, ",\n\r") {
		return finish(errors.New("Docker mount path contains unsupported characters"))
	}
	uid, gid := os.Getuid(), os.Getgid()
	if uid < 1 {
		uid = 65534
	}
	if gid < 1 {
		gid = 65534
	}
	// Existing uploaded files and work directory are owned by the same non-root uid.
	if os.Getuid() == 0 {
		if err = os.Chown(work, uid, gid); err != nil {
			return finish(err)
		}
		for _, f := range job.Files {
			_ = os.Chown(filepath.Join(work, filepath.FromSlash(f.Path)), uid, gid)
		}
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return finish(errors.New("Docker is required on the MCPBox server"))
	}
	network := "none"
	if p.Network {
		network = "bridge"
	}
	container := "mcpbox-skill-" + id
	scriptRoot := "/package/"
	if p.DependencyFile != "" {
		scriptRoot = "/skill/"
	}
	dockerArgs := []string{"run", "--rm", "--pull=never", "--name", container, "--init", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=128", "--memory=512m", "--cpus=1", "--ulimit", "fsize=33554432:33554432", "--network", network, "--user", fmt.Sprintf("%d:%d", uid, gid), "--tmpfs", "/tmp:rw,nosuid,nodev,size=64m", "--mount", "type=bind,src=" + pkg + ",dst=/package,readonly", "--mount", "type=bind,src=" + work + ",dst=/work", "--workdir", "/work", "--env", "HOME=/tmp", "--env", "PYTHONDONTWRITEBYTECODE=1", "--env", "PYTHONPATH=/package", "--env", "NODE_PATH=/skill/node_modules", "--entrypoint", map[string]string{"python": "python", "node": "node", "bash": "bash"}[p.Runtime], p.PreparedImage, scriptRoot + script}
	dockerArgs = append(dockerArgs, args...)
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()
	out, stderr := &boundedOutput{max: 1 << 20}, &boundedOutput{max: 1 << 20}
	cmd := exec.CommandContext(runCtx, "docker", dockerArgs...)
	cmd.Stdout = out
	cmd.Stderr = stderr
	cmd.WaitDelay = 2 * time.Second
	monitorDone := make(chan struct{})
	monitorResult := make(chan error, 1)
	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-monitorDone:
				monitorResult <- nil
				return
			case <-tick.C:
				if _, quotaErr := listFiles(work, MaxJobBytes); quotaErr != nil {
					cancel()
					monitorResult <- quotaErr
					return
				}
			}
		}
	}()
	err = cmd.Run()
	close(monitorDone)
	quotaErr := <-monitorResult
	if quotaErr != nil {
		err = quotaErr
	}
	// Killing the docker client alone does not stop the container. Always remove it.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	cleanupErr := removeContainer(cleanupCtx, container)
	cleanupCancel()
	if cleanupErr != nil {
		job.Error = "Cannot confirm container termination; restart MCPBox after Docker is available to recover this job"
		saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer saveCancel()
		if saveErr := s.db.WithContext(saveCtx).Model(job).Update("error", job.Error).Error; saveErr != nil {
			return nil, saveErr
		}
		return job, errors.New(job.Error)
	}
	job.Stdout = out.String()
	job.Stderr = stderr.String()
	job.Truncated = out.truncated || stderr.truncated
	job.TimedOut = errors.Is(runCtx.Err(), context.DeadlineExceeded)
	job.ExitCode = 0
	if err != nil {
		job.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			job.ExitCode = exit.ExitCode()
		}
		return finish(err)
	}
	return finish(nil)
}

func (s *Service) RuntimeStatus(ctx context.Context) map[string]any {
	result := map[string]any{"engine": "docker", "available": false, "root": s.root}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	output, err := cmd.Output()
	if err != nil {
		result["error"] = "Docker is not available on the MCPBox server"
	} else {
		result["available"] = true
		result["version"] = strings.TrimSpace(string(output))
	}
	return result
}

// RecoverJobs marks unfinished runs after a server restart. Containers are named by
// job ID and are removed before the job becomes accessible for reading/deletion.
func (s *Service) RecoverJobs(ctx context.Context) error {
	var jobs []Job
	if err := s.db.WithContext(ctx).Where("status = ?", "running").Find(&jobs).Error; err != nil {
		return err
	}
	for _, j := range jobs {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := removeContainer(cleanupCtx, "mcpbox-skill-"+j.ID)
		cancel()
		if err != nil {
			continue
		}
		if err = s.db.WithContext(ctx).Model(&j).Updates(map[string]any{"status": "failed", "error": "MCPBox restarted during execution"}).Error; err != nil {
			return err
		}
	}

	return nil
}

// Keep gorm's not-found error available to HTTP callers without string matching.
var ErrNotFound = gorm.ErrRecordNotFound

func removeContainer(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "--force", name)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err == nil {
		return nil
	}
	// A failed rm may simply mean --rm already removed a completed container.
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.ID}}").Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) != "" {
		return errors.New("container is still present")
	}
	return nil
}
