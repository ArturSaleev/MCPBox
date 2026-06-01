package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/rag"
	"github.com/google/uuid"
)

type createRAGCollectionRequest struct {
	Name        string `json:"name"`
	SourcePath  string `json:"source_path"`
	AutoReindex bool   `json:"auto_reindex"`
}

type updateRAGCollectionRequest struct {
	Name        string `json:"name"`
	SourcePath  string `json:"source_path"`
	AutoReindex bool   `json:"auto_reindex"`
}

type linkRAGCollectionRequest struct {
	CollectionID string `json:"collection_id"`
}

type indexRAGCollectionRequest struct {
	DirPath string `json:"dir_path"`
}

type searchRAGCollectionRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type ragCollectionResponse struct {
	ID           uint   `json:"id"`
	CollectionID string `json:"collection_id"`
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	SourcePath   string `json:"source_path"`
	AutoReindex  bool   `json:"auto_reindex"`
	IndexPath    string `json:"index_path"`
}

type ragSearchResponse struct {
	Items []rag.Chunk `json:"items"`
}

func (s *Server) handleListRAGCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := s.store.ListRAGCollections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := make([]ragCollectionResponse, 0, len(collections))
	for _, collection := range collections {
		response = append(response, mapRAGCollection(collection))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": response})
}

func (s *Server) handleCreateRAGCollection(w http.ResponseWriter, r *http.Request) {
	var req createRAGCollectionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	sourcePath := strings.TrimSpace(req.SourcePath)
	if sourcePath == "" {
		writeError(w, http.StatusBadRequest, errors.New("source_path is required"))
		return
	}

	collectionID := uuid.NewString()

	indexPath, err := s.resolveRAGIndexPath("", collectionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	collection := &models.RAGCollection{
		CollectionID: collectionID,
		Name:         name,
		DataType:     models.RAGDataTypeCode,
		SourcePath:   sourcePath,
		AutoReindex:  req.AutoReindex,
		IndexPath:    indexPath,
	}

	if err := reindexCollection(*collection, sourcePath); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.CreateRAGCollection(r.Context(), collection); err != nil {
		_ = os.RemoveAll(collection.IndexPath)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "rag_collection_created", clientActor(r), collectionID)
	s.logAudit(r.Context(), nil, nil, "rag_collection_indexed", clientActor(r), collectionID)
	writeJSON(w, http.StatusCreated, mapRAGCollection(*collection))
}

func (s *Server) handleRAGCollectionAction(w http.ResponseWriter, r *http.Request) {
	collectionID, tail, ok := parseStringIDTail(r.URL.Path, "/api/rag/collections/")
	if !ok || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	collection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	switch tail {
	case "index":
		s.handleIndexRAGCollection(w, r, *collection)
	case "search":
		s.handleSearchRAGCollection(w, r, *collection)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleDeleteRAGCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, ok := parseSingleStringID(r.URL.Path, "/api/rag/collections/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	collection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	if err := os.RemoveAll(collection.IndexPath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.store.DeleteRAGCollection(r.Context(), collectionID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "rag_collection_deleted", clientActor(r), collectionID)
	writeJSON(w, http.StatusOK, map[string]any{"collection_id": collectionID, "deleted": true})
}

func (s *Server) handleUpdateRAGCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, ok := parseSingleStringID(r.URL.Path, "/api/rag/collections/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	collection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	var req updateRAGCollectionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	sourcePath := strings.TrimSpace(req.SourcePath)
	if sourcePath == "" {
		writeError(w, http.StatusBadRequest, errors.New("source_path is required"))
		return
	}

	updatedPreview := *collection
	updatedPreview.Name = name
	updatedPreview.SourcePath = sourcePath
	updatedPreview.AutoReindex = req.AutoReindex

	if err := reindexCollection(updatedPreview, sourcePath); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := s.store.UpdateRAGCollectionConfig(r.Context(), collectionID, name, sourcePath, req.AutoReindex); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	updatedCollection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if updatedCollection == nil {
		http.NotFound(w, r)
		return
	}

	s.logAudit(r.Context(), nil, nil, "rag_collection_updated", clientActor(r), collectionID)
	s.logAudit(r.Context(), nil, nil, "rag_collection_indexed", clientActor(r), collectionID)
	writeJSON(w, http.StatusOK, mapRAGCollection(*updatedCollection))
}

func (s *Server) handleProjectRAGCollectionAction(w http.ResponseWriter, r *http.Request, projectID uint) {
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}

	var req linkRAGCollectionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	collection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), strings.TrimSpace(req.CollectionID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	if err := s.store.LinkRAGCollectionToProject(r.Context(), projectID, collection.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, nil, "rag_collection_linked", clientActor(r), collection.CollectionID)
	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleProjectRAGCollectionDelete(w http.ResponseWriter, r *http.Request, projectID uint, collectionID string) {
	collection, err := s.store.GetRAGCollectionByCollectionID(r.Context(), collectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	if err := s.store.UnlinkRAGCollectionFromProject(r.Context(), projectID, collection.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if updatedProject == nil {
		http.NotFound(w, r)
		return
	}

	s.logAudit(r.Context(), &projectID, nil, "rag_collection_unlinked", clientActor(r), collection.CollectionID)
	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleIndexRAGCollection(w http.ResponseWriter, r *http.Request, collection models.RAGCollection) {
	var req indexRAGCollectionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	dirPath := strings.TrimSpace(req.DirPath)
	if dirPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("dir_path is required"))
		return
	}

	if err := reindexCollection(collection, dirPath); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.UpdateRAGCollectionConfig(r.Context(), collection.CollectionID, collection.Name, dirPath, collection.AutoReindex); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "rag_collection_indexed", clientActor(r), collection.CollectionID)
	writeJSON(w, http.StatusOK, map[string]any{
		"collection_id": collection.CollectionID,
		"indexed":       true,
		"dir_path":      dirPath,
	})
}

func (s *Server) handleSearchRAGCollection(w http.ResponseWriter, r *http.Request, collection models.RAGCollection) {
	var req searchRAGCollectionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	index, err := rag.NewCollection(collection.CollectionID, collection.Name, collection.IndexPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer func() { _ = index.Close() }()

	items, err := index.Search(strings.TrimSpace(req.Query), req.Limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, ragSearchResponse{Items: items})
}

func mapRAGCollection(collection models.RAGCollection) ragCollectionResponse {
	return ragCollectionResponse{
		ID:           collection.ID,
		CollectionID: collection.CollectionID,
		Name:         collection.Name,
		DataType:     normalizedRAGDataType(collection.DataType),
		SourcePath:   collection.SourcePath,
		AutoReindex:  collection.AutoReindex,
		IndexPath:    collection.IndexPath,
	}
}

func reindexCollection(collection models.RAGCollection, dirPath string) error {
	index, err := rag.NewCollection(collection.CollectionID, collection.Name, collection.IndexPath)
	if err != nil {
		return err
	}
	defer func() { _ = index.Close() }()

	return index.IndexFolder(dirPath)
}

func (s *Server) resolveRAGIndexPath(indexPath, collectionID string) (string, error) {
	if strings.TrimSpace(indexPath) != "" {
		return filepath.Abs(indexPath)
	}

	baseRoot := "."
	if s != nil && s.store != nil && strings.TrimSpace(s.store.DataRoot()) != "" {
		baseRoot = s.store.DataRoot()
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		baseRoot = cwd
	}
	return filepath.Join(baseRoot, "knowledge_base", "indexes", sanitizeRAGPathSegment(collectionID)), nil
}

func sanitizeRAGPathSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "collection"
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", " ", "-")
	return replacer.Replace(value)
}

func parseUintParam(raw string) (uint, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, errors.New("invalid uint")
	}
	return uint(parsed), nil
}

func parseSingleStringID(rawPath, prefix string) (string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", false
	}
	return trimmed, true
}

func parseProjectStringTail(rawPath, prefix, middle string) (uint, string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 || strings.TrimSpace(parts[1]) != middle {
		return 0, "", false
	}
	projectID, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, "", false
	}
	collectionID := strings.TrimSpace(parts[2])
	if collectionID == "" {
		return 0, "", false
	}
	return uint(projectID), collectionID, true
}

func normalizedRAGDataType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", models.RAGDataTypeCode:
		return models.RAGDataTypeCode
	case models.RAGDataTypeDocuments:
		return models.RAGDataTypeDocuments
	case models.RAGDataTypeDialogs:
		return models.RAGDataTypeDialogs
	default:
		return models.RAGDataTypeCode
	}
}
