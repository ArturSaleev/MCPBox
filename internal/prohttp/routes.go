package prohttp

import (
	"net/http"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"github.com/ArturSaleev/MCPBox/internal/prosso"
	"github.com/ArturSaleev/MCPBox/internal/prostorage"
)

func RegisterRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service, sso *prosso.Service, ragbox *proragbox.Service, ragboxBindings *proragbox.BindingService, storageConfig *prostorage.Service, storageManager *prostorage.Manager, projectShadows *proprojects.Service) {
	registerChatGPTOAuthRoutes(runtime, mux)
	registerAuthRoutes(mux, auth, sso)
	registerMetaRoutes(mux, auth, sso)
	registerUserRoutes(runtime, mux, auth)
	registerTokenRoutes(runtime, mux, auth)
	registerSessionRoutes(runtime, mux, auth)
	registerRagBoxRoutes(runtime, mux, auth, ragbox, ragboxBindings)
	registerRagBoxProjectRoutes(runtime, mux, auth, ragbox, ragboxBindings, projectShadows)
	registerRagBoxKnowledgeBaseRoutes(runtime, mux, auth, ragbox, ragboxBindings, projectShadows)
	registerStorageRoutes(runtime, mux, auth, storageConfig, storageManager, ragbox, ragboxBindings, projectShadows)
}

type updateStorageConfigRequest struct {
	OperationalBackend       string `json:"operational_backend"`
	PostgresHost             string `json:"postgres_host"`
	PostgresPort             int    `json:"postgres_port"`
	PostgresDatabase         string `json:"postgres_database"`
	PostgresSchema           string `json:"postgres_schema"`
	PostgresUsername         string `json:"postgres_username"`
	PostgresPassword         string `json:"postgres_password"`
	ClearPostgresDSN         bool   `json:"clear_postgres_dsn"`
	LogsBackend              string `json:"logs_backend"`
	ClickHouseURL            string `json:"clickhouse_url"`
	ClickHouseDatabase       string `json:"clickhouse_database"`
	ClickHouseUsername       string `json:"clickhouse_username"`
	ClickHousePassword       string `json:"clickhouse_password"`
	ClearClickHousePassword  bool   `json:"clear_clickhouse_password"`
	ElasticsearchURL         string `json:"elasticsearch_url"`
	ElasticsearchUsername    string `json:"elasticsearch_username"`
	ElasticsearchPassword    string `json:"elasticsearch_password"`
	ElasticsearchIndex       string `json:"elasticsearch_index"`
	ClearElasticsearchPasswd bool   `json:"clear_elasticsearch_password"`
}

func registerStorageRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service, storageConfig *prostorage.Service, storageManager *prostorage.Manager, ragbox *proragbox.Service, ragboxBindings *proragbox.BindingService, projectShadows *proprojects.Service) {
	mux.HandleFunc("GET /api/pro/storage/config", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if storageConfig == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage config service is not configured"})
			return
		}

		config, err := storageConfig.GetPublicConfig(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, config)
	}))

	mux.HandleFunc("PUT /api/pro/storage/config", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		if storageConfig == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage config service is not configured"})
			return
		}

		var req updateStorageConfigRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		config, err := storageConfig.UpdateConfig(r.Context(), prostorage.UpdateConfigInput{
			OperationalBackend:       req.OperationalBackend,
			PostgresHost:             req.PostgresHost,
			PostgresPort:             req.PostgresPort,
			PostgresDatabase:         req.PostgresDatabase,
			PostgresSchema:           req.PostgresSchema,
			PostgresUsername:         req.PostgresUsername,
			PostgresPassword:         req.PostgresPassword,
			ClearPostgresDSN:         req.ClearPostgresDSN,
			LogsBackend:              req.LogsBackend,
			ClickHouseURL:            req.ClickHouseURL,
			ClickHouseDatabase:       req.ClickHouseDatabase,
			ClickHouseUsername:       req.ClickHouseUsername,
			ClickHousePassword:       req.ClickHousePassword,
			ClearClickHousePassword:  req.ClearClickHousePassword,
			ElasticsearchURL:         req.ElasticsearchURL,
			ElasticsearchUsername:    req.ElasticsearchUsername,
			ElasticsearchPassword:    req.ElasticsearchPassword,
			ElasticsearchIndex:       req.ElasticsearchIndex,
			ClearElasticsearchPasswd: req.ClearElasticsearchPasswd,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if storageManager != nil {
			if _, err := storageManager.Reload(r.Context()); err != nil {
				proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
		}
		if ragbox != nil && storageManager != nil {
			if err := ragbox.RebindDB(r.Context(), storageManager.OperationalDB()); err != nil {
				proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
		}
		if ragboxBindings != nil && storageManager != nil {
			if err := ragboxBindings.RebindStateDB(r.Context(), storageManager.OperationalDB()); err != nil {
				proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
		}
		if projectShadows != nil && storageManager != nil {
			if err := projectShadows.RebindStateDB(r.Context(), storageManager.OperationalDB()); err != nil {
				proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			if _, err := projectShadows.SyncAll(r.Context()); err != nil {
				proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "storage_config_updated",
		})

		proauth.WriteJSON(w, http.StatusOK, config)
	}))

	mux.HandleFunc("GET /api/pro/storage/runtime-status", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if storageManager == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage runtime manager is not configured"})
			return
		}

		status, err := storageManager.Reload(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, status)
	}))

	mux.HandleFunc("GET /api/pro/storage/shadow-status", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		if projectShadows == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "project shadow service is not configured"})
			return
		}

		status, err := projectShadows.Status(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, status)
	}))

}
