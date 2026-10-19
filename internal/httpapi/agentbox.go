package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

// Retired routes are explicit tombstones: /connect/runtime must never be
// interpreted as a project token after the PolicyBox adapter is removed.
func (s *Server) registerAgentBoxRoutes() {
	retired := func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusGone, errors.New("PolicyBox runtime has been removed; use the project's /mcp/<token> endpoint and bearer authentication"))
	}
	s.mux.HandleFunc("/connect/runtime", retired)
	s.mux.HandleFunc("/connect/runtime/", retired)
	s.mux.HandleFunc("/api/agentbox/bootstrap", retired)
	s.mux.HandleFunc("/api/pro/policybox/", retired)
	s.mux.HandleFunc("POST /api/agentbox/direct-bootstrap", s.handleAgentBoxDirectBootstrap)
}

// Direct AgentBox clients use the same project credentials and isolation gates
// as MCP connections. No caller-supplied identity becomes a trusted context.
func (s *Server) handleAgentBoxDirectBootstrap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MCPURL    string `json:"mcp_url"`
		PageURL   string `json:"page_url,omitempty"`
		PageTitle string `json:"page_title,omitempty"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	parsed, err := url.Parse(strings.TrimSpace(req.MCPURL))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid mcp_url"))
		return
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || (parts[0] != "mcp" && parts[0] != "connect") || parts[1] == "" || parts[1] == "runtime" {
		writeError(w, http.StatusBadRequest, errors.New("mcp_url must contain /mcp/<project-token> or /connect/<project-token>"))
		return
	}
	project, err := s.store.GetProjectByToken(r.Context(), parts[1])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}
	if _, ok := s.authorizeProjectConnection(w, r, project); !ok {
		return
	}
	prompt := strings.TrimSpace(project.Prompt)
	if prompt == "" {
		prompt = "You are an embedded project assistant. Use the available runtime context, stay concise, and ask follow-up questions only when required."
	}
	profiles := decodeProjectPromptProfiles(project.PromptProfilesJSON)
	defaultID := ""
	for _, profile := range profiles {
		if profile.IsDefault {
			defaultID = profile.ID
			break
		}
	}
	writeJSON(w, http.StatusOK, struct {
		Success         bool                          `json:"success"`
		ProjectName     string                        `json:"project_name"`
		ProjectToken    string                        `json:"project_token"`
		Prompt          string                        `json:"prompt"`
		PromptProfiles  []models.ProjectPromptProfile `json:"prompt_profiles,omitempty"`
		DefaultPromptID string                        `json:"default_prompt_profile_id,omitempty"`
	}{true, project.Name, project.Token, prompt, profiles, defaultID})
}

func (s *Server) handleMigratePolicyBox(w http.ResponseWriter, r *http.Request, projectID uint) {
	if err := s.store.MigratePolicyBoxProject(r.Context(), projectID); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logAudit(r.Context(), &projectID, nil, "project_policybox_migrated", clientActor(r), "bearer credential rotated; project paused for access review")
	writeJSON(w, http.StatusOK, s.projectStatus(r, *project))
}
