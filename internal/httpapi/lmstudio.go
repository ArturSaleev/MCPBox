package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"os/exec"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"MCPBox/internal/models"
)

type lmStudioLaunchResponse struct {
	ProjectID  uint   `json:"project_id"`
	ServerName string `json:"server_name"`
	Deeplink   string `json:"deeplink"`
}

func (s *Server) handleLaunchProjectLMStudio(w http.ResponseWriter, r *http.Request, project models.Project) {
	if project.IsPaused {
		writeError(w, http.StatusBadRequest, errors.New("project is paused"))
		return
	}

	if !s.projectConnectionReady(project) {
		writeError(w, http.StatusBadRequest, errors.New("project has no enabled MCP servers or connected knowledge bases configured"))
		return
	}

	serverName := lmStudioServerName(project)
	deeplink, err := buildLMStudioDeeplink(serverName, s.connectURL(r, project.Token))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(
		r.Context(),
		&project.ID,
		nil,
		"project_lmstudio_launch_prepared",
		clientActor(r),
		truncateDetail(fmt.Sprintf("server=%s | deeplink=%s", serverName, deeplink)),
	)

	if err := s.urlLauncher(deeplink); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.logAudit(r.Context(), &project.ID, nil, "project_lmstudio_launched", clientActor(r), truncateDetail(serverName))
	writeJSON(w, http.StatusOK, lmStudioLaunchResponse{
		ProjectID:  project.ID,
		ServerName: serverName,
		Deeplink:   deeplink,
	})
}

func buildLMStudioDeeplink(serverName, connectURL string) (string, error) {
	serverName = strings.TrimSpace(serverName)
	connectURL = strings.TrimSpace(connectURL)
	if serverName == "" {
		return "", errors.New("LM Studio server name is required")
	}
	if connectURL == "" {
		return "", errors.New("LM Studio connect URL is required")
	}

	configBytes, err := json.Marshal(map[string]string{"url": connectURL})
	if err != nil {
		return "", fmt.Errorf("marshal LM Studio config: %w", err)
	}

	query := neturl.Values{}
	query.Set("name", serverName)
	query.Set("config", base64.StdEncoding.EncodeToString(configBytes))

	return "lmstudio://add_mcp?" + query.Encode(), nil
}

func lmStudioServerName(project models.Project) string {
	slug := normalizeClientIntegrationName(project.Name)
	if slug == "" {
		slug = "project"
	}

	maxSlugRunes := 48
	if utf8.RuneCountInString(slug) > maxSlugRunes {
		slug = string([]rune(slug)[:maxSlugRunes])
		slug = strings.Trim(slug, "_")
	}

	return fmt.Sprintf("mcpbox_%d_%s", project.ID, slug)
}

func normalizeClientIntegrationName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(strings.ToValidUTF8(raw, "")))
	if raw == "" {
		return ""
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return strings.Trim(builder.String(), "_")
}

func launchExternalURL(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("target URL is required")
	}

	switch runtime.GOOS {
	case "darwin":
		if err := exec.Command("open", target).Start(); err != nil {
			return fmt.Errorf("open URL via Launch Services: %w", err)
		}
		return nil
	case "linux":
		candidates := [][]string{
			{"xdg-open", target},
			{"gio", "open", target},
		}
		for _, candidate := range candidates {
			path, err := execLookPath(candidate[0])
			if err != nil {
				continue
			}
			if runErr := exec.Command(path, candidate[1:]...).Start(); runErr == nil {
				return nil
			}
		}
		return errors.New("no supported URL opener found")
	case "windows":
		command := fmt.Sprintf("Start-Process %s", powerShellQuote(target))
		if err := exec.Command("powershell", "-NoProfile", "-Command", command).Start(); err != nil {
			return fmt.Errorf("open URL via PowerShell: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("automatic URL launch is not supported on %s", runtime.GOOS)
	}
}
