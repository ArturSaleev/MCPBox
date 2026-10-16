package httpapi

import (
	"errors"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/ArturSaleev/MCPBox/internal/cli"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"gorm.io/gorm"
)

func (s *Server) registerCLIRoutes() {
	s.mux.HandleFunc("GET /api/cli/catalog", s.handleCLICatalog)
	s.mux.HandleFunc("POST /api/cli/catalog/sync", s.handleCLICatalog)
	s.mux.HandleFunc("POST /api/cli/catalog/{slug}/apply", s.handleApplyCLICatalog)
	s.mux.HandleFunc("GET /api/cli", s.handleListCLI)
	s.mux.HandleFunc("POST /api/cli", s.handleSaveCLI)
	s.mux.HandleFunc("GET /api/cli/{cliID}", s.handleGetCLI)
	s.mux.HandleFunc("PUT /api/cli/{cliID}", s.handleSaveCLI)
	s.mux.HandleFunc("DELETE /api/cli/{cliID}", s.handleDeleteCLI)
	s.mux.HandleFunc("GET /api/cli/{cliID}/commands", s.handleCLICommands)
	s.mux.HandleFunc("GET /api/projects/{projectID}/cli", s.handleProjectCLIList)
	s.mux.HandleFunc("PUT /api/projects/{projectID}/cli/{cliID}", s.handleProjectCLILink)
	s.mux.HandleFunc("DELETE /api/projects/{projectID}/cli/{cliID}", s.handleProjectCLILink)
	s.mux.HandleFunc("POST /api/projects/{projectID}/cli/execute", s.handleCLIExecute)
}
func cliPathID(r *http.Request, name string) (uint, error) {
	id, err := strconv.ParseUint(r.PathValue(name), 10, 32)
	if err != nil || id == 0 {
		return 0, errors.New("invalid " + name)
	}
	return uint(id), nil
}
func cliHTTPError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if errors.Is(err, gorm.ErrRecordNotFound) {
		code = http.StatusNotFound
	}
	if errors.Is(err, cli.ErrDenied) {
		code = http.StatusForbidden
	}
	writeError(w, code, err)
}
func (s *Server) handleListCLI(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCLIs(r.Context())
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	type availableCLI struct {
		models.CLI
		BinaryAvailable bool   `json:"binary_available"`
		BinaryPath      string `json:"binary_path,omitempty"`
	}
	response := make([]availableCLI, 0, len(items))
	for _, item := range items {
		path, err := exec.LookPath(item.Binary)
		response = append(response, availableCLI{CLI: item, BinaryAvailable: err == nil, BinaryPath: path})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": response})
}
func (s *Server) handleGetCLI(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "cliID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.store.GetCLI(r.Context(), id)
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) handleCLICommands(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "cliID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.store.GetCLI(r.Context(), id)
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": item.Operations})
}
func (s *Server) handleSaveCLI(w http.ResponseWriter, r *http.Request) {
	// IDs and timestamps cannot be selected by the caller.
	var req struct {
		Name           string                `json:"name"`
		Slug           string                `json:"slug"`
		Description    string                `json:"description"`
		Binary         string                `json:"binary"`
		IsEnabled      bool                  `json:"is_enabled"`
		Operations     []models.CLIOperation `json:"operations"`
		TimeoutSeconds int                   `json:"timeout_seconds"`
		MaxOutputBytes int                   `json:"max_output_bytes"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item := models.CLI{Name: req.Name, Slug: req.Slug, Description: req.Description, Binary: req.Binary, IsEnabled: req.IsEnabled, Operations: req.Operations, TimeoutSeconds: req.TimeoutSeconds, MaxOutputBytes: req.MaxOutputBytes}
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, err := cliPathID(r, "cliID")
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		existing, err := s.store.GetCLI(r.Context(), id)
		if err != nil {
			cliHTTPError(w, err)
			return
		}
		item.ID, item.CreatedAt = id, existing.CreatedAt
		status = http.StatusOK
	}
	if err := cli.Validate(&item); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.SaveCLI(r.Context(), &item); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	s.logAudit(r.Context(), nil, nil, "cli_registry_save", clientActor(r), mustJSON(map[string]any{"cli_id": item.ID, "slug": item.Slug}))
	writeJSON(w, status, item)
}
func (s *Server) handleDeleteCLI(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "cliID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.DeleteCLI(r.Context(), id); err != nil {
		cliHTTPError(w, err)
		return
	}
	s.logAudit(r.Context(), nil, nil, "cli_registry_delete", clientActor(r), mustJSON(map[string]any{"cli_id": id}))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (s *Server) handleProjectCLIList(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "projectID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	items, err := s.store.ListProjectCLIs(r.Context(), id, false)
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) handleProjectCLILink(w http.ResponseWriter, r *http.Request) {
	projectID, err := cliPathID(r, "projectID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cliID, err := cliPathID(r, "cliID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	enabled := r.Method == http.MethodPut
	if err := s.store.SetProjectCLI(r.Context(), projectID, cliID, enabled); err != nil {
		cliHTTPError(w, err)
		return
	}
	s.logAudit(r.Context(), &projectID, nil, "cli_project_permission", clientActor(r), mustJSON(map[string]any{"cli_id": cliID, "enabled": enabled}))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (s *Server) executeProjectCLI(r *http.Request, projectID uint, req cli.Request) (*cli.Result, error) {
	return s.executeCLI(r.Context(), projectID, req, clientActor(r))
}
func (s *Server) handleCLIExecute(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "projectID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req cli.Request
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.executeProjectCLI(r, id, req)
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCLICatalog(w http.ResponseWriter, r *http.Request) {
	service := cli.CatalogService{Store: s.store}
	var result *cli.CatalogStatus
	var err error
	if r.Method == http.MethodPost {
		result, err = service.Sync(r.Context())
	} else {
		result, err = service.Inspect(r.Context())
	}
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	if r.Method == http.MethodPost {
		s.logAudit(r.Context(), nil, nil, "cli_catalog_sync", clientActor(r), mustJSON(map[string]any{"added": result.Added, "source": result.Source, "version": result.Version}))
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) handleApplyCLICatalog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Revision string `json:"revision"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := (cli.CatalogService{Store: s.store}).Apply(r.Context(), r.PathValue("slug"), req.Revision); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	s.logAudit(r.Context(), nil, nil, "cli_catalog_apply", clientActor(r), mustJSON(map[string]any{"slug": r.PathValue("slug")}))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
