# Knowledge bases, logging, and integrations

[Documentation](README.md) | [Русский](../ru/knowledge-integrations.md)

## Local RAG knowledge base

A knowledge base is a searchable collection of documents. In local mode MCPBox extracts text, splits it into overlapping chunks, and indexes the chunks with the embedded Bleve search engine. At query time, the most relevant chunks can be returned to an AI workflow as context.

RAG improves answers grounded in your organization’s documents, but it does not guarantee correctness. Keep source documents current, test representative queries, and retain the original source of record.

### Create a local knowledge base

1. Open **Knowledge → Create knowledge base**.
2. Provide a name.
3. Select **Local / Bleve only** (`bleve_only`) as the service mode.
4. Optionally set a source folder that the MCPBox service account can read.
5. Save the base, upload files or index the selected source folder, then associate the knowledge base with the required project.

MCPBox always creates a managed folder for uploaded files. Upload mode **Keep** retains the uploaded original for future reindexing. **Transient** processes the file and removes the original after successful processing. Archive uploads (`.zip`, `.tar`, `.tgz`, and `.tar.gz`) are unpacked and processed as batches.

### Supported local file types and limits

The local extractor supports `.txt`, `.md`, `.csv`, `.docx`, `.xlsx`, `.pptx`, `.pdf`, `.go`, `.js`, and `.php`. It indexes extractable text, not visual layout; scanned PDFs without a text layer may produce no usable content.

Local indexing splits content into chunks of up to 2,000 characters with overlap. Search returns a limited set of relevant chunks (10 by default). This is a lightweight local knowledge base, not a substitute for a distributed vector database. Capacity depends on available disk, RAM, document size, and query load; monitor these resources and divide very large or unrelated document sets into separate collections.

Automatic reindexing runs every 10 minutes only in `bleve_only` mode. Do not point a source folder at large build, dependency, or generated directories: known folders such as `node_modules`, `vendor`, `.git`, `dist`, and `build` are skipped, but an intentionally small and curated source folder is still best.

## RagBox integration

Use RagBox when you need a separate ingestion/retrieval service, workspaces, or an external vector backend. Configure it in **Settings → RagBox**:

- RagBox base URL and tenant ID;
- least-privilege **service token** for workspace, ingestion, and search operations;
- separate **admin token** only for administrative actions such as vector connections;
- request timeout;
- write mode: local only or dual write;
- search mode: local only, dual search, or hybrid.

MCPBox checks the connection when saving the configuration. For a RagBox-backed knowledge base, choose `ragbox_only` or `dual`, then select an existing RagBox workspace or a vector connection for a managed workspace. Supported vector connection options in the admin UI include SQLite, Qdrant, pgvector, Milvus, Weaviate, OpenSearch, Elasticsearch, Pinecone, and Chroma.

`dual` keeps local Bleve and RagBox in sync. `ragbox_only` relies on RagBox. Automatic local reindexing is disabled in both modes to prevent duplicate background work; use uploads, source synchronization, or a manual sync instead.

## Logging and analytics

Open **Logs** to review the audit trail and **Analytics** to review request volume, errors, latency, traffic, and recent failures. Logs can be filtered by project and by Access/Policy activity, refreshed, and exported from the visible list.

Use logs to investigate failed MCP calls, server health checks, knowledge-base actions, configuration changes, and access activity. Avoid writing secrets or sensitive personal data into project names, prompts, custom headers, or request metadata because operational logs may retain those values.

## Other settings

### PolicyBox

In **Settings → PolicyBox**, provide the PolicyBox base URL and its admin token. Bind the PolicyBox project to the MCPBox project. When project identity verification is enabled, callers to the project MCP endpoint must present a valid PolicyBox runtime bearer token whose project binding matches. Test ticket, exchange, and validation flows in the runtime playground before enabling this for users.

### Storage

In **Settings → Storage**, MCPBox keeps its bootstrap/system layer on SQLite and can use PostgreSQL as an operational backend. The log backend can remain operational storage or be configured for ClickHouse or Elasticsearch. Change these settings only with a backup and a tested rollback plan; confirm the runtime and shadow status after saving.

### Access administration

The administration interface also manages users, roles, sessions, and API tokens. Grant the smallest practical role, revoke inactive sessions/tokens, and use OIDC SSO where it is required by the organization.
