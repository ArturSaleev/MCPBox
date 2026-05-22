# MCPBox 1.2.0

## Release Notes

MCPBox 1.2.0 introduces the first built-in Knowledge Base / RAG workflow for the free edition.

### Highlights

- Built-in Knowledge Base page
  - MCPBox now includes a dedicated `Knowledge Base` area in the UI.
  - Collections can be created, renamed, deleted, indexed, and searched directly from the control plane.

- Global collections with project-level linking
  - Knowledge collections are now global records, not project-owned data.
  - One collection can be connected to one or many projects.
  - Projects can have several connected knowledge bases at the same time.

- Local full-text RAG for the free edition
  - Collections are indexed locally with Bleve full-text search.
  - No external vector database, Docker service, or embedding pipeline is required.
  - The current free edition is intentionally optimized as an embedded local search layer.

- Multi-format document ingestion
  - MCPBox can now index code and plain text files.
  - It also supports `CSV`, `XLSX`, `DOCX`, `PPTX`, and text-based `PDF` documents.
  - Search results can include document section hints such as `Sheet`, `Slide`, and `Page`.

- Project-level MCP knowledge tool
  - Connected collections are exposed through the built-in MCP tool `search_project_knowledge`.
  - The project endpoint can search across all connected knowledge bases without creating one tool per collection.

- Better audit visibility
  - Knowledge search tool calls now appear in the audit log with query details, selected collections, and result counts.

### What Changed

- Added global Knowledge Base collection storage and project-to-collection linking
- Added local Bleve-backed indexing and keyword search for the first RAG workflow
- Added collection management, indexing, search, and search-result modals in the embedded UI
- Added built-in `search_project_knowledge` MCP tool generation for project endpoints
- Added document extraction support for `CSV`, `XLSX`, `DOCX`, `PPTX`, and text-based `PDF`
- Added document section metadata in search results for better navigation
- Added `knowledge_base/` local runtime storage for on-disk indexes

### Upgrade Notes

- The current Knowledge Base layer uses local full-text search with Bleve, not embedding indexes.
- Existing installations with older project-owned `rag_collections` schemas are migrated to the new global collection model on startup.
- `PDF` support in `1.2.0` is limited to files with a text layer. OCR for scanned documents is planned for the future `Pro` edition.

## Short Release Text

MCPBox 1.2.0 adds a built-in Knowledge Base / RAG workflow for the free edition. It introduces global collections, project-level knowledge search through `search_project_knowledge`, local full-text indexing with Bleve, and support for code, text, CSV, XLSX, DOCX, PPTX, and text-based PDF files.

## Russian Release Text

MCPBox 1.2.0 добавляет встроенный workflow Базы знаний / RAG для бесплатной версии. В релиз вошли глобальные коллекции, project-level поиск через `search_project_knowledge`, локальная полнотекстовая индексация на Bleve и поддержка кода, текста, CSV, XLSX, DOCX, PPTX и PDF с текстовым слоем.
