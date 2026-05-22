# MCPBox Pro Roadmap

This document describes the planned `Pro` direction for MCPBox. It is a product roadmap, not a statement of current implementation.

Status:
- `Free / 1.2.0` is implemented today
- `Pro` is not implemented yet
- this file documents future capabilities only

## Free Now

The current free edition focuses on an embedded local MCP control plane and a built-in Knowledge Base / RAG search layer.

Current free capabilities include:
- local Knowledge Base collections
- local indexing with Bleve full-text search
- project-level `search_project_knowledge`
- support for code, text, `CSV`, `XLSX`, `DOCX`, `PPTX`, and text-based `PDF`
- no external vector database
- no embedding index generation
- no OCR pipeline

## Planned Pro Capabilities

The following areas are planned for the future `Pro` edition. None of them are part of `1.2.0`.

### 1. Advanced RAG And External Data Sources

Goal:
- extend the current local search layer into a richer enterprise knowledge system

Planned direction:
- REST API for external context enrichment such as `POST /api/v1/knowledge-base/enrich`
- one-click connectors for systems such as Notion, Confluence, Jira, Google Drive, and GitHub
- hybrid search that combines Bleve lexical retrieval with semantic or vector retrieval

Difference from Free:
- `Free` uses local Bleve full-text search only
- `Pro` is expected to add external sync and semantic retrieval layers

### 2. Auth, SSO, RBAC, And Agent Tokens

Goal:
- support multi-user and enterprise access control scenarios

Planned direction:
- SSO and OAuth2 login through providers such as Google Workspace, GitHub, GitLab, LDAP, or Active Directory
- RBAC for admins, developers, analysts, and other internal roles
- scoped API tokens for agents and automations with TTL and limited permissions

Difference from Free:
- `Free` has no edition-level auth or role system
- `Pro` is expected to add identity, access policies, and delegated tokens

### 3. Advanced OCR RAG

Goal:
- make scanned or image-based business documents searchable

Planned direction:
- OCR ingestion for scanned PDFs, invoices, contracts, acts, and image files
- local processing through engines such as Tesseract or local multimodal models through Ollama
- searchable extraction without requiring third-party cloud OCR

Difference from Free:
- `Free` supports text-based PDFs only
- `Pro` is expected to add OCR for scans and images

### 4. Team Control Plane

Goal:
- make MCPBox usable as a centrally managed team gateway

Planned direction:
- shared server-side MCPBox deployments for teams
- centrally managed configs and secrets for internal services
- grouping of MCP servers by environment such as Development, Staging, and Production

Difference from Free:
- `Free` is focused on local single-user and small-team embedded workflows
- `Pro` is expected to add centrally managed team operations

### 5. Security And Guardrails

Goal:
- add stronger operational safety before model actions reach sensitive systems

Planned direction:
- blocking or confirmation flows for destructive actions such as dangerous SQL operations
- PII masking before logs or tool context are sent into model-visible payloads
- stronger audit and policy enforcement around model actions

Difference from Free:
- `Free` includes audit visibility but not advanced policy enforcement
- `Pro` is expected to add active safety controls and data redaction

### 6. Code Awareness And AST Parsing

Goal:
- improve code retrieval quality beyond plain text chunking

Planned direction:
- AST-aware parsing through tools such as tree-sitter
- code chunks aligned to classes, methods, functions, and syntactic boundaries
- better retrieval precision for large or complex codebases

Difference from Free:
- `Free` uses practical line and text chunking with Bleve
- `Pro` is expected to add syntax-aware code understanding

## Edition Boundary

`Free / 1.2.0`:
- embedded local control plane
- local full-text Knowledge Base search
- no semantic vector pipeline
- no OCR
- no SSO or RBAC

`Pro / future`:
- enterprise knowledge ingestion and connectors
- hybrid lexical plus semantic retrieval
- OCR for scanned documents and images
- multi-user auth and role controls
- centralized team deployment and security guardrails
- AST-aware code intelligence
