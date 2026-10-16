package proragbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	tenantID   string
	apiToken   string
	adminToken string
	httpClient *http.Client
}

type ErrorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

func (e *APIError) Error() string {
	if e == nil {
		return "ragbox api error"
	}
	if strings.TrimSpace(e.Code) == "" {
		return fmt.Sprintf("ragbox api error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ragbox api error (%d, %s): %s", e.StatusCode, e.Code, e.Message)
}

type HealthResponse struct {
	Status string `json:"status"`
}

type VersionResponse struct {
	Service    string `json:"service"`
	Version    string `json:"version"`
	APIVersion string `json:"api_version"`
}

type Workspace struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenant_id"`
	VectorConnectionID string `json:"vector_connection_id"`
	AccessMode         string `json:"access_mode"`
	ExternalRef        string `json:"external_ref"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type AdminWorkspaceListResponse struct {
	Items []AdminWorkspace `json:"items"`
}

type AdminWorkspace struct {
	ID                 string         `json:"id"`
	TenantID           string         `json:"tenant_id"`
	VectorConnectionID string         `json:"vector_connection_id"`
	AccessMode         string         `json:"access_mode"`
	ExternalRef        string         `json:"external_ref"`
	Name               string         `json:"name"`
	Slug               string         `json:"slug"`
	Description        string         `json:"description,omitempty"`
	Status             string         `json:"status"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          string         `json:"created_at"`
}

type CreateWorkspaceRequest struct {
	TenantID           string         `json:"tenant_id,omitempty"`
	ExternalRef        string         `json:"external_ref"`
	Name               string         `json:"name"`
	Slug               string         `json:"slug"`
	VectorConnectionID string         `json:"vector_connection_id,omitempty"`
	AccessMode         string         `json:"access_mode,omitempty"`
	Description        string         `json:"description,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type Collection struct {
	ID                 string `json:"id"`
	WorkspaceID        string `json:"workspace_id"`
	VectorConnectionID string `json:"vector_connection_id"`
	ExternalRef        string `json:"external_ref"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type CreateCollectionRequest struct {
	ExternalRef     string         `json:"external_ref"`
	Name            string         `json:"name"`
	Slug            string         `json:"slug"`
	Description     string         `json:"description,omitempty"`
	DefaultLanguage string         `json:"default_language,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type VectorConnectionListResponse struct {
	BuiltinFallbackBackend string             `json:"builtin_fallback_backend"`
	Items                  []VectorConnection `json:"items"`
}

type VectorConnection struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	BackendType string         `json:"backend_type"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	IsDefault   bool           `json:"is_default"`
	Config      map[string]any `json:"config"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type CreateVectorConnectionRequest struct {
	Name          string         `json:"name"`
	BackendType   string         `json:"backend_type"`
	Description   string         `json:"description,omitempty"`
	IsDefault     bool           `json:"is_default,omitempty"`
	Qdrant        map[string]any `json:"qdrant,omitempty"`
	PgVector      map[string]any `json:"pgvector,omitempty"`
	Milvus        map[string]any `json:"milvus,omitempty"`
	Weaviate      map[string]any `json:"weaviate,omitempty"`
	OpenSearch    map[string]any `json:"opensearch,omitempty"`
	Elasticsearch map[string]any `json:"elasticsearch,omitempty"`
	Pinecone      map[string]any `json:"pinecone,omitempty"`
	Chroma        map[string]any `json:"chroma,omitempty"`
}

type FileUploadRequest struct {
	FilePath   string
	SourceName string
	ExternalID string
	Title      string
	Metadata   map[string]any
}

type FileUploadResponse struct {
	DocumentID string `json:"document_id"`
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
}

type IngestionSource struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name"`
}

type RecordInput struct {
	ExternalID      string         `json:"external_id"`
	RecordType      string         `json:"record_type"`
	Title           string         `json:"title,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Content         string         `json:"content"`
	Fields          map[string]any `json:"fields,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	SourceUpdatedAt string         `json:"source_updated_at,omitempty"`
}

type RecordBatchRequest struct {
	Source  *IngestionSource `json:"source,omitempty"`
	Records []RecordInput    `json:"records"`
}

type RecordDeleteRequest struct {
	Source      *IngestionSource `json:"source,omitempty"`
	ExternalIDs []string         `json:"external_ids"`
}

type RecordBatchResponse struct {
	JobID           string `json:"job_id"`
	Status          string `json:"status"`
	AcceptedRecords int64  `json:"accepted_records"`
}

type SearchFilters struct {
	OwnerTypes    []string       `json:"owner_types,omitempty"`
	ExternalIDs   []string       `json:"external_ids,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	ChunkProfiles []string       `json:"chunk_profiles,omitempty"`
}

type SearchRequest struct {
	Query          string         `json:"query"`
	TopK           int            `json:"top_k,omitempty"`
	Offset         int            `json:"offset,omitempty"`
	MinScore       float32        `json:"min_score,omitempty"`
	IncludeContent *bool          `json:"include_content,omitempty"`
	Filters        *SearchFilters `json:"filters,omitempty"`
}

type SearchResponse struct {
	Results     []SearchResult    `json:"results"`
	Total       int               `json:"total"`
	Limit       int               `json:"limit"`
	Offset      int               `json:"offset"`
	HasMore     bool              `json:"has_more"`
	Diagnostics SearchDiagnostics `json:"diagnostics"`
}

type SearchResult struct {
	ChunkID  string           `json:"chunk_id"`
	Score    float32          `json:"score"`
	RawScore float32          `json:"raw_score"`
	Text     string           `json:"text"`
	Owner    SearchOwner      `json:"owner"`
	Metadata map[string]any   `json:"metadata"`
	Chunk    SearchChunkStats `json:"chunk"`
}

type SearchOwner struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	Title      string `json:"title"`
}

type SearchChunkStats struct {
	Ordinal       int64  `json:"ordinal"`
	TokenEstimate int64  `json:"token_estimate"`
	StartOffset   *int64 `json:"start_offset"`
	EndOffset     *int64 `json:"end_offset"`
	ChunkProfile  string `json:"chunk_profile"`
	ChunkStrategy string `json:"chunk_strategy"`
	HasHeading    bool   `json:"has_heading"`
}

type SearchDiagnostics struct {
	Query             string `json:"query"`
	FormattedQuery    string `json:"formatted_query"`
	EmbeddingProvider string `json:"embedding_provider"`
	EmbeddingModelID  string `json:"embedding_model_id"`
	CandidateCount    int    `json:"candidate_count"`
	MatchedCount      int    `json:"matched_count"`
}

type JobResponse struct {
	ID           string           `json:"id"`
	JobType      string           `json:"job_type"`
	TargetType   string           `json:"target_type"`
	TargetID     string           `json:"target_id"`
	Status       string           `json:"status"`
	AttemptCount int64            `json:"attempt_count"`
	Stats        map[string]any   `json:"stats"`
	Error        *JobError        `json:"error"`
	CreatedAt    string           `json:"created_at"`
	StartedAt    *string          `json:"started_at"`
	CompletedAt  *string          `json:"completed_at"`
	UpdatedAt    string           `json:"updated_at"`
	Attempts     []JobAttemptInfo `json:"attempts"`
}

type JobError struct {
	Code    *string `json:"code"`
	Message *string `json:"message"`
}

type JobAttemptInfo struct {
	AttemptNumber int64          `json:"attempt_number"`
	Status        string         `json:"status"`
	StartedAt     *string        `json:"started_at"`
	FinishedAt    *string        `json:"finished_at"`
	ErrorCode     *string        `json:"error_code"`
	ErrorMessage  *string        `json:"error_message"`
	Stats         map[string]any `json:"stats"`
}

func NewClient(config ClientConfig) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("ragbox base url is not configured")
	}
	apiToken := strings.TrimSpace(config.APIToken)
	if apiToken == "" {
		apiToken = strings.TrimSpace(config.AdminToken)
	}
	if apiToken == "" {
		return nil, errors.New("ragbox admin token is not configured")
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid ragbox base url: %w", err)
	}

	timeout := time.Duration(normalizeTimeout(config.RequestTimeoutSeconds)) * time.Second
	return &Client{
		baseURL:    baseURL,
		tenantID:   normalizeTenantID(config.TenantID, ""),
		apiToken:   apiToken,
		adminToken: strings.TrimSpace(config.AdminToken),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var response HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) Version(ctx context.Context) (*VersionResponse, error) {
	var response VersionResponse
	if err := c.doJSON(ctx, http.MethodGet, "/version", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateWorkspace(ctx context.Context, request CreateWorkspaceRequest) (*Workspace, error) {
	if strings.TrimSpace(request.TenantID) == "" {
		request.TenantID = c.tenantID
	}
	var response Workspace
	if err := c.doJSON(ctx, http.MethodPost, "/v1/workspaces", request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	var response Workspace
	if err := c.doJSON(ctx, http.MethodGet, "/v1/workspaces/"+url.PathEscape(strings.TrimSpace(workspaceID)), nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ListAdminWorkspaces(ctx context.Context) (*AdminWorkspaceListResponse, error) {
	var response AdminWorkspaceListResponse
	if err := c.doJSONWithToken(ctx, http.MethodGet, "/api/workspaces", nil, &response, c.adminAuthToken()); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateCollection(ctx context.Context, workspaceID string, request CreateCollectionRequest) (*Collection, error) {
	var response Collection
	path := "/v1/workspaces/" + url.PathEscape(strings.TrimSpace(workspaceID)) + "/collections"
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetCollection(ctx context.Context, collectionID string) (*Collection, error) {
	var response Collection
	if err := c.doJSON(ctx, http.MethodGet, "/v1/collections/"+url.PathEscape(strings.TrimSpace(collectionID)), nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteAdminCollection(ctx context.Context, collectionID string) error {
	path := "/api/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/delete"
	return c.doJSONWithToken(ctx, http.MethodPost, path, nil, nil, c.adminAuthToken())
}

func (c *Client) ListVectorConnections(ctx context.Context) (*VectorConnectionListResponse, error) {
	var response VectorConnectionListResponse
	if err := c.doJSONWithToken(ctx, http.MethodGet, "/v1/admin/vector-connections", nil, &response, c.adminAuthToken()); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateVectorConnection(ctx context.Context, request CreateVectorConnectionRequest) (*VectorConnection, error) {
	response, err := c.createVectorConnectionWithPayload(ctx, request)
	if err == nil {
		return response, nil
	}

	if !isUnknownBackendVariantError(err) {
		return nil, err
	}

	compatRequest, ok := vectorConnectionCompatRequest(request)
	if !ok {
		return nil, err
	}

	return c.createVectorConnectionWithPayload(ctx, compatRequest)
}

func (c *Client) createVectorConnectionWithPayload(ctx context.Context, request CreateVectorConnectionRequest) (*VectorConnection, error) {
	var response VectorConnection
	if err := c.doJSONWithToken(ctx, http.MethodPost, "/v1/admin/vector-connections", request, &response, c.adminAuthToken()); err != nil {
		return nil, err
	}
	return &response, nil
}

func isUnknownBackendVariantError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "backend_type") && strings.Contains(message, "unknown variant")
}

func vectorConnectionCompatRequest(request CreateVectorConnectionRequest) (CreateVectorConnectionRequest, bool) {
	compat := request
	switch strings.TrimSpace(request.BackendType) {
	case "pgvector":
		compat.BackendType = "pg_vector"
		return compat, true
	case "pg_vector":
		compat.BackendType = "pgvector"
		return compat, true
	case "opensearch":
		compat.BackendType = "open_search"
		return compat, true
	case "open_search":
		compat.BackendType = "opensearch"
		return compat, true
	default:
		return CreateVectorConnectionRequest{}, false
	}
}

func (c *Client) SetDefaultVectorConnection(ctx context.Context, connectionID string) (*VectorConnection, error) {
	var response VectorConnection
	path := "/v1/admin/vector-connections/" + url.PathEscape(strings.TrimSpace(connectionID)) + "/set-default"
	if err := c.doJSONWithToken(ctx, http.MethodPost, path, nil, &response, c.adminAuthToken()); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UploadFile(ctx context.Context, collectionID string, request FileUploadRequest) (*FileUploadResponse, error) {
	file, err := os.Open(request.FilePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(request.FilePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if strings.TrimSpace(request.SourceName) != "" {
		_ = writer.WriteField("source_name", strings.TrimSpace(request.SourceName))
	}
	if strings.TrimSpace(request.ExternalID) != "" {
		_ = writer.WriteField("external_id", strings.TrimSpace(request.ExternalID))
	}
	if strings.TrimSpace(request.Title) != "" {
		_ = writer.WriteField("title", strings.TrimSpace(request.Title))
	}
	if len(request.Metadata) > 0 {
		raw, err := json.Marshal(request.Metadata)
		if err != nil {
			return nil, err
		}
		_ = writer.WriteField("metadata", string(raw))
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/collections/"+url.PathEscape(strings.TrimSpace(collectionID))+"/files:upload",
		&body,
	)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", writer.FormDataContentType())
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiToken)

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode >= http.StatusBadRequest {
		return nil, decodeAPIError(httpResponse)
	}

	var response FileUploadResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) IngestRecords(ctx context.Context, collectionID string, request RecordBatchRequest) (*RecordBatchResponse, error) {
	var response RecordBatchResponse
	path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/records:ingest"
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpsertRecords(ctx context.Context, collectionID string, request RecordBatchRequest) (*RecordBatchResponse, error) {
	var response RecordBatchResponse
	path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/records:upsert"
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteRecords(ctx context.Context, collectionID string, request RecordDeleteRequest) (*RecordBatchResponse, error) {
	var response RecordBatchResponse
	path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/records:delete"
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteDocuments(ctx context.Context, collectionID string, externalIDs []string) (*RecordBatchResponse, error) {
	var response RecordBatchResponse
	path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/documents:delete"
	request := map[string]any{
		"external_ids": externalIDs,
	}
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) SearchCollection(ctx context.Context, collectionID string, request SearchRequest) (*SearchResponse, error) {
	var response SearchResponse
	path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(collectionID)) + "/search"
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetJob(ctx context.Context, jobID string) (*JobResponse, error) {
	var response JobResponse
	path := "/v1/jobs/" + url.PathEscape(strings.TrimSpace(jobID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	return c.doJSONWithToken(ctx, method, path, requestBody, responseBody, c.apiToken)
}

func (c *Client) doJSONWithToken(ctx context.Context, method, path string, requestBody any, responseBody any, token string) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	if requestBody != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode >= http.StatusBadRequest {
		return decodeAPIError(httpResponse)
	}
	if responseBody == nil {
		return nil
	}
	return json.NewDecoder(httpResponse.Body).Decode(responseBody)
}

func (c *Client) adminAuthToken() string {
	if strings.TrimSpace(c.adminToken) != "" {
		return c.adminToken
	}
	return c.apiToken
}

func decodeAPIError(response *http.Response) error {
	if response == nil {
		return errors.New("ragbox api request failed")
	}

	body, _ := io.ReadAll(response.Body)
	var envelope ErrorResponse
	if err := json.Unmarshal(body, &envelope); err == nil && strings.TrimSpace(envelope.Error.Message) != "" {
		return &APIError{
			StatusCode: response.StatusCode,
			Code:       envelope.Error.Code,
			Message:    envelope.Error.Message,
			Details:    envelope.Error.Details,
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return &APIError{
		StatusCode: response.StatusCode,
		Message:    message,
	}
}
