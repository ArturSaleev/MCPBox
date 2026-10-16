package prohttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"gorm.io/gorm"
)

type ensureProRagBoxWorkspaceRequest struct {
	VectorConnectionID string `json:"vector_connection_id"`
	WorkspaceID        string `json:"workspace_id"`
}

type proRagBoxRecordBatchRequest struct {
	Source  *proragbox.IngestionSource `json:"source,omitempty"`
	Records []proragbox.RecordInput    `json:"records"`
}

func registerRagBoxProjectRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service, ragbox *proragbox.Service, ragboxBindings *proragbox.BindingService, projectShadows *proprojects.Service) {
	mux.HandleFunc("POST /api/pro/ragbox/ingest/", func(w http.ResponseWriter, r *http.Request) {
		handleProRagBoxIngest(w, r, runtime, ragbox, ragboxBindings)
	})

	mux.HandleFunc("POST /api/pro/ragbox/projects/", func(w http.ResponseWriter, r *http.Request) {
		if projectShadows != nil {
			if projectID, collectionID, _, err := ragBoxProjectCollectionActionFromPath(r.URL.Path); err == nil {
				if syncErr := projectShadows.SyncProjectByID(r.Context(), projectID); syncErr != nil && !errors.Is(syncErr, gorm.ErrRecordNotFound) {
					proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": syncErr.Error()})
					return
				}
				if syncErr := projectShadows.SyncProjectRAGLink(r.Context(), projectID, collectionID); syncErr != nil && !errors.Is(syncErr, gorm.ErrRecordNotFound) {
					proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": syncErr.Error()})
					return
				}
			} else if projectID, action, err := ragBoxProjectActionFromPath(r.URL.Path); err == nil && action == "workspace/ensure" {
				if syncErr := projectShadows.SyncProjectByID(r.Context(), projectID); syncErr != nil && !errors.Is(syncErr, gorm.ErrRecordNotFound) {
					proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": syncErr.Error()})
					return
				}
			}
		}
		if isRagBoxSearchPath(r.URL.Path) {
			proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
				handleProRagBoxSearch(w, r, runtime, ragbox, ragboxBindings)
			})(w, r)
			return
		}
		proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
			handleProRagBoxProjectAdmin(w, r, runtime, ragbox, ragboxBindings)
		})(w, r)
	})
}

func handleProRagBoxProjectAdmin(
	w http.ResponseWriter,
	r *http.Request,
	runtime *app.RuntimeContext,
	ragbox *proragbox.Service,
	ragboxBindings *proragbox.BindingService,
) {
	if ragbox == nil || ragboxBindings == nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox services are not configured"})
		return
	}

	projectID, action, err := ragBoxProjectActionFromPath(r.URL.Path)
	if err == nil && action == "workspace/ensure" {
		var req ensureProRagBoxWorkspaceRequest
		if r.ContentLength > 0 {
			if decodeErr := decodeJSONBody(r, &req); decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
				proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": decodeErr.Error()})
				return
			}
		}
		client, clientErr := ragBoxClientFromService(r.Context(), ragbox)
		if clientErr != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": clientErr.Error()})
			return
		}
		result, ensureErr := ragboxBindings.EnsureProjectWorkspace(
			r.Context(),
			client,
			projectID,
			strings.TrimSpace(req.VectorConnectionID),
			strings.TrimSpace(req.WorkspaceID),
		)
		if ensureErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(ensureErr), map[string]string{"error": ensureErr.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			ProjectID: &projectID,
			Action:    "ragbox_workspace_bound",
			Detail:    fmt.Sprintf(`{"project_id":%d,"ragbox_workspace_id":"%s"}`, projectID, result.Binding.RagBoxWorkspaceID),
		})

		proauth.WriteJSON(w, http.StatusCreated, result)
		return
	}

	var collectionID string
	projectID, collectionID, action, err = ragBoxProjectCollectionActionFromPath(r.URL.Path)
	if err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	switch action {
	case "ensure":
		var req ensureProRagBoxWorkspaceRequest
		if r.ContentLength > 0 {
			if decodeErr := decodeJSONBody(r, &req); decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
				proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": decodeErr.Error()})
				return
			}
		}
		client, clientErr := ragBoxClientFromService(r.Context(), ragbox)
		if clientErr != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": clientErr.Error()})
			return
		}
		result, ensureErr := ragboxBindings.EnsureProjectCollection(r.Context(), client, projectID, collectionID)
		if ensureErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(ensureErr), map[string]string{"error": ensureErr.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			ProjectID: &projectID,
			Action:    "ragbox_collection_bound",
			Detail: fmt.Sprintf(
				`{"project_id":%d,"collection_id":"%s","ragbox_collection_id":"%s"}`,
				projectID,
				collectionID,
				result.Binding.RagBoxCollectionID,
			),
		})

		proauth.WriteJSON(w, http.StatusCreated, result)
	case "files:upload", "records:upsert":
		sharedCollection, sharedErr := ragboxBindings.GetSharedRAGCollectionByCollectionID(r.Context(), collectionID)
		if sharedErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": sharedErr.Error()})
			return
		}
		if sharedCollection == nil {
			proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
			return
		}

		linked, linkErr := ragboxBindings.ProjectHasCollectionLink(r.Context(), projectID, sharedCollection.ID)
		if linkErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": linkErr.Error()})
			return
		}
		if !linked {
			proauth.WriteJSON(w, http.StatusConflict, map[string]string{"error": "knowledge base is not linked to the selected project"})
			return
		}

		sourcePath, sourceErr := ensureManagedKnowledgeBasePath(*sharedCollection)
		if sourceErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": sourceErr.Error()})
			return
		}

		switch action {
		case "files:upload":
			response, uploadErr := handleProRagBoxFileUpload(r.Context(), runtime, ragbox, ragboxBindings, *sharedCollection, sourcePath, r)
			if uploadErr != nil {
				proauth.WriteJSON(w, ragBoxHTTPStatusFromError(uploadErr), map[string]string{"error": uploadErr.Error()})
				return
			}
			proauth.WriteJSON(w, http.StatusAccepted, response)
		case "records:upsert":
			var req proRagBoxRecordBatchRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			response, upsertErr := handleProRagBoxRecordUpsert(r.Context(), runtime, ragbox, ragboxBindings, *sharedCollection, sourcePath, req)
			if upsertErr != nil {
				proauth.WriteJSON(w, ragBoxHTTPStatusFromError(upsertErr), map[string]string{"error": upsertErr.Error()})
				return
			}
			proauth.WriteJSON(w, http.StatusAccepted, response)
		default:
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func handleProRagBoxSearch(
	w http.ResponseWriter,
	r *http.Request,
	runtime *app.RuntimeContext,
	ragbox *proragbox.Service,
	ragboxBindings *proragbox.BindingService,
) {
	if ragbox == nil || ragboxBindings == nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox services are not configured"})
		return
	}

	projectID, collectionID, action, err := ragBoxProjectCollectionActionFromPath(r.URL.Path)
	if err != nil || action != "search" {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ragbox search path"})
		return
	}

	var req proragbox.ProSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sharedCollection, err := ragboxBindings.GetSharedRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sharedCollection == nil {
		proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
		return
	}

	linked, linkErr := ragboxBindings.ProjectHasCollectionLink(r.Context(), projectID, sharedCollection.ID)
	if linkErr != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": linkErr.Error()})
		return
	}
	if !linked {
		proauth.WriteJSON(w, http.StatusConflict, map[string]string{"error": "knowledge base is not linked to the selected project"})
		return
	}

	clientConfig, err := ragbox.GetClientConfig(r.Context())
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	publicConfig, err := ragbox.GetPublicConfig(r.Context())
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	searchMode := clientConfig.SearchMode
	if !publicConfig.Enabled || !publicConfig.Configured {
		searchMode = "bleve_only"
	}

	var client *proragbox.Client
	if publicConfig.Enabled && strings.TrimSpace(clientConfig.BaseURL) != "" && strings.TrimSpace(clientConfig.APIToken) != "" {
		client, err = proragbox.NewClient(clientConfig)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	binding, err := ragboxBindings.GetProjectCollectionBinding(r.Context(), projectID, sharedCollection.ID)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	adminProxy := proragbox.RuntimeAdminProxy{}
	if runtime != nil {
		adminProxy.Host = runtime.Edition.AdminHost
		adminProxy.Port = runtime.Edition.AdminPort
	}

	response, searchErr := proragbox.SearchKnowledge(
		r.Context(),
		adminProxy,
		clientConfig,
		searchMode,
		client,
		binding,
		collectionID,
		req.Query,
		req.Limit,
	)
	if searchErr != nil {
		proauth.WriteJSON(w, ragBoxHTTPStatusFromError(searchErr), map[string]string{"error": searchErr.Error()})
		return
	}

	proauth.WriteJSON(w, http.StatusOK, response)
}

func handleProRagBoxGlobalSearch(
	w http.ResponseWriter,
	r *http.Request,
	runtime *app.RuntimeContext,
	ragbox *proragbox.Service,
	ragboxBindings *proragbox.BindingService,
) {
	if ragbox == nil || ragboxBindings == nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox services are not configured"})
		return
	}

	collectionID, action, err := ragBoxCollectionActionFromPath(r.URL.Path)
	if err != nil || action != "search" {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ragbox search path"})
		return
	}

	var req proragbox.ProSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sharedCollection, err := ragboxBindings.GetSharedRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sharedCollection == nil {
		proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
		return
	}

	clientConfig, err := ragbox.GetClientConfig(r.Context())
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	publicConfig, err := ragbox.GetPublicConfig(r.Context())
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	searchMode := clientConfig.SearchMode
	var client *proragbox.Client
	var binding *proragbox.KnowledgeBaseBinding

	switch normalizeKnowledgeBaseServiceMode(sharedCollection.ServiceMode) {
	case "bleve_only":
		searchMode = "bleve_only"
	case "ragbox_only":
		searchMode = "ragbox_only"
	default:
		searchMode = clientConfig.SearchMode
	}

	if !publicConfig.Enabled || !publicConfig.Configured {
		if searchMode == "ragbox_only" {
			proauth.WriteJSON(w, http.StatusConflict, map[string]string{"error": "RagBox search is required for this knowledge base but RagBox is not configured"})
			return
		}
		searchMode = "bleve_only"
	} else {
		client, err = proragbox.NewClient(clientConfig)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if proragboxUsesRagBox(sharedCollection.ServiceMode) {
			binding, _, err = ensureGlobalKnowledgeBaseBinding(r.Context(), ragboxBindings, client, collectionID)
			if err != nil {
				proauth.WriteJSON(w, ragBoxHTTPStatusFromError(err), map[string]string{"error": err.Error()})
				return
			}
		}
	}

	adminProxy := proragbox.RuntimeAdminProxy{}
	if runtime != nil {
		adminProxy.Host = runtime.Edition.AdminHost
		adminProxy.Port = runtime.Edition.AdminPort
	}

	var searchBinding *proragbox.ProjectCollectionBinding
	if binding != nil && strings.TrimSpace(binding.RagBoxCollectionID) != "" {
		searchBinding = &proragbox.ProjectCollectionBinding{RagBoxCollectionID: binding.RagBoxCollectionID}
	}

	response, searchErr := proragbox.SearchKnowledge(
		r.Context(),
		adminProxy,
		clientConfig,
		searchMode,
		client,
		searchBinding,
		collectionID,
		req.Query,
		req.Limit,
	)
	if searchErr != nil {
		proauth.WriteJSON(w, ragBoxHTTPStatusFromError(searchErr), map[string]string{"error": searchErr.Error()})
		return
	}

	proauth.WriteJSON(w, http.StatusOK, response)
}

func handleProRagBoxIngest(
	w http.ResponseWriter,
	r *http.Request,
	runtime *app.RuntimeContext,
	ragbox *proragbox.Service,
	ragboxBindings *proragbox.BindingService,
) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if ragbox == nil || ragboxBindings == nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "ragbox services are not configured"})
		return
	}

	publicID, err := ragBoxIngestPublicIDFromPath(r.URL.Path)
	if err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	token, err := ragboxBindings.GetKnowledgeBaseIngestTokenByPublicID(r.Context(), publicID)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if token == nil {
		proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "ingest endpoint was not found"})
		return
	}

	if bearerTokenFromRequest(r) != strings.TrimSpace(token.SecretToken) {
		proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid ingest token"})
		return
	}

	collection, err := ragboxBindings.GetSharedRAGCollectionByDBID(r.Context(), token.RAGCollectionID)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if collection == nil {
		proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
		return
	}

	var req proRagBoxRecordBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sourcePath, err := ragboxBindings.EnsureShadowSourcePath(*collection)
	if err != nil {
		proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response, upsertErr := handleProRagBoxRecordUpsert(r.Context(), runtime, ragbox, ragboxBindings, *collection, sourcePath, req)
	if upsertErr != nil {
		proauth.WriteJSON(w, ragBoxHTTPStatusFromError(upsertErr), map[string]string{"error": upsertErr.Error()})
		return
	}

	proauth.WriteJSON(w, http.StatusAccepted, response)
}

func syncAllRagBoxKnowledgeBases(ctx context.Context, runtime *app.RuntimeContext, service *proragbox.Service, bindings *proragbox.BindingService) ([]proragbox.EnsureKnowledgeBaseResult, error) {
	if service == nil || bindings == nil {
		return nil, errors.New("ragbox services are not configured")
	}
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	client, err := ragBoxClientFromService(ctx, service)
	if err != nil {
		return nil, err
	}
	statuses, err := bindings.ListKnowledgeBaseIntegrationStatuses(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]proragbox.EnsureKnowledgeBaseResult, 0, len(statuses))
	for _, item := range statuses {
		if !proragboxUsesRagBox(item.Collection.ServiceMode) {
			continue
		}
		result, err := bindings.EnsureKnowledgeBase(ctx, client, item.Collection.CollectionID, "")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.Collection.SourcePath) != "" {
			if _, err := syncKnowledgeBaseSourceToRagBox(ctx, bindings, client, &result.Binding, result.Collection); err != nil {
				return nil, err
			}
		}
		items = append(items, *result)
	}
	return items, nil
}
