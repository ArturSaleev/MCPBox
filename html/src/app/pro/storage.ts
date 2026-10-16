export type ProStorageConfig = {
  bootstrap_backend: string;
  operational_backend: 'sqlite' | 'postgres';
  operational_configured: boolean;
  operational_postgres_configured: boolean;
  operational_postgres_host?: string;
  operational_postgres_port?: number;
  operational_postgres_database?: string;
  operational_postgres_schema?: string;
  operational_postgres_username?: string;
  operational_postgres_preview?: string;
  operational_postgres_password_preview?: string;
  logs_backend: 'operational' | 'clickhouse' | 'elasticsearch';
  logs_configured: boolean;
  effective_logs_backend: 'operational' | 'clickhouse' | 'elasticsearch';
  clickhouse_configured: boolean;
  clickhouse_url?: string;
  clickhouse_database?: string;
  clickhouse_username?: string;
  clickhouse_password_preview?: string;
  elasticsearch_configured: boolean;
  elasticsearch_url?: string;
  elasticsearch_username?: string;
  elasticsearch_password_preview?: string;
  elasticsearch_index?: string;
};

export type ProStorageConnectionStatus = {
  configured: boolean;
  reachable: boolean;
  message?: string;
};

export type ProStorageRuntimeStatus = {
  bootstrap_backend: string;
  desired_operational_backend: string;
  active_operational_backend: string;
  desired_logs_backend: string;
  active_logs_backend: string;
  operational_fallback_reason?: string;
  logs_fallback_reason?: string;
  postgres: ProStorageConnectionStatus;
  clickhouse: ProStorageConnectionStatus;
  elasticsearch: ProStorageConnectionStatus;
};

export type ProStorageShadowStatus = {
  projects_count: number;
  rag_collections_count: number;
  project_rag_links_count: number;
  last_project_sync_at?: string;
  last_rag_collection_sync_at?: string;
  last_project_rag_link_sync_at?: string;
};
