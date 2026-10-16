package prohttp

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
)

type updateRagBoxConfigRequest struct {
	BaseURL               string `json:"base_url"`
	TenantID              string `json:"tenant_id"`
	ServiceToken          string `json:"service_token"`
	ClearServiceToken     bool   `json:"clear_service_token"`
	AdminToken            string `json:"admin_token"`
	ClearAdminToken       bool   `json:"clear_admin_token"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
	WriteMode             string `json:"write_mode"`
	SearchMode            string `json:"search_mode"`
}

type createProRagBoxVectorConnectionRequest struct {
	Name          string         `json:"name"`
	BackendType   string         `json:"backend_type"`
	Description   string         `json:"description"`
	IsDefault     bool           `json:"is_default"`
	BackendConfig map[string]any `json:"backend_config,omitempty"`
	Qdrant        *struct {
		URL              string  `json:"url"`
		APIKey           *string `json:"api_key"`
		CollectionPrefix string  `json:"collection_prefix"`
	} `json:"qdrant,omitempty"`
	PgVector *struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Database    string `json:"database"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Schema      string `json:"schema"`
		TablePrefix string `json:"table_prefix"`
	} `json:"pgvector,omitempty"`
	Milvus *struct {
		URL              string  `json:"url"`
		Token            *string `json:"token"`
		Database         *string `json:"database"`
		CollectionPrefix string  `json:"collection_prefix"`
	} `json:"milvus,omitempty"`
	Weaviate *struct {
		URL              string  `json:"url"`
		APIKey           *string `json:"api_key"`
		CollectionPrefix string  `json:"collection_prefix"`
	} `json:"weaviate,omitempty"`
	OpenSearch *struct {
		URL         string  `json:"url"`
		Username    *string `json:"username"`
		Password    *string `json:"password"`
		IndexPrefix string  `json:"index_prefix"`
	} `json:"opensearch,omitempty"`
	Elasticsearch *struct {
		URL         string  `json:"url"`
		APIKey      *string `json:"api_key"`
		Username    *string `json:"username"`
		Password    *string `json:"password"`
		IndexPrefix string  `json:"index_prefix"`
	} `json:"elasticsearch,omitempty"`
	Pinecone *struct {
		Host            string `json:"host"`
		APIKey          string `json:"api_key"`
		NamespacePrefix string `json:"namespace_prefix"`
	} `json:"pinecone,omitempty"`
	Chroma *struct {
		URL              string  `json:"url"`
		Tenant           *string `json:"tenant"`
		Database         *string `json:"database"`
		APIKey           *string `json:"api_key"`
		CollectionPrefix string  `json:"collection_prefix"`
	} `json:"chroma,omitempty"`
}

func registerRagBoxRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service, ragbox *proragbox.Service, ragboxBindings *proragbox.BindingService) {
	mux.HandleFunc("GET /api/pro/ragbox/config", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		config, err := ragbox.GetPublicConfig(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, config)
	}))

	mux.HandleFunc("PUT /api/pro/ragbox/config", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		var req updateRagBoxConfigRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		config, err := ragbox.UpdateConfig(r.Context(), proragbox.UpdateConfigInput{
			BaseURL:               req.BaseURL,
			TenantID:              req.TenantID,
			APIToken:              req.ServiceToken,
			ClearAPIToken:         req.ClearServiceToken,
			AdminToken:            req.AdminToken,
			ClearAdminToken:       req.ClearAdminToken,
			RequestTimeoutSeconds: req.RequestTimeoutSeconds,
			WriteMode:             req.WriteMode,
			SearchMode:            req.SearchMode,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "ragbox_config_updated",
			Detail: fmt.Sprintf(`{"enabled":%t,"base_url":"%s","admin_token_configured":%t,"timeout_seconds":%d,"write_mode":"%s","search_mode":"%s"}`,
				config.Enabled,
				config.BaseURL,
				config.AdminTokenConfigured,
				config.RequestTimeoutSeconds,
				config.WriteMode,
				config.SearchMode,
			),
		})

		if config.Enabled && config.Configured && ragbox != nil && ragboxBindings != nil {
			_, _ = syncAllRagBoxKnowledgeBases(r.Context(), runtime, ragbox, ragboxBindings)
		}

		proauth.WriteJSON(w, http.StatusOK, config)
	}))

	mux.HandleFunc("GET /api/pro/ragbox/admin/vector-connections", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		client, err := ragBoxClientFromService(r.Context(), ragbox)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		response, err := client.ListVectorConnections(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, response)
	}))

	mux.HandleFunc("GET /api/pro/ragbox/admin/workspaces", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		client, err := ragBoxClientFromService(r.Context(), ragbox)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		response, err := client.ListAdminWorkspaces(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, response)
	}))

	mux.HandleFunc("POST /api/pro/ragbox/admin/vector-connections", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		var req createProRagBoxVectorConnectionRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		payload := buildProRagBoxVectorConnectionPayload(req)

		client, err := ragBoxClientFromService(r.Context(), ragbox)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		response, err := client.CreateVectorConnection(r.Context(), payload)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusCreated, response)
	}))

	mux.HandleFunc("POST /api/pro/ragbox/admin/vector-connections/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		if ragbox == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox service is not configured"})
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/pro/ragbox/admin/vector-connections/")
		if path == "" || !strings.HasSuffix(path, "/set-default") {
			http.NotFound(w, r)
			return
		}

		connectionID := strings.Trim(strings.TrimSuffix(path, "/set-default"), "/")
		if connectionID == "" {
			http.NotFound(w, r)
			return
		}

		client, err := ragBoxClientFromService(r.Context(), ragbox)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		response, err := client.SetDefaultVectorConnection(r.Context(), connectionID)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, response)
	}))

	mux.HandleFunc("GET /api/pro/ragbox/knowledge-bases", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if ragboxBindings == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox bindings are not configured"})
			return
		}
		statuses, err := ragboxBindings.ListKnowledgeBaseIntegrationStatuses(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, map[string]any{"items": statuses})
	}))

	mux.HandleFunc("POST /api/pro/ragbox/knowledge-bases/sync", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		items, err := syncAllRagBoxKnowledgeBases(r.Context(), runtime, ragbox, ragboxBindings)
		if err != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(err), map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	}))
}

func normalizeProRagBoxBackendType(value string) string {
	switch strings.TrimSpace(value) {
	case "pg_vector":
		return "pgvector"
	case "open_search":
		return "opensearch"
	default:
		return strings.TrimSpace(value)
	}
}

func cloneTrimmedMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(value))
	for key, raw := range value {
		switch typed := raw.(type) {
		case string:
			cloned[key] = strings.TrimSpace(typed)
		default:
			cloned[key] = raw
		}
	}
	return cloned
}

func optionalTrimmedString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func buildProRagBoxVectorConnectionPayload(req createProRagBoxVectorConnectionRequest) proragbox.CreateVectorConnectionRequest {
	payload := proragbox.CreateVectorConnectionRequest{
		Name:        strings.TrimSpace(req.Name),
		BackendType: normalizeProRagBoxBackendType(req.BackendType),
		Description: strings.TrimSpace(req.Description),
		IsDefault:   req.IsDefault,
	}

	if config := cloneTrimmedMap(req.BackendConfig); len(config) > 0 {
		switch payload.BackendType {
		case "qdrant":
			payload.Qdrant = config
		case "pgvector":
			payload.PgVector = config
		case "milvus":
			payload.Milvus = config
		case "weaviate":
			payload.Weaviate = config
		case "opensearch":
			payload.OpenSearch = config
		case "elasticsearch":
			payload.Elasticsearch = config
		case "pinecone":
			payload.Pinecone = config
		case "chroma":
			payload.Chroma = config
		}
		return payload
	}

	switch payload.BackendType {
	case "qdrant":
		if req.Qdrant != nil {
			payload.Qdrant = map[string]any{
				"url":               strings.TrimSpace(req.Qdrant.URL),
				"collection_prefix": strings.TrimSpace(req.Qdrant.CollectionPrefix),
			}
			if value := optionalTrimmedString(req.Qdrant.APIKey); value != nil {
				payload.Qdrant["api_key"] = value
			}
		}
	case "pgvector":
		if req.PgVector != nil {
			password := url.QueryEscape(req.PgVector.Password)
			username := url.QueryEscape(strings.TrimSpace(req.PgVector.Username))
			database := url.QueryEscape(strings.TrimSpace(req.PgVector.Database))
			host := strings.TrimSpace(req.PgVector.Host)
			port := req.PgVector.Port
			if port <= 0 {
				port = 5432
			}
			databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", username, password, host, port, database)
			payload.PgVector = map[string]any{
				"database_url": databaseURL,
				"schema":       strings.TrimSpace(req.PgVector.Schema),
				"table_prefix": strings.TrimSpace(req.PgVector.TablePrefix),
			}
		}
	case "milvus":
		if req.Milvus != nil {
			payload.Milvus = map[string]any{
				"url":               strings.TrimSpace(req.Milvus.URL),
				"collection_prefix": strings.TrimSpace(req.Milvus.CollectionPrefix),
			}
			if value := optionalTrimmedString(req.Milvus.Token); value != nil {
				payload.Milvus["token"] = value
			}
			if value := optionalTrimmedString(req.Milvus.Database); value != nil {
				payload.Milvus["database"] = value
			}
		}
	case "weaviate":
		if req.Weaviate != nil {
			payload.Weaviate = map[string]any{
				"url":               strings.TrimSpace(req.Weaviate.URL),
				"collection_prefix": strings.TrimSpace(req.Weaviate.CollectionPrefix),
			}
			if value := optionalTrimmedString(req.Weaviate.APIKey); value != nil {
				payload.Weaviate["api_key"] = value
			}
		}
	case "opensearch":
		if req.OpenSearch != nil {
			payload.OpenSearch = map[string]any{
				"url":          strings.TrimSpace(req.OpenSearch.URL),
				"index_prefix": strings.TrimSpace(req.OpenSearch.IndexPrefix),
			}
			if value := optionalTrimmedString(req.OpenSearch.Username); value != nil {
				payload.OpenSearch["username"] = value
			}
			if value := optionalTrimmedString(req.OpenSearch.Password); value != nil {
				payload.OpenSearch["password"] = value
			}
		}
	case "elasticsearch":
		if req.Elasticsearch != nil {
			payload.Elasticsearch = map[string]any{
				"url":          strings.TrimSpace(req.Elasticsearch.URL),
				"index_prefix": strings.TrimSpace(req.Elasticsearch.IndexPrefix),
			}
			if value := optionalTrimmedString(req.Elasticsearch.APIKey); value != nil {
				payload.Elasticsearch["api_key"] = value
			}
			if value := optionalTrimmedString(req.Elasticsearch.Username); value != nil {
				payload.Elasticsearch["username"] = value
			}
			if value := optionalTrimmedString(req.Elasticsearch.Password); value != nil {
				payload.Elasticsearch["password"] = value
			}
		}
	case "pinecone":
		if req.Pinecone != nil {
			payload.Pinecone = map[string]any{
				"host":             strings.TrimSpace(req.Pinecone.Host),
				"api_key":          strings.TrimSpace(req.Pinecone.APIKey),
				"namespace_prefix": strings.TrimSpace(req.Pinecone.NamespacePrefix),
			}
		}
	case "chroma":
		if req.Chroma != nil {
			payload.Chroma = map[string]any{
				"url":               strings.TrimSpace(req.Chroma.URL),
				"collection_prefix": strings.TrimSpace(req.Chroma.CollectionPrefix),
			}
			if value := optionalTrimmedString(req.Chroma.Tenant); value != nil {
				payload.Chroma["tenant"] = value
			}
			if value := optionalTrimmedString(req.Chroma.Database); value != nil {
				payload.Chroma["database"] = value
			}
			if value := optionalTrimmedString(req.Chroma.APIKey); value != nil {
				payload.Chroma["api_key"] = value
			}
		}
	}

	return payload
}
