package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/cli"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
)

func isCLITool(name string) bool {
	return name == "cli_list" || name == "cli_describe" || name == "cli_execute"
}
func projectCLITools() []aggregateTool {
	tools := make([]aggregateTool, 0, 3)
	for _, name := range []string{"cli_list", "cli_describe", "cli_execute"} {
		props := map[string]any{}
		required := []string{}
		description := "List CLI IDs and descriptions available to this project. Use cli_describe for operations."
		if name != "cli_list" {
			props["cli"] = map[string]any{"type": "string", "description": "CLI ID returned by cli_list."}
			required = append(required, "cli")
			description = "Describe one connected CLI and its exact allowed operations and argument sets."
		}
		if name == "cli_execute" {
			props["working_directory"] = map[string]any{"type": "string", "description": "Optional absolute working folder on the MCPBox host. Use cli_describe directory_policy; allowed folders include descendants. This does not change the operation/argument allowlist."}
			props["command"] = map[string]any{"type": "string", "description": "Exact operation from cli_describe (including multiword operations)."}
			props["args"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Omit or use an exact approved argument set."}
			required = append(required, "command")
			description = "Execute an approved project CLI operation. Returns stdout, stderr and exit_code."
		}
		schema, _ := json.Marshal(map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false})
		tools = append(tools, aggregateTool{Alias: name, Origin: orchestrator.InspectionTool{Name: name, Description: description, InputSchema: schema}})
	}
	return tools
}
func (s *Server) executeCLI(ctx context.Context, projectID uint, req cli.Request, actor string) (*cli.Result, error) {
	result, err := cli.Execute(ctx, s.store, projectID, req)
	detail := map[string]any{"cli": req.CLI, "command": req.Command}
	// Do not record stdout or argument values: these may contain secrets.
	if err != nil {
		detail["error"] = err.Error()
	}
	if result != nil {
		detail["exit_code"] = result.ExitCode
		detail["timed_out"] = result.TimedOut
		detail["truncated"] = result.Truncated
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	s.logAudit(auditCtx, &projectID, nil, "cli_execute", actor, mustJSON(detail))
	return result, err
}
func (s *Server) callCLITool(ctx context.Context, project models.Project, name string, input map[string]any, actor string) map[string]any {
	var value any
	var err error
	var req cli.Request
	arguments, marshalErr := json.Marshal(input)
	if marshalErr != nil {
		return map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": marshalErr.Error()}}}
	}
	if input == nil {
		arguments = []byte(`{}`)
	}
	var fields map[string]json.RawMessage
	err = json.Unmarshal(arguments, &fields)
	if err == nil {
		for key := range fields {
			if (key == "cli" && name != "cli_list") || ((key == "command" || key == "args" || key == "working_directory") && name == "cli_execute") {
				continue
			}
			err = fmt.Errorf("unknown argument: %s", key)
			break
		}
	}
	if err == nil {
		err = json.Unmarshal(arguments, &req)
	}
	if err == nil {
		switch name {
		case "cli_list":
			var items []models.CLI
			items, err = s.store.ListProjectCLIs(ctx, project.ID, true)
			summaries := make([]map[string]any, 0, len(items))
			for _, item := range items {
				summaries = append(summaries, map[string]any{"cli": item.Slug, "name": item.Name, "description": item.Description})
			}
			value = map[string]any{"items": summaries}
		case "cli_describe":
			var item *models.CLI
			item, err = cli.Describe(ctx, s.store, project.ID, req.CLI)
			if err == nil {
				current, loadErr := s.store.GetProject(ctx, project.ID)
				if loadErr != nil {
					err = loadErr
					break
				}
				value = map[string]any{"directory_policy": cli.ProjectDirectoryPolicy(current), "cli": item.Slug, "description": item.Description, "operations": item.Operations, "timeout_seconds": item.TimeoutSeconds, "max_output_bytes": item.MaxOutputBytes}
			}
		case "cli_execute":
			value, err = s.executeCLI(ctx, project.ID, req, actor)
		}
	}
	if err != nil {
		return map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": err.Error()}}}
	}
	body, _ := json.Marshal(value)
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": string(body)}}, "structuredContent": value}
	if execution, ok := value.(*cli.Result); ok && (execution.ExitCode != 0 || execution.TimedOut) {
		result["isError"] = true
	}
	return result
}

func projectHasActiveCLI(project models.Project) bool {
	for _, item := range project.CLIs {
		if item.IsEnabled {
			return true
		}
	}
	return false
}
