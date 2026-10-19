package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/skills"
)

func (s *Server) registerSkillRoutes() {
	s.skills = skills.New(s.store.DB(), filepath.Join(s.store.DataRoot(), "skill_store"))
	s.mux.HandleFunc("GET /api/skills", s.handleSkillsList)
	s.mux.HandleFunc("GET /api/skills/runtime", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.skills.RuntimeStatus(r.Context())) })
	s.mux.HandleFunc("GET /api/skills/catalog", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.skills.Catalog(r.Context())
		skillResponse(w, v, e)
	})
	s.mux.HandleFunc("POST /api/skills/catalog/sync", s.handleSkillCatalogSync)
	s.mux.HandleFunc("POST /api/skills/import", s.handleSkillImport)
	s.mux.HandleFunc("POST /api/skills/install", s.handleSkillInstall)
	s.mux.HandleFunc("GET /api/skills/{skillID}/file", s.handleSkillFile)
	s.mux.HandleFunc("PUT /api/skills/{skillID}/profile", s.handleSkillProfile)
	s.mux.HandleFunc("POST /api/skills/{skillID}/prepare", s.handleSkillPrepare)
	s.mux.HandleFunc("DELETE /api/skills/{skillID}", func(w http.ResponseWriter, r *http.Request) {
		skillResponse(w, map[string]bool{"ok": true}, s.skills.Delete(r.Context(), r.PathValue("skillID")))
	})
	s.mux.HandleFunc("GET /api/projects/{projectID}/skills", s.handleProjectSkills)
	s.mux.HandleFunc("PUT /api/projects/{projectID}/skills/{skillID}", s.handleSkillBinding)
	s.mux.HandleFunc("DELETE /api/projects/{projectID}/skills/{skillID}", s.handleSkillBinding)
	s.mux.HandleFunc("GET /api/projects/{projectID}/skill-jobs", s.handleSkillJobs)
	s.mux.HandleFunc("POST /api/projects/{projectID}/skill-jobs", s.handleSkillJobs)
	s.mux.HandleFunc("GET /api/projects/{projectID}/skill-jobs/{jobID}", s.handleSkillJob)
	s.mux.HandleFunc("DELETE /api/projects/{projectID}/skill-jobs/{jobID}", s.handleSkillJob)
	s.mux.HandleFunc("POST /api/projects/{projectID}/skill-jobs/{jobID}/files", s.handleSkillJobUpload)
	s.mux.HandleFunc("GET /api/projects/{projectID}/skill-jobs/{jobID}/file", s.handleSkillJobDownload)
	s.mux.HandleFunc("POST /api/projects/{projectID}/skill-jobs/{jobID}/run", s.handleSkillJobRun)
}
func skillResponse(w http.ResponseWriter, value any, err error) {
	if err != nil {
		status := 400
		if errors.Is(err, skills.ErrNotFound) {
			status = 404
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, 200, value)
}
func skillDecode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}
func skillProjectID(r *http.Request) (uint, error) {
	v, err := strconv.ParseUint(r.PathValue("projectID"), 10, 32)
	if err != nil || v == 0 {
		return 0, errors.New("invalid project ID")
	}
	return uint(v), nil
}
func (s *Server) handleSkillsList(w http.ResponseWriter, r *http.Request) {
	v, e := s.skills.List(r.Context(), 0)
	skillResponse(w, map[string]any{"items": v}, e)
}
func (s *Server) handleSkillCatalogSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string          `json:"url"`
		Content json.RawMessage `json:"content"`
	}
	if e := skillDecode(w, r, &req); e != nil {
		skillResponse(w, nil, e)
		return
	}
	if len(req.Content) > 0 {
		v, e := s.skills.ImportCatalog(r.Context(), req.Content, "local-file")
		skillResponse(w, v, e)
	} else {
		v, e := s.skills.SyncCatalog(r.Context(), req.URL)
		skillResponse(w, v, e)
	}
}
func (s *Server) handleSkillInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CatalogID string         `json:"catalog_id"`
		Source    *skills.Source `json:"source,omitempty"`
	}
	if e := skillDecode(w, r, &req); e != nil {
		skillResponse(w, nil, e)
		return
	}
	var entry skills.Entry
	if req.Source != nil {
		entry.Source = *req.Source
	} else {
		cache, e := s.skills.Catalog(r.Context())
		if e != nil {
			skillResponse(w, nil, e)
			return
		}
		for _, item := range cache.Content.Items {
			if item.ID == req.CatalogID {
				entry = item
				break
			}
		}
		if entry.ID == "" {
			skillResponse(w, nil, errors.New("catalog item not found"))
			return
		}
	}
	v, e := s.skills.Install(r.Context(), entry)
	if e == nil {
		s.logAudit(r.Context(), nil, nil, "skill_install", clientActor(r), mustJSON(map[string]any{"skill_id": v.ID, "commit": v.Source.Commit}))
	}
	skillResponse(w, v, e)
}
func (s *Server) handleSkillImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, skills.MaxPackageBytes+(1<<20))
	if e := r.ParseMultipartForm(1 << 20); e != nil {
		skillResponse(w, nil, e)
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, e := r.FormFile("file")
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, skills.MaxPackageBytes+1))
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	v, e := s.skills.ImportZIP(r.Context(), data, r.FormValue("path"))
	skillResponse(w, v, e)
}
func (s *Server) handleSkillFile(w http.ResponseWriter, r *http.Request) {
	data, e := s.skills.ReadPackageFile(r.Context(), r.PathValue("skillID"), r.URL.Query().Get("path"))
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	writeJSON(w, 200, map[string]string{"path": r.URL.Query().Get("path"), "content": string(data)})
}
func (s *Server) handleSkillProfile(w http.ResponseWriter, r *http.Request) {
	var p skills.Profile
	if e := skillDecode(w, r, &p); e != nil {
		skillResponse(w, nil, e)
		return
	}
	v, e := s.skills.SaveProfile(r.Context(), r.PathValue("skillID"), p)
	if e == nil {
		s.logAudit(r.Context(), nil, nil, "skill_profile", clientActor(r), mustJSON(map[string]any{"skill_id": v.ID, "enabled": p.Enabled, "network": p.Network}))
	}
	skillResponse(w, v, e)
}
func (s *Server) handleSkillPrepare(w http.ResponseWriter, r *http.Request) {
	v, e := s.skills.Prepare(r.Context(), r.PathValue("skillID"))
	if e == nil {
		s.logAudit(r.Context(), nil, nil, "skill_prepare", clientActor(r), mustJSON(map[string]string{"skill_id": v.ID}))
	}
	skillResponse(w, v, e)
}
func (s *Server) handleProjectSkills(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	if _, e = s.store.GetProject(r.Context(), p); e != nil {
		skillResponse(w, nil, e)
		return
	}
	v, e := s.skills.List(r.Context(), p)
	skillResponse(w, map[string]any{"items": v}, e)
}
func (s *Server) handleSkillBinding(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e == nil {
		var changed bool
		changed, e = s.skills.SetBinding(r.Context(), p, r.PathValue("skillID"), r.Method == http.MethodPut)
		if e == nil && changed {
			s.notifyProjectToolsChanged(p)
		}
	}
	skillResponse(w, map[string]bool{"ok": e == nil}, e)
}
func (s *Server) handleSkillJobs(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	if r.Method == http.MethodGet {
		v, e := s.skills.ListJobs(r.Context(), p)
		skillResponse(w, map[string]any{"items": v}, e)
		return
	}
	var req struct {
		SkillID string `json:"skill_id"`
	}
	if e = skillDecode(w, r, &req); e != nil {
		skillResponse(w, nil, e)
		return
	}
	v, e := s.skills.NewJob(r.Context(), p, req.SkillID)
	skillResponse(w, v, e)
}
func (s *Server) handleSkillJob(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	if r.Method == http.MethodDelete {
		skillResponse(w, map[string]bool{"ok": true}, s.skills.DeleteJob(r.Context(), p, r.PathValue("jobID")))
		return
	}
	v, e := s.skills.GetJob(r.Context(), p, r.PathValue("jobID"))
	skillResponse(w, v, e)
}
func (s *Server) handleSkillJobUpload(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, skills.MaxFileBytes+(1<<20))
	if e = r.ParseMultipartForm(1 << 20); e != nil {
		skillResponse(w, nil, e)
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, h, e := r.FormFile("file")
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	defer f.Close()
	v, e := s.skills.Upload(r.Context(), p, r.PathValue("jobID"), h.Filename, f)
	skillResponse(w, v, e)
}
func (s *Server) handleSkillJobDownload(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	name := r.URL.Query().Get("path")
	data, e := s.skills.ReadJobFile(r.Context(), p, r.PathValue("jobID"), name)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(name)}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(data))
}
func (s *Server) handleSkillJobRun(w http.ResponseWriter, r *http.Request) {
	p, e := skillProjectID(r)
	if e != nil {
		skillResponse(w, nil, e)
		return
	}
	var req struct {
		Script string   `json:"script"`
		Args   []string `json:"args"`
	}
	if e = skillDecode(w, r, &req); e != nil {
		skillResponse(w, nil, e)
		return
	}
	v, e := s.runSkillJob(r.Context(), p, r.PathValue("jobID"), req.Script, req.Args, clientActor(r))
	skillResponse(w, v, e)
}
