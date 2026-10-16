package prohttp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArturSaleev/MCPBox/connectruntime"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/propolicy"
	"gorm.io/gorm"
)

const runtimeConnectPath = "/connect/runtime"
const agentBoxBootstrapPath = "/api/agentbox/bootstrap"
const agentBoxDirectBootstrapPath = "/api/agentbox/direct-bootstrap"
const policyBoxRuntimeHeader = "X-PolicyBox-Runtime"

type agentBoxBootstrapRequest struct {
	PageURL   string `json:"page_url,omitempty"`
	PageTitle string `json:"page_title,omitempty"`
}

type agentBoxDirectBootstrapRequest struct {
	MCPURL    string `json:"mcp_url"`
	PageURL   string `json:"page_url,omitempty"`
	PageTitle string `json:"page_title,omitempty"`
}

type agentBoxSummaryBlock struct {
	Key     string   `json:"key"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Sources []string `json:"sources,omitempty"`
}

type agentBoxActionDefinition struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind"`
	Prompt      string `json:"prompt,omitempty"`
}

type agentBoxPromptProfile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Prompt         string `json:"prompt"`
	ResponseFormat string `json:"response_format,omitempty"`
	ResponseSchema string `json:"response_schema,omitempty"`
	IsDefault      bool   `json:"is_default,omitempty"`
}

type agentBoxToolLimits struct {
	TimeoutMS         int `json:"timeout_ms,omitempty"`
	MaxResultBytes    int `json:"max_result_bytes,omitempty"`
	MaxCallsPerMinute int `json:"max_calls_per_minute,omitempty"`
}

type agentBoxToolManifest struct {
	Name                string             `json:"name"`
	Version             int                `json:"version"`
	Description         string             `json:"description,omitempty"`
	Risk                string             `json:"risk"`
	Confirmation        string             `json:"confirmation"`
	RequiredPermissions []string           `json:"required_permissions,omitempty"`
	InputSchema         json.RawMessage    `json:"input_schema"`
	Limits              agentBoxToolLimits `json:"limits"`
}

type agentBoxRuntimeAccess struct {
	SessionID   string            `json:"session_id"`
	ProjectID   string            `json:"project_id"`
	ProjectSlug string            `json:"project_slug"`
	SubjectID   string            `json:"subject_id"`
	Role        string            `json:"role"`
	Scopes      []string          `json:"scopes"`
	Context     map[string]string `json:"context"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type agentBoxBootstrapResponse struct {
	Success           bool                       `json:"success"`
	AgentName         string                     `json:"agent_name"`
	ProjectName       string                     `json:"project_name"`
	ProjectToken      string                     `json:"project_token"`
	Prompt            string                     `json:"prompt"`
	ConnectURL        string                     `json:"connect_url"`
	ProviderPolicy    *agentBoxProviderPolicy    `json:"provider_policy,omitempty"`
	SummaryBlocks     []agentBoxSummaryBlock     `json:"summary_blocks,omitempty"`
	RecommendedHints  []string                   `json:"recommended_hints,omitempty"`
	ActionDefinitions []agentBoxActionDefinition `json:"action_definitions,omitempty"`
	PromptProfiles    []agentBoxPromptProfile    `json:"prompt_profiles,omitempty"`
	DefaultPromptID   string                     `json:"default_prompt_profile_id,omitempty"`
	ConfigVersion     string                     `json:"config_version,omitempty"`
	ToolManifests     []agentBoxToolManifest     `json:"tool_manifests,omitempty"`
	RuntimeAccess     agentBoxRuntimeAccess      `json:"runtime_access"`
}

type agentBoxProviderPolicy struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type sharedProjectBootstrapRow struct {
	ID                 uint   `gorm:"column:id"`
	Name               string `gorm:"column:name"`
	Description        string `gorm:"column:description"`
	Token              string `gorm:"column:token"`
	Prompt             string `gorm:"column:prompt"`
	PromptProfilesJSON string `gorm:"column:prompt_profiles_json"`
	LlamaCppModelPath  string `gorm:"column:llama_cpp_model_path"`
	LlamaCppModelName  string `gorm:"column:llama_cpp_model_name"`
	IsPaused           bool   `gorm:"column:is_paused"`
}

func (sharedProjectBootstrapRow) TableName() string {
	return "projects"
}

func ProjectAuthorizer(policy *propolicy.Service) connectruntime.ProjectAuthorizer {
	return func(r *http.Request, project connectruntime.Project) (*connectruntime.Access, error) {
		if !project.IdentityVerification {
			return nil, nil
		}
		if policy == nil || !policy.Enabled() {
			userID := policyBoxRuntimeToken(r)
			if userID == "" {
				return nil, &connectruntime.AuthorizationError{StatusCode: http.StatusUnauthorized, Message: "missing test user id"}
			}
			return buildInsecureRuntimeAccess(project, userID), nil
		}

		token := policyBoxRuntimeToken(r)
		if token == "" {
			return nil, &connectruntime.AuthorizationError{
				StatusCode: http.StatusUnauthorized,
				Message:    "missing runtime bearer token",
			}
		}

		validated, err := policy.ValidateRuntimeToken(r.Context(), token)
		if err != nil {
			return nil, &connectruntime.AuthorizationError{
				StatusCode: http.StatusUnauthorized,
				Message:    err.Error(),
			}
		}

		resolvedProject, err := policy.ResolveProject(r.Context(), validated)
		if err != nil {
			return nil, &connectruntime.AuthorizationError{
				StatusCode: http.StatusForbidden,
				Message:    "runtime project is not mapped to MCPBox",
			}
		}
		if resolvedProject.ID != project.ID {
			return nil, &connectruntime.AuthorizationError{
				StatusCode: http.StatusForbidden,
				Message:    "runtime token does not grant access to this project",
			}
		}

		access := buildRuntimeAccess(validated)
		if access != nil {
			access.PublicConnectPath = runtimeConnectPath
		}
		return access, nil
	}
}

func registerPolicyBoxRuntimeRoute(mux *http.ServeMux, policy *propolicy.Service) {
	if mux == nil || policy == nil {
		return
	}

	mux.HandleFunc("POST "+agentBoxDirectBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if !policy.Enabled() {
			if policyBoxRuntimeToken(r) == "" {
				proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing test user id"})
				return
			}
			var request agentBoxDirectBootstrapRequest
			if r.Body != nil {
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
					proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
					return
				}
			}
			record, err := loadProjectBootstrapRecordByToken(r.Context(), policy, projectTokenFromDirectMCPURL(request.MCPURL))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, gorm.ErrRecordNotFound) {
					status = http.StatusNotFound
				}
				proauth.WriteJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			if record.IsPaused {
				proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "project is paused"})
				return
			}
			proauth.WriteJSON(w, http.StatusOK, buildAgentBoxDirectBootstrapResponse(record))
			return
		}
		token := policyBoxRuntimeToken(r)
		if token == "" {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing runtime bearer token"})
			return
		}
		validated, err := policy.ValidateRuntimeToken(r.Context(), token)
		if err != nil {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "runtime token is invalid"})
			return
		}
		resolvedProject, err := policy.ResolveProject(r.Context(), validated)
		if err != nil {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "runtime project is not mapped to MCPBox"})
			return
		}
		var request agentBoxDirectBootstrapRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
				proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
				return
			}
		}

		projectToken := projectTokenFromDirectMCPURL(request.MCPURL)
		if projectToken == "" {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "mcp_url does not contain a project token"})
			return
		}

		record, err := loadProjectBootstrapRecordByToken(r.Context(), policy, projectToken)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusNotFound
			}
			proauth.WriteJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		if record.IsPaused {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "project is paused"})
			return
		}
		if record.ID != resolvedProject.ID {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "runtime token does not grant access to this project"})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, buildAgentBoxDirectBootstrapResponse(record))
	})

	if !policy.Enabled() {
		return
	}

	mux.HandleFunc(runtimeConnectPath, func(w http.ResponseWriter, r *http.Request) {
		token := policyBoxRuntimeToken(r)
		if token == "" {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing runtime bearer token"})
			return
		}

		validated, err := policy.ValidateRuntimeToken(r.Context(), token)
		if err != nil {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		project, err := policy.ResolveProject(r.Context(), validated)
		if err != nil {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "runtime project is not mapped to MCPBox"})
			return
		}

		rewrittenURL := cloneURL(r.URL)
		rewrittenURL.Path = "/mcp/" + project.Token
		rewrittenURL.RawPath = rewrittenURL.Path

		access := buildRuntimeAccess(validated)
		access.PublicConnectPath = runtimeConnectPath

		rewrittenRequest := r.Clone(connectruntime.WithAccess(r.Context(), access))
		rewrittenRequest.URL = rewrittenURL
		rewrittenRequest.RequestURI = rewrittenURL.RequestURI()

		mux.ServeHTTP(w, rewrittenRequest)
	})

	mux.HandleFunc("POST "+agentBoxBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		token := policyBoxRuntimeToken(r)
		if token == "" {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing runtime bearer token"})
			return
		}

		validated, err := policy.ValidateRuntimeToken(r.Context(), token)
		if err != nil {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		project, err := policy.ResolveProject(r.Context(), validated)
		if err != nil {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "runtime project is not mapped to MCPBox"})
			return
		}

		record, err := loadProjectBootstrapRecord(r.Context(), policy, project.ID)
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if record.IsPaused {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "project is paused"})
			return
		}

		var request agentBoxBootstrapRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
				proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
				return
			}
		}

		access := buildRuntimeAccess(validated)
		if access == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "runtime access could not be constructed"})
			return
		}
		access.PublicConnectPath = runtimeConnectPath
		manifestSet, err := policy.PublishedToolManifests(r.Context(), project.ID)
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "published tool policy could not be loaded"})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, buildAgentBoxBootstrapResponse(r, request, record, access, manifestSet))
	})
}

// buildInsecureRuntimeAccess is the temporary no-PolicyBox path. The bearer
// value is deliberately treated as a plain user ID and must never be enabled
// for an internet-facing installation: a caller can choose any value.
func buildInsecureRuntimeAccess(project connectruntime.Project, userID string) *connectruntime.Access {
	userID = strings.TrimSpace(userID)
	projectID := fmt.Sprintf("%d", project.ID)
	payload := map[string]any{
		"project_id": projectID,
		"subject_id": userID,
		"role":       "operator",
		"scopes":     []string{"chat:open", "chat:message", "context:read", "employee_data:read"},
		"context":    map[string]string{"user_id": userID},
		"data":       map[string]any{"user_id": userID},
	}
	payloadJSON, _ := json.Marshal(payload)
	headers := map[string]string{
		"X-MCPBox-Runtime-Project-ID": projectID,
		"X-MCPBox-Runtime-Subject-ID": userID,
		"X-MCPBox-Runtime-Role":       "operator",
		"X-MCPBox-Runtime-Scopes":     "chat:open,chat:message,context:read,employee_data:read",
		"X-MCPBox-Runtime-Context":    string(payloadJSON),
	}
	if signature := signRuntimeContext(payloadJSON); signature != "" {
		headers["X-MCPBox-Runtime-Signature"] = signature
	}
	return &connectruntime.Access{Actor: "agentbox-test:" + userID, ProjectID: projectID, SubjectID: userID, Role: "operator", Scopes: []string{"chat:open", "chat:message", "context:read", "employee_data:read"}, Context: map[string]string{"user_id": userID}, UpstreamHeaders: headers}
}

func buildRuntimeAccess(validated *propolicy.ValidateTokenResponse) *connectruntime.Access {
	if validated == nil {
		return nil
	}

	contextMap := map[string]string{}
	for key, value := range validated.Session.Context.Context {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			contextMap[key] = value
		}
	}
	for key, value := range validated.Session.Context.User.Attributes {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			contextMap["user_attribute."+key] = value
		}
	}

	payload := map[string]any{
		"project_id":   strings.TrimSpace(validated.Session.Context.ProjectID),
		"project_slug": strings.TrimSpace(validated.Session.Context.ProjectSlug),
		"session_id":   strings.TrimSpace(validated.Session.ID),
		"subject_id":   strings.TrimSpace(validated.Claims.Subject),
		"role":         strings.TrimSpace(validated.Session.Context.User.Role),
		"scopes":       append([]string(nil), validated.Session.Context.GrantedScopes...),
		"context":      contextMap,
		"data":         validated.Session.Context.Data,
		"metadata":     validated.Session.Context.Metadata,
	}
	payloadJSON, _ := json.Marshal(payload)

	upstreamHeaders := map[string]string{
		"X-MCPBox-Runtime-Project-ID":   strings.TrimSpace(validated.Session.Context.ProjectID),
		"X-MCPBox-Runtime-Project-Slug": strings.TrimSpace(validated.Session.Context.ProjectSlug),
		"X-MCPBox-Runtime-Session-ID":   strings.TrimSpace(validated.Session.ID),
		"X-MCPBox-Runtime-Subject-ID":   strings.TrimSpace(validated.Claims.Subject),
		"X-MCPBox-Runtime-Role":         strings.TrimSpace(validated.Session.Context.User.Role),
		"X-MCPBox-Runtime-Scopes":       strings.Join(validated.Session.Context.GrantedScopes, ","),
		"X-MCPBox-Runtime-Context":      string(payloadJSON),
	}
	if signature := signRuntimeContext(payloadJSON); signature != "" {
		upstreamHeaders["X-MCPBox-Runtime-Signature"] = signature
	}

	return &connectruntime.Access{
		Actor:           "policybox:" + strings.TrimSpace(validated.Claims.Subject),
		ProjectID:       strings.TrimSpace(validated.Session.Context.ProjectID),
		ProjectSlug:     strings.TrimSpace(validated.Session.Context.ProjectSlug),
		SessionID:       strings.TrimSpace(validated.Session.ID),
		SubjectID:       strings.TrimSpace(validated.Claims.Subject),
		Role:            strings.TrimSpace(validated.Session.Context.User.Role),
		Scopes:          append([]string(nil), validated.Session.Context.GrantedScopes...),
		Context:         contextMap,
		Metadata:        validated.Session.Context.Metadata,
		UpstreamHeaders: upstreamHeaders,
	}
}

func policyBoxRuntimeToken(r *http.Request) string {
	if r != nil {
		if token := strings.TrimSpace(r.Header.Get(policyBoxRuntimeHeader)); token != "" {
			return token
		}
	}
	return bearerTokenFromRequest(r)
}

// The shared secret is configured only in MCPBoxPro and user-filtering MCPs.
// A missing secret intentionally means no signature is emitted; MCPs that
// require policy data must fail closed in that case.
func signRuntimeContext(payload []byte) string {
	secret := strings.TrimSpace(os.Getenv("MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET"))
	if secret == "" || len(payload) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func cloneURL(source *url.URL) *url.URL {
	if source == nil {
		return &url.URL{}
	}
	cloned := *source
	return &cloned
}

func loadProjectBootstrapRecord(ctx context.Context, policy *propolicy.Service, projectID uint) (*sharedProjectBootstrapRow, error) {
	if policy == nil || policy.DB() == nil {
		return nil, errors.New("policy runtime database is not configured")
	}
	if projectID == 0 {
		return nil, errors.New("project id is required")
	}

	var record sharedProjectBootstrapRow
	if err := policy.DB().WithContext(ctx).First(&record, "id = ?", projectID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func loadProjectBootstrapRecordByToken(ctx context.Context, policy *propolicy.Service, projectToken string) (*sharedProjectBootstrapRow, error) {
	if policy == nil || policy.DB() == nil {
		return nil, errors.New("policy runtime database is not configured")
	}
	projectToken = strings.TrimSpace(projectToken)
	if projectToken == "" {
		return nil, errors.New("project token is required")
	}

	var record sharedProjectBootstrapRow
	if err := policy.DB().WithContext(ctx).First(&record, "token = ?", projectToken).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func decodeAgentBoxPromptProfiles(raw string) []agentBoxPromptProfile {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var profiles []agentBoxPromptProfile
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
		return nil
	}
	normalized := make([]agentBoxPromptProfile, 0, len(profiles))
	defaultID := ""
	for _, profile := range profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Description = strings.TrimSpace(profile.Description)
		profile.Prompt = strings.TrimSpace(profile.Prompt)
		profile.ResponseFormat = strings.TrimSpace(strings.ToLower(profile.ResponseFormat))
		profile.ResponseSchema = strings.TrimSpace(profile.ResponseSchema)
		if profile.Name == "" || profile.Prompt == "" {
			continue
		}
		if profile.ID == "" {
			profile.ID = strings.ToLower(strings.ReplaceAll(profile.Name, " ", "-"))
		}
		if profile.ResponseFormat == "" {
			profile.ResponseFormat = "text"
		}
		if profile.IsDefault && defaultID == "" {
			defaultID = profile.ID
		} else {
			profile.IsDefault = false
		}
		normalized = append(normalized, profile)
	}
	return normalized
}

func buildAgentBoxBootstrapResponse(r *http.Request, request agentBoxBootstrapRequest, project *sharedProjectBootstrapRow, access *connectruntime.Access, manifestSet *propolicy.PublishedToolManifestSet) agentBoxBootstrapResponse {
	projectName := strings.TrimSpace(project.Name)
	if projectName == "" {
		projectName = strings.TrimSpace(access.ProjectSlug)
	}

	prompt := strings.TrimSpace(project.Prompt)
	if prompt == "" {
		prompt = "You are an embedded project assistant. Use the available runtime context, stay concise, and ask follow-up questions only when required."
	}

	summaryBlocks := []agentBoxSummaryBlock{
		{
			Key:     "project",
			Title:   "Project",
			Content: fmt.Sprintf("Connected to MCPBox project %s.", projectName),
			Sources: []string{"mcpboxpro-project"},
		},
		{
			Key:     "subject",
			Title:   "Current Subject",
			Content: fmt.Sprintf("Runtime subject %s is working in role %s.", strings.TrimSpace(access.SubjectID), strings.TrimSpace(access.Role)),
			Sources: []string{"policybox-runtime"},
		},
	}

	if description := strings.TrimSpace(project.Description); description != "" {
		summaryBlocks = append(summaryBlocks, agentBoxSummaryBlock{
			Key:     "project_description",
			Title:   "Project Description",
			Content: description,
			Sources: []string{"mcpboxpro-project"},
		})
	}
	if pageTitle := strings.TrimSpace(request.PageTitle); pageTitle != "" {
		summaryBlocks = append(summaryBlocks, agentBoxSummaryBlock{
			Key:     "page",
			Title:   "Current Page",
			Content: pageTitle,
			Sources: []string{"host-page"},
		})
	}
	if pageURL := strings.TrimSpace(request.PageURL); pageURL != "" {
		summaryBlocks = append(summaryBlocks, agentBoxSummaryBlock{
			Key:     "page_url",
			Title:   "Page URL",
			Content: pageURL,
			Sources: []string{"host-page"},
		})
	}

	promptProfiles := decodeAgentBoxPromptProfiles(project.PromptProfilesJSON)
	defaultPromptID := ""
	for _, profile := range promptProfiles {
		if profile.IsDefault {
			defaultPromptID = profile.ID
			break
		}
	}

	var toolManifests []agentBoxToolManifest
	configVersion := ""
	if manifestSet != nil && json.Unmarshal([]byte(manifestSet.ManifestsJSON), &toolManifests) == nil {
		configVersion = strings.TrimSpace(manifestSet.ConfigVersion)
	}

	return agentBoxBootstrapResponse{
		Success:        true,
		AgentName:      "AgentBox",
		ProjectName:    projectName,
		ProjectToken:   strings.TrimSpace(project.Token),
		Prompt:         prompt,
		ConnectURL:     publicURLForRequest(r, runtimeConnectPath),
		ProviderPolicy: buildAgentBoxProviderPolicy(project),
		SummaryBlocks:  summaryBlocks,
		RecommendedHints: []string{
			"Summarize the current runtime context",
			"What should I review before starting?",
			"Which MCP data should I load next?",
		},
		ActionDefinitions: []agentBoxActionDefinition{
			{
				ID:          "runtime-summary",
				Label:       "Runtime Summary",
				Description: "Generate a concise project-aware summary before proceeding.",
				Kind:        "prompt",
				Prompt:      "Summarize the current runtime context, the current page, and the most important project details before any action is taken.",
			},
			{
				ID:          "review-checklist",
				Label:       "Review Checklist",
				Description: "Prepare a short checklist for the operator.",
				Kind:        "prompt",
				Prompt:      "Create a short checklist of what the operator should review next, using the current runtime context and project summary.",
			},
		},
		PromptProfiles:  promptProfiles,
		DefaultPromptID: defaultPromptID,
		ConfigVersion:   configVersion,
		ToolManifests:   toolManifests,
		RuntimeAccess: agentBoxRuntimeAccess{
			SessionID:   strings.TrimSpace(access.SessionID),
			ProjectID:   strings.TrimSpace(access.ProjectID),
			ProjectSlug: strings.TrimSpace(access.ProjectSlug),
			SubjectID:   strings.TrimSpace(access.SubjectID),
			Role:        strings.TrimSpace(access.Role),
			Scopes:      append([]string(nil), access.Scopes...),
			Context:     cloneStringMap(access.Context),
			Metadata:    cloneStringMap(access.Metadata),
		},
	}
}

func buildAgentBoxDirectBootstrapResponse(project *sharedProjectBootstrapRow) agentBoxBootstrapResponse {
	projectName := strings.TrimSpace(project.Name)
	if projectName == "" {
		projectName = "AgentBox"
	}

	prompt := strings.TrimSpace(project.Prompt)
	if prompt == "" {
		prompt = "You are an embedded project assistant. Use the available runtime context, stay concise, and ask follow-up questions only when required."
	}

	promptProfiles := decodeAgentBoxPromptProfiles(project.PromptProfilesJSON)
	defaultPromptID := ""
	for _, profile := range promptProfiles {
		if profile.IsDefault {
			defaultPromptID = profile.ID
			break
		}
	}

	return agentBoxBootstrapResponse{
		Success:         true,
		ProjectName:     projectName,
		ProjectToken:    strings.TrimSpace(project.Token),
		Prompt:          prompt,
		PromptProfiles:  promptProfiles,
		DefaultPromptID: defaultPromptID,
	}
}

func projectTokenFromDirectMCPURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	cleanedPath := strings.Trim(parsed.EscapedPath(), "/")
	parts := strings.Split(cleanedPath, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "mcp" || parts[i] == "connect" {
			if token, err := url.PathUnescape(parts[i+1]); err == nil {
				return strings.TrimSpace(token)
			}
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func buildAgentBoxProviderPolicy(project *sharedProjectBootstrapRow) *agentBoxProviderPolicy {
	if project == nil {
		return nil
	}

	model := strings.TrimSpace(project.LlamaCppModelName)
	if model == "" {
		model = modelNameFromPath(strings.TrimSpace(project.LlamaCppModelPath))
	}
	if model == "" {
		return nil
	}

	return &agentBoxProviderPolicy{
		Provider: "llama.cpp",
		Model:    model,
	}
}

func modelNameFromPath(modelPath string) string {
	base := filepath.Base(strings.TrimSpace(modelPath))
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func publicURLForRequest(r *http.Request, routePath string) string {
	scheme := "http"
	if r != nil {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
			scheme = forwarded
		} else if r.TLS != nil {
			scheme = "https"
		}
	}

	host := ""
	if r != nil {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return routePath
	}
	return scheme + "://" + host + routePath
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		cloned[key] = value
	}
	return cloned
}
