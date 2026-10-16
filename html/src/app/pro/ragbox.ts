export type ProRagBoxConfig = {
  enabled: boolean;
  base_url: string;
  admin_token_configured: boolean;
  admin_token_preview?: string;
  request_timeout_seconds: number;
  write_mode: 'bleve_only' | 'dual_write';
  search_mode: 'bleve_only' | 'dual_search' | 'hybrid';
  admin_configured: boolean;
  configured: boolean;
};

export type ProRagBoxVectorConnection = {
  id: string;
  name: string;
  backend_type:
    | 'sqlite'
    | 'qdrant'
    | 'pgvector'
    | 'pg_vector'
    | 'milvus'
    | 'weaviate'
    | 'opensearch'
    | 'open_search'
    | 'elasticsearch'
    | 'pinecone'
    | 'chroma'
    | string;
  description?: string;
  status: string;
  is_default: boolean;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type ProRagBoxVectorConnectionListResponse = {
  builtin_fallback_backend: string;
  items: ProRagBoxVectorConnection[];
};

export type ProRagBoxWorkspace = {
  id: string;
  vector_connection_id: string;
  access_mode: string;
  external_ref: string;
  name: string;
  slug: string;
  description?: string;
  status: string;
  metadata?: Record<string, unknown>;
  created_at: string;
};

export type ProRagBoxWorkspaceListResponse = {
  items: ProRagBoxWorkspace[];
};

export type ProRagBoxWriteResponse = {
  local: {
    mode: string;
    indexed: boolean;
    dir_path: string;
    written_files?: string[];
  };
  ragbox: {
    skipped: boolean;
    reason?: string;
    collection_id?: string;
    document_id?: string;
    job_id?: string;
    status?: string;
    accepted_records?: number;
  };
};

export type ProRagBoxSearchHit = {
  id: string;
  source: 'bleve' | 'ragbox' | 'hybrid' | string;
  score?: number;
  file_path?: string;
  section?: string;
  title?: string;
  content: string;
  dedupe_key?: string;
};

export type ProRagBoxSearchDiagnostics = {
  mode: string;
  bleve_count: number;
  ragbox_count: number;
  merged_count: number;
  ragbox_skipped?: boolean;
  ragbox_reason?: string;
};

export type ProRagBoxSearchResponse = {
  items: ProRagBoxSearchHit[];
  diagnostics: ProRagBoxSearchDiagnostics;
};

export type ProRagBoxKnowledgeBaseStatus = {
  collection: {
    id: number;
    collection_id: string;
    name: string;
    data_type: string;
    source_path: string;
    managed_source_path?: string;
    service_mode: 'bleve_only' | 'ragbox_only' | 'dual' | string;
  };
  binding?: {
    rag_collection_id: number;
    ragbox_workspace_id: string;
    ragbox_workspace_vector_connection_id?: string;
    ragbox_collection_id: string;
    sync_status: string;
    last_job_id?: string;
    last_error?: string;
    last_synced_at?: string;
  };
  token_public_id?: string;
  token_secret?: string;
  token_configured: boolean;
  ingest_url_path?: string;
  shadow_source_path?: string;
  uses_shadow_storage: boolean;
  requested_vector_connection_id?: string;
};
