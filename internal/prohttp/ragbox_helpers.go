package prohttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"gorm.io/gorm"
)

func ragBoxClientFromService(ctx context.Context, service *proragbox.Service) (*proragbox.Client, error) {
	if service == nil {
		return nil, errors.New("ragbox service is not configured")
	}
	config, err := service.GetClientConfig(ctx)
	if err != nil {
		return nil, err
	}
	return proragbox.NewClient(config)
}

func ragBoxHTTPStatusFromError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound
	}

	if apiErr, ok := err.(*proragbox.APIError); ok && apiErr.StatusCode > 0 {
		return apiErr.StatusCode
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "not linked") || strings.Contains(message, "does not have a ragbox workspace") {
		return http.StatusConflict
	}
	if strings.Contains(message, "not configured") || strings.Contains(message, "invalid") {
		return http.StatusBadRequest
	}
	return http.StatusBadGateway
}

func ragBoxCollectionActionFromPath(path string) (string, string, error) {
	prefix := "/api/pro/ragbox/collections/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", errors.New("invalid ragbox collection path")
	}
	trimmed := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, prefix)), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", "", errors.New("invalid ragbox collection action path")
	}
	return strings.TrimSpace(parts[0]), strings.Join(parts[1:], "/"), nil
}

func ragBoxCollectionIDFromDeletePath(path string) (string, error) {
	prefix := "/api/pro/ragbox/collections/"
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid ragbox collection path")
	}

	trimmed := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, prefix)), "/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", errors.New("invalid ragbox collection delete path")
	}
	return trimmed, nil
}

func ragBoxIngestPublicIDFromPath(path string) (string, error) {
	prefix := "/api/pro/ragbox/ingest/"
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid ragbox ingest path")
	}
	publicID := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, prefix)), "/")
	if publicID == "" {
		return "", errors.New("invalid ragbox ingest path")
	}
	return publicID, nil
}

func ragBoxProjectActionFromPath(path string) (uint, string, error) {
	prefix := "/api/pro/ragbox/projects/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", errors.New("invalid ragbox project path")
	}

	trimmed := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, prefix)), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return 0, "", errors.New("invalid ragbox project action path")
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", errors.New("invalid project id")
	}

	return uint(id), strings.Join(parts[1:], "/"), nil
}

func ragBoxProjectCollectionActionFromPath(path string) (uint, string, string, error) {
	prefix := "/api/pro/ragbox/projects/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", "", errors.New("invalid ragbox collection path")
	}

	trimmed := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, prefix)), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 4 || parts[1] != "collections" {
		return 0, "", "", errors.New("invalid ragbox collection action path")
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", "", errors.New("invalid project id")
	}

	return uint(id), strings.TrimSpace(parts[2]), strings.Join(parts[3:], "/"), nil
}

func isRagBoxSearchPath(path string) bool {
	_, _, action, err := ragBoxProjectCollectionActionFromPath(path)
	return err == nil && action == "search"
}

func isRagBoxGlobalSearchPath(path string) bool {
	_, action, err := ragBoxCollectionActionFromPath(path)
	return err == nil && action == "search"
}

func normalizeKnowledgeBaseServiceMode(value string) string {
	switch strings.TrimSpace(value) {
	case "bleve_only", "ragbox_only", "dual":
		return strings.TrimSpace(value)
	default:
		return "bleve_only"
	}
}

func proragboxUsesRagBox(mode string) bool {
	switch normalizeKnowledgeBaseServiceMode(mode) {
	case "ragbox_only", "dual":
		return true
	default:
		return false
	}
}

func proragboxUsesBleve(mode string) bool {
	switch normalizeKnowledgeBaseServiceMode(mode) {
	case "bleve_only", "dual":
		return true
	default:
		return false
	}
}

func bearerTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func ragBoxWriteTarget(
	ctx context.Context,
	service *proragbox.Service,
	bindings *proragbox.BindingService,
	collection proragbox.SharedRAGCollection,
) (*proragbox.Client, *proragbox.KnowledgeBaseBinding, bool, error) {
	if service == nil || bindings == nil {
		return nil, nil, false, errors.New("ragbox services are not configured")
	}

	config, err := service.GetClientConfig(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	if !proragboxUsesRagBox(collection.ServiceMode) {
		return nil, nil, false, nil
	}

	client, err := proragbox.NewClient(config)
	if err != nil {
		return nil, nil, false, err
	}

	binding, _, err := ensureGlobalKnowledgeBaseBinding(ctx, bindings, client, collection.CollectionID)
	if err != nil {
		return nil, nil, false, err
	}
	if binding == nil || strings.TrimSpace(binding.RagBoxCollectionID) == "" {
		return nil, nil, false, errors.New("knowledge base does not have a ragbox collection binding yet")
	}

	return client, binding, true, nil
}

func ensureGlobalKnowledgeBaseBinding(ctx context.Context, bindings *proragbox.BindingService, client *proragbox.Client, collectionID string) (*proragbox.KnowledgeBaseBinding, *proragbox.SharedRAGCollection, error) {
	result, err := bindings.EnsureKnowledgeBase(ctx, client, collectionID, "")
	if err != nil {
		return nil, nil, err
	}
	return &result.Binding, &result.Collection, nil
}

func uniquePath(candidate string) string {
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate
	}

	ext := filepath.Ext(candidate)
	base := strings.TrimSuffix(candidate, ext)
	for idx := 1; ; idx++ {
		next := fmt.Sprintf("%s-%d%s", base, idx, ext)
		if _, err := os.Stat(next); errors.Is(err, os.ErrNotExist) {
			return next
		}
	}
}

func safeLocalFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.Trim(name, ". ")
	return name
}

func safeRecordFileName(externalID string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", " ", "-")
	value := replacer.Replace(strings.TrimSpace(externalID))
	value = strings.Trim(value, "-.")
	if value == "" {
		return fmt.Sprintf("record-%d", time.Now().UnixNano())
	}
	return value
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

func isOptionalStateTableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "no such table") ||
		strings.Contains(message, "does not exist")
}

func deleteOptionalStateRows(tx *gorm.DB, model any, query string, args ...any) error {
	if tx == nil {
		return errors.New("database transaction is not available")
	}
	err := tx.Where(query, args...).Delete(model).Error
	if err == nil || isOptionalStateTableError(err) {
		return nil
	}
	return err
}

func normalizeKnowledgeBaseUploadMode(raw string) proKnowledgeBaseUploadMode {
	switch strings.TrimSpace(raw) {
	case string(proKnowledgeBaseUploadModeTransient):
		return proKnowledgeBaseUploadModeTransient
	default:
		return proKnowledgeBaseUploadModeKeep
	}
}
