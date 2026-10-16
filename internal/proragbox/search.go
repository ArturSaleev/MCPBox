package proragbox

import (
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
	"strconv"
	"strings"
	"sync"
)

type ProSearchRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ProSearchHit struct {
	ID        string  `json:"id"`
	Source    string  `json:"source"`
	Score     float32 `json:"score,omitempty"`
	FilePath  string  `json:"file_path,omitempty"`
	Section   string  `json:"section,omitempty"`
	Title     string  `json:"title,omitempty"`
	Content   string  `json:"content"`
	DedupeKey string  `json:"dedupe_key,omitempty"`
}

type ProSearchDiagnostics struct {
	Mode          string `json:"mode"`
	BleveCount    int    `json:"bleve_count"`
	RagBoxCount   int    `json:"ragbox_count"`
	MergedCount   int    `json:"merged_count"`
	RagBoxSkipped bool   `json:"ragbox_skipped,omitempty"`
	RagBoxReason  string `json:"ragbox_reason,omitempty"`
}

type ProSearchResponse struct {
	Items       []ProSearchHit       `json:"items"`
	Diagnostics ProSearchDiagnostics `json:"diagnostics"`
}

type localBleveChunk struct {
	ID       string `json:"id"`
	FilePath string `json:"file_path"`
	Section  string `json:"section,omitempty"`
	Content  string `json:"content"`
}

func SearchKnowledge(
	ctx context.Context,
	runtime AdminRuntime,
	config ClientConfig,
	searchMode string,
	client *Client,
	binding *ProjectCollectionBinding,
	collectionID string,
	query string,
	limit int,
) (ProSearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return ProSearchResponse{}, errors.New("query is required")
	}

	limit = normalizeSearchLimit(limit)
	mode := normalizeSearchMode(searchMode)

	response := ProSearchResponse{
		Diagnostics: ProSearchDiagnostics{Mode: mode},
	}

	var bleveHits []ProSearchHit
	var ragboxHits []ProSearchHit
	var bleveErr error
	var ragboxErr error

	var wg sync.WaitGroup
	shouldQueryBleve := mode != "ragbox_only"
	if shouldQueryBleve {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bleveHits, bleveErr = searchLocalBleve(ctx, runtime, collectionID, query, limit)
		}()
	}

	shouldQueryRagBox := mode != "bleve_only" && client != nil && binding != nil && strings.TrimSpace(binding.RagBoxCollectionID) != ""
	if shouldQueryRagBox {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ragboxHits, ragboxErr = searchRagBoxCollection(ctx, client, binding.RagBoxCollectionID, query, limit)
		}()
	} else if mode != "bleve_only" {
		response.Diagnostics.RagBoxSkipped = true
		switch {
		case client == nil:
			response.Diagnostics.RagBoxReason = "ragbox is not configured"
		case binding == nil || strings.TrimSpace(binding.RagBoxCollectionID) == "":
			response.Diagnostics.RagBoxReason = "ragbox collection binding is missing"
		default:
			response.Diagnostics.RagBoxReason = "ragbox search unavailable"
		}
	}

	wg.Wait()

	if bleveErr != nil {
		return ProSearchResponse{}, bleveErr
	}
	response.Diagnostics.BleveCount = len(bleveHits)

	if shouldQueryRagBox {
		if ragboxErr != nil {
			return ProSearchResponse{}, ragboxErr
		}
		response.Diagnostics.RagBoxCount = len(ragboxHits)
	}

	switch mode {
	case "bleve_only":
		response.Items = bleveHits
		response.Diagnostics.MergedCount = len(response.Items)
	case "ragbox_only":
		response.Items = ragboxHits
		response.Diagnostics.MergedCount = len(response.Items)
	case "dual_search":
		response.Items = append(append([]ProSearchHit{}, bleveHits...), ragboxHits...)
		sortSearchHits(response.Items)
		if len(response.Items) > limit {
			response.Items = response.Items[:limit]
		}
		response.Diagnostics.MergedCount = len(response.Items)
	default:
		response.Items = mergeHybridHits(bleveHits, ragboxHits, limit)
		response.Diagnostics.MergedCount = len(response.Items)
	}

	return response, nil
}

func searchLocalBleve(ctx context.Context, runtime AdminRuntime, collectionID, query string, limit int) ([]ProSearchHit, error) {
	if runtime == nil {
		return nil, errors.New("runtime is not available")
	}

	adminHost, adminPort := resolveAdminRuntimeEndpoint(runtime)

	payload, err := json.Marshal(map[string]any{
		"query": query,
		"limit": limit,
	})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("http://%s:%d/api/rag/collections/%s/search", adminHost, adminPort, collectionID)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer func() { _ = httpResponse.Body.Close() }()

	body, _ := io.ReadAll(httpResponse.Body)
	if httpResponse.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = "local bleve search request failed"
		}
		return nil, errors.New(message)
	}

	var envelope struct {
		Items []localBleveChunk `json:"items"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	hits := make([]ProSearchHit, 0, len(envelope.Items))
	for idx, item := range envelope.Items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("bleve-%d", idx)
		}
		filePath := strings.TrimSpace(item.FilePath)
		hits = append(hits, ProSearchHit{
			ID:        id,
			Source:    "bleve",
			Score:     float32(len(envelope.Items) - idx),
			FilePath:  filePath,
			Section:   strings.TrimSpace(item.Section),
			Content:   content,
			DedupeKey: dedupeKeyForBleve(filePath, content),
		})
	}
	normalizeHitScores(hits)
	return hits, nil
}

func searchRagBoxCollection(ctx context.Context, client *Client, collectionID, query string, limit int) ([]ProSearchHit, error) {
	if client == nil {
		return nil, errors.New("ragbox client is not configured")
	}

	includeContent := true
	response, err := client.SearchCollection(ctx, collectionID, SearchRequest{
		Query:          query,
		TopK:           limit,
		IncludeContent: &includeContent,
	})
	if err != nil {
		return nil, err
	}

	hits := make([]ProSearchHit, 0, len(response.Results))
	for idx, item := range response.Results {
		content := strings.TrimSpace(item.Text)
		if content == "" {
			continue
		}
		id := strings.TrimSpace(item.ChunkID)
		if id == "" {
			id = fmt.Sprintf("ragbox-%d", idx)
		}
		title := strings.TrimSpace(item.Owner.Title)
		externalID := strings.TrimSpace(item.Owner.ExternalID)
		filePath := externalID
		if filePath == "" {
			filePath = title
		}
		hits = append(hits, ProSearchHit{
			ID:        id,
			Source:    "ragbox",
			Score:     item.Score,
			FilePath:  filePath,
			Title:     title,
			Content:   content,
			DedupeKey: dedupeKeyForRagBox(externalID, title, content),
		})
	}
	normalizeHitScores(hits)
	return hits, nil
}

func mergeHybridHits(bleveHits, ragboxHits []ProSearchHit, limit int) []ProSearchHit {
	merged := make(map[string]ProSearchHit)
	order := make([]string, 0, len(bleveHits)+len(ragboxHits))

	add := func(hit ProSearchHit) {
		key := strings.TrimSpace(hit.DedupeKey)
		if key == "" {
			key = dedupeKeyFromContent(hit.Content)
		}
		if existing, ok := merged[key]; ok {
			if hit.Score > existing.Score {
				existing.Score = hit.Score
			}
			if existing.Source != hit.Source {
				existing.Source = "hybrid"
			}
			if existing.FilePath == "" {
				existing.FilePath = hit.FilePath
			}
			if existing.Title == "" {
				existing.Title = hit.Title
			}
			if existing.Section == "" {
				existing.Section = hit.Section
			}
			if len(existing.Content) < len(hit.Content) {
				existing.Content = hit.Content
			}
			merged[key] = existing
			return
		}
		merged[key] = hit
		order = append(order, key)
	}

	for _, hit := range bleveHits {
		add(hit)
	}
	for _, hit := range ragboxHits {
		add(hit)
	}

	items := make([]ProSearchHit, 0, len(order))
	for _, key := range order {
		items = append(items, merged[key])
	}
	sortSearchHits(items)
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func sortSearchHits(hits []ProSearchHit) {
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].Score > hits[i].Score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
}

func normalizeHitScores(hits []ProSearchHit) {
	if len(hits) == 0 {
		return
	}
	var maxScore float32
	for _, hit := range hits {
		if hit.Score > maxScore {
			maxScore = hit.Score
		}
	}
	if maxScore <= 0 {
		for idx := range hits {
			hits[idx].Score = float32(len(hits) - idx)
		}
		return
	}
	for idx := range hits {
		hits[idx].Score = hits[idx].Score / maxScore
	}
}

func dedupeKeyForBleve(filePath, content string) string {
	filePath = strings.ToLower(strings.TrimSpace(filePath))
	if filePath != "" {
		return "path:" + filePath
	}
	return dedupeKeyFromContent(content)
}

func dedupeKeyForRagBox(externalID, title, content string) string {
	externalID = strings.ToLower(strings.TrimSpace(externalID))
	if externalID != "" {
		return "ext:" + externalID
	}
	title = strings.ToLower(strings.TrimSpace(title))
	if title != "" {
		return "title:" + title
	}
	return dedupeKeyFromContent(content)
}

func dedupeKeyFromContent(content string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(content)), " "))
	if len(normalized) > 240 {
		normalized = normalized[:240]
	}
	sum := sha256.Sum256([]byte(normalized))
	return "content:" + hex.EncodeToString(sum[:8])
}

func normalizeSearchLimit(limit int) int {
	switch {
	case limit <= 0:
		return 5
	case limit > 50:
		return 50
	default:
		return limit
	}
}

// AdminRuntime exposes admin listener coordinates for local Bleve proxy calls.
type AdminRuntime interface {
	AdminHost() string
	AdminPort() int
}

// RuntimeAdminProxy adapts shared MCPBox runtime edition metadata for local Bleve calls.
type RuntimeAdminProxy struct {
	Host string
	Port int
}

func (p RuntimeAdminProxy) AdminHost() string {
	return p.Host
}

func (p RuntimeAdminProxy) AdminPort() int {
	return p.Port
}

func resolveAdminRuntimeEndpoint(runtime AdminRuntime) (string, int) {
	adminHost, adminPort := adminRuntimeOverrideFromArgs()
	if adminHost == "" {
		adminHost = strings.TrimSpace(os.Getenv("MCPBOX_HOST"))
	}
	if adminHost == "" {
		adminHost = strings.TrimSpace(runtime.AdminHost())
	}
	if adminHost == "" || adminHost == "0.0.0.0" || adminHost == "::" {
		adminHost = "127.0.0.1"
	}
	if ip := net.ParseIP(adminHost); ip != nil && ip.IsUnspecified() {
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
		adminPort = runtime.AdminPort()
	}
	if adminPort == 0 {
		adminPort = 38181
	}
	return adminHost, adminPort
}

func adminRuntimeOverrideFromArgs() (string, int) {
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
