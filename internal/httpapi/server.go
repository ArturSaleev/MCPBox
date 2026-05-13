package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

type Server struct {
	store    *storage.Store
	registry *orchestrator.Registry
	mux      *http.ServeMux
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addServerRequest struct {
	Name          string `json:"name"`
	LaunchCommand string `json:"launch_command"`
	WorkingDir    string `json:"working_dir"`
	AutoStart     bool   `json:"auto_start"`
}

type projectStatusResponse struct {
	ProjectID uint                 `json:"project_id"`
	Name      string               `json:"name"`
	Token     string               `json:"token"`
	Servers   []serverStatusRecord `json:"servers"`
}

type serverStatusRecord struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	LaunchCommand string `json:"launch_command"`
	Status        string `json:"status"`
}

func NewServer(store *storage.Store, registry *orchestrator.Registry) *Server {
	s := &Server{
		store:    store,
		registry: registry,
		mux:      http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects/", s.handleProjectStatus)
	s.mux.HandleFunc("POST /api/projects/", s.handleAddServer)
	s.mux.HandleFunc("POST /api/servers/", s.handleServerAction)
	s.mux.HandleFunc("/connect/", s.handleConnect)
	s.mux.Handle("/", s.handleUI())
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	project := &models.Project{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	}
	if project.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	if err := s.store.CreateProject(r.Context(), project); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := make([]projectStatusResponse, 0, len(projects))
	for _, project := range projects {
		response = append(response, s.projectStatus(project))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleProjectStatus(w http.ResponseWriter, r *http.Request) {
	projectID, tail, ok := parseIDTail(r.URL.Path, "/api/projects/")
	if !ok || tail != "status" || r.Method != http.MethodGet {
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

	writeJSON(w, http.StatusOK, s.projectStatus(*project))
}

func (s *Server) handleAddServer(w http.ResponseWriter, r *http.Request) {
	projectID, tail, ok := parseIDTail(r.URL.Path, "/api/projects/")
	if !ok || tail != "servers" || r.Method != http.MethodPost {
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

	var req addServerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	server := &models.MCPServer{
		ProjectID:     projectID,
		Name:          strings.TrimSpace(req.Name),
		LaunchCommand: strings.TrimSpace(req.LaunchCommand),
		WorkingDir:    strings.TrimSpace(req.WorkingDir),
		AutoStart:     req.AutoStart,
	}
	if server.Name == "" || server.LaunchCommand == "" {
		writeError(w, http.StatusBadRequest, errors.New("name and launch_command are required"))
		return
	}

	if err := s.store.AddServer(r.Context(), server); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if server.AutoStart {
		if err := s.registry.StartServer(r.Context(), *server); err != nil {
			log.Printf("auto-start after create failed for server %d: %v", server.ID, err)
		}
	}

	writeJSON(w, http.StatusCreated, server)
}

func (s *Server) handleServerAction(w http.ResponseWriter, r *http.Request) {
	serverID, tail, ok := parseIDTail(r.URL.Path, "/api/servers/")
	if !ok || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	server, err := s.store.GetServer(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if server == nil {
		http.NotFound(w, r)
		return
	}

	switch tail {
	case "start":
		err = s.registry.StartServer(r.Context(), *server)
	case "stop":
		stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		err = s.registry.StopServer(stopCtx, server.ID)
	default:
		http.NotFound(w, r)
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"server_id": server.ID,
		"status":    s.registry.Status(server.ID),
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	token := strings.Trim(strings.TrimPrefix(path.Clean(r.URL.Path), "/connect/"), "/")
	if token == "" || token == "." {
		http.NotFound(w, r)
		return
	}

	project, err := s.store.GetProjectByToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}

	runner, _, err := s.registry.RunnerForProject(r.Context(), *project)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.serveSSE(w, r, runner)
	case http.MethodPost:
		s.forwardJSONRPC(w, r, runner)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request, runner *orchestrator.ServerRunner) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	stream, unsubscribe := runner.Subscribe()
	defer unsubscribe()

	fmt.Fprint(w, "event: ready\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-stream:
			if !ok {
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) forwardJSONRPC(w http.ResponseWriter, r *http.Request, runner *orchestrator.ServerRunner) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var raw json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("body must be valid JSON-RPC payload"))
		return
	}

	sendCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := runner.Send(sendCtx, payload); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "forwarded"})
}

func (s *Server) handleUI() http.Handler {
	fileServer := http.FileServer(http.FS(uiFS()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/connect/") {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, uiFS(), "index.html")
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) projectStatus(project models.Project) projectStatusResponse {
	response := projectStatusResponse{
		ProjectID: project.ID,
		Name:      project.Name,
		Token:     project.Token,
		Servers:   make([]serverStatusRecord, 0, len(project.Servers)),
	}

	for _, server := range project.Servers {
		response.Servers = append(response.Servers, serverStatusRecord{
			ID:            server.ID,
			Name:          server.Name,
			LaunchCommand: server.LaunchCommand,
			Status:        s.registry.Status(server.ID),
		})
	}

	return response
}

func parseIDTail(rawPath, prefix string) (uint, string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return 0, "", false
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}

	return uint(id), parts[1], true
}

func decodeJSON(body io.ReadCloser, target any) error {
	defer body.Close()
	decoder := json.NewDecoder(io.LimitReader(body, 1024*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
