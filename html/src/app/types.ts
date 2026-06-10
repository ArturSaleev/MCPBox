export type ProjectStatus = {
  project_id: number;
  name: string;
  description: string;
  root_path: string;
  token: string;
  is_paused: boolean;
  identity_verification_enabled: boolean;
  bearer_auth_enabled: boolean;
  bearer_token: string;
  llama_cpp_model_path: string;
  llama_cpp_model_name: string;
  connect_url: string;
  connect_urls: string[];
  connection_ready: boolean;
  servers: ServerStatus[];
  rag_collections: RAGCollection[];
  installed_integrations: InstalledIntegration[];
  package_instances?: ProjectPackageInstance[];
  prompt: string;
};

export type ServerStatus = {
  id: number;
  name: string;
  transport: 'stdio' | 'http_stream' | string;
  launch_command: string;
  launch_command_display: string;
  command: string;
  args: string[];
  env_vars: KeyValuePair[];
  env_passthrough: string[];
  working_dir: string;
  url: string;
  bearer_token_env_var: string;
  headers: KeyValuePair[];
  header_env_vars: KeyValuePair[];
  auth_type: 'none' | 'oauth2' | string;
  oauth_provider: string;
  oauth_authorize_url: string;
  oauth_token_url: string;
  oauth_refresh_url: string;
  oauth_client_id: string;
  oauth_client_secret: string;
  oauth_scopes: string[];
  disabled_tool_names: string[];
  oauth_connected: boolean;
  oauth_connected_at: string;
  oauth_last_error: string;
  auto_start: boolean;
  status: 'Running' | 'Stopped' | 'Remote' | string;
  health_status: 'healthy' | 'failed' | 'unknown' | string;
  health_error: string;
  health_checked_at: string;
  is_enabled: boolean;
};

export type KeyValuePair = {
  key: string;
  value: string;
};

export type ProjectFormState = {
  name: string;
  description: string;
  root_path: string;
  identity_verification_enabled: boolean;
  bearer_auth_enabled: boolean;
  bearer_token: string;
};

export type ServerFormState = {
  name: string;
  transport: 'stdio' | 'http_stream';
  command: string;
  args: string[];
  env_vars: KeyValuePair[];
  env_passthrough: string[];
  working_dir: string;
  url: string;
  bearer_token_env_var: string;
  headers: KeyValuePair[];
  header_env_vars: KeyValuePair[];
  auth_type: 'none' | 'oauth2';
  oauth_provider: string;
  oauth_authorize_url: string;
  oauth_token_url: string;
  oauth_refresh_url: string;
  oauth_client_id: string;
  oauth_client_secret: string;
  oauth_scopes: string[];
  auto_start: boolean;
};

export type RAGCollection = {
  id: number;
  collection_id: string;
  name: string;
  data_type: string;
  source_path: string;
  auto_reindex: boolean;
  index_path: string;
};

export type RAGSearchResult = {
  id: string;
  file_path: string;
  section?: string;
  content: string;
};

export type InstalledIntegration = {
  id: number;
  project_id: number;
  catalog_item_id: string;
  server_id: number | null;
  name: string;
  transport: string;
  status: string;
  enabled: boolean;
  version: string;
  config: Record<string, unknown>;
  last_synced_at: string;
  created_at: string;
  updated_at: string;
};

export type ProjectPackageInstance = {
  id: number;
  project_id: number;
  installed_package_id: number;
  server_id: number | null;
  catalog_item_id: string;
  name: string;
  status: string;
  config_json: string;
};

export type OllamaStatus = {
  installed: boolean;
  models: string[];
  default_model: string;
};

export type LlamaCppStatus = {
  installed: boolean;
  configured: boolean;
  model_path: string;
  model_name: string;
  server_url: string;
  chat_template_file: string;
  running: boolean;
  managed: boolean;
  active_model_path: string;
  active_model_name: string;
};

export type OllamaLaunchResponse = {
  project_id: number;
  model: string;
  config_path: string;
  command_preview: string;
};

export type LlamaCppLaunchResponse = {
  project_id: number;
  model_path: string;
  model_name: string;
  server_url: string;
  web_ui_url: string;
  command_preview: string;
};

export type AuditLog = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  action: string;
  actor: string;
  detail: string;
  created_at: string;
};

type PerformanceSummary = {
  request_count: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  traffic_in: number;
  traffic_out: number;
};

type PerformanceTrendBucket = {
  timestamp: string;
  request_count: number;
  error_count: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  traffic_in: number;
  traffic_out: number;
};

type PerformanceServerMetricRecord = {
  server_id: number;
  request_count: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  request_bytes: number;
  response_bytes: number;
  total_traffic: number;
};

type PerformanceFailureRecord = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  operation: string;
  transport: string;
  latency_ms: number;
  request_bytes: number;
  response_bytes: number;
  error_detail: string;
  created_at: string;
};

export type PerformanceMetricsResponse = {
  window: MetricsWindow;
  summary: PerformanceSummary;
  trends: PerformanceTrendBucket[];
  top_slow_servers: PerformanceServerMetricRecord[];
  top_error_servers: PerformanceServerMetricRecord[];
  top_traffic_servers: PerformanceServerMetricRecord[];
  recent_failures: PerformanceFailureRecord[];
};

export type MetricsWindow = '5m' | '1h' | '24h';

export type EditionMeta = {
  edition_id: string;
  edition_name: string;
  capabilities: string[];
};

export type LogsFilterMode = 'all' | 'pro';

export type ServerInspection = {
  protocol_version: string;
  server_info: {
    name: string;
    title: string;
    version: string;
  };
  instructions: string;
  capabilities: string[];
  tools: Array<{
    name: string;
    title: string;
    description: string;
    input_schema?: unknown;
    output_schema?: unknown;
  }>;
  resources: Array<{
    name: string;
    title: string;
    description: string;
    uri: string;
    mime_type: string;
  }>;
  prompts: Array<{
    name: string;
    title: string;
    description: string;
    arguments: Array<{
      name: string;
      description: string;
      required: boolean;
    }>;
  }>;
  readme_path: string;
  readme: string;
};

export type ServerToolStatus = {
  name: string;
  title: string;
  description: string;
  input_schema?: unknown;
  output_schema?: unknown;
  enabled: boolean;
};
