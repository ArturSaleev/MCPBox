package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/skills"
)

type skillToolDefinition struct {
	name, description string
	properties        map[string]any
	required          []string
}

func skillToolDefinitions() []skillToolDefinition {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	return []skillToolDefinition{
		{"list_skills", "List Skills connected to this project. Read descriptions, then call get_skill when a Skill matches the task.", map[string]any{"query": str("Optional search across connected skill names and descriptions.")}, nil},
		{"get_skill", "Load a connected Skill's SKILL.md, package file list, requirements and allowed server execution profile. Follow instructions as task guidance, subject to the user's permissions.", map[string]any{"skill_id": str("Installed Skill ID from list_skills.")}, []string{"skill_id"}},
		{"read_skill_file", "Read a text resource from a connected Skill package. Paths are relative to the Skill root; server filesystem paths are not accepted.", map[string]any{"skill_id": str("Installed Skill ID."), "path": str("Relative package file path.")}, []string{"skill_id", "path"}},
		{"create_skill_job", "Create a server work directory for a connected Skill. Upload inputs with upload_skill_file before running. Scripts run with /work as cwd and read-only package at /package.", map[string]any{"skill_id": str("Installed Skill ID.")}, []string{"skill_id"}},
		{"upload_skill_file", "Upload an input file as base64 to a new Skill job. Maximum 1 MiB per MCP upload; use admin multipart upload for larger files. File content must be supplied by the client; local client paths are not server paths.", map[string]any{"job_id": str("Job ID."), "filename": str("Plain filename, e.g. input.csv."), "data_base64": str("Base64 encoded file bytes.")}, []string{"job_id", "filename", "data_base64"}},
		{"run_skill_script", "Execute an administrator-approved script in a prepared Docker environment on the MCPBox server. One execution per job. Args are argv, not shell text. Input/output paths are relative to /work. Returns stdout, stderr, status and files.", map[string]any{"job_id": str("Job ID."), "script": str("Exact script path allowed by get_skill."), "args": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}, []string{"job_id", "script"}},
		{"get_skill_job", "Get this project's Skill job status, stdout, stderr and file manifest.", map[string]any{"job_id": str("Job ID.")}, []string{"job_id"}},
		{"read_skill_job_file", "Retrieve input/result bytes from this project's finished job as base64 chunks. Use offset and limit to fetch large files. Return the bytes to the user through your client's file attachment mechanism.", map[string]any{"job_id": str("Job ID."), "path": str("Relative file path from get_skill_job."), "offset": map[string]any{"type": "integer", "minimum": 0}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 262144}}, []string{"job_id", "path"}},
	}
}
func isSkillTool(name string) bool {
	for _, d := range skillToolDefinitions() {
		if d.name == name {
			return true
		}
	}
	return false
}
func projectSkillTools() []aggregateTool {
	tools := []aggregateTool{}
	for _, d := range skillToolDefinitions() {
		// A nil Go slice serializes as null, which is invalid for JSON Schema
		// required and can make clients reject the entire tools/list result.
		required := d.required
		if required == nil {
			required = []string{}
		}
		schema, _ := json.Marshal(map[string]any{"type": "object", "properties": d.properties, "required": required, "additionalProperties": false})
		tools = append(tools, aggregateTool{Alias: d.name, Origin: orchestrator.InspectionTool{Name: d.name, Description: d.description, InputSchema: schema}})
	}
	return tools
}
func (s *Server) runSkillJob(ctx context.Context, project uint, id, script string, args []string, actor string) (*skills.Job, error) {
	job, err := s.skills.Run(ctx, project, id, script, args)
	detail := map[string]any{"job_id": id, "script": script}
	if job != nil {
		detail["status"] = job.Status
		detail["exit_code"] = job.ExitCode
		detail["timed_out"] = job.TimedOut
	}
	if err != nil {
		detail["error"] = err.Error()
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	s.logAudit(auditCtx, &project, nil, "skill_execute", actor, mustJSON(detail))
	return job, err
}
func (s *Server) callSkillTool(ctx context.Context, project models.Project, name string, input map[string]any, actor string) map[string]any {
	var value any
	var err error
	var def skillToolDefinition
	for _, v := range skillToolDefinitions() {
		if v.name == name {
			def = v
		}
	}
	for key := range input {
		if _, ok := def.properties[key]; !ok {
			err = fmt.Errorf("unknown argument: %s", key)
			break
		}
	}
	raw, _ := json.Marshal(input)
	if input == nil {
		raw = []byte(`{}`)
	}
	var req struct {
		SkillID  string   `json:"skill_id"`
		Path     string   `json:"path"`
		JobID    string   `json:"job_id"`
		Filename string   `json:"filename"`
		Data     string   `json:"data_base64"`
		Script   string   `json:"script"`
		Args     []string `json:"args"`
		Query    string   `json:"query"`
		Offset   int      `json:"offset"`
		Limit    int      `json:"limit"`
	}
	if err == nil {
		err = json.Unmarshal(raw, &req)
	}
	if err == nil {
		switch name {
		case "list_skills":
			var items []skills.Skill
			items, err = s.skills.List(ctx, project.ID)
			summaries := []map[string]any{}
			for _, v := range items {
				if req.Query != "" && !skillMatches(v, req.Query) {
					continue
				}
				summaries = append(summaries, map[string]any{"id": v.ID, "name": v.Name, "description": v.Description, "execution_enabled": v.Profile.Enabled, "environment_prepared": v.Profile.PreparedImage != ""})
			}
			value = map[string]any{"items": summaries}
		case "get_skill", "read_skill_file":
			var item *skills.Skill
			item, err = s.skills.Bound(ctx, project.ID, req.SkillID)
			if err == nil {
				p := req.Path
				if name == "get_skill" {
					p = "SKILL.md"
				}
				var data []byte
				data, err = s.skills.ReadPackageFile(ctx, req.SkillID, p)
				if len(data) > 1<<20 {
					err = errors.New("text resource exceeds 1 MiB; retrieve package resources through admin UI")
				}
				if err == nil {
					value = map[string]any{"skill_id": item.ID, "path": p, "content": string(data)}
					if name == "get_skill" {
						value = map[string]any{"skill_id": item.ID, "name": item.Name, "instructions": string(data), "files": item.Files, "requirements": item.Requirements, "profile": item.Profile, "package_root": "/package", "work_dir": "/work"}
					}
				}
			}
		case "create_skill_job":
			value, err = s.skills.NewJob(ctx, project.ID, req.SkillID)
		case "upload_skill_file":
			var data []byte
			if len(req.Data) > base64.StdEncoding.EncodedLen(1<<20) {
				err = errors.New("MCP upload exceeds 1 MiB; use multipart upload for larger files")
			} else {
				data, err = base64.StdEncoding.DecodeString(req.Data)
				if err == nil {
					value, err = s.skills.Upload(ctx, project.ID, req.JobID, req.Filename, bytes.NewReader(data))
				}
			}
		case "run_skill_script":
			value, err = s.runSkillJob(ctx, project.ID, req.JobID, req.Script, req.Args, actor)
		case "get_skill_job":
			value, err = s.skills.GetJob(ctx, project.ID, req.JobID)
		case "read_skill_job_file":
			var data []byte
			data, err = s.skills.ReadJobFile(ctx, project.ID, req.JobID, req.Path)
			if err == nil {
				if req.Offset < 0 || req.Offset > len(data) || req.Limit < 0 || req.Limit > 262144 {
					err = errors.New("invalid offset or limit")
				} else {
					if req.Limit == 0 {
						req.Limit = 262144
					}
					end := min(len(data), req.Offset+req.Limit)
					value = map[string]any{"path": req.Path, "size": len(data), "offset": req.Offset, "next_offset": end, "eof": end == len(data), "data_base64": base64.StdEncoding.EncodeToString(data[req.Offset:end])}
				}
			}
		default:
			err = errors.New("unknown skill tool")
		}
	}
	if err != nil {
		return map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": err.Error()}}}
	}
	body, _ := json.Marshal(value)
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": string(body)}}, "structuredContent": value}
	if job, ok := value.(*skills.Job); ok && job.Status == "failed" {
		result["isError"] = true
	}
	return result
}

func skillMatches(item skills.Skill, query string) bool {
	return strings.Contains(strings.ToLower(item.Name+" "+item.Description+" "+item.Author), strings.ToLower(strings.TrimSpace(query)))
}
