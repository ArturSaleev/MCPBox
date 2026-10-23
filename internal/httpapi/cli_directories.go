package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ArturSaleev/MCPBox/internal/cli"
	"github.com/ArturSaleev/MCPBox/internal/models"
)

func (s *Server) handleProjectCLIDirectories(w http.ResponseWriter, r *http.Request) {
	id, err := cliPathID(r, "projectID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		cliHTTPError(w, err)
		return
	}
	if r.Method == http.MethodPut {
		var policy cli.DirectoryPolicy
		if err := decodeJSON(r.Body, &policy); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := cli.ValidateDirectoryPolicy(&policy); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		directories, err := json.Marshal(policy.AllowedDirectories)
		if err != nil {
			cliHTTPError(w, err)
			return
		}
		err = s.store.DB().WithContext(r.Context()).Model(&models.Project{}).Where("id = ?", id).Updates(map[string]any{"cli_workdir_mode": policy.Mode, "cli_allowed_directories": string(directories)}).Error
		if err != nil {
			cliHTTPError(w, err)
			return
		}
		project.CLIWorkdirMode, project.CLIAllowedDirectories = policy.Mode, policy.AllowedDirectories
		s.logAudit(r.Context(), &id, nil, "cli_project_directories", clientActor(r), mustJSON(map[string]any{"mode": policy.Mode, "directory_count": len(policy.AllowedDirectories)}))
	}
	writeJSON(w, http.StatusOK, cli.ProjectDirectoryPolicy(project))
}
