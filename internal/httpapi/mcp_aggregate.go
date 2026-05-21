package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/rag"
)

type projectRequestEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type projectResponseEnvelope struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      json.RawMessage  `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *projectRPCError `json:"error,omitempty"`
}

type projectRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type aggregateTool struct {
	Server models.MCPServer
	Alias  string
	Origin orchestrator.InspectionTool
}

type aggregatePrompt struct {
	Server models.MCPServer
	Alias  string
	Origin orchestrator.InspectionPrompt
}

type aggregateResource struct {
	Server models.MCPServer
	Alias  string
	Origin orchestrator.InspectionItem
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type promptGetParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

type listToolsResult struct {
	Tools      []orchestrator.InspectionTool `json:"tools"`
	NextCursor string                        `json:"nextCursor,omitempty"`
}

type aggregatedToolResult struct {
	Name         string          `json:"name"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
}

type projectKnowledgeSearchArguments struct {
	Query       string   `json:"query"`
	Limit       int      `json:"limit,omitempty"`
	Collections []string `json:"collections,omitempty"`
}

type projectKnowledgeSearchResultItem struct {
	CollectionID string `json:"collection_id"`
	FilePath     string `json:"file_path"`
	Content      string `json:"content"`
}

const projectKnowledgeSearchToolName = "search_project_knowledge"

type listResourcesResult struct {
	Resources  []orchestrator.InspectionItem `json:"resources"`
	NextCursor string                        `json:"nextCursor,omitempty"`
}

type listPromptsResult struct {
	Prompts    []orchestrator.InspectionPrompt `json:"prompts"`
	NextCursor string                          `json:"nextCursor,omitempty"`
}

var aggregateAliasPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (s *Server) projectConnectServers(project models.Project) []models.MCPServer {
	servers := make([]models.MCPServer, 0, len(project.Servers))
	for _, server := range project.Servers {
		if !server.IsEnabled {
			continue
		}
		servers = append(servers, server)
	}
	return servers
}

func (s *Server) serveProjectSSE(w http.ResponseWriter, r *http.Request, projectToken string, projectID uint) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sessionID, err := newConnectSessionID()
	if err != nil {
		http.Error(w, "failed to create connect session", http.StatusInternalServerError)
		return
	}

	stream := make(chan []byte, 64)
	s.registerConnectSession(connectSession{
		ID:           sessionID,
		ProjectToken: projectToken,
		ProjectID:    projectID,
		CreatedAt:    time.Now().UTC(),
		Stream:       stream,
	})
	defer s.unregisterConnectSession(sessionID)

	endpointURL := s.connectMessageURL(r, projectToken, sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-stream:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", line)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) loadConnectSession(sessionID string) (connectSession, bool) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}

func (s *Server) publishConnectSession(sessionID string, payload []byte) error {
	session, ok := s.loadConnectSession(sessionID)
	if !ok || session.Stream == nil {
		return errors.New("invalid or expired connect session")
	}

	select {
	case session.Stream <- payload:
		return nil
	default:
		return errors.New("connect session stream is full")
	}
}

func (s *Server) dispatchProjectJSONRPC(
	ctx context.Context,
	project models.Project,
	servers []models.MCPServer,
	payload []byte,
) ([]byte, bool, error) {
	var request projectRequestEnvelope
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, false, errors.New("body must be valid JSON-RPC payload")
	}

	if strings.TrimSpace(request.Method) == "" {
		return nil, false, errors.New("json-rpc method is required")
	}

	switch request.Method {
	case "initialize":
		result := map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
				"prompts":   map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mcpbox-project",
				"title":   project.Name,
				"version": "1.0.0",
			},
			"instructions": "MCPBox project aggregation endpoint.",
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "notifications/initialized":
		return nil, false, nil
	case "ping":
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  map[string]any{},
		}), true, nil
	case "tools/list":
		tools, err := s.aggregateTools(ctx, servers)
		if err != nil {
			return nil, false, err
		}
		if len(project.RAGCollections) > 0 {
			tools = append(tools, s.projectKnowledgeAggregateTool())
		}
		result := struct {
			Tools []aggregatedToolResult `json:"tools"`
		}{
			Tools: make([]aggregatedToolResult, 0, len(tools)),
		}
		for _, tool := range tools {
			item := tool.Origin
			result.Tools = append(result.Tools, aggregatedToolResult{
				Name:         tool.Alias,
				Title:        item.Title,
				Description:  item.Description,
				InputSchema:  ensureJSONSchema(item.InputSchema),
				OutputSchema: normalizeOptionalJSON(item.OutputSchema),
			})
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, false, errors.New("invalid tools/call params")
		}
		if params.Name == projectKnowledgeSearchToolName {
			result, err := s.callProjectKnowledgeTool(ctx, project, params.Arguments)
			if err != nil {
				return nil, false, err
			}
			return mustMarshal(projectResponseEnvelope{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result:  result,
			}), true, nil
		}
		tool, err := s.resolveAggregateTool(ctx, servers, params.Name)
		if err != nil {
			return nil, false, err
		}
		params.Name = tool.Origin.Name
		result, err := s.callServerMethod(ctx, tool.Server, "tools/call", params)
		if err != nil {
			return nil, false, err
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "resources/list":
		resources, err := s.aggregateResources(ctx, servers)
		if err != nil {
			return nil, false, err
		}
		result := listResourcesResult{
			Resources: make([]orchestrator.InspectionItem, 0, len(resources)),
		}
		for _, resource := range resources {
			item := resource.Origin
			item.URI = resource.Alias
			result.Resources = append(result.Resources, item)
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "resources/read":
		var params resourceReadParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, false, errors.New("invalid resources/read params")
		}
		resource, err := s.resolveAggregateResource(ctx, servers, params.URI)
		if err != nil {
			return nil, false, err
		}
		params.URI = resource.Origin.URI
		result, err := s.callServerMethod(ctx, resource.Server, "resources/read", params)
		if err != nil {
			return nil, false, err
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "prompts/list":
		prompts, err := s.aggregatePrompts(ctx, servers)
		if err != nil {
			return nil, false, err
		}
		result := listPromptsResult{
			Prompts: make([]orchestrator.InspectionPrompt, 0, len(prompts)),
		}
		for _, prompt := range prompts {
			item := prompt.Origin
			item.Name = prompt.Alias
			result.Prompts = append(result.Prompts, item)
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	case "prompts/get":
		var params promptGetParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, false, errors.New("invalid prompts/get params")
		}
		prompt, err := s.resolveAggregatePrompt(ctx, servers, params.Name)
		if err != nil {
			return nil, false, err
		}
		params.Name = prompt.Origin.Name
		result, err := s.callServerMethod(ctx, prompt.Server, "prompts/get", params)
		if err != nil {
			return nil, false, err
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		}), true, nil
	default:
		if len(bytes.TrimSpace(request.ID)) == 0 {
			return nil, false, nil
		}
		return mustMarshal(projectResponseEnvelope{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &projectRPCError{
				Code:    -32601,
				Message: "method not supported by MCPBox project aggregation",
			},
		}), true, nil
	}
}

func (s *Server) callServerMethod(ctx context.Context, server models.MCPServer, method string, params any) (json.RawMessage, error) {
	server, err := s.ensureServerInitialized(ctx, server)
	if err != nil {
		return nil, err
	}

	requestID := fmt.Sprintf("mcpbox-project-%d", time.Now().UnixNano())
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	payload := mustMarshal(request)

	switch server.Transport {
	case models.ServerTransportSTDIO:
		runner := s.registry.Runner(server.ID)
		if runner == nil {
			return nil, fmt.Errorf("server %d runner is unavailable", server.ID)
		}
		response, err := runner.SendAndWait(ctx, payload)
		if err != nil {
			return nil, err
		}
		var envelope struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(response, &envelope); err != nil {
			return nil, err
		}
		if envelope.Error != nil {
			return nil, errors.New(envelope.Error.Message)
		}
		return envelope.Result, nil
	case models.ServerTransportHTTPStream:
		return s.callHTTPServerMethod(ctx, server, payload)
	default:
		return nil, errors.New("unsupported server transport")
	}
}

func (s *Server) ensureServerInitialized(ctx context.Context, server models.MCPServer) (models.MCPServer, error) {
	s.oauthMu.RLock()
	initialized := s.initializedServers[server.ID]
	s.oauthMu.RUnlock()

	if server.Transport == models.ServerTransportHTTPStream {
		var err error
		server, err = s.ensureOAuthAccessToken(ctx, server)
		if err != nil {
			return server, err
		}
	}

	if initialized {
		if server.Transport == models.ServerTransportSTDIO {
			if err := s.registry.StartServer(ctx, server); err != nil {
				return server, err
			}
		}
		return server, nil
	}

	initParams := map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "MCPBox",
			"title":   "MCPBox",
			"version": "1.0.0",
		},
	}

	switch server.Transport {
	case models.ServerTransportSTDIO:
		if err := s.registry.StartServer(ctx, server); err != nil {
			return server, err
		}
		runner := s.registry.Runner(server.ID)
		if runner == nil {
			return server, fmt.Errorf("server %d runner is unavailable", server.ID)
		}
		if err := s.callRunnerMethod(ctx, runner, "initialize", initParams); err != nil {
			return server, err
		}
		_ = runner.Send(ctx, mustMarshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}))
	case models.ServerTransportHTTPStream:
		request := map[string]any{
			"jsonrpc": "2.0",
			"id":      fmt.Sprintf("mcpbox-init-%d", server.ID),
			"method":  "initialize",
			"params":  initParams,
		}
		if _, err := s.callHTTPServerMethod(ctx, server, mustMarshal(request)); err != nil {
			return server, err
		}
		_, _ = s.callHTTPServerMethod(ctx, server, mustMarshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}))
	default:
		return server, errors.New("unsupported server transport")
	}

	s.oauthMu.Lock()
	s.initializedServers[server.ID] = true
	s.oauthMu.Unlock()

	return server, nil
}

func (s *Server) callRunnerMethod(ctx context.Context, runner *orchestrator.ServerRunner, method string, params any) error {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("mcpbox-init-%d", time.Now().UnixNano()),
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}
	response, err := runner.SendAndWait(ctx, mustMarshal(request))
	if err != nil {
		return err
	}
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return errors.New(envelope.Error.Message)
	}
	return nil
}

func (s *Server) callHTTPServerMethod(ctx context.Context, server models.MCPServer, payload []byte) (json.RawMessage, error) {
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	applyConfiguredHeaders(upstreamReq.Header, server)

	response, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(body) == 0 {
		return nil, nil
	}

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != nil {
		return nil, errors.New(envelope.Error.Message)
	}
	return envelope.Result, nil
}

func (s *Server) aggregateTools(ctx context.Context, servers []models.MCPServer) ([]aggregateTool, error) {
	tools := make([]aggregateTool, 0)
	for _, server := range servers {
		server, err := s.ensureServerInitialized(ctx, server)
		if err != nil {
			return nil, err
		}
		var result listToolsResult
		if err := s.fetchServerList(ctx, server, "tools/list", &result); err != nil {
			return nil, err
		}
		for _, item := range result.Tools {
			tools = append(tools, aggregateTool{Server: server, Origin: item})
		}
	}
	assignToolAliases(tools)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Alias < tools[j].Alias })
	return tools, nil
}

func (s *Server) projectKnowledgeAggregateTool() aggregateTool {
	return aggregateTool{
		Origin: orchestrator.InspectionTool{
			Name:        projectKnowledgeSearchToolName,
			Title:       "Project Knowledge Search",
			Description: "Searches across all knowledge base collections connected to the current project.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"query":{"type":"string","description":"What to search for in the connected project knowledge bases."},
					"limit":{"type":"integer","minimum":1,"maximum":20,"description":"Maximum number of chunks to return."},
					"collections":{"type":"array","items":{"type":"string"},"description":"Optional subset of connected collection ids to search in."}
				},
				"required":["query"]
			}`),
		},
		Alias: projectKnowledgeSearchToolName,
	}
}

func (s *Server) callProjectKnowledgeTool(ctx context.Context, project models.Project, arguments map[string]any) (map[string]any, error) {
	rawArgs, err := json.Marshal(arguments)
	if err != nil {
		return nil, err
	}

	var args projectKnowledgeSearchArguments
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, errors.New("invalid search_project_knowledge arguments")
	}

	query := strings.TrimSpace(args.Query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	allowed := make(map[string]models.RAGCollection, len(project.RAGCollections))
	for _, collection := range project.RAGCollections {
		allowed[collection.CollectionID] = collection
	}
	if len(allowed) == 0 {
		return nil, errors.New("project has no connected knowledge bases")
	}

	targets := make([]models.RAGCollection, 0, len(allowed))
	if len(args.Collections) == 0 {
		for _, collection := range project.RAGCollections {
			targets = append(targets, collection)
		}
	} else {
		for _, collectionID := range args.Collections {
			collectionID = strings.TrimSpace(collectionID)
			collection, ok := allowed[collectionID]
			if !ok {
				return nil, fmt.Errorf("collection %q is not connected to the project", collectionID)
			}
			targets = append(targets, collection)
		}
	}

	items := make([]projectKnowledgeSearchResultItem, 0)
	for _, collection := range targets {
		index, err := rag.NewCollection(collection.CollectionID, collection.Name, collection.IndexPath)
		if err != nil {
			return nil, err
		}

		chunks, searchErr := index.Search(query, limit)
		closeErr := index.Close()
		if searchErr != nil {
			return nil, searchErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		for _, chunk := range chunks {
			items = append(items, projectKnowledgeSearchResultItem{
				CollectionID: collection.CollectionID,
				FilePath:     chunk.FilePath,
				Content:      chunk.Content,
			})
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}

	textParts := make([]string, 0, len(items))
	for _, item := range items {
		textParts = append(textParts, fmt.Sprintf("[%s] %s\n%s", item.CollectionID, item.FilePath, item.Content))
	}
	text := "No relevant knowledge chunks found."
	if len(textParts) > 0 {
		text = strings.Join(textParts, "\n\n")
	}

	s.logAudit(ctx, &project.ID, nil, "tool_call_project_knowledge_search", "mcp-client", mustJSON(map[string]any{
		"tool":        projectKnowledgeSearchToolName,
		"query":       truncateDetail(query),
		"collections": collectionIDs(targets),
		"results":     len(items),
	}))

	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"structuredContent": map[string]any{
			"query": query,
			"items": items,
		},
	}, nil
}

func collectionIDs(collections []models.RAGCollection) []string {
	result := make([]string, 0, len(collections))
	for _, collection := range collections {
		result = append(result, collection.CollectionID)
	}
	return result
}

func (s *Server) aggregateResources(ctx context.Context, servers []models.MCPServer) ([]aggregateResource, error) {
	resources := make([]aggregateResource, 0)
	for _, server := range servers {
		server, err := s.ensureServerInitialized(ctx, server)
		if err != nil {
			return nil, err
		}
		var result listResourcesResult
		if err := s.fetchServerList(ctx, server, "resources/list", &result); err != nil {
			return nil, err
		}
		for _, item := range result.Resources {
			resources = append(resources, aggregateResource{Server: server, Origin: item})
		}
	}
	assignResourceAliases(resources)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Alias < resources[j].Alias })
	return resources, nil
}

func (s *Server) aggregatePrompts(ctx context.Context, servers []models.MCPServer) ([]aggregatePrompt, error) {
	prompts := make([]aggregatePrompt, 0)
	for _, server := range servers {
		server, err := s.ensureServerInitialized(ctx, server)
		if err != nil {
			return nil, err
		}
		var result listPromptsResult
		if err := s.fetchServerList(ctx, server, "prompts/list", &result); err != nil {
			return nil, err
		}
		for _, item := range result.Prompts {
			prompts = append(prompts, aggregatePrompt{Server: server, Origin: item})
		}
	}
	assignPromptAliases(prompts)
	sort.Slice(prompts, func(i, j int) bool { return prompts[i].Alias < prompts[j].Alias })
	return prompts, nil
}

func (s *Server) fetchServerList(ctx context.Context, server models.MCPServer, method string, result any) error {
	cursor := ""
	switch method {
	case "tools/list":
		all := make([]orchestrator.InspectionTool, 0)
		for {
			raw, err := s.callServerMethod(ctx, server, method, cursorParams(cursor))
			if err != nil {
				return err
			}
			page, err := decodeListToolsResult(raw)
			if err != nil {
				return err
			}
			all = append(all, page.Tools...)
			if strings.TrimSpace(page.NextCursor) == "" {
				break
			}
			cursor = page.NextCursor
		}
		typed := result.(*listToolsResult)
		typed.Tools = all
		return nil
	case "resources/list":
		all := make([]orchestrator.InspectionItem, 0)
		for {
			raw, err := s.callServerMethod(ctx, server, method, cursorParams(cursor))
			if err != nil {
				return err
			}
			var page listResourcesResult
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &page); err != nil {
					return err
				}
			}
			all = append(all, page.Resources...)
			if strings.TrimSpace(page.NextCursor) == "" {
				break
			}
			cursor = page.NextCursor
		}
		typed := result.(*listResourcesResult)
		typed.Resources = all
		return nil
	case "prompts/list":
		all := make([]orchestrator.InspectionPrompt, 0)
		for {
			raw, err := s.callServerMethod(ctx, server, method, cursorParams(cursor))
			if err != nil {
				return err
			}
			var page listPromptsResult
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &page); err != nil {
					return err
				}
			}
			all = append(all, page.Prompts...)
			if strings.TrimSpace(page.NextCursor) == "" {
				break
			}
			cursor = page.NextCursor
		}
		typed := result.(*listPromptsResult)
		typed.Prompts = all
		return nil
	default:
		return errors.New("unsupported list method")
	}
}

func (s *Server) resolveAggregateTool(ctx context.Context, servers []models.MCPServer, alias string) (*aggregateTool, error) {
	tools, err := s.aggregateTools(ctx, servers)
	if err != nil {
		return nil, err
	}
	for _, tool := range tools {
		if tool.Alias == alias {
			return &tool, nil
		}
	}
	return nil, fmt.Errorf("tool %q was not found", alias)
}

func (s *Server) resolveAggregateResource(ctx context.Context, servers []models.MCPServer, alias string) (*aggregateResource, error) {
	resources, err := s.aggregateResources(ctx, servers)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		if resource.Alias == alias {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("resource %q was not found", alias)
}

func (s *Server) resolveAggregatePrompt(ctx context.Context, servers []models.MCPServer, alias string) (*aggregatePrompt, error) {
	prompts, err := s.aggregatePrompts(ctx, servers)
	if err != nil {
		return nil, err
	}
	for _, prompt := range prompts {
		if prompt.Alias == alias {
			return &prompt, nil
		}
	}
	return nil, fmt.Errorf("prompt %q was not found", alias)
}

func assignToolAliases(tools []aggregateTool) {
	counts := make(map[string]int)
	for _, tool := range tools {
		counts[tool.Origin.Name]++
	}
	for index := range tools {
		if counts[tools[index].Origin.Name] == 1 {
			tools[index].Alias = tools[index].Origin.Name
			continue
		}
		tools[index].Alias = fmt.Sprintf("%s/%s", serverAliasPrefix(tools[index].Server), tools[index].Origin.Name)
	}
}

func assignPromptAliases(prompts []aggregatePrompt) {
	counts := make(map[string]int)
	for _, prompt := range prompts {
		counts[prompt.Origin.Name]++
	}
	for index := range prompts {
		if counts[prompts[index].Origin.Name] == 1 {
			prompts[index].Alias = prompts[index].Origin.Name
			continue
		}
		prompts[index].Alias = fmt.Sprintf("%s/%s", serverAliasPrefix(prompts[index].Server), prompts[index].Origin.Name)
	}
}

func assignResourceAliases(resources []aggregateResource) {
	counts := make(map[string]int)
	for _, resource := range resources {
		counts[resource.Origin.URI]++
	}
	for index := range resources {
		if counts[resources[index].Origin.URI] == 1 {
			resources[index].Alias = resources[index].Origin.URI
			continue
		}
		resources[index].Alias = fmt.Sprintf("mcpbox://%s/%s", serverAliasPrefix(resources[index].Server), url.PathEscape(resources[index].Origin.URI))
	}
}

func serverAliasPrefix(server models.MCPServer) string {
	base := strings.ToLower(strings.TrimSpace(server.Name))
	base = aggregateAliasPattern.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = fmt.Sprintf("server-%d", server.ID)
	}
	return base
}

func mustMarshal(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return payload
}

func decodeListToolsResult(raw json.RawMessage) (listToolsResult, error) {
	var envelope map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return listToolsResult{}, err
		}
	}

	toolsValue, _ := envelope["tools"].([]any)
	tools := make([]orchestrator.InspectionTool, 0, len(toolsValue))
	for _, rawTool := range toolsValue {
		toolMap, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		tools = append(tools, orchestrator.InspectionTool{
			Name:         stringValue(toolMap["name"]),
			Title:        stringValue(toolMap["title"]),
			Description:  stringValue(toolMap["description"]),
			InputSchema:  firstJSONValue(toolMap, "inputSchema", "input_schema"),
			OutputSchema: firstJSONValue(toolMap, "outputSchema", "output_schema"),
		})
	}

	nextCursor := stringValue(envelope["nextCursor"])
	if nextCursor == "" {
		nextCursor = stringValue(envelope["next_cursor"])
	}

	return listToolsResult{
		Tools:      tools,
		NextCursor: nextCursor,
	}, nil
}

func firstJSONValue(source map[string]any, keys ...string) json.RawMessage {
	for _, key := range keys {
		value, ok := source[key]
		if !ok {
			continue
		}
		payload, err := json.Marshal(value)
		if err == nil && len(bytes.TrimSpace(payload)) > 0 && string(bytes.TrimSpace(payload)) != "null" {
			return payload
		}
	}
	return nil
}

func ensureJSONSchema(raw json.RawMessage) json.RawMessage {
	normalized := normalizeOptionalJSON(raw)
	if len(normalized) == 0 {
		return json.RawMessage(`{}`)
	}
	return normalized
}

func normalizeOptionalJSON(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	return json.RawMessage(trimmed)
}

func stringValue(value any) string {
	casted, _ := value.(string)
	return strings.TrimSpace(casted)
}

func cursorParams(cursor string) map[string]any {
	if strings.TrimSpace(cursor) == "" {
		return map[string]any{}
	}
	return map[string]any{"cursor": cursor}
}
