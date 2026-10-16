package prohttp

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"gorm.io/gorm"
)

type proKnowledgeBaseDeletePreviewResponse struct {
	CollectionID       string `json:"collection_id"`
	ManagedSourcePath  string `json:"managed_source_path"`
	HasUploadedFiles   bool   `json:"has_uploaded_files"`
	UploadedFilesCount int    `json:"uploaded_files_count"`
}

type proKnowledgeBaseUploadMode string

const (
	proKnowledgeBaseUploadModeKeep      proKnowledgeBaseUploadMode = "keep"
	proKnowledgeBaseUploadModeTransient proKnowledgeBaseUploadMode = "transient"
)

type proRagBoxWriteResponse struct {
	Local struct {
		Mode         string   `json:"mode"`
		Indexed      bool     `json:"indexed"`
		DirPath      string   `json:"dir_path"`
		WrittenFiles []string `json:"written_files,omitempty"`
	} `json:"local"`
	RagBox struct {
		Skipped         bool   `json:"skipped"`
		Reason          string `json:"reason,omitempty"`
		CollectionID    string `json:"collection_id,omitempty"`
		DocumentID      string `json:"document_id,omitempty"`
		JobID           string `json:"job_id,omitempty"`
		Status          string `json:"status,omitempty"`
		AcceptedRecords int64  `json:"accepted_records,omitempty"`
	} `json:"ragbox"`
}

type knowledgeBaseDeleteTarget struct {
	ID           uint   `gorm:"column:id"`
	CollectionID string `gorm:"column:collection_id"`
	IndexPath    string `gorm:"column:index_path"`
}

func (knowledgeBaseDeleteTarget) TableName() string {
	return "rag_collections"
}

type knowledgeBaseDeleteBinding struct {
	RAGCollectionID    uint   `gorm:"column:rag_collection_id"`
	RagBoxCollectionID string `gorm:"column:rag_box_collection_id"`
}

func (knowledgeBaseDeleteBinding) TableName() string {
	return "knowledge_base_bindings"
}

func registerRagBoxKnowledgeBaseRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service, ragbox *proragbox.Service, ragboxBindings *proragbox.BindingService, projectShadows *proprojects.Service) {
	mux.HandleFunc("POST /api/pro/ragbox/collections/", func(w http.ResponseWriter, r *http.Request) {
		if projectShadows != nil {
			if collectionID, _, err := ragBoxCollectionActionFromPath(r.URL.Path); err == nil {
				if syncErr := projectShadows.SyncRAGCollectionByCollectionID(r.Context(), collectionID); syncErr != nil && !errors.Is(syncErr, gorm.ErrRecordNotFound) {
					proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": syncErr.Error()})
					return
				}
			}
		}
		if isRagBoxGlobalSearchPath(r.URL.Path) {
			proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
				handleProRagBoxGlobalSearch(w, r, runtime, ragbox, ragboxBindings)
			})(w, r)
			return
		}
		proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
			handleProRagBoxKnowledgeBaseAdmin(w, r, runtime, ragbox, ragboxBindings)
		})(w, r)
	})

	mux.HandleFunc("GET /api/pro/ragbox/collections/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		collectionID, action, err := ragBoxCollectionActionFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if action != "delete-preview" {
			http.NotFound(w, r)
			return
		}

		preview, previewErr := buildKnowledgeBaseDeletePreview(r.Context(), runtime, collectionID)
		if previewErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(previewErr), map[string]string{"error": previewErr.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, preview)
	}))

	mux.HandleFunc("DELETE /api/pro/ragbox/collections/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		collectionID, err := ragBoxCollectionIDFromDeletePath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		removeUploadedFiles := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("remove_uploaded_files")), "true")
		preview, deleteErr := deleteKnowledgeBaseWithArtifacts(r.Context(), runtime, ragbox, collectionID, removeUploadedFiles)
		if deleteErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(deleteErr), map[string]string{"error": deleteErr.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "knowledge_base_deleted",
			Detail: fmt.Sprintf(`{"collection_id":%q,"remove_uploaded_files":%t,"uploaded_files_count":%d}`,
				preview.CollectionID,
				removeUploadedFiles,
				preview.UploadedFilesCount,
			),
		})

		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"collection_id":         preview.CollectionID,
			"deleted":               true,
			"remove_uploaded_files": removeUploadedFiles,
			"uploaded_files_count":  preview.UploadedFilesCount,
		})
	}))
}

func handleProRagBoxKnowledgeBaseAdmin(
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
		sharedCollection, sharedErr := ragboxBindings.GetSharedRAGCollectionByCollectionID(r.Context(), collectionID)
		if sharedErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": sharedErr.Error()})
			return
		}
		if sharedCollection == nil {
			proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
			return
		}
		if !proragboxUsesRagBox(sharedCollection.ServiceMode) {
			proauth.WriteJSON(w, http.StatusConflict, map[string]string{"error": "knowledge base is configured without RagBox"})
			return
		}
		result, ensureErr := ragboxBindings.EnsureKnowledgeBase(r.Context(), client, collectionID, strings.TrimSpace(req.WorkspaceID))
		if ensureErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(ensureErr), map[string]string{"error": ensureErr.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusCreated, result)
	case "files:upload":
		sharedCollection, sharedErr := ragboxBindings.GetSharedRAGCollectionByCollectionID(r.Context(), collectionID)
		if sharedErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": sharedErr.Error()})
			return
		}
		if sharedCollection == nil {
			proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "knowledge base was not found"})
			return
		}
		sourcePath, sourceErr := ensureManagedKnowledgeBasePath(*sharedCollection)
		if sourceErr != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": sourceErr.Error()})
			return
		}
		response, uploadErr := handleProKnowledgeBaseFileUpload(r.Context(), runtime, ragbox, ragboxBindings, *sharedCollection, sourcePath, r)
		if uploadErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(uploadErr), map[string]string{"error": uploadErr.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusAccepted, response)
	case "sync-source":
		client, clientErr := ragBoxClientFromService(r.Context(), ragbox)
		if clientErr != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": clientErr.Error()})
			return
		}
		binding, collection, resolveErr := ensureGlobalKnowledgeBaseBinding(r.Context(), ragboxBindings, client, collectionID)
		if resolveErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(resolveErr), map[string]string{"error": resolveErr.Error()})
			return
		}
		response, syncErr := syncKnowledgeBaseSourceToRagBox(r.Context(), ragboxBindings, client, binding, *collection)
		if syncErr != nil {
			proauth.WriteJSON(w, ragBoxHTTPStatusFromError(syncErr), map[string]string{"error": syncErr.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusAccepted, response)
	default:
		http.NotFound(w, r)
	}
}

func buildKnowledgeBaseDeletePreview(ctx context.Context, runtime *app.RuntimeContext, collectionID string) (*proKnowledgeBaseDeletePreviewResponse, error) {
	target, err := loadKnowledgeBaseDeleteTarget(ctx, runtime, collectionID)
	if err != nil {
		return nil, err
	}

	managedPath := resolveManagedKnowledgeBasePath(target.CollectionID)
	filesCount, err := countKnowledgeBaseManagedFiles(managedPath)
	if err != nil {
		return nil, err
	}

	return &proKnowledgeBaseDeletePreviewResponse{
		CollectionID:       target.CollectionID,
		ManagedSourcePath:  managedPath,
		HasUploadedFiles:   filesCount > 0,
		UploadedFilesCount: filesCount,
	}, nil
}

func deleteKnowledgeBaseWithArtifacts(ctx context.Context, runtime *app.RuntimeContext, ragboxService *proragbox.Service, collectionID string, removeUploadedFiles bool) (*proKnowledgeBaseDeletePreviewResponse, error) {
	target, err := loadKnowledgeBaseDeleteTarget(ctx, runtime, collectionID)
	if err != nil {
		return nil, err
	}
	binding, err := loadKnowledgeBaseDeleteBinding(ctx, runtime, target.ID)
	if err != nil {
		return nil, err
	}

	managedPath := resolveManagedKnowledgeBasePath(target.CollectionID)
	filesCount, err := countKnowledgeBaseManagedFiles(managedPath)
	if err != nil {
		return nil, err
	}

	if binding != nil && strings.TrimSpace(binding.RagBoxCollectionID) != "" {
		client, err := ragBoxClientFromService(ctx, ragboxService)
		if err != nil {
			return nil, err
		}
		if err := client.DeleteAdminCollection(ctx, binding.RagBoxCollectionID); err != nil && !isRagBoxCollectionMissing(err) {
			return nil, err
		}
	}

	if strings.TrimSpace(target.IndexPath) != "" {
		if err := os.RemoveAll(target.IndexPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if removeUploadedFiles {
		if err := os.RemoveAll(managedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	if err := runtime.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := deleteOptionalStateRows(tx, &proprojects.ProjectRAGCollectionShadow{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		if err := deleteOptionalStateRows(tx, &proprojects.RAGCollectionShadow{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		if err := deleteOptionalStateRows(tx, &proragbox.ProjectCollectionBinding{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		if err := deleteOptionalStateRows(tx, &proragbox.KnowledgeBaseBinding{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		if err := deleteOptionalStateRows(tx, &proragbox.KnowledgeBaseIngestToken{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		if err := deleteOptionalStateRows(tx, &proragbox.SharedProjectRAGCollection{}, "rag_collection_id = ?", target.ID); err != nil {
			return err
		}
		return tx.Delete(&knowledgeBaseDeleteTarget{}, target.ID).Error
	}); err != nil {
		return nil, err
	}

	return &proKnowledgeBaseDeletePreviewResponse{
		CollectionID:       target.CollectionID,
		ManagedSourcePath:  managedPath,
		HasUploadedFiles:   filesCount > 0,
		UploadedFilesCount: filesCount,
	}, nil
}

func loadKnowledgeBaseDeleteTarget(ctx context.Context, runtime *app.RuntimeContext, collectionID string) (*knowledgeBaseDeleteTarget, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}

	var target knowledgeBaseDeleteTarget
	err := runtime.DB.WithContext(ctx).
		Where("collection_id = ?", strings.TrimSpace(collectionID)).
		First(&target).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func loadKnowledgeBaseDeleteBinding(ctx context.Context, runtime *app.RuntimeContext, ragCollectionID uint) (*knowledgeBaseDeleteBinding, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}

	var binding knowledgeBaseDeleteBinding
	err := runtime.DB.WithContext(ctx).
		Where("rag_collection_id = ?", ragCollectionID).
		First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || isOptionalStateTableError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func isRagBoxCollectionMissing(err error) bool {
	var apiErr *proragbox.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func handleProRagBoxFileUpload(
	ctx context.Context,
	runtime *app.RuntimeContext,
	ragboxService *proragbox.Service,
	bindings *proragbox.BindingService,
	collection proragbox.SharedRAGCollection,
	sourcePath string,
	request *http.Request,
) (*proRagBoxWriteResponse, error) {
	if err := request.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		return nil, errors.New("file is required")
	}
	defer func() { _ = file.Close() }()

	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		return nil, err
	}

	fileName := safeLocalFileName(header.Filename)
	if fileName == "" {
		fileName = fmt.Sprintf("upload-%d.bin", time.Now().UnixNano())
	}
	targetPath := uniquePath(filepath.Join(sourcePath, fileName))
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(targetFile, file); err != nil {
		_ = targetFile.Close()
		return nil, err
	}
	if err := targetFile.Close(); err != nil {
		return nil, err
	}

	response := &proRagBoxWriteResponse{}
	response.Local.Mode = "disabled"
	response.Local.Indexed = false
	response.Local.DirPath = sourcePath
	response.Local.WrittenFiles = []string{targetPath}
	if proragboxUsesBleve(collection.ServiceMode) {
		if err := reindexLocalKnowledgeBase(ctx, runtime, collection.CollectionID); err != nil {
			return nil, err
		}
		response.Local.Mode = "bleve"
		response.Local.Indexed = true
	}

	client, binding, shouldWriteRemote, err := ragBoxWriteTarget(ctx, ragboxService, bindings, collection)
	if err != nil {
		return nil, err
	}
	if !shouldWriteRemote {
		response.RagBox.Skipped = true
		response.RagBox.Reason = "knowledge base does not use RagBox"
		return response, nil
	}

	sourceName := strings.TrimSpace(request.FormValue("source_name"))
	externalID := strings.TrimSpace(request.FormValue("external_id"))
	title := strings.TrimSpace(request.FormValue("title"))
	var metadata map[string]any
	rawMetadata := strings.TrimSpace(request.FormValue("metadata"))
	if rawMetadata != "" {
		if err := json.Unmarshal([]byte(rawMetadata), &metadata); err != nil {
			return nil, errors.New("metadata must be valid json")
		}
	}

	uploadResponse, err := client.UploadFile(ctx, binding.RagBoxCollectionID, proragbox.FileUploadRequest{
		FilePath:   targetPath,
		SourceName: sourceName,
		ExternalID: externalID,
		Title:      title,
		Metadata:   metadata,
	})
	if err != nil {
		return nil, err
	}

	response.RagBox.CollectionID = binding.RagBoxCollectionID
	response.RagBox.DocumentID = uploadResponse.DocumentID
	response.RagBox.JobID = uploadResponse.JobID
	response.RagBox.Status = uploadResponse.Status
	return response, nil
}

func handleProKnowledgeBaseFileUpload(
	ctx context.Context,
	runtime *app.RuntimeContext,
	ragboxService *proragbox.Service,
	bindings *proragbox.BindingService,
	collection proragbox.SharedRAGCollection,
	sourcePath string,
	request *http.Request,
) (*proRagBoxWriteResponse, error) {
	if err := request.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		return nil, errors.New("file is required")
	}
	defer func() { _ = file.Close() }()

	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		return nil, err
	}

	fileName := safeLocalFileName(header.Filename)
	if fileName == "" {
		fileName = fmt.Sprintf("upload-%d.bin", time.Now().UnixNano())
	}

	uploadMode := normalizeKnowledgeBaseUploadMode(request.FormValue("storage_mode"))
	originalPath := uniquePath(filepath.Join(sourcePath, fileName))
	targetFile, err := os.Create(originalPath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(targetFile, file); err != nil {
		_ = targetFile.Close()
		return nil, err
	}
	if err := targetFile.Close(); err != nil {
		return nil, err
	}

	finalLocalFiles := []string{originalPath}
	transientPaths := []string{originalPath}
	if extractedRoot, extractedFiles, extractErr := maybeExpandKnowledgeBaseArchive(originalPath, sourcePath); extractErr != nil {
		return nil, extractErr
	} else if extractedRoot != "" {
		finalLocalFiles = extractedFiles
		transientPaths = []string{extractedRoot}
		if err := os.Remove(originalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	response := &proRagBoxWriteResponse{}
	response.Local.Mode = string(uploadMode)
	response.Local.Indexed = false
	response.Local.DirPath = sourcePath
	response.Local.WrittenFiles = finalLocalFiles
	if proragboxUsesBleve(collection.ServiceMode) {
		if err := reindexLocalKnowledgeBase(ctx, runtime, collection.CollectionID); err != nil {
			return nil, err
		}
		response.Local.Indexed = true
	}

	client, binding, shouldWriteRemote, err := ragBoxWriteTarget(ctx, ragboxService, bindings, collection)
	if err != nil {
		return nil, err
	}
	if !shouldWriteRemote {
		response.RagBox.Skipped = true
		response.RagBox.Reason = "knowledge base does not use RagBox"
		if uploadMode == proKnowledgeBaseUploadModeTransient {
			for _, transientPath := range transientPaths {
				if strings.TrimSpace(transientPath) == "" {
					continue
				}
				if err := os.RemoveAll(transientPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					return nil, err
				}
			}
			response.Local.WrittenFiles = nil
		}
		return response, nil
	}

	sourceName := strings.TrimSpace(request.FormValue("source_name"))
	externalID := strings.TrimSpace(request.FormValue("external_id"))
	title := strings.TrimSpace(request.FormValue("title"))
	var metadata map[string]any
	rawMetadata := strings.TrimSpace(request.FormValue("metadata"))
	if rawMetadata != "" {
		if err := json.Unmarshal([]byte(rawMetadata), &metadata); err != nil {
			return nil, errors.New("metadata must be valid json")
		}
	}

	response.RagBox.CollectionID = binding.RagBoxCollectionID
	for idx, localPath := range finalLocalFiles {
		effectiveMetadata := map[string]any{
			"mcpbox_origin":        "managed_upload",
			"mcpbox_scope":         "knowledge_base",
			"mcpbox_collection_id": collection.CollectionID,
			"mcpbox_relative_path": filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(localPath, sourcePath), string(os.PathSeparator))),
		}
		for key, value := range metadata {
			effectiveMetadata[key] = value
		}
		uploadRequest := proragbox.FileUploadRequest{
			FilePath:   localPath,
			SourceName: sourceName,
			Metadata:   effectiveMetadata,
		}
		if len(finalLocalFiles) == 1 {
			uploadRequest.ExternalID = strings.TrimSpace(externalID)
			if uploadRequest.ExternalID == "" {
				uploadRequest.ExternalID = knowledgeBaseManagedUploadExternalID(collection.CollectionID, filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(localPath, sourcePath), string(os.PathSeparator))))
			}
			uploadRequest.Title = title
		} else {
			uploadRequest.ExternalID = knowledgeBaseManagedUploadExternalID(collection.CollectionID, filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(localPath, sourcePath), string(os.PathSeparator))))
			uploadRequest.Title = filepath.Base(localPath)
		}
		uploadResponse, err := client.UploadFile(ctx, binding.RagBoxCollectionID, uploadRequest)
		if err != nil {
			return nil, err
		}
		if idx == len(finalLocalFiles)-1 {
			response.RagBox.DocumentID = uploadResponse.DocumentID
			response.RagBox.JobID = uploadResponse.JobID
			response.RagBox.Status = uploadResponse.Status
		}
	}
	if uploadMode == proKnowledgeBaseUploadModeTransient {
		for _, transientPath := range transientPaths {
			if strings.TrimSpace(transientPath) == "" {
				continue
			}
			if err := os.RemoveAll(transientPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		response.Local.WrittenFiles = nil
	}
	return response, nil
}

func handleProRagBoxRecordUpsert(
	ctx context.Context,
	runtime *app.RuntimeContext,
	ragboxService *proragbox.Service,
	bindings *proragbox.BindingService,
	collection proragbox.SharedRAGCollection,
	sourcePath string,
	request proRagBoxRecordBatchRequest,
) (*proRagBoxWriteResponse, error) {
	if len(request.Records) == 0 {
		return nil, errors.New("records must not be empty")
	}

	recordRoot := filepath.Join(sourcePath, ".mcpboxpro-records")
	if err := os.MkdirAll(recordRoot, 0o755); err != nil {
		return nil, err
	}

	writtenFiles := make([]string, 0, len(request.Records))
	for _, record := range request.Records {
		if strings.TrimSpace(record.ExternalID) == "" {
			return nil, errors.New("record external_id is required")
		}
		if strings.TrimSpace(record.RecordType) == "" {
			return nil, errors.New("record record_type is required")
		}
		if strings.TrimSpace(record.Content) == "" {
			return nil, errors.New("record content is required")
		}

		filePath := filepath.Join(recordRoot, safeRecordFileName(record.ExternalID)+".md")
		if err := os.WriteFile(filePath, []byte(renderLocalRecordShadow(record)), 0o644); err != nil {
			return nil, err
		}
		writtenFiles = append(writtenFiles, filePath)
	}

	response := &proRagBoxWriteResponse{}
	response.Local.Mode = "disabled"
	response.Local.Indexed = false
	response.Local.DirPath = sourcePath
	response.Local.WrittenFiles = writtenFiles
	if proragboxUsesBleve(collection.ServiceMode) {
		if err := reindexLocalKnowledgeBase(ctx, runtime, collection.CollectionID); err != nil {
			return nil, err
		}
		response.Local.Mode = "bleve"
		response.Local.Indexed = true
	}

	client, binding, shouldWriteRemote, err := ragBoxWriteTarget(ctx, ragboxService, bindings, collection)
	if err != nil {
		return nil, err
	}
	if !shouldWriteRemote {
		response.RagBox.Skipped = true
		response.RagBox.Reason = "knowledge base does not use RagBox"
		return response, nil
	}

	upsertResponse, err := client.UpsertRecords(ctx, binding.RagBoxCollectionID, proragbox.RecordBatchRequest{
		Source:  request.Source,
		Records: request.Records,
	})
	if err != nil {
		return nil, err
	}

	response.RagBox.CollectionID = binding.RagBoxCollectionID
	response.RagBox.JobID = upsertResponse.JobID
	response.RagBox.Status = upsertResponse.Status
	response.RagBox.AcceptedRecords = upsertResponse.AcceptedRecords
	return response, nil
}

func syncKnowledgeBaseSourceToRagBox(
	ctx context.Context,
	bindings *proragbox.BindingService,
	client *proragbox.Client,
	binding *proragbox.KnowledgeBaseBinding,
	collection proragbox.SharedRAGCollection,
) (*proRagBoxWriteResponse, error) {
	if bindings == nil || client == nil || binding == nil {
		return nil, errors.New("ragbox sync dependencies are not configured")
	}
	if bindings.StateDB() == nil {
		return nil, errors.New("ragbox binding database is not available")
	}

	sourceRoot := strings.TrimSpace(collection.SourcePath)
	response := &proRagBoxWriteResponse{}
	response.Local.Mode = "filesystem_sync"
	response.Local.Indexed = false
	response.Local.DirPath = sourceRoot
	response.RagBox.CollectionID = binding.RagBoxCollectionID
	if proragboxUsesBleve(collection.ServiceMode) {
		response.Local.Mode = "bleve"
		response.Local.Indexed = true
	}

	if sourceRoot == "" {
		response.RagBox.Skipped = true
		response.RagBox.Reason = "knowledge base source_path is not configured"
		return response, nil
	}

	candidates, err := collectKnowledgeBaseFilesystemSyncCandidates(sourceRoot, collection.CollectionID)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		files = append(files, candidate.AbsolutePath)
	}
	response.Local.WrittenFiles = files

	existingItems, err := loadKnowledgeBaseFilesystemSyncItems(ctx, bindings, binding.RAGCollectionID)
	if err != nil {
		return nil, err
	}
	existingByRelativePath := make(map[string]proragbox.KnowledgeBaseFilesystemSyncItem, len(existingItems))
	for _, item := range existingItems {
		existingByRelativePath[item.RelativePath] = item
	}

	deleteExternalIDs := make([]string, 0)
	uploadCandidates := make([]knowledgeBaseFilesystemSyncCandidate, 0, len(candidates))
	nextItems := make([]proragbox.KnowledgeBaseFilesystemSyncItem, 0, len(candidates))
	for _, candidate := range candidates {
		existing, exists := existingByRelativePath[candidate.RelativePath]
		if exists && strings.TrimSpace(existing.ContentHash) == candidate.ContentHash {
			nextItems = append(nextItems, proragbox.KnowledgeBaseFilesystemSyncItem{
				RAGCollectionID: binding.RAGCollectionID,
				RelativePath:    candidate.RelativePath,
				ExternalID:      existing.ExternalID,
				ContentHash:     candidate.ContentHash,
				SizeBytes:       candidate.SizeBytes,
			})
			delete(existingByRelativePath, candidate.RelativePath)
			continue
		}
		if exists && strings.TrimSpace(existing.ExternalID) != "" {
			deleteExternalIDs = append(deleteExternalIDs, existing.ExternalID)
			delete(existingByRelativePath, candidate.RelativePath)
		}
		nextItems = append(nextItems, proragbox.KnowledgeBaseFilesystemSyncItem{
			RAGCollectionID: binding.RAGCollectionID,
			RelativePath:    candidate.RelativePath,
			ExternalID:      candidate.ExternalID,
			ContentHash:     candidate.ContentHash,
			SizeBytes:       candidate.SizeBytes,
		})
		uploadCandidates = append(uploadCandidates, candidate)
	}
	for _, orphan := range existingByRelativePath {
		if strings.TrimSpace(orphan.ExternalID) != "" {
			deleteExternalIDs = append(deleteExternalIDs, orphan.ExternalID)
		}
	}

	if len(deleteExternalIDs) > 0 {
		for _, batch := range chunkStrings(deleteExternalIDs, 100) {
			if len(batch) == 0 {
				continue
			}
			if _, err := client.DeleteDocuments(ctx, binding.RagBoxCollectionID, batch); err != nil {
				return nil, err
			}
		}
	}

	uploadedCount := int64(0)
	for idx, candidate := range uploadCandidates {
		uploadResponse, err := client.UploadFile(ctx, binding.RagBoxCollectionID, proragbox.FileUploadRequest{
			FilePath:   candidate.AbsolutePath,
			SourceName: "mcpboxpro-kb-filesystem",
			ExternalID: candidate.ExternalID,
			Title:      filepath.Base(candidate.AbsolutePath),
			Metadata: map[string]any{
				"mcpbox_origin":        "filesystem",
				"mcpbox_scope":         "knowledge_base",
				"mcpbox_collection_id": collection.CollectionID,
				"mcpbox_relative_path": candidate.RelativePath,
				"mcpbox_content_hash":  candidate.ContentHash,
			},
		})
		if err != nil {
			return nil, err
		}
		uploadedCount++
		if idx == len(uploadCandidates)-1 {
			response.RagBox.DocumentID = uploadResponse.DocumentID
			response.RagBox.JobID = uploadResponse.JobID
			response.RagBox.Status = uploadResponse.Status
		}
	}

	if err := replaceKnowledgeBaseFilesystemSyncItems(ctx, bindings, binding.RAGCollectionID, nextItems); err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		response.RagBox.Skipped = true
		response.RagBox.Reason = "no files were found in the knowledge base source path"
	}
	response.RagBox.AcceptedRecords = uploadedCount
	return response, nil
}

type knowledgeBaseFilesystemSyncCandidate struct {
	AbsolutePath string
	RelativePath string
	ExternalID   string
	ContentHash  string
	SizeBytes    int64
}

func collectKnowledgeBaseFilesystemSyncCandidates(root, collectionID string) ([]knowledgeBaseFilesystemSyncCandidate, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []knowledgeBaseFilesystemSyncCandidate{}, nil
		}
		return nil, err
	}

	candidates := make([]knowledgeBaseFilesystemSyncCandidate, 0)
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(strings.TrimSpace(relativePath))
		contentHash, err := fileSHA256(path)
		if err != nil {
			return err
		}
		candidates = append(candidates, knowledgeBaseFilesystemSyncCandidate{
			AbsolutePath: path,
			RelativePath: relativePath,
			ExternalID:   knowledgeBaseFilesystemExternalID(collectionID, relativePath),
			ContentHash:  contentHash,
			SizeBytes:    info.Size(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return candidates, nil
}

func loadKnowledgeBaseFilesystemSyncItems(
	ctx context.Context,
	bindings *proragbox.BindingService,
	ragCollectionID uint,
) ([]proragbox.KnowledgeBaseFilesystemSyncItem, error) {
	if bindings == nil || bindings.StateDB() == nil {
		return nil, errors.New("ragbox binding database is not available")
	}
	items := make([]proragbox.KnowledgeBaseFilesystemSyncItem, 0)
	err := bindings.StateDB().WithContext(ctx).
		Where("rag_collection_id = ?", ragCollectionID).
		Order("relative_path asc").
		Find(&items).Error
	return items, err
}

func replaceKnowledgeBaseFilesystemSyncItems(
	ctx context.Context,
	bindings *proragbox.BindingService,
	ragCollectionID uint,
	items []proragbox.KnowledgeBaseFilesystemSyncItem,
) error {
	if bindings == nil || bindings.StateDB() == nil {
		return errors.New("ragbox binding database is not available")
	}
	return bindings.StateDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("rag_collection_id = ?", ragCollectionID).Delete(&proragbox.KnowledgeBaseFilesystemSyncItem{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			item.ID = 0
			item.RAGCollectionID = ragCollectionID
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func knowledgeBaseFilesystemExternalID(collectionID, relativePath string) string {
	sum := sha256.Sum256([]byte(collectionID + "::" + relativePath))
	return "mcpboxpro:kbfs:" + hex.EncodeToString(sum[:16])
}

func knowledgeBaseManagedUploadExternalID(collectionID, relativePath string) string {
	sum := sha256.Sum256([]byte(collectionID + "::managed::" + relativePath))
	return "mcpboxpro:kbup:" + hex.EncodeToString(sum[:16])
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func reindexLocalKnowledgeBase(ctx context.Context, runtime *app.RuntimeContext, collectionID string) error {
	if runtime == nil {
		return errors.New("runtime is not available")
	}

	targetHost, adminPort := resolveLocalAdminEndpoint(runtime.Edition.AdminHost, runtime.Edition.AdminPort)

	endpoint := fmt.Sprintf("http://%s:%d/api/rag/collections/%s/reindex", targetHost, adminPort, collectionID)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = "local bleve reindex request failed"
		}
		return errors.New(message)
	}
	return nil
}

func resolveLocalAdminEndpoint(defaultHost string, defaultPort int) (string, int) {
	adminHost, adminPort := adminEndpointOverrideFromArgs()
	if adminHost == "" {
		adminHost = strings.TrimSpace(os.Getenv("MCPBOX_HOST"))
	}
	if adminHost == "" {
		adminHost = strings.TrimSpace(defaultHost)
	}
	if adminHost == "" || adminHost == "0.0.0.0" || adminHost == "::" {
		adminHost = "127.0.0.1"
	}

	if adminPort == 0 {
		if rawPort := strings.TrimSpace(os.Getenv("MCPBOX_PORT")); rawPort != "" {
			if parsedPort, err := strconv.Atoi(rawPort); err == nil && parsedPort > 0 {
				adminPort = parsedPort
			}
		}
	}
	if adminPort == 0 {
		adminPort = defaultPort
	}
	if adminPort == 0 {
		adminPort = 38181
	}

	targetHost := adminHost
	if ip := net.ParseIP(adminHost); ip != nil && ip.IsUnspecified() {
		targetHost = "127.0.0.1"
	}
	return targetHost, adminPort
}

func adminEndpointOverrideFromArgs() (string, int) {
	args := os.Args[1:]
	host := ""
	port := 0
	for idx := 0; idx < len(args); idx++ {
		arg := strings.TrimSpace(args[idx])
		switch {
		case arg == "--host" && idx+1 < len(args):
			host = strings.TrimSpace(args[idx+1])
			idx++
		case strings.HasPrefix(arg, "--host="):
			host = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		case arg == "--port" && idx+1 < len(args):
			if parsedPort, err := strconv.Atoi(strings.TrimSpace(args[idx+1])); err == nil && parsedPort > 0 {
				port = parsedPort
			}
			idx++
		case strings.HasPrefix(arg, "--port="):
			if parsedPort, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--port="))); err == nil && parsedPort > 0 {
				port = parsedPort
			}
		}
	}
	return host, port
}

func resolveManagedKnowledgeBasePath(collectionID string) string {
	return filepath.Join(".", "knowledge_base", "managed", sanitizeKnowledgeBasePathSegment(collectionID))
}

func ensureManagedKnowledgeBasePath(collection proragbox.SharedRAGCollection) (string, error) {
	path := resolveManagedKnowledgeBasePath(collection.CollectionID)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func countKnowledgeBaseManagedFiles(root string) (int, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0, nil
	}
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	count := 0
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}
		count++
		return nil
	}); err != nil {
		return 0, err
	}
	return count, nil
}

func sanitizeKnowledgeBasePathSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "collection"
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", " ", "-")
	return replacer.Replace(value)
}

func maybeExpandKnowledgeBaseArchive(archivePath string, sourcePath string) (string, []string, error) {
	format := detectKnowledgeBaseArchiveFormat(archivePath)
	if format == "" {
		return "", nil, nil
	}

	baseName := archiveExtractionBaseName(filepath.Base(archivePath))
	targetRoot := uniquePath(filepath.Join(sourcePath, baseName))
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return "", nil, err
	}

	var extractedFiles []string
	var err error
	switch format {
	case "zip":
		extractedFiles, err = extractZipArchive(archivePath, targetRoot)
	case "tar":
		extractedFiles, err = extractTarArchive(archivePath, targetRoot, false)
	case "tgz":
		extractedFiles, err = extractTarArchive(archivePath, targetRoot, true)
	default:
		err = fmt.Errorf("unsupported archive format: %s", format)
	}
	if err != nil {
		_ = os.RemoveAll(targetRoot)
		return "", nil, err
	}
	if len(extractedFiles) == 0 {
		_ = os.RemoveAll(targetRoot)
		return "", nil, errors.New("archive did not contain any regular files")
	}

	return targetRoot, extractedFiles, nil
}

func detectKnowledgeBaseArchiveFormat(path string) string {
	lower := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "tgz"
	case strings.HasSuffix(lower, ".tar"):
		return "tar"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	default:
		return ""
	}
}

func archiveExtractionBaseName(name string) string {
	base := safeRecordFileName(trimKnowledgeBaseArchiveSuffix(name))
	if base == "" {
		return fmt.Sprintf("archive-%d", time.Now().UnixNano())
	}
	return base
}

func trimKnowledgeBaseArchiveSuffix(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return name[:len(name)-len(".tar.gz")]
	case strings.HasSuffix(lower, ".tgz"):
		return name[:len(name)-len(".tgz")]
	case strings.HasSuffix(lower, ".tar"):
		return name[:len(name)-len(".tar")]
	case strings.HasSuffix(lower, ".zip"):
		return name[:len(name)-len(".zip")]
	default:
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
}

func extractZipArchive(archivePath string, targetRoot string) ([]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	files := make([]string, 0)
	for _, entry := range reader.File {
		entryPath, err := safeArchiveDestination(targetRoot, entry.Name)
		if err != nil {
			return nil, err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(entryPath, 0o755); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
			return nil, err
		}
		in, err := entry.Open()
		if err != nil {
			return nil, err
		}
		if err := writeArchiveEntryFile(entryPath, in); err != nil {
			_ = in.Close()
			return nil, err
		}
		if err := in.Close(); err != nil {
			return nil, err
		}
		files = append(files, entryPath)
	}
	return files, nil
}

func extractTarArchive(archivePath string, targetRoot string, gzipped bool) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var reader io.Reader = file
	if gzipped {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzipReader.Close() }()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)
	files := make([]string, 0)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		entryPath, err := safeArchiveDestination(targetRoot, header.Name)
		if err != nil {
			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(entryPath, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
				return nil, err
			}
			if err := writeArchiveEntryFile(entryPath, tarReader); err != nil {
				return nil, err
			}
			files = append(files, entryPath)
		default:
			continue
		}
	}
	return files, nil
}

func safeArchiveDestination(root string, entryName string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(entryName))
	if cleaned == "." || cleaned == "" {
		return "", errors.New("archive contains an empty entry name")
	}
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive entry: %s", entryName)
	}

	fullPath := filepath.Join(root, cleaned)
	relative, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive entry: %s", entryName)
	}
	return fullPath, nil
}

func writeArchiveEntryFile(path string, reader io.Reader) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func renderLocalRecordShadow(record proragbox.RecordInput) string {
	var builder strings.Builder
	builder.WriteString("# ")
	title := strings.TrimSpace(record.Title)
	if title == "" {
		title = record.ExternalID
	}
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString("- External ID: ")
	builder.WriteString(record.ExternalID)
	builder.WriteString("\n")
	builder.WriteString("- Record Type: ")
	builder.WriteString(record.RecordType)
	builder.WriteString("\n")
	if strings.TrimSpace(record.Summary) != "" {
		builder.WriteString("\n## Summary\n\n")
		builder.WriteString(strings.TrimSpace(record.Summary))
		builder.WriteString("\n")
	}
	builder.WriteString("\n## Content\n\n")
	builder.WriteString(strings.TrimSpace(record.Content))
	builder.WriteString("\n")
	if len(record.Fields) > 0 {
		raw, _ := json.MarshalIndent(record.Fields, "", "  ")
		builder.WriteString("\n## Fields\n\n```json\n")
		builder.Write(raw)
		builder.WriteString("\n```\n")
	}
	if len(record.Metadata) > 0 {
		raw, _ := json.MarshalIndent(record.Metadata, "", "  ")
		builder.WriteString("\n## Metadata\n\n```json\n")
		builder.Write(raw)
		builder.WriteString("\n```\n")
	}
	return builder.String()
}
