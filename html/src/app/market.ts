export type CatalogSettings = {
  catalog_source_url: string;
  last_sync_at: string;
  last_sync_status: string;
  last_sync_error: string;
  last_manifest_url: string;
  last_schema_version: string;
};

export type CatalogItem = {
  id: string;
  name: string;
  category: string;
  description: string;
  icon: string;
  icon_url: string;
  runtime: {
    type: string;
    version: string;
  };
  source: {
    type: string;
    package: string;
    version: string;
    url: string;
  };
  install: {
    strategy: string;
    metadata: Record<string, unknown>;
  };
  launch: {
    command: string;
    args: string[];
    working_dir: string;
    entry_point: string;
  };
  shared_install: boolean;
  supports_multi_project: boolean;
  transport: string;
  mcp_url: string;
  command?: string;
  args?: string[];
  env?: Array<{ key: string; value: string }>;
  default_env?: Array<{ key: string; value: string }>;
  env_schema?: Record<string, unknown>;
  env_passthrough?: string[];
  working_dir?: string;
  default_auto_start?: boolean;
  auth_type: string;
  auth_provider: string;
  oauth_authorize_url?: string;
  oauth_token_url?: string;
  oauth_refresh_url?: string;
  default_oauth_scopes?: string[];
  system_dependencies?: Array<{
    executable: string;
    min_version: string;
    critical: boolean;
    install_hint: string;
  }>;
  health_check?: {
    enabled?: boolean;
    required?: boolean;
    timeout_seconds?: number;
  };
  config_schema: Record<string, unknown>;
  capabilities: string[];
  tags: string[];
  website: string;
  docs_url: string;
  enabled: boolean;
  version: string;
  manifest_source_url: string;
  schema_version: string;
  last_synced_at: string;
};

export type InstalledPackage = {
  id: number;
  catalog_item_id: string;
  name: string;
  version: string;
  runtime_type: string;
  source_type: string;
  install_strategy: string;
  install_dir: string;
  entry_point: string;
  status: string;
  last_error: string;
  installed_at: string;
  project_use_count: number;
  created_at: string;
  updated_at: string;
};

export type ProjectOption = {
  project_id: number;
  name: string;
  root_path: string;
};

export type CatalogConfigField = {
  key: string;
  label: string;
  type: 'string' | 'array';
  required: boolean;
  secret: boolean;
  envVar: string;
  defaultValue: string;
  description: string;
};

const projectPathConfigKeys = new Set(['root_path', 'project_path', 'workspace_path']);

function schemaFields(
  schema: Record<string, unknown> | undefined,
  defaultScopes?: string[],
): CatalogConfigField[] {
  const properties =
    schema && typeof schema === 'object' && 'properties' in schema && schema.properties && typeof schema.properties === 'object'
      ? (schema.properties as Record<string, unknown>)
      : {};
  const requiredSet = new Set(
    schema && typeof schema === 'object' && 'required' in schema && Array.isArray(schema.required)
      ? schema.required.filter((value): value is string => typeof value === 'string')
      : [],
  );

  return Object.entries(properties).flatMap(([key, rawProperty]) => {
    if (!rawProperty || typeof rawProperty !== 'object') {
      return [];
    }

    const property = rawProperty as Record<string, unknown>;
    const rawType = typeof property.type === 'string' ? property.type : 'string';
    if (rawType !== 'string' && rawType !== 'array') {
      return [];
    }

    const title = typeof property.title === 'string' && property.title.trim() !== '' ? property.title.trim() : key;
    const description =
      typeof property.description === 'string' && property.description.trim() !== '' ? property.description.trim() : '';
    const defaultValue =
      rawType === 'array'
        ? Array.isArray(property.default)
          ? property.default.filter((value): value is string => typeof value === 'string').join('\n')
          : key === 'oauth_scopes' && Array.isArray(defaultScopes)
            ? defaultScopes.join('\n')
            : ''
        : typeof property.default === 'string'
          ? property.default
          : '';

    return [{
      key,
      label: title,
      type: rawType,
      required: requiredSet.has(key),
      secret:
        property.secret === true ||
        key.toLowerCase().includes('secret') ||
        key.toLowerCase().includes('token'),
      envVar: typeof property.env_var === 'string' ? property.env_var.trim() : '',
      defaultValue,
      description,
    }];
  });
}

export function catalogConfigFields(item: CatalogItem): CatalogConfigField[] {
  return schemaFields(item.config_schema, item.default_oauth_scopes);
}

export function catalogEnvFields(item: CatalogItem): CatalogConfigField[] {
  const fields = schemaFields(item.env_schema);
  const defaults = new Map((item.default_env ?? []).map((entry) => [entry.key, entry.value]));
  return fields.map((field) => ({
    ...field,
    defaultValue: defaults.get(field.key) ?? field.defaultValue,
  }));
}

export function normalizeInstallConfig(
  fields: CatalogConfigField[],
  rawValues: Record<string, string>,
): Record<string, unknown> {
  const config: Record<string, unknown> = {};
  const env: Record<string, string> = {};

  for (const field of fields) {
    const rawValue = rawValues[field.key] ?? '';
    if (field.type === 'array') {
      const values = rawValue
        .split(/\r?\n|,/)
        .map((value) => value.trim())
        .filter(Boolean);
      if (values.length > 0) {
        config[field.key] = values;
      }
      continue;
    }

    const value = rawValue.trim();
    if (value !== '') {
      if (field.secret && field.envVar) {
        env[field.envVar] = value;
        continue;
      }
      config[field.key] = value;
    }
  }

  if (Object.keys(env).length > 0) {
    config.env = env;
  }

  return config;
}

export function normalizeEnvConfig(
  fields: CatalogConfigField[],
  rawValues: Record<string, string>,
): Record<string, string> {
  const env: Record<string, string> = {};
  for (const field of fields) {
    const value = (rawValues[field.key] ?? '').trim();
    if (value !== '') {
      env[field.key] = value;
    }
  }
  return env;
}

export function applyProjectDefaultsToInstallValues(
  values: Record<string, string>,
  fields: CatalogConfigField[],
  project: ProjectOption | null,
) {
  if (!project || project.root_path.trim() === '') {
    return values;
  }

  const nextValues = { ...values };
  for (const field of fields) {
    if (field.type !== 'string' || !projectPathConfigKeys.has(field.key)) {
      continue;
    }
    nextValues[field.key] = project.root_path;
  }

  return nextValues;
}
