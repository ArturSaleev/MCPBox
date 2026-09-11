package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"gorm.io/gorm"
)

type createProjectOAuthClientRequest struct {
	Name        string `json:"name"`
	RedirectURI string `json:"redirect_uri"`
}

type projectOAuthClientResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) handleListProjectOAuthClients(w http.ResponseWriter, r *http.Request) {
	projectID, ok := requestPathUint(r, "projectID")
	if !ok || !s.projectExists(r, projectID) {
		http.NotFound(w, r)
		return
	}

	clients, err := s.store.ListProjectOAuthClients(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response := make([]projectOAuthClientResponse, 0, len(clients))
	for _, client := range clients {
		response = append(response, mapProjectOAuthClient(client))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateProjectOAuthClient(w http.ResponseWriter, r *http.Request) {
	projectID, ok := requestPathUint(r, "projectID")
	if !ok {
		http.NotFound(w, r)
		return
	}
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}
	if !project.BearerAuthEnabled {
		writeError(w, http.StatusBadRequest, errors.New("project authorization must be enabled before adding OAuth clients"))
		return
	}

	var req createProjectOAuthClientRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.RedirectURI = strings.TrimSpace(req.RedirectURI)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if len([]rune(req.Name)) > 255 {
		writeError(w, http.StatusBadRequest, errors.New("name is too long"))
		return
	}
	if !validOAuthRedirectURI(req.RedirectURI) {
		writeError(w, http.StatusBadRequest, errors.New("a valid OAuth callback URL is required"))
		return
	}
	existing, err := s.store.GetProjectOAuthClientByName(r.Context(), projectID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, errors.New("an OAuth client with this name already exists"))
		return
	}

	client := &models.ProjectOAuthClient{
		ProjectID:   projectID,
		Name:        req.Name,
		RedirectURI: req.RedirectURI,
		IsEnabled:   true,
	}
	if err := s.store.CreateProjectOAuthClient(r.Context(), client); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logAudit(r.Context(), &projectID, nil, "project_oauth_client_created", clientActor(r), client.Name)
	writeJSON(w, http.StatusCreated, mapProjectOAuthClient(*client))
}

func (s *Server) handleRevealProjectOAuthClientToken(w http.ResponseWriter, r *http.Request) {
	projectID, clientID, ok := projectOAuthClientPathIDs(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	client, err := s.store.GetProjectOAuthClient(r.Context(), projectID, clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		http.NotFound(w, r)
		return
	}
	s.logAudit(r.Context(), &projectID, nil, "project_oauth_client_token_revealed", clientActor(r), client.Name)
	writeJSON(w, http.StatusOK, map[string]string{"token": client.Token})
}

func (s *Server) handleEnableProjectOAuthClient(w http.ResponseWriter, r *http.Request) {
	s.handleSetProjectOAuthClientEnabled(w, r, true)
}

func (s *Server) handleDisableProjectOAuthClient(w http.ResponseWriter, r *http.Request) {
	s.handleSetProjectOAuthClientEnabled(w, r, false)
}

func (s *Server) handleSetProjectOAuthClientEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	projectID, clientID, ok := projectOAuthClientPathIDs(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := s.store.SetProjectOAuthClientEnabled(r.Context(), projectID, clientID, enabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	client, err := s.store.GetProjectOAuthClient(r.Context(), projectID, clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	action := "project_oauth_client_disabled"
	if enabled {
		action = "project_oauth_client_enabled"
	}
	s.logAudit(r.Context(), &projectID, nil, action, clientActor(r), client.Name)
	writeJSON(w, http.StatusOK, mapProjectOAuthClient(*client))
}

func (s *Server) handleDeleteProjectOAuthClient(w http.ResponseWriter, r *http.Request) {
	projectID, clientID, ok := projectOAuthClientPathIDs(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	client, err := s.store.GetProjectOAuthClient(r.Context(), projectID, clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if client == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteProjectOAuthClient(r.Context(), projectID, clientID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logAudit(r.Context(), &projectID, nil, "project_oauth_client_deleted", clientActor(r), client.Name)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func requestPathUint(r *http.Request, name string) (uint, bool) {
	raw := strings.TrimSpace(r.PathValue(name))
	value, err := strconv.ParseUint(raw, 10, 64)
	return uint(value), err == nil && value > 0
}

func projectOAuthClientPathIDs(r *http.Request) (uint, uint, bool) {
	projectID, projectOK := requestPathUint(r, "projectID")
	clientID, clientOK := requestPathUint(r, "clientID")
	return projectID, clientID, projectOK && clientOK
}

func (s *Server) projectExists(r *http.Request, projectID uint) bool {
	project, err := s.store.GetProject(r.Context(), projectID)
	return err == nil && project != nil
}

func validOAuthRedirectURI(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}

func mapProjectOAuthClient(client models.ProjectOAuthClient) projectOAuthClientResponse {
	return projectOAuthClientResponse{
		ID:        client.ID,
		Name:      client.Name,
		IsEnabled: client.IsEnabled,
		CreatedAt: client.CreatedAt.UTC().Format(time.RFC3339),
	}
}
