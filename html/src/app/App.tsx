import { FormEvent, useEffect, useRef, useState } from 'react';
import {
  AlertCircle,
  Bot,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Copy,
  Database,
  FolderKanban,
  Info,
  LoaderCircle,
  Pause,
  Pencil,
  Play,
  Plus,
  Radio,
  RefreshCw,
  Server,
  ShoppingBag,
  Square,
  TextSearch,
  Trash2,
} from 'lucide-react';
import {
  defaultLanguage,
  detectInitialLanguage,
  dictionaries,
  languageStorageKey,
  Language,
} from './i18n';
import logo from '../styles/logo.png';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './components/ui/select';
import { Tooltip, TooltipContent, TooltipTrigger } from './components/ui/tooltip';

type ServerStatus = {
  id: number;
  name: string;
  transport: 'stdio' | 'http_stream' | string;
  launch_command: string;
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

type ProjectStatus = {
  project_id: number;
  name: string;
  description: string;
  root_path: string;
  token: string;
  is_paused: boolean;
  connect_url: string;
  connect_urls: string[];
  connection_ready: boolean;
  servers: ServerStatus[];
  rag_collections: RAGCollection[];
  installed_integrations: InstalledIntegration[];
  package_instances?: ProjectPackageInstance[];
};

type RAGCollection = {
  id: number;
  collection_id: string;
  name: string;
  data_type: string;
  source_path: string;
  index_path: string;
};

type RAGSearchResult = {
  id: string;
  file_path: string;
  section?: string;
  content: string;
};

type InstalledIntegration = {
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

type ProjectPackageInstance = {
  id: number;
  project_id: number;
  installed_package_id: number;
  server_id: number | null;
  catalog_item_id: string;
  name: string;
  status: string;
  config_json: string;
};

type ProjectOption = {
  project_id: number;
  name: string;
  root_path: string;
};

type CatalogSettings = {
  catalog_source_url: string;
  last_sync_at: string;
  last_sync_status: string;
  last_sync_error: string;
  last_manifest_url: string;
  last_schema_version: string;
};

type CatalogItem = {
  id: string;
  name: string;
  category: string;
  description: string;
  icon: string;
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
  env?: KeyValuePair[];
  env_passthrough?: string[];
  working_dir?: string;
  default_auto_start?: boolean;
  auth_type: string;
  auth_provider: string;
  oauth_authorize_url?: string;
  oauth_token_url?: string;
  oauth_refresh_url?: string;
  default_oauth_scopes?: string[];
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

type CatalogResponse = {
  settings: CatalogSettings;
  items: CatalogItem[];
};

type InstalledPackage = {
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

type InstalledPackageListResponse = {
  items: InstalledPackage[];
};

type InstallPackageResponse = {
  package: InstalledPackage;
};

type AuditLog = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  action: string;
  actor: string;
  detail: string;
  created_at: string;
};

type ApiError = {
  error: string;
};

type KnowledgeSearchAuditDetail = {
  tool?: string;
  query?: string;
  collections?: string[];
  results?: number;
};

type OllamaLaunchResponse = {
  project_id: number;
  model: string;
  config_path: string;
  command_preview: string;
};

type OllamaStatus = {
  installed: boolean;
  models: string[];
  default_model: string;
};

type CatalogConfigField = {
  key: string;
  label: string;
  type: 'string' | 'array';
  required: boolean;
  secret: boolean;
  defaultValue: string;
};

const projectPathConfigKeys = new Set(['root_path', 'project_path', 'workspace_path']);

type ServerInspection = {
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

type ProjectFormState = {
  name: string;
  description: string;
  root_path: string;
};

type ServerFormState = {
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

type KeyValuePair = {
  key: string;
  value: string;
};

const emptyProjectForm: ProjectFormState = {
  name: '',
  description: '',
  root_path: '',
};

const emptyServerForm: ServerFormState = {
  name: '',
  transport: 'stdio',
  command: '',
  args: [''],
  env_vars: [{ key: '', value: '' }],
  env_passthrough: [''],
  working_dir: '',
  url: '',
  bearer_token_env_var: '',
  headers: [{ key: '', value: '' }],
  header_env_vars: [{ key: '', value: '' }],
  auth_type: 'none',
  oauth_provider: '',
  oauth_authorize_url: '',
  oauth_token_url: '',
  oauth_refresh_url: '',
  oauth_client_id: '',
  oauth_client_secret: '',
  oauth_scopes: [''],
  auto_start: false,
};

const emptyRAGCollectionForm = {
  name: '',
};

async function apiRequest<T>(
  input: RequestInfo,
  requestFailedMessage: (status: number) => string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(input, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    let message = requestFailedMessage(response.status);
    try {
      const payload = (await response.json()) as ApiError;
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // Ignore JSON parsing errors and keep fallback message.
    }

    throw new Error(message);
  }

  return (await response.json()) as T;
}

function statusTone(status: string) {
  return status === 'Running'
    ? 'bg-status-running/15 text-status-running border-status-running/30'
    : status === 'Disabled'
      ? 'bg-amber-500/10 text-amber-600 border-amber-500/30'
    : status === 'Remote'
      ? 'bg-electric-blue/12 text-electric-blue border-electric-blue/30'
    : 'bg-muted text-muted-foreground border-border';
}

function statusIcon(status: string) {
  return status === 'Running' ? (
    <Play className="h-3.5 w-3.5 fill-current" />
  ) : status === 'Disabled' ? (
    <Pause className="h-3.5 w-3.5" />
  ) : status === 'Remote' ? (
    <Radio className="h-3.5 w-3.5" />
  ) : (
    <Square className="h-3.5 w-3.5" />
  );
}

function healthTone(status: string) {
  return status === 'healthy'
    ? 'bg-status-running/15 text-status-running border-status-running/30'
    : status === 'failed'
      ? 'bg-destructive/10 text-destructive border-destructive/30'
      : 'bg-muted text-muted-foreground border-border';
}

function healthLabel(status: string, labels: typeof dictionaries.en.labels) {
  if (status === 'healthy') {
    return labels.healthy;
  }
  if (status === 'failed') {
    return labels.failed;
  }

  return labels.unknown;
}

function formatSchema(schema: unknown) {
  if (!schema) {
    return '';
  }

  try {
    return JSON.stringify(schema, null, 2);
  } catch {
    return String(schema);
  }
}

function formatAuditAction(action: string) {
  if (action === 'tool_call_project_knowledge_search') {
    return 'tool_call -> search_project_knowledge';
  }
  return action;
}

function formatAuditDetail(entry: AuditLog) {
  if (entry.action !== 'tool_call_project_knowledge_search') {
    return entry.detail;
  }

  try {
    const parsed = JSON.parse(entry.detail) as KnowledgeSearchAuditDetail;
    const collections = parsed.collections?.length ? `[${parsed.collections.join(', ')}]` : '[all connected collections]';
    const query = parsed.query ? `query="${parsed.query}"` : '';
    const results =
      typeof parsed.results === 'number' ? `results=${parsed.results}` : '';
    return [collections, query, results]
      .filter((part) => part && part.trim() !== '')
      .join(' ');
  } catch {
    return entry.detail;
  }
}

function catalogConfigFields(item: CatalogItem): CatalogConfigField[] {
  const schema = item.config_schema;
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
    const defaultValue =
      rawType === 'array'
        ? Array.isArray(property.default)
          ? property.default.filter((value): value is string => typeof value === 'string').join('\n')
          : key === 'oauth_scopes' && Array.isArray(item.default_oauth_scopes)
            ? item.default_oauth_scopes.join('\n')
            : ''
        : typeof property.default === 'string'
          ? property.default
          : '';

    return [{
      key,
      label: title,
      type: rawType,
      required: requiredSet.has(key),
      secret: key.toLowerCase().includes('secret') || key.toLowerCase().includes('token'),
      defaultValue,
    }];
  });
}

function normalizeInstallConfig(
  fields: CatalogConfigField[],
  rawValues: Record<string, string>,
): Record<string, string | string[]> {
  const config: Record<string, string | string[]> = {};

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
      config[field.key] = value;
    }
  }

  return config;
}

function OllamaIcon({ className }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center justify-center rounded-md border border-current/20 bg-current/10 ${className ?? ''}`}
    >
      <Bot className="h-[0.9em] w-[0.9em]" />
    </span>
  );
}

export default function App() {
  const [view, setView] = useState<'projects' | 'knowledge' | 'market' | 'logs'>('projects');
  const [language, setLanguage] = useState<Language>(detectInitialLanguage);
  const [projects, setProjects] = useState<ProjectStatus[]>([]);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [selectedLogsProjectId, setSelectedLogsProjectId] = useState<number | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [projectForm, setProjectForm] = useState<ProjectFormState>(emptyProjectForm);
  const [serverForm, setServerForm] = useState<ServerFormState>(emptyServerForm);
  const [allRAGCollections, setAllRAGCollections] = useState<RAGCollection[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [creatingProject, setCreatingProject] = useState(false);
  const [createProjectOpen, setCreateProjectOpen] = useState(false);
  const [editingProjectId, setEditingProjectId] = useState<number | null>(null);
  const [addingServer, setAddingServer] = useState(false);
  const [addServerOpen, setAddServerOpen] = useState(false);
  const [, setOAuthAdvancedOpen] = useState(false);
  const [editingServerId, setEditingServerId] = useState<number | null>(null);
  const [editingRAGCollectionId, setEditingRAGCollectionId] = useState<string | null>(null);
  const [createRAGCollectionOpen, setCreateRAGCollectionOpen] = useState(false);
  const [connectRAGCollectionOpen, setConnectRAGCollectionOpen] = useState(false);
  const [creatingRAGCollection, setCreatingRAGCollection] = useState(false);
  const [linkingCollectionId, setLinkingCollectionId] = useState<string | null>(null);
  const [ragCollectionForm, setRAGCollectionForm] = useState(emptyRAGCollectionForm);
  const [ragIndexPaths, setRAGIndexPaths] = useState<Record<string, string>>({});
  const [ragSearchQueries, setRAGSearchQueries] = useState<Record<string, string>>({});
  const [ragSearchResults, setRAGSearchResults] = useState<Record<string, RAGSearchResult[]>>({});
  const [ragSearchResultsOpen, setRAGSearchResultsOpen] = useState(false);
  const [activeRAGSearchCollectionId, setActiveRAGSearchCollectionId] = useState<string | null>(null);
  const [indexingCollectionId, setIndexingCollectionId] = useState<string | null>(null);
  const [searchingCollectionId, setSearchingCollectionId] = useState<string | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [catalogSettings, setCatalogSettings] = useState<CatalogSettings | null>(null);
  const [installedPackages, setInstalledPackages] = useState<InstalledPackage[]>([]);
  const [ollamaStatus, setOllamaStatus] = useState<OllamaStatus | null>(null);
  const [selectedOllamaModel, setSelectedOllamaModel] = useState('');
  const [catalogURL, setCatalogURL] = useState('https://webeasy.kz/mcpbox/catalog.json');
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogSyncing, setCatalogSyncing] = useState(false);
  const [catalogURLVisible, setCatalogURLVisible] = useState(false);
  const [selectedCatalogCategory, setSelectedCatalogCategory] = useState('all');
  const [installingCatalogItemId, setInstallingCatalogItemId] = useState<string | null>(null);
  const [addingCatalogItemId, setAddingCatalogItemId] = useState<string | null>(null);
  const [installDialogOpen, setInstallDialogOpen] = useState(false);
  const [installDialogItem, setInstallDialogItem] = useState<CatalogItem | null>(null);
  const [installDialogProjectId, setInstallDialogProjectId] = useState<number | null>(null);
  const [installDialogValues, setInstallDialogValues] = useState<Record<string, string>>({});
  const [busyProjectId, setBusyProjectId] = useState<number | null>(null);
  const [launchingOllamaProjectId, setLaunchingOllamaProjectId] = useState<number | null>(null);
  const [busyServerId, setBusyServerId] = useState<number | null>(null);
  const [inspectOpen, setInspectOpen] = useState(false);
  const [inspectingServerId, setInspectingServerId] = useState<number | null>(null);
  const [inspection, setInspection] = useState<ServerInspection | null>(null);
  const [inspectionServerName, setInspectionServerName] = useState('');
  const [inspectionError, setInspectionError] = useState<string | null>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [authServerId, setAuthServerId] = useState<number | null>(null);
  const [copied, setCopied] = useState(false);
  const [connectionURLsExpanded, setConnectionURLsExpanded] = useState(false);
  const logsViewportRef = useRef<HTMLDivElement | null>(null);
  const dictionary = dictionaries[language];
  const { labels, messages } = dictionary;
  const languageOptions: Array<{ value: Language; label: string }> = [
    { value: 'en', label: labels.english },
    { value: 'ru', label: labels.russian },
  ];
  const navigationItems = [
    { id: 'projects' as const, label: labels.projects, icon: FolderKanban },
    { id: 'knowledge' as const, label: labels.knowledgeBase, icon: Database },
    { id: 'market' as const, label: labels.market, icon: ShoppingBag },
    { id: 'logs' as const, label: labels.logs, icon: TextSearch },
  ];

  const selectedProject =
    projects.find((project) => project.project_id === selectedProjectId) ?? null;
  const alternativeConnectURLs = (selectedProject?.connect_urls ?? []).filter(
    (url) => url !== selectedProject?.connect_url,
  );
  const filteredLogsProject =
    selectedLogsProjectId !== null
      ? projects.find((project) => project.project_id === selectedLogsProjectId) ?? null
      : null;
  const serverNamesById = Object.fromEntries(
    projects.flatMap((project) =>
      project.servers.map((server) => [server.id, server.name] as const),
    ),
  );
  const selectedProjectHealthyCount = selectedProject
    ? selectedProject.servers.filter((server) => server.health_status === 'healthy').length
    : 0;
  const selectedProjectOAuthConnectedCount = selectedProject
    ? selectedProject.servers.filter(
        (server) => server.transport === 'http_stream' && server.auth_type === 'oauth2' && server.oauth_connected,
      ).length
    : 0;
  const serverIntegrationsByServerID = Object.fromEntries(
    projects.flatMap((project) =>
      (project.installed_integrations ?? [])
        .filter((integration) => integration.server_id !== null)
        .map((integration) => [integration.server_id as number, integration] as const),
    ),
  );
  const authServer =
    selectedProject?.servers.find((server) => server.id === authServerId) ?? null;
  const installedCatalogIDs = new Set(
    (selectedProject?.installed_integrations ?? []).map((integration) => integration.catalog_item_id),
  );
  const installedPackageCatalogIDs = new Set(
    installedPackages
      .filter((pkg) => pkg.status === 'installed')
      .map((pkg) => pkg.catalog_item_id),
  );
  const connectedRAGCollectionIDs = new Set(
    (selectedProject?.rag_collections ?? []).map((collection) => collection.collection_id),
  );
  const availableRAGCollections = allRAGCollections.filter(
    (collection) => !connectedRAGCollectionIDs.has(collection.collection_id),
  );
  const activeRAGSearchCollection =
    activeRAGSearchCollectionId !== null
      ? allRAGCollections.find((collection) => collection.collection_id === activeRAGSearchCollectionId) ?? null
      : null;
  const activeRAGSearchResults = activeRAGSearchCollectionId
    ? ragSearchResults[activeRAGSearchCollectionId] ?? []
    : [];
  const editingServerIntegration =
    editingServerId !== null ? serverIntegrationsByServerID[editingServerId] ?? null : null;
  const catalogCategories = ['all', ...Array.from(new Set(catalogItems.map((item) => item.category || labels.generalCategory))).sort((left, right) => left.localeCompare(right))];
  const filteredCatalogItems = selectedCatalogCategory === 'all'
    ? catalogItems
    : catalogItems.filter((item) => (item.category || labels.generalCategory) === selectedCatalogCategory);
  const installDialogFields = installDialogItem ? catalogConfigFields(installDialogItem) : [];
  const projectOptions: ProjectOption[] = projects.map((project) => ({
    project_id: project.project_id,
    name: project.name,
    root_path: project.root_path,
  }));
  const installDialogProject =
    projectOptions.find((project) => project.project_id === installDialogProjectId) ?? null;
  const shouldShowOllamaControls = ollamaStatus?.installed ?? false;
  const canLaunchOllama =
    !!selectedOllamaModel &&
    (ollamaStatus?.models.length ?? 0) > 0;

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    window.localStorage.setItem(languageStorageKey, language);
    document.documentElement.lang = language;
  }, [language]);

  useEffect(() => {
    setConnectionURLsExpanded(false);
  }, [selectedProjectId]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      const isToggleCombo =
        (event.metaKey || event.ctrlKey) &&
        event.shiftKey &&
        event.key.toLowerCase() === 'u';

      if (isToggleCombo) {
        event.preventDefault();
        setCatalogURLVisible((current) => !current);
      }

      if (event.key === 'Escape') {
        setCatalogURLVisible(false);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) {
        return;
      }
      if (event.data?.type !== 'mcpbox-oauth-complete') {
        return;
      }

      void loadProjects();
      void loadLogs({ silent: true });
    };

    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, [selectedLogsProjectId]);

  useEffect(() => {
    void Promise.all([loadProjects(true), loadRAGCollections(), loadCatalog(true), loadInstalledPackages(), loadOllamaStatus()]);
  }, []);

  useEffect(() => {
    if (!ollamaStatus?.installed) {
      setSelectedOllamaModel('');
      return;
    }

    if (
      selectedOllamaModel &&
      ollamaStatus.models.includes(selectedOllamaModel)
    ) {
      return;
    }

    setSelectedOllamaModel(ollamaStatus.default_model || ollamaStatus.models[0] || '');
  }, [ollamaStatus, selectedOllamaModel]);

  useEffect(() => {
    if (view === 'logs') {
      void loadLogs();
    }
  }, [view, selectedLogsProjectId]);

  useEffect(() => {
    if (view !== 'logs') {
      return;
    }

    const intervalID = window.setInterval(() => {
      void loadLogs({ silent: true });
    }, 5000);

    return () => window.clearInterval(intervalID);
  }, [view, selectedLogsProjectId]);

  useEffect(() => {
    if (view !== 'logs' || !logsViewportRef.current) {
      return;
    }

    logsViewportRef.current.scrollTop = logsViewportRef.current.scrollHeight
  }, [logs, view]);

  useEffect(() => {
    if (projects.length === 0) {
      setSelectedProjectId(null);
      return;
    }

    if (
      !selectedProjectId ||
      !projects.some((project) => project.project_id === selectedProjectId)
    ) {
      setSelectedProjectId(projects[0].project_id);
    }
  }, [projects, selectedProjectId]);

  useEffect(() => {
    if (projects.length === 0) {
      setInstallDialogProjectId(null);
      return;
    }

    if (
      installDialogProjectId &&
      projects.some((project) => project.project_id === installDialogProjectId)
    ) {
      return;
    }

    setInstallDialogProjectId(selectedProjectId ?? projects[0].project_id);
  }, [projects, selectedProjectId, installDialogProjectId]);

  async function loadProjects(initial = false) {
    if (initial) {
      setLoading(true);
    } else {
      setRefreshing(true);
    }

    try {
      setError(null);
      const nextProjects = await apiRequest<ProjectStatus[]>(
        '/api/projects',
        messages.requestFailed,
      );
      setProjects(nextProjects);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }

  async function loadRAGCollections() {
    try {
      const response = await apiRequest<{ items: RAGCollection[] }>(
        '/api/rag/collections',
        messages.requestFailed,
      );
      setAllRAGCollections(response.items);
      setRAGIndexPaths((current) => {
        const next = { ...current };
        for (const collection of response.items) {
          if (collection.source_path?.trim()) {
            next[collection.collection_id] = collection.source_path;
          }
        }
        return next;
      });
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    }
  }

  async function loadLogs(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setLogsLoading(true);
    }
    try {
      const query =
        selectedLogsProjectId !== null ? `?project_id=${selectedLogsProjectId}` : '';
      const nextLogs = await apiRequest<AuditLog[]>(
        `/api/logs${query}`,
        messages.requestFailed,
      );
      setLogs(nextLogs);
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      if (!options?.silent) {
        setLogsLoading(false);
      }
    }
  }

  async function loadCatalog(initial = false) {
    if (initial) {
      setCatalogLoading(true);
    }

    try {
      const response = await apiRequest<CatalogResponse>(
        '/api/catalog/items',
        messages.requestFailed,
      );
      setCatalogItems(response.items);
      setCatalogSettings(response.settings);
      if (response.settings.catalog_source_url) {
        setCatalogURL(response.settings.catalog_source_url);
      }
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : 'Failed to load catalog');
    } finally {
      if (initial) {
        setCatalogLoading(false);
      }
    }
  }

  async function loadOllamaStatus() {
    try {
      const nextStatus = await apiRequest<OllamaStatus>(
        '/api/ollama/status',
        messages.requestFailed,
      );
      setOllamaStatus(nextStatus);
    } catch {
      setOllamaStatus(null);
    }
  }

  async function loadInstalledPackages() {
    try {
      const response = await apiRequest<InstalledPackageListResponse>(
        '/api/packages',
        messages.requestFailed,
      );
      setInstalledPackages(response.items);
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadPackagesError);
    }
  }

  async function syncCatalog() {
    setCatalogSyncing(true);
    setActionError(null);

    try {
      const response = await apiRequest<CatalogResponse>(
        '/api/catalog/sync',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ url: catalogURL }),
        },
      );
      setCatalogItems(response.items);
      setCatalogSettings(response.settings);
      await loadInstalledPackages();
      await loadLogs({ silent: true });
    } catch (syncError) {
      setActionError(syncError instanceof Error ? syncError.message : 'Failed to sync catalog');
    } finally {
      setCatalogSyncing(false);
    }
  }

  function applyProjectDefaultsToInstallValues(
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

  function openInstallDialog(item: CatalogItem, preferredProjectId?: number | null) {
    const fields = catalogConfigFields(item);
    const baseValues = Object.fromEntries(
      fields.map((field) => [field.key, field.defaultValue]),
    );
    const resolvedProjectId =
      preferredProjectId && projects.some((project) => project.project_id === preferredProjectId)
        ? preferredProjectId
        : selectedProjectId && projects.some((project) => project.project_id === selectedProjectId)
          ? selectedProjectId
          : projects[0]?.project_id ?? null;
    const project =
      projectOptions.find((entry) => entry.project_id === resolvedProjectId) ?? null;
    setInstallDialogItem(item);
    setInstallDialogProjectId(resolvedProjectId);
    setInstallDialogValues(applyProjectDefaultsToInstallValues(baseValues, fields, project));
    setInstallDialogOpen(true);
  }

  async function installCatalogPackage(item: CatalogItem) {
    setInstallingCatalogItemId(item.id);
    setActionError(null);

    try {
      await apiRequest<InstallPackageResponse>(
        `/api/catalog/items/${item.id}/install`,
        messages.requestFailed,
        {
          method: 'POST',
        },
      );
      await loadInstalledPackages();
      await loadLogs({ silent: true });
      return true;
    } catch (installError) {
      setActionError(
        installError instanceof Error ? installError.message : messages.installPackageError,
      );
      return false;
    } finally {
      setInstallingCatalogItemId(null);
    }
  }

  async function performCatalogInstall(
    item: CatalogItem,
    projectId: number,
    config: Record<string, string | string[]>,
  ) {
    const targetProject = projects.find((project) => project.project_id === projectId);
    if (!targetProject) {
      setActionError(messages.selectProjectBeforeInstall);
      return false;
    }

    setAddingCatalogItemId(item.id);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/catalog/items/${item.id}/add-to-project`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            project_id: projectId,
            name: item.name,
            config,
          }),
        },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      await loadInstalledPackages();
      await loadLogs({ silent: true });
      return true;
    } catch (installError) {
      setActionError(
        installError instanceof Error ? installError.message : messages.addPackageToProjectError,
      );
      return false;
    } finally {
      setAddingCatalogItemId(null);
    }
  }

  async function installCatalogItem(item: CatalogItem, preferredProjectId?: number | null) {
    const targetProjectId =
      preferredProjectId && projects.some((project) => project.project_id === preferredProjectId)
        ? preferredProjectId
        : selectedProjectId && projects.some((project) => project.project_id === selectedProjectId)
          ? selectedProjectId
          : projects[0]?.project_id ?? null;
    if (!targetProjectId) {
      setActionError(messages.selectProjectBeforeInstall);
      return;
    }

    const packageInstalled = installedPackageCatalogIDs.has(item.id);
    if (!packageInstalled) {
      const ok = await installCatalogPackage(item);
      if (!ok) {
        return;
      }
    }
    const fields = catalogConfigFields(item);
    if (fields.length > 0) {
      openInstallDialog(item, targetProjectId);
      return;
    }

    await performCatalogInstall(item, targetProjectId, {});
  }

  function projectNameFromLog(projectId: number | null) {
    if (!projectId) {
      return null;
    }

    return (
      projects.find((project) => project.project_id === projectId)?.name ??
      messages.projectTag(projectId)
    );
  }

  function serverNameFromLog(serverId: number | null) {
    if (!serverId) {
      return null;
    }

    return serverNamesById[serverId] ?? messages.serverTag(serverId);
  }

  const projectActivity = Object.entries(
    logs.reduce<Record<string, number>>((accumulator, entry) => {
      if (entry.project_id) {
        const key = String(entry.project_id);
        accumulator[key] = (accumulator[key] ?? 0) + 1;
      }
      return accumulator;
    }, {}),
  )
    .map(([projectId, count]) => ({
      id: Number(projectId),
      name: projectNameFromLog(Number(projectId)) ?? messages.projectTag(Number(projectId)),
      count,
    }))
    .sort((left, right) => right.count - left.count);

  const serverActivity = Object.entries(
    logs.reduce<Record<string, number>>((accumulator, entry) => {
      if (entry.server_id) {
        const key = String(entry.server_id);
        accumulator[key] = (accumulator[key] ?? 0) + 1;
      }
      return accumulator;
    }, {}),
  )
    .map(([serverId, count]) => ({
      id: Number(serverId),
      name: serverNameFromLog(Number(serverId)) ?? messages.serverTag(Number(serverId)),
      count,
    }))
    .sort((left, right) => right.count - left.count);

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatingProject(true);
    setActionError(null);

    try {
      if (editingProjectId) {
        const updatedProject = await apiRequest<ProjectStatus>(
          `/api/projects/${editingProjectId}`,
          messages.requestFailed,
          {
            method: 'PUT',
            body: JSON.stringify(projectForm),
          },
        );

        setProjects((current) =>
          current.map((project) =>
            project.project_id === updatedProject.project_id ? updatedProject : project,
          ),
        );
      } else {
        await apiRequest('/api/projects', messages.requestFailed, {
          method: 'POST',
          body: JSON.stringify(projectForm),
        });
        await loadProjects();
      }

      setProjectForm(emptyProjectForm);
      setEditingProjectId(null);
      setCreateProjectOpen(false);
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.createProjectError,
      );
    } finally {
      setCreatingProject(false);
    }
  }

  async function addServer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }

    setAddingServer(true);
    setActionError(null);

    try {
      const payload = {
        name: serverForm.name,
        transport: serverForm.transport,
        command: serverForm.command,
        args: serverForm.args,
        env_vars: serverForm.env_vars,
        env_passthrough: serverForm.env_passthrough,
        working_dir: serverForm.working_dir,
        url: serverForm.url,
        bearer_token_env_var: serverForm.bearer_token_env_var,
        headers: serverForm.headers,
        header_env_vars: serverForm.header_env_vars,
        auth_type: serverForm.transport === 'http_stream' ? serverForm.auth_type : 'none',
        oauth_provider: serverForm.transport === 'http_stream' ? serverForm.oauth_provider : '',
        oauth_authorize_url:
          serverForm.transport === 'http_stream' ? serverForm.oauth_authorize_url : '',
        oauth_token_url: serverForm.transport === 'http_stream' ? serverForm.oauth_token_url : '',
        oauth_refresh_url:
          serverForm.transport === 'http_stream' ? serverForm.oauth_refresh_url : '',
        oauth_client_id: serverForm.transport === 'http_stream' ? serverForm.oauth_client_id : '',
        oauth_client_secret:
          serverForm.transport === 'http_stream' ? serverForm.oauth_client_secret : '',
        oauth_scopes: serverForm.transport === 'http_stream' ? serverForm.oauth_scopes : [],
        auto_start: serverForm.transport === 'stdio' ? serverForm.auto_start : false,
      };

      const targetServerIdBeforeReset = editingServerId;
      const shouldStartOAuth =
        payload.transport === 'http_stream' && payload.auth_type === 'oauth2';

      const updatedProject = editingServerId
        ? await apiRequest<ProjectStatus>(
            `/api/servers/${editingServerId}`,
            messages.requestFailed,
            {
              method: 'PUT',
              body: JSON.stringify(payload),
            },
          )
        : await apiRequest<ProjectStatus>(
            `/api/projects/${selectedProject.project_id}/servers`,
            messages.requestFailed,
            {
              method: 'POST',
              body: JSON.stringify(payload),
            },
          );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );

      let oauthServerId: number | null = null;
      if (shouldStartOAuth) {
        oauthServerId = resolveOAuthServerId(
          updatedProject,
          targetServerIdBeforeReset,
          payload.name,
          payload.url,
        );
      }

      setServerForm(emptyServerForm);
      setEditingServerId(null);
      setAddServerOpen(false);
      setOAuthAdvancedOpen(false);

      if (oauthServerId) {
        await connectOAuth(oauthServerId);
      }
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.addServerError);
    } finally {
      setAddingServer(false);
    }
  }

  async function runServerAction(serverId: number, action: 'start' | 'stop') {
    setBusyServerId(serverId);
    setActionError(null);

    try {
      await apiRequest<{ server_id: number; status: string }>(
        `/api/servers/${serverId}/${action}`,
        messages.requestFailed,
        {
          method: 'POST',
        },
      );

      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(
        submitError instanceof Error
          ? submitError.message
          : action === 'start'
            ? messages.startServerError
            : messages.stopServerError,
      );
      await loadProjects();
      await loadLogs();
    } finally {
      setBusyServerId(null);
    }
  }

  async function setProjectPaused(projectId: number, paused: boolean) {
    setBusyProjectId(projectId);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${projectId}/${paused ? 'pause' : 'resume'}`,
        messages.requestFailed,
        { method: 'POST' },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.setProjectPausedError);
    } finally {
      setBusyProjectId(null);
    }
  }

  async function launchProjectOllama(projectId: number) {
    setLaunchingOllamaProjectId(projectId);
    setActionError(null);

    try {
      await apiRequest<OllamaLaunchResponse>(
        `/api/projects/${projectId}/launch-ollama`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ model: selectedOllamaModel }),
        },
      );
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.launchOllamaError);
    } finally {
      setLaunchingOllamaProjectId(null);
    }
  }

  async function copyConnectURL() {
    if (!selectedProject?.connect_url) {
      return;
    }

    await navigator.clipboard.writeText(selectedProject.connect_url);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }

  async function setServerEnabled(serverId: number, enabled: boolean) {
    setBusyServerId(serverId);
    setActionError(null);

    try {
      await apiRequest<{ server_id: number; status: string }>(
        `/api/servers/${serverId}/${enabled ? 'enable' : 'disable'}`,
        messages.requestFailed,
        { method: 'POST' },
      );

      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.setServerEnabledError);
    } finally {
      setBusyServerId(null);
    }
  }

  async function inspectServer(server: ServerStatus) {
    setInspectOpen(true);
    setInspectingServerId(server.id);
    setInspection(null);
    setInspectionError(null);
    setInspectionServerName(server.name);

    try {
      const nextInspection = await apiRequest<ServerInspection>(
        `/api/servers/${server.id}/inspect`,
        messages.requestFailed,
      );
      setInspection(nextInspection);
    } catch (loadError) {
      setInspectionError(
        loadError instanceof Error ? loadError.message : messages.inspectServerError,
      );
    } finally {
      setInspectingServerId(null);
    }
  }

  function updateServerForm<K extends keyof ServerFormState>(key: K, value: ServerFormState[K]) {
    setServerForm((current) => ({ ...current, [key]: value }));
  }

  function updateServerLastArg(value: string) {
    setServerForm((current) => {
      const nextArgs = current.args.length > 0 ? [...current.args] : [''];
      if (nextArgs.length === 0) {
        nextArgs.push(value);
      } else {
        nextArgs[nextArgs.length - 1] = value;
      }
      return { ...current, args: nextArgs };
    });
  }

  function startEditProject() {
    if (!selectedProject) {
      return;
    }

    setProjectForm({
      name: selectedProject.name,
      description: selectedProject.description,
      root_path: selectedProject.root_path,
    });
    setEditingProjectId(selectedProject.project_id);
    setCreateProjectOpen(true);
  }

  async function deleteProject(projectId: number) {
    const confirmed = window.confirm('Delete this project and all its servers?');
    if (!confirmed) {
      return;
    }

    setBusyProjectId(projectId);
    setActionError(null);

    try {
      await apiRequest<{ deleted: boolean }>(`/api/projects/${projectId}`, messages.requestFailed, {
        method: 'DELETE',
      });
      if (selectedProjectId === projectId) {
        setSelectedProjectId(null);
      }
      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setBusyProjectId(null);
    }
  }

  async function checkServerHealth(serverId: number) {
    setBusyServerId(serverId);
    setActionError(null);

    try {
      await apiRequest<{ server_id: number; status: string }>(
        `/api/servers/${serverId}/check`,
        messages.requestFailed,
        {
          method: 'POST',
        },
      );

      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.checkServerError,
      );
      await loadProjects();
      await loadLogs();
    } finally {
      setBusyServerId(null);
    }
  }

  async function connectOAuth(serverId: number) {
    setBusyServerId(serverId);
    setActionError(null);

    try {
      const payload = await apiRequest<{ auth_url: string }>(
        `/api/servers/${serverId}/oauth-start`,
        messages.requestFailed,
        { method: 'POST' },
      );

      window.open(payload.auth_url, '_blank', 'noopener,noreferrer');
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.checkServerError,
      );
    } finally {
      setBusyServerId(null);
    }
  }

  async function disconnectOAuth(serverId: number) {
    setBusyServerId(serverId);
    setActionError(null);

    try {
      await apiRequest<{ server_id: number; status: string }>(
        `/api/servers/${serverId}/oauth-disconnect`,
        messages.requestFailed,
        { method: 'POST' },
      );
      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.checkServerError,
      );
    } finally {
      setBusyServerId(null);
    }
  }

  function openAuthModal(serverId: number) {
    setAuthServerId(serverId);
    setAuthOpen(true);
  }

  function resolveOAuthServerId(
    project: ProjectStatus,
    existingServerId: number | null,
    name: string,
    url: string,
  ) {
    if (existingServerId) {
      return existingServerId;
    }

    const matches = project.servers.filter(
      (server) =>
        server.transport === 'http_stream' &&
        server.name === name &&
        server.url === url,
    );
    if (matches.length === 0) {
      return null;
    }

    return matches.reduce((latest, server) => (server.id > latest ? server.id : latest), matches[0].id);
  }

  function startEditServer(server: ServerStatus) {
    const args = Array.isArray(server.args) && server.args.length > 0 ? server.args : [''];
    const envVars =
      Array.isArray(server.env_vars) && server.env_vars.length > 0
        ? server.env_vars
        : [{ key: '', value: '' }];
    const envPassthrough =
      Array.isArray(server.env_passthrough) && server.env_passthrough.length > 0
        ? server.env_passthrough
        : [''];
    const headers =
      Array.isArray(server.headers) && server.headers.length > 0
        ? server.headers
        : [{ key: '', value: '' }];
    const oauthScopes =
      Array.isArray(server.oauth_scopes) && server.oauth_scopes.length > 0
        ? server.oauth_scopes
        : [''];
    const headerEnvVars =
      Array.isArray(server.header_env_vars) && server.header_env_vars.length > 0
        ? server.header_env_vars
        : [{ key: '', value: '' }];

    setServerForm({
      name: server.name,
      transport: server.transport === 'http_stream' ? 'http_stream' : 'stdio',
      command: server.command,
      args,
      env_vars: envVars,
      env_passthrough: envPassthrough,
      working_dir: server.working_dir,
      url: server.url,
      bearer_token_env_var: server.bearer_token_env_var,
      headers,
      header_env_vars: headerEnvVars,
      auth_type: server.auth_type === 'oauth2' ? 'oauth2' : 'none',
      oauth_provider: server.oauth_provider,
      oauth_authorize_url: server.oauth_authorize_url,
      oauth_token_url: server.oauth_token_url,
      oauth_refresh_url: server.oauth_refresh_url,
      oauth_client_id: server.oauth_client_id,
      oauth_client_secret: server.oauth_client_secret,
      oauth_scopes: oauthScopes,
      auto_start: server.auto_start,
    });
    setOAuthAdvancedOpen(true);
    setEditingServerId(server.id);
    setAddServerOpen(true);
  }

  async function deleteServer(serverId: number) {
    const confirmed = window.confirm('Delete this MCP server?');
    if (!confirmed) {
      return;
    }

    setBusyServerId(serverId);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(`/api/servers/${serverId}`, messages.requestFailed, {
        method: 'DELETE',
      });
      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setBusyServerId(null);
    }
  }

  async function deleteRAGCollection(collectionId: string) {
    const confirmed = window.confirm(messages.deleteKnowledgeBaseConfirm);
    if (!confirmed) {
      return;
    }

    setActionError(null);
    setLinkingCollectionId(collectionId);

    try {
      await apiRequest<{ deleted: boolean }>(
        `/api/rag/collections/${collectionId}`,
        messages.requestFailed,
        {
          method: 'DELETE',
        },
      );
      await loadRAGCollections();
      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setLinkingCollectionId(null);
    }
  }

  function startEditRAGCollection(collection: RAGCollection) {
    setEditingRAGCollectionId(collection.collection_id);
    setRAGCollectionForm({ name: collection.name });
    setCreateRAGCollectionOpen(true);
  }

  async function createRAGCollection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setCreatingRAGCollection(true);
    setActionError(null);

    try {
      const savedCollection = await apiRequest<RAGCollection>(
        editingRAGCollectionId ? `/api/rag/collections/${editingRAGCollectionId}` : '/api/rag/collections',
        messages.requestFailed,
        {
          method: editingRAGCollectionId ? 'PUT' : 'POST',
          body: JSON.stringify({
            name: ragCollectionForm.name,
          }),
        },
      );

      setAllRAGCollections((current) =>
        editingRAGCollectionId
          ? current.map((collection) =>
              collection.collection_id === editingRAGCollectionId ? savedCollection : collection,
            )
          : [...current, savedCollection],
      );
      setRAGCollectionForm(emptyRAGCollectionForm);
      setEditingRAGCollectionId(null);
      setCreateRAGCollectionOpen(false);
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setCreatingRAGCollection(false);
    }
  }

  async function indexRAGCollection(collectionId: string) {
    const dirPath = (ragIndexPaths[collectionId] ?? selectedProject?.root_path ?? '').trim();
    if (!dirPath) {
      setActionError('Directory path is required for indexing.');
      return;
    }

    setIndexingCollectionId(collectionId);
    setActionError(null);

    try {
      await apiRequest<{ indexed: boolean }>(
        `/api/rag/collections/${collectionId}/index`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ dir_path: dirPath }),
        },
      );
      setAllRAGCollections((current) =>
        current.map((collection) =>
          collection.collection_id === collectionId
            ? { ...collection, source_path: dirPath }
            : collection,
        ),
      );
      setRAGIndexPaths((current) => ({
        ...current,
        [collectionId]: dirPath,
      }));
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setIndexingCollectionId(null);
    }
  }

  async function searchRAGCollection(collectionId: string) {
    const query = (ragSearchQueries[collectionId] ?? '').trim();
    if (!query) {
      setActionError(messages.searchQueryRequired);
      return;
    }

    setSearchingCollectionId(collectionId);
    setActionError(null);

    try {
      const response = await apiRequest<{ items: RAGSearchResult[] }>(
        `/api/rag/collections/${collectionId}/search`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ query, limit: 5 }),
        },
      );
      setRAGSearchResults((current) => ({
        ...current,
        [collectionId]: response.items,
      }));
      setActiveRAGSearchCollectionId(collectionId);
      setRAGSearchResultsOpen(true);
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setSearchingCollectionId(null);
    }
  }

  async function connectRAGCollectionToProject(collectionId: string) {
    if (!selectedProject) {
      return;
    }

    setLinkingCollectionId(collectionId);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}/rag-collections`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ collection_id: collectionId }),
        },
      );
      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      setConnectRAGCollectionOpen(false);
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setLinkingCollectionId(null);
    }
  }

  async function disconnectRAGCollectionFromProject(collectionId: string) {
    if (!selectedProject) {
      return;
    }

    setBusyProjectId(selectedProject.project_id);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}/rag-collections/${collectionId}`,
        messages.requestFailed,
        {
          method: 'DELETE',
        },
      );
      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setBusyProjectId(null);
    }
  }

  function updateStringListField(
    field: 'args' | 'env_passthrough',
    index: number,
    value: string,
  ) {
    setServerForm((current) => ({
      ...current,
      [field]: current[field].map((item, itemIndex) => (itemIndex === index ? value : item)),
    }));
  }

  function addStringListField(field: 'args' | 'env_passthrough') {
    setServerForm((current) => ({
      ...current,
      [field]: [...current[field], ''],
    }));
  }

  function removeStringListField(field: 'args' | 'env_passthrough', index: number) {
    setServerForm((current) => {
      const next = current[field].filter((_, itemIndex) => itemIndex !== index);
      return {
        ...current,
        [field]: next.length > 0 ? next : [''],
      };
    });
  }

  function updateKeyValueField(
    field: 'env_vars' | 'headers' | 'header_env_vars',
    index: number,
    key: 'key' | 'value',
    value: string,
  ) {
    setServerForm((current) => ({
      ...current,
      [field]: current[field].map((item, itemIndex) =>
        itemIndex === index ? { ...item, [key]: value } : item,
      ),
    }));
  }

  function addKeyValueField(field: 'env_vars' | 'headers' | 'header_env_vars') {
    setServerForm((current) => ({
      ...current,
      [field]: [...current[field], { key: '', value: '' }],
    }));
  }

  function removeKeyValueField(
    field: 'env_vars' | 'headers' | 'header_env_vars',
    index: number,
  ) {
    setServerForm((current) => {
      const next = current[field].filter((_, itemIndex) => itemIndex !== index);
      return {
        ...current,
        [field]: next.length > 0 ? next : [{ key: '', value: '' }],
      };
    });
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="flex min-h-screen w-full">
        <aside className="sticky top-0 flex h-screen w-20 shrink-0 flex-col items-center border-r border-border bg-sidebar/55 px-3 py-6">
          <div className="flex h-full flex-col items-center gap-3">
            <div className="mb-2 flex h-12 w-12 items-center justify-center">
              <img src={logo} alt={labels.appTitle} className="max-h-full w-auto object-contain" />
            </div>
            {navigationItems.map((item) => {
              const Icon = item.icon;
              const isActive = view === item.id;

              return (
                <Tooltip key={item.id}>
                  <TooltipTrigger asChild>
                    <button
                      onClick={() => setView(item.id)}
                      aria-label={item.label}
                      className={`inline-flex h-12 w-12 items-center justify-center rounded-2xl border transition-colors ${
                        isActive
                          ? 'border-electric-blue/40 bg-electric-blue text-white shadow-[0_12px_30px_rgba(38,132,255,0.22)]'
                          : 'border-transparent bg-card text-muted-foreground hover:border-border hover:bg-accent hover:text-foreground'
                      }`}
                    >
                      <Icon className="h-5 w-5" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="right" sideOffset={10}>
                    {item.label}
                  </TooltipContent>
                </Tooltip>
              );
            })}
          </div>
        </aside>

        {view === 'projects' ? (
        <aside className="w-full max-w-sm border-r border-border bg-sidebar/40">
          <div className="border-b border-border px-6 py-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">{labels.appTitle}</p>
              </div>
              <div className="flex items-center gap-2">
                <Select value={language} onValueChange={(value) => setLanguage(value as Language)}>
                  <SelectTrigger
                    className="h-10 w-[150px] rounded-md border-border bg-card text-xs"
                    aria-label={labels.language}
                  >
                    <SelectValue placeholder={labels.language} />
                  </SelectTrigger>
                  <SelectContent>
                    {languageOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <button
                  onClick={() => void Promise.all([loadProjects(), loadOllamaStatus()])}
                  className="rounded-md border border-border bg-card p-2 transition-colors hover:bg-accent"
                  aria-label="Refresh projects"
                >
                  <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
                </button>
              </div>
            </div>
            <p className="mt-3 text-sm text-muted-foreground">
              {labels.appDescription}
            </p>
          </div>

          <div className="space-y-6 p-6">
            <section className="space-y-3">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-medium text-muted-foreground">{labels.projects}</h2>
                <div className="flex items-center gap-2">
                  <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
                    {projects.length}
                  </span>
                  <Dialog
                    open={createProjectOpen}
                    onOpenChange={(open) => {
                      setCreateProjectOpen(open);
                      if (!open) {
                        setEditingProjectId(null);
                        setProjectForm(emptyProjectForm);
                      }
                    }}
                  >
                    <DialogTrigger asChild>
                      <button className="inline-flex h-8 items-center justify-center gap-2 rounded-md bg-electric-blue px-3 text-xs font-medium text-white transition-colors hover:bg-electric-blue/90">
                        <Plus className="h-3.5 w-3.5" />
                        {labels.createProject}
                      </button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                          <DialogTitle>{editingProjectId ? 'Edit Project' : labels.createProject}</DialogTitle>
                        <DialogDescription>{messages.projectHelper}</DialogDescription>
                      </DialogHeader>

                      <form className="space-y-3" onSubmit={createProject}>
                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{labels.name}</span>
                          <input
                            required
                            value={projectForm.name}
                            onChange={(event) =>
                              setProjectForm((current) => ({ ...current, name: event.target.value }))
                            }
                            className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={messages.projectNamePlaceholder}
                          />
                        </label>

                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{labels.description}</span>
                          <textarea
                            value={projectForm.description}
                            onChange={(event) =>
                              setProjectForm((current) => ({
                                ...current,
                                description: event.target.value,
                              }))
                            }
                            rows={3}
                            className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={messages.projectDescriptionPlaceholder}
                          />
                        </label>

                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{labels.workingDirectory}</span>
                          <input
                            value={projectForm.root_path}
                            onChange={(event) =>
                              setProjectForm((current) => ({
                                ...current,
                                root_path: event.target.value,
                              }))
                            }
                            className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={messages.workingDirectoryPlaceholder}
                          />
                        </label>

                        <button
                          type="submit"
                          disabled={creatingProject}
                          className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                              {creatingProject ? (
                                <LoaderCircle className="h-4 w-4 animate-spin" />
                              ) : (
                                <Plus className="h-4 w-4" />
                              )}
                              {editingProjectId ? 'Save Project' : labels.createProject}
                            </button>
                          </form>
                        </DialogContent>
                  </Dialog>
                </div>
              </div>

              {loading ? (
                <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-4 py-5 text-sm text-muted-foreground">
                  <LoaderCircle className="h-4 w-4 animate-spin" />
                  {messages.loadingProjects}
                </div>
              ) : projects.length === 0 ? (
                <div className="rounded-lg border border-dashed border-border bg-card px-4 py-5 text-sm text-muted-foreground">
                  {messages.noProjects}
                </div>
              ) : (
                <div className="space-y-2">
                  {projects.map((project) => {
                    const runningCount = project.servers.filter(
                      (server) => server.status === 'Running',
                    ).length;

                    return (
                      <button
                        key={project.project_id}
                        onClick={() => setSelectedProjectId(project.project_id)}
                        className={`w-full rounded-xl border p-4 text-left transition-colors ${
                          selectedProjectId === project.project_id
                            ? 'border-electric-blue bg-electric-blue/8'
                            : 'border-border bg-card hover:bg-accent/60'
                        }`}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <div className="font-medium">{project.name}</div>
                            <div className="mt-1 text-sm text-muted-foreground">
                              {project.description || messages.workspaceGroupFallback}
                            </div>
                          </div>
                          {project.connection_ready ? (
                            <Radio className="mt-0.5 h-4 w-4 text-status-running" />
                          ) : (
                            <AlertCircle className="mt-0.5 h-4 w-4 text-amber-500" />
                          )}
                        </div>
                        <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
                          <span>{messages.serverCount(project.servers.length)}</span>
                          <span>•</span>
                          <span>{messages.runningCount(runningCount)}</span>
                          {project.is_paused ? (
                            <>
                              <span>•</span>
                              <span className="text-amber-600">{labels.paused}</span>
                            </>
                          ) : null}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </section>

          </div>
        </aside>
        ) : null}

        <main className="flex-1 p-6 md:p-8">
          {error ? (
            <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
              {error}
            </div>
          ) : null}

          {actionError ? (
            <div className="mb-6 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
              {actionError}
            </div>
          ) : null}

          {view === 'logs' ? (
            <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
              <div className="rounded-2xl border border-border bg-card p-6">
                <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
                <div>
                  <h2 className="text-2xl font-semibold">{labels.auditLogs}</h2>
                </div>
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
                    <div className="min-w-[220px]">
                      <Select
                        value={selectedLogsProjectId === null ? 'all' : String(selectedLogsProjectId)}
                        onValueChange={(value) =>
                          setSelectedLogsProjectId(value === 'all' ? null : Number(value))
                        }
                      >
                        <SelectTrigger className="h-10 rounded-md border-border bg-background text-sm">
                          <SelectValue placeholder={labels.filterByProject} />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">{labels.allProjects}</SelectItem>
                          {projects.map((project) => (
                            <SelectItem key={`logs-project-${project.project_id}`} value={String(project.project_id)}>
                              {project.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <button
                      onClick={() => void loadLogs()}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                    >
                      <RefreshCw className={`h-4 w-4 ${logsLoading ? 'animate-spin' : ''}`} />
                      {labels.refresh}
                    </button>
                  </div>
                </div>

                <div className="mb-4 flex items-center justify-between gap-3 rounded-xl border border-border bg-background px-4 py-3 text-sm">
                  <div>
                    <div className="font-medium">{labels.consoleFeed}</div>
                    <div className="text-muted-foreground">
                      {filteredLogsProject?.name ?? labels.allProjects}
                    </div>
                  </div>
                  <div className="rounded-full border border-border bg-card px-3 py-1 text-xs text-muted-foreground">
                    {logs.length} {labels.requests}
                  </div>
                </div>

                {logsLoading ? (
                  <div className="flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                    {messages.loadingLogs}
                  </div>
                ) : logs.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-center text-muted-foreground">
                    {messages.noLogs}
                  </div>
                ) : (
                  <div className="overflow-hidden rounded-xl border border-border bg-[#0b0f14]">
                    <div ref={logsViewportRef} className="max-h-[70vh] overflow-y-auto">
                      {logs.map((entry) => (
                        <div
                          key={entry.id}
                          className="border-b border-white/5 px-4 py-3 font-mono text-xs text-slate-200 last:border-b-0"
                        >
                          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                            <span className="text-slate-500">
                              {new Date(entry.created_at).toLocaleTimeString()}
                            </span>
                            <span className="text-electric-blue">{formatAuditAction(entry.action)}</span>
                            <span className="text-slate-400">
                              {entry.actor || labels.unknownActor}
                            </span>
                            {entry.project_id ? (
                              <span className="text-emerald-400">
                                {projectNameFromLog(entry.project_id)}
                              </span>
                            ) : null}
                            {entry.server_id ? (
                              <span className="text-amber-300">
                                {serverNameFromLog(entry.server_id)}
                              </span>
                            ) : null}
                          </div>
                          {entry.detail ? (
                            <div className="mt-1 break-words text-slate-300/85">
                              {formatAuditDetail(entry)}
                            </div>
                          ) : null}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              <aside className="space-y-4">
                <div className="rounded-2xl border border-border bg-card p-5">
                  <h3 className="text-lg font-semibold">{labels.activityOverview}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {messages.popularityDescription}
                  </p>
                </div>

                <div className="rounded-2xl border border-border bg-card p-5">
                  <div className="text-sm text-muted-foreground">{labels.hottestProject}</div>
                  <div className="mt-2 text-lg font-semibold">
                    {projectActivity[0]?.name ?? labels.noActivity}
                  </div>
                  <div className="mt-1 text-sm text-electric-blue">
                    {projectActivity[0]?.count ?? 0} {labels.requests}
                  </div>
                </div>

                <div className="rounded-2xl border border-border bg-card p-5">
                  <div className="text-sm text-muted-foreground">{labels.hottestServer}</div>
                  <div className="mt-2 text-lg font-semibold">
                    {serverActivity[0]?.name ?? labels.noActivity}
                  </div>
                  <div className="mt-1 text-sm text-electric-blue">
                    {serverActivity[0]?.count ?? 0} {labels.requests}
                  </div>
                </div>

                <div className="rounded-2xl border border-border bg-card p-5">
                  <div className="text-sm font-medium">{labels.projects}</div>
                  <div className="mt-3 space-y-2">
                    {projectActivity.slice(0, 5).map((entry) => (
                      <div
                        key={`project-activity-${entry.id}`}
                        className="flex items-center justify-between gap-3 rounded-lg bg-background px-3 py-2 text-sm"
                      >
                        <span className="truncate">{entry.name}</span>
                        <span className="text-muted-foreground">{entry.count}</span>
                      </div>
                    ))}
                    {projectActivity.length === 0 ? (
                      <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                        {labels.noActivity}
                      </div>
                    ) : null}
                  </div>
                </div>

                <div className="rounded-2xl border border-border bg-card p-5">
                  <div className="text-sm font-medium">{labels.servers}</div>
                  <div className="mt-3 space-y-2">
                    {serverActivity.slice(0, 5).map((entry) => (
                      <div
                        key={`server-activity-${entry.id}`}
                        className="flex items-center justify-between gap-3 rounded-lg bg-background px-3 py-2 text-sm"
                      >
                        <span className="truncate">{entry.name}</span>
                        <span className="text-muted-foreground">{entry.count}</span>
                      </div>
                    ))}
                    {serverActivity.length === 0 ? (
                      <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                        {labels.noActivity}
                      </div>
                    ) : null}
                  </div>
                </div>
              </aside>
            </section>
          ) : view === 'knowledge' ? (
            <section className="space-y-6">
              <div className="rounded-2xl border border-border bg-card p-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div>
                    <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
                      {labels.knowledgeBase}
                    </p>
                    <h2 className="mt-2 text-3xl font-semibold">{messages.knowledgeBaseHeroTitle}</h2>
                    <p className="mt-2 max-w-3xl text-muted-foreground">
                      {messages.knowledgeBaseHeroDescription}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="rounded-xl border border-border bg-background px-4 py-3">
                      <div className="text-sm text-muted-foreground">{labels.collections}</div>
                      <div className="mt-1 text-2xl font-semibold">{allRAGCollections.length}</div>
                    </div>
                    <Dialog
                      open={createRAGCollectionOpen}
                      onOpenChange={(open) => {
                        setCreateRAGCollectionOpen(open);
                        if (!open) {
                          setRAGCollectionForm(emptyRAGCollectionForm);
                          setEditingRAGCollectionId(null);
                        }
                      }}
                    >
                      <DialogTrigger asChild>
                        <button className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
                          <Plus className="h-4 w-4" />
                          {labels.createCollection}
                        </button>
                      </DialogTrigger>
                      <DialogContent className="sm:max-w-xl">
                        <DialogHeader>
                          <DialogTitle>{editingRAGCollectionId ? 'Edit Knowledge Base' : messages.createKnowledgeBaseTitle}</DialogTitle>
                          <DialogDescription>
                            {editingRAGCollectionId ? 'Update the collection name. The indexed files and search data will stay intact.' : messages.createKnowledgeBaseDescription}
                          </DialogDescription>
                        </DialogHeader>
                        <form className="space-y-4" onSubmit={createRAGCollection}>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{labels.name}</span>
                            <input
                              required
                              value={ragCollectionForm.name}
                              onChange={(event) =>
                                setRAGCollectionForm((current) => ({ ...current, name: event.target.value }))
                              }
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                              placeholder={messages.collectionNamePlaceholder}
                            />
                          </label>
                          <DialogFooter>
                            <button
                              type="submit"
                              disabled={creatingRAGCollection}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              {creatingRAGCollection ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                              {editingRAGCollectionId ? 'Save Knowledge Base' : labels.create}
                            </button>
                          </DialogFooter>
                        </form>
                      </DialogContent>
                    </Dialog>
                  </div>
                </div>
              </div>

              <section className="rounded-2xl border border-border bg-card p-6">
                {allRAGCollections.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
                    {messages.noKnowledgeBasesCreated}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {allRAGCollections.map((collection) => (
                      <div key={collection.collection_id} className="rounded-xl border border-border bg-background p-4">
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                          <div className="min-w-0">
                            <div className="flex items-center gap-2">
                              <Database className="h-4 w-4 text-electric-blue" />
                              <div className="font-medium">{collection.name}</div>
                            </div>
                            <div className="mt-2 flex flex-wrap items-center gap-2">
                              <span className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                                {messages.supportedFormatsLabel}: {messages.supportedFormatsValue}
                              </span>
                            </div>
                            <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                              {collection.collection_id} · {collection.data_type}
                            </div>
                            <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">
                              {collection.index_path}
                            </code>
                          </div>
                          <div className="flex flex-wrap items-center gap-2">
                            <button
                              onClick={() => startEditRAGCollection(collection)}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                            >
                              <Pencil className="h-4 w-4" />
                              Edit
                            </button>
                            <button
                              onClick={() => void deleteRAGCollection(collection.collection_id)}
                              disabled={linkingCollectionId === collection.collection_id}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              <Trash2 className="h-4 w-4" />
                              {labels.delete}
                            </button>
                          </div>
                        </div>

                        <div className="mt-4 grid gap-4 xl:grid-cols-2">
                          <div className="rounded-xl border border-border bg-card p-4">
                            <div className="text-sm font-medium">{messages.indexFolderTitle}</div>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {messages.indexFolderDescription}
                            </p>
                            <input
                              value={ragIndexPaths[collection.collection_id] ?? ''}
                              onChange={(event) =>
                                setRAGIndexPaths((current) => ({
                                  ...current,
                                  [collection.collection_id]: event.target.value,
                                }))
                              }
                              className="mt-3 h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                              placeholder={messages.indexFolderPlaceholder}
                            />
                            <button
                              onClick={() => void indexRAGCollection(collection.collection_id)}
                              disabled={indexingCollectionId === collection.collection_id}
                              className="mt-3 inline-flex h-10 items-center justify-center gap-2 rounded-md bg-foreground px-4 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              {indexingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                              {labels.index}
                            </button>
                          </div>

                          <div className="rounded-xl border border-border bg-card p-4">
                            <div className="text-sm font-medium">{messages.searchCollectionTitle}</div>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {messages.searchCollectionDescription}
                            </p>
                            <div className="mt-3 flex gap-3">
                              <input
                                value={ragSearchQueries[collection.collection_id] ?? ''}
                                onChange={(event) =>
                                  setRAGSearchQueries((current) => ({
                                    ...current,
                                    [collection.collection_id]: event.target.value,
                                  }))
                                }
                                className="h-10 min-w-0 flex-1 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                placeholder={messages.searchCollectionPlaceholder}
                              />
                              <button
                                onClick={() => void searchRAGCollection(collection.collection_id)}
                                disabled={searchingCollectionId === collection.collection_id}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                              >
                                {searchingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <TextSearch className="h-4 w-4" />}
                                {labels.search}
                              </button>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              <Dialog
                open={ragSearchResultsOpen}
                onOpenChange={(open) => {
                  setRAGSearchResultsOpen(open);
                  if (!open) {
                    setActiveRAGSearchCollectionId(null);
                  }
                }}
              >
                <DialogContent className="sm:max-w-4xl">
                  <DialogHeader>
                    <DialogTitle>{messages.searchResultsTitle}</DialogTitle>
                    <DialogDescription>
                      {messages.searchResultsDescription(activeRAGSearchCollection?.name ?? labels.knowledgeBase)}
                    </DialogDescription>
                  </DialogHeader>

                  {activeRAGSearchCollection ? (
                    <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
                      <div className="font-medium">{activeRAGSearchCollection.name}</div>
                      <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                        {activeRAGSearchCollection.collection_id} · {activeRAGSearchCollection.data_type}
                      </div>
                    </div>
                  ) : null}

                  {activeRAGSearchResults.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-sm text-muted-foreground">
                      {messages.searchResultsEmpty}
                    </div>
                  ) : (
                    <div className="max-h-[65vh] space-y-3 overflow-y-auto pr-1">
                      {activeRAGSearchResults.map((item) => (
                        <div key={item.id} className="rounded-lg border border-border bg-background p-3">
                          <code className="block overflow-x-auto text-xs text-electric-blue">{item.file_path}</code>
                          {item.section ? (
                            <div className="mt-2 inline-flex rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                              {item.section}
                            </div>
                          ) : null}
                          <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-foreground/85">{item.content}</pre>
                        </div>
                      ))}
                    </div>
                  )}
                </DialogContent>
              </Dialog>
            </section>
          ) : view === 'market' ? (
            <section className="grid gap-6 xl:grid-cols-[260px_minmax(0,1fr)] xl:items-start">
              <aside className="rounded-2xl border border-border bg-card p-4 xl:sticky xl:top-8 xl:h-[calc(100vh-4rem)] xl:overflow-hidden">
                <div className="flex h-full flex-col gap-4">
                  <div>
                    <div className="text-sm uppercase tracking-[0.24em] text-electric-blue">
                      {labels.catalog}
                    </div>
                    <div className="mt-2 text-lg font-semibold">{labels.integrations}</div>
                  </div>

                  <div className="space-y-2 overflow-y-auto pr-1 xl:flex-1">
                    {catalogCategories.map((category) => (
                      <button
                        key={`catalog-category-${category}`}
                        onClick={() => setSelectedCatalogCategory(category)}
                        className={`flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-xs font-medium transition-colors ${
                          selectedCatalogCategory === category
                            ? 'border-electric-blue bg-electric-blue text-white'
                            : 'border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground'
                        }`}
                      >
                        <span>{category === 'all' ? labels.allCategories : category}</span>
                      </button>
                    ))}
                  </div>

                  <div className="mt-auto pt-2">
                    <button
                      onClick={() => void syncCatalog()}
                      disabled={catalogSyncing}
                      className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {catalogSyncing ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : (
                        <RefreshCw className="h-4 w-4" />
                      )}
                      {labels.syncCatalog}
                    </button>
                  </div>
                </div>
              </aside>

              <div className="space-y-6">
              <div className="rounded-2xl border border-border bg-card p-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
                      {labels.integrations}
                    </p>
                    <h2 className="mt-2 text-3xl font-semibold">{labels.market} / {labels.catalog}</h2>
                    <p className="mt-2 max-w-3xl text-muted-foreground">
                      {messages.marketDescription}
                    </p>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-3">
                    <div className="rounded-xl border border-border bg-background px-4 py-3">
                      <div className="text-sm text-muted-foreground">{labels.catalogItems}</div>
                      <div className="mt-1 text-2xl font-semibold">{catalogItems.length}</div>
                    </div>
                    <div className="rounded-xl border border-border bg-background px-4 py-3">
                      <div className="text-sm text-muted-foreground">{labels.installed}</div>
                      <div className="mt-1 text-2xl font-semibold">
                        {selectedProject?.installed_integrations.length ?? 0}
                      </div>
                    </div>
                    <div className="rounded-xl border border-border bg-background px-4 py-3">
                      <div className="text-sm text-muted-foreground">{labels.lastSync}</div>
                      <div className="mt-1 text-sm font-medium">
                        {catalogSettings?.last_sync_at
                          ? new Date(catalogSettings.last_sync_at).toLocaleString()
                          : messages.notSynced}
                      </div>
                    </div>
                  </div>
                </div>

                <div className={`mt-6 grid gap-4 ${catalogURLVisible ? 'lg:grid-cols-[minmax(0,1fr)]' : 'lg:grid-cols-[auto]'}`}>
                  {catalogURLVisible ? (
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{labels.externalManifestUrl}</span>
                      <input
                        value={catalogURL}
                        onChange={(event) => setCatalogURL(event.target.value)}
                        className="h-11 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder="https://webeasy.kz/mcpbox/catalog.json"
                      />
                    </label>
                  ) : null}
                </div>

                {catalogURLVisible ? (
                  <div className="mt-3 text-xs text-muted-foreground">
                    {messages.advancedModeEnabled}
                  </div>
                ) : null}

                {catalogSettings?.last_sync_status === 'failed' && catalogSettings.last_sync_error ? (
                  <div className="mt-4 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                    {catalogSettings.last_sync_error}
                  </div>
                ) : null}
              </div>

              {!selectedProject ? (
                <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
                  {messages.selectProjectBeforeInstall}
                </div>
              ) : null}

              <div className="grid gap-4">
                {filteredCatalogItems.map((item) => {
                  const packageInstalling = installingCatalogItemId === item.id;
                  const addingToProject = addingCatalogItemId === item.id;
                  const packageInstalled = installedPackageCatalogIDs.has(item.id);
                  const addedToProject = installedCatalogIDs.has(item.id);
                  const transportLabel = item.transport === 'stdio' ? 'STDIO' : item.transport;
                  const primaryInfoLabel = item.transport === 'stdio' ? labels.command : labels.endpoint;
                  const primaryInfoValue = item.transport === 'stdio'
                    ? [item.command, ...(item.args ?? [])].filter(Boolean).join(' ')
                    : item.mcp_url || messages.noValue;
                  const authLabel = item.auth_type === 'mcp_discovery' ? labels.mcpDiscovery : item.auth_type;

                  return (
                    <div key={item.id} className="rounded-2xl border border-border bg-card p-5">
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <h3 className="text-lg font-semibold">{item.name}</h3>
                            <div className="flex flex-wrap justify-end gap-2">
                              <button
                                onClick={() => void installCatalogPackage(item)}
                                disabled={packageInstalling || packageInstalled || !item.enabled}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-electric-blue bg-electric-blue/10 px-4 text-sm font-medium text-electric-blue transition-colors hover:bg-electric-blue/20 disabled:cursor-not-allowed disabled:opacity-60"
                              >
                                {packageInstalling ? (
                                  <LoaderCircle className="h-4 w-4 animate-spin" />
                                ) : (
                                  <Plus className="h-4 w-4" />
                                )}
                                {packageInstalled ? labels.packageInstalled : labels.installPackage}
                              </button>
                              <button
                                onClick={() => void installCatalogItem(item)}
                                disabled={projects.length === 0 || addingToProject || !packageInstalled || addedToProject || !item.enabled}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-60"
                              >
                                {addingToProject ? (
                                  <LoaderCircle className="h-4 w-4 animate-spin" />
                                ) : (
                                  <Plus className="h-4 w-4" />
                                )}
                                {addedToProject ? labels.addedToProject : labels.addToProject}
                              </button>
                            </div>
                          </div>
                          <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                {transportLabel}
                              </span>
                              <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                {authLabel}
                              </span>
                              <span className={`rounded-full border px-2 py-1 text-xs ${packageInstalled ? 'border-status-running/30 bg-status-running/12 text-status-running' : 'border-border bg-muted text-muted-foreground'}`}>
                                {packageInstalled ? labels.packageInstalled : labels.packageNotInstalled}
                              </span>
                              {selectedProject ? (
                                <span className={`rounded-full border px-2 py-1 text-xs ${addedToProject ? 'border-electric-blue/30 bg-electric-blue/12 text-electric-blue' : 'border-border bg-muted text-muted-foreground'}`}>
                                  {addedToProject ? labels.addedToProject : labels.notInProject}
                                </span>
                              ) : null}
                            </div>
                            <div className="flex flex-wrap items-center justify-end gap-3 text-sm">
                              {item.docs_url ? (
                                <a
                                  className="text-electric-blue underline-offset-4 hover:underline"
                                  href={item.docs_url}
                                  target="_blank"
                                  rel="noreferrer"
                                >
                                  {labels.docs}
                                </a>
                              ) : null}
                              {item.website ? (
                                <a
                                  className="text-electric-blue underline-offset-4 hover:underline"
                                  href={item.website}
                                  target="_blank"
                                  rel="noreferrer"
                                >
                                  {labels.website}
                                </a>
                              ) : null}
                            </div>
                          </div>
                          <div className="mt-1 text-sm text-electric-blue">{item.category || labels.generalCategory}</div>
                        </div>
                      </div>

                      <p className="mt-4 text-sm text-muted-foreground">
                        {item.description || messages.noDescriptionProvided}
                      </p>
                      {item.auth_type === 'mcp_discovery' ? (
                        <div className="mt-4 rounded-xl border border-electric-blue/20 bg-electric-blue/8 px-4 py-3 text-sm text-muted-foreground">
                          {messages.upstreamAuthNotice}
                        </div>
                      ) : null}

                      <div className="mt-4 rounded-xl border border-border bg-background p-3">
                        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{primaryInfoLabel}</div>
                        <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">
                          {primaryInfoValue || messages.noValue}
                        </code>
                        {item.transport === 'stdio' && item.working_dir ? (
                          <div className="mt-3 text-sm text-muted-foreground">
                            {messages.workingDirectoryValue(item.working_dir)}
                          </div>
                        ) : null}
                        {item.transport === 'stdio' && item.default_auto_start ? (
                          <div className="mt-2 text-sm text-muted-foreground">
                            {messages.autoStartAfterInstall}
                          </div>
                        ) : null}
                      </div>

                      {(item.tags?.length ?? 0) > 0 ? (
                        <div className="mt-4 flex flex-wrap gap-2">
                          {item.tags.map((tag) => (
                            <span
                              key={`${item.id}-${tag}`}
                              className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  );
                })}
              </div>

              {!catalogLoading && catalogItems.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
                  {messages.syncManifestToPopulateCatalog}
                </div>
              ) : null}
              {!catalogLoading && catalogItems.length > 0 && filteredCatalogItems.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
                  {messages.noIntegrationsInCategory}
                </div>
              ) : null}

              <Dialog
                open={installDialogOpen}
                onOpenChange={(open) => {
                  setInstallDialogOpen(open);
                  if (!open) {
                    setInstallDialogItem(null);
                    setInstallDialogProjectId(null);
                    setInstallDialogValues({});
                  }
                }}
              >
                <DialogContent className="sm:max-w-xl">
                  <DialogHeader>
                    <DialogTitle>
                      {installDialogItem
                        ? messages.addPackageDialogTitle(installDialogItem.name)
                        : messages.addPackageDialogFallbackTitle}
                    </DialogTitle>
                    <DialogDescription>
                      {messages.addPackageDialogDescription}
                    </DialogDescription>
                  </DialogHeader>

                  {installDialogItem ? (
                    <form
                      className="space-y-4"
                      onSubmit={(event) => {
                        event.preventDefault();
                        if (!installDialogProjectId) {
                          setActionError(messages.selectProjectBeforeInstall);
                          return;
                        }
                        const config = normalizeInstallConfig(installDialogFields, installDialogValues);
                        void performCatalogInstall(installDialogItem, installDialogProjectId, config).then((success) => {
                          if (!success) {
                            return;
                          }
                          setInstallDialogOpen(false);
                          setInstallDialogItem(null);
                          setInstallDialogProjectId(null);
                          setInstallDialogValues({});
                        });
                      }}
                    >
                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{labels.projects}</span>
                        <Select
                          value={installDialogProjectId ? String(installDialogProjectId) : undefined}
                          onValueChange={(value) => {
                            const nextProjectId = Number(value);
                            const nextProject =
                              projectOptions.find((project) => project.project_id === nextProjectId) ?? null;
                            setInstallDialogProjectId(nextProjectId);
                            setInstallDialogValues((current) =>
                              applyProjectDefaultsToInstallValues(current, installDialogFields, nextProject),
                            );
                          }}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue placeholder={labels.notSelected} />
                          </SelectTrigger>
                          <SelectContent>
                            {projectOptions.map((project) => (
                              <SelectItem key={`install-project-${project.project_id}`} value={String(project.project_id)}>
                                {project.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        {installDialogProject?.root_path ? (
                          <div className="text-xs text-muted-foreground">
                            {messages.workingDirectoryValue(installDialogProject.root_path)}
                          </div>
                        ) : null}
                      </label>

                      {installDialogFields.map((field) => (
                        <label key={`install-field-${field.key}`} className="block space-y-2">
                          <span className="text-sm text-muted-foreground">
                            {field.label}
                            {field.required ? ' *' : ''}
                          </span>
                          {field.type === 'array' ? (
                            <textarea
                              value={installDialogValues[field.key] ?? ''}
                              onChange={(event) =>
                                setInstallDialogValues((current) => ({
                                  ...current,
                                  [field.key]: event.target.value,
                                }))
                              }
                              rows={4}
                              className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue"
                              placeholder={messages.oneValuePerLine}
                              required={field.required}
                            />
                          ) : (
                            <input
                              type={field.secret ? 'password' : 'text'}
                              value={installDialogValues[field.key] ?? ''}
                              onChange={(event) =>
                                setInstallDialogValues((current) => ({
                                  ...current,
                                  [field.key]: event.target.value,
                                }))
                              }
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                              required={field.required}
                            />
                          )}
                        </label>
                      ))}

                      <button
                        type="submit"
                        disabled={!installDialogItem || !installDialogProjectId || addingCatalogItemId === installDialogItem.id}
                        className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {installDialogItem && addingCatalogItemId === installDialogItem.id ? (
                          <LoaderCircle className="h-4 w-4 animate-spin" />
                        ) : (
                          <Plus className="h-4 w-4" />
                        )}
                        {labels.addToProject}
                      </button>
                    </form>
                  ) : null}
                </DialogContent>
              </Dialog>
              </div>
            </section>
          ) : !selectedProject ? (
            <div className="flex min-h-[60vh] items-center justify-center rounded-2xl border border-dashed border-border bg-card/50">
              <div className="max-w-md text-center">
                <Server className="mx-auto h-12 w-12 text-electric-blue" />
                <h2 className="mt-4 text-2xl font-semibold">{labels.noProjectSelected}</h2>
                <p className="mt-2 text-muted-foreground">
                  {messages.emptySelectionBody}
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              <section className="rounded-2xl border border-border bg-card p-6">
                <div className="space-y-4">
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
                        {labels.projectOverview}
                      </p>
                      <h2 className="mt-2 text-3xl font-semibold">{selectedProject.name}</h2>
                      <p className="mt-2 max-w-2xl text-muted-foreground">
                        {selectedProject.description || messages.overviewFallbackDescription}
                      </p>
                    </div>

                    <div className="grid gap-3 sm:grid-cols-4">
                      <div className="rounded-xl border border-border bg-background px-4 py-3">
                        <div className="text-sm text-muted-foreground">{labels.servers}</div>
                        <div className="mt-1 text-2xl font-semibold">
                          {selectedProject.servers.length}
                        </div>
                      </div>
                      <div className="rounded-xl border border-border bg-background px-4 py-3">
                        <div className="text-sm text-muted-foreground">{labels.running}</div>
                        <div className="mt-1 text-2xl font-semibold">
                          {
                            selectedProject.servers.filter((server) => server.status === 'Running')
                              .length
                          }
                        </div>
                      </div>
                      <div className="rounded-xl border border-border bg-background px-4 py-3">
                        <div className="text-sm text-muted-foreground">{labels.healthy}</div>
                        <div className="mt-1 text-2xl font-semibold">
                          {selectedProjectHealthyCount}
                        </div>
                      </div>
                      {/*<div className="rounded-xl border border-border bg-background px-4 py-3">*/}
                      {/*  <div className="text-sm text-muted-foreground">{labels.oauthConnected}</div>*/}
                      {/*  <div className="mt-1 text-2xl font-semibold">*/}
                      {/*    {selectedProjectOAuthConnectedCount}*/}
                      {/*  </div>*/}
                      {/*</div>*/}
                      <div className="rounded-xl border border-border bg-background px-4 py-3">
                        <div className="text-sm text-muted-foreground">{labels.connectedKnowledgeBases}</div>
                        <div className="mt-1 text-2xl font-semibold">
                          {selectedProject.rag_collections.length}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-wrap justify-end gap-3">
                    {shouldShowOllamaControls ? (
                      <>
                        <div className="min-w-[220px]">
                          <Select
                            value={selectedOllamaModel || undefined}
                            onValueChange={setSelectedOllamaModel}
                          >
                            <SelectTrigger className="h-11 rounded-xl border-border bg-background">
                              <SelectValue placeholder={labels.ollamaModel} />
                            </SelectTrigger>
                            <SelectContent>
                              {ollamaStatus?.models.map((model) => (
                                <SelectItem key={`ollama-model-${model}`} value={model}>
                                  {model}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                        <button
                          onClick={() => void launchProjectOllama(selectedProject.project_id)}
                          disabled={
                            launchingOllamaProjectId === selectedProject.project_id ||
                            !selectedProject.connection_ready ||
                            selectedProject.is_paused ||
                            !canLaunchOllama
                          }
                          className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {launchingOllamaProjectId === selectedProject.project_id ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <OllamaIcon className="h-5 w-5" />
                          )}
                          {labels.launchOllama}
                        </button>
                      </>
                    ) : null}
                    <button
                      onClick={() =>
                        void setProjectPaused(selectedProject.project_id, !selectedProject.is_paused)
                      }
                      disabled={busyProjectId === selectedProject.project_id}
                      className={`inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-70 ${
                        selectedProject.is_paused
                          ? 'bg-status-running text-white hover:bg-status-running/90'
                          : 'bg-amber-500 text-white hover:bg-amber-500/90'
                      }`}
                    >
                      {busyProjectId === selectedProject.project_id ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : selectedProject.is_paused ? (
                        <Play className="h-4 w-4" />
                      ) : (
                        <Pause className="h-4 w-4" />
                      )}
                      {selectedProject.is_paused ? labels.resumeProject : labels.pauseProject}
                    </button>
                    <button
                      onClick={startEditProject}
                      className="inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                    >
                      <Pencil className="h-4 w-4" />
                      Edit
                    </button>
                    <button
                      onClick={() => void deleteProject(selectedProject.project_id)}
                      disabled={busyProjectId === selectedProject.project_id}
                      className="inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      <Trash2 className="h-4 w-4" />
                      Delete
                    </button>
                  </div>
                  {shouldShowOllamaControls && (ollamaStatus?.models.length ?? 0) === 0 ? (
                    <p className="text-sm text-muted-foreground">{messages.noOllamaModels}</p>
                  ) : null}
                </div>
              </section>

              <section>
                <div className="rounded-2xl border border-border bg-card p-6">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <h3 className="text-lg font-semibold">{labels.connectionEndpoint}</h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                      {messages.connectionDescription}
                    </p>
                  </div>
                    <div
                      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium ${
                        selectedProject.connection_ready
                          ? 'border-status-running/30 bg-status-running/12 text-status-running'
                          : 'border-amber-500/30 bg-amber-500/10 text-amber-600'
                      }`}
                    >
                      {selectedProject.connection_ready ? (
                        <CheckCircle2 className="h-3.5 w-3.5" />
                      ) : (
                        <AlertCircle className="h-3.5 w-3.5" />
                      )}
                      {selectedProject.connection_ready ? labels.ready : labels.notSelected}
                    </div>
                  </div>

                  <div className="mt-5 flex flex-col gap-3 rounded-xl border border-border bg-background p-4 sm:flex-row sm:items-center">
                    <code className="flex-1 overflow-x-auto text-sm text-electric-blue">
                      {selectedProject.connect_url}
                    </code>
                    <button
                      onClick={() => void copyConnectURL()}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                    >
                      {copied ? <CheckCircle2 className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                      {copied ? labels.copied : labels.copyUrl}
                    </button>
                  </div>

                  {alternativeConnectURLs.length > 0 ? (
                    <div className="mt-3 rounded-xl border border-border bg-background">
                      <button
                        type="button"
                        onClick={() => setConnectionURLsExpanded((current) => !current)}
                        className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left text-sm font-medium transition-colors hover:bg-accent"
                      >
                        <span>{messages.otherConnectionOptions}</span>
                        <span className="inline-flex items-center gap-2 text-muted-foreground">
                          {connectionURLsExpanded ? labels.hide : labels.showMore}
                          {connectionURLsExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                        </span>
                      </button>
                      {connectionURLsExpanded ? (
                        <div className="border-t border-border px-4 py-4">
                          <p className="text-sm text-muted-foreground">
                            {messages.otherConnectionOptionsDescription}
                          </p>
                          <div className="mt-3 space-y-2">
                            {alternativeConnectURLs.map((url) => (
                              <code
                                key={url}
                                className="block overflow-x-auto rounded-lg border border-border bg-card px-3 py-2 text-xs text-electric-blue"
                              >
                                {url}
                              </code>
                            ))}
                          </div>
                        </div>
                      ) : null}
                    </div>
                  ) : null}

                  {!selectedProject.connection_ready ? (
                    <div className="mt-4 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-700">
                      {messages.connectionWarning(selectedProject.token)}
                    </div>
                  ) : null}
                </div>
              </section>

              <section className="rounded-2xl border border-border bg-card p-6">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <div>
                    <h3 className="text-lg font-semibold">{labels.connectedKnowledgeBases}</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {messages.connectedKnowledgeBasesDescription}
                    </p>
                  </div>
                  <Dialog open={connectRAGCollectionOpen} onOpenChange={setConnectRAGCollectionOpen}>
                    <DialogTrigger asChild>
                      <button className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
                        <Plus className="h-4 w-4" />
                        {messages.connectKnowledgeBaseTitle}
                      </button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                        <DialogTitle>{messages.connectKnowledgeBaseTitle}</DialogTitle>
                        <DialogDescription>
                          {messages.connectKnowledgeBaseDescription}
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-3">
                        {availableRAGCollections.length === 0 ? (
                          <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
                            {messages.noAvailableCollections}
                          </div>
                        ) : (
                          availableRAGCollections.map((collection) => (
                            <div key={collection.collection_id} className="flex items-center justify-between gap-3 rounded-xl border border-border bg-background p-4">
                              <div className="min-w-0">
                                <div className="font-medium">{collection.name}</div>
                                <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                                  {collection.collection_id} · {collection.data_type}
                                </div>
                              </div>
                              <button
                                onClick={() => void connectRAGCollectionToProject(collection.collection_id)}
                                disabled={linkingCollectionId === collection.collection_id}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                              >
                                {linkingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                                {labels.connect}
                              </button>
                            </div>
                          ))
                        )}
                      </div>
                    </DialogContent>
                  </Dialog>
                </div>

                {selectedProject.rag_collections.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
                    {messages.noKnowledgeBasesConnected}
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="rounded-xl border border-electric-blue/20 bg-electric-blue/8 p-4">
                      <div className="flex items-center gap-2 text-sm font-medium text-electric-blue">
                        <Database className="h-4 w-4" />
                        {labels.mcpToolReady}
                      </div>
                      <p className="mt-2 text-sm text-foreground/85">
                        {messages.mcpToolReadyIntro}<code>search_project_knowledge</code>{messages.mcpToolReadyOutro}
                      </p>
                      <div className="mt-3 rounded-lg border border-border bg-background p-3">
                        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                          {labels.toolContract}
                        </div>
                        <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-foreground/85">{`search_project_knowledge({
  query: "payment gateway",
  limit: 5,
  collections: ["crm_gym"]
})`}</pre>
                      </div>
                    </div>

                    <div className="space-y-3">
                    {selectedProject.rag_collections.map((collection) => (
                      <div key={collection.collection_id} className="flex flex-col gap-3 rounded-xl border border-border bg-background p-4 lg:flex-row lg:items-center lg:justify-between">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <Database className="h-4 w-4 text-electric-blue" />
                            <div className="font-medium">{collection.name}</div>
                          </div>
                          <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                            {collection.collection_id} · {collection.data_type}
                          </div>
                        </div>
                        <button
                          onClick={() => void disconnectRAGCollectionFromProject(collection.collection_id)}
                          disabled={busyProjectId === selectedProject.project_id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          <Trash2 className="h-4 w-4" />
                          {labels.disconnect}
                        </button>
                      </div>
                    ))}
                    </div>
                  </div>
                )}
              </section>

              <section className="rounded-2xl border border-border bg-card p-6">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <div>
                    <h3 className="text-lg font-semibold">{labels.servers}</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {messages.serverControlDescription}
                    </p>
                  </div>
                  <Dialog
                    open={addServerOpen}
                    onOpenChange={(open) => {
                      setAddServerOpen(open);
                      if (!open) {
                        setEditingServerId(null);
                        setServerForm(emptyServerForm);
                        setOAuthAdvancedOpen(false);
                      }
                    }}
                  >
                    <DialogTrigger asChild>
                      <button className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
                        <Plus className="h-4 w-4" />
                        {labels.addServer}
                      </button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                          <DialogTitle>{editingServerId ? 'Edit MCP Server' : labels.addServer}</DialogTitle>
                        <DialogDescription>{messages.addServerDescription}</DialogDescription>
                      </DialogHeader>

                      <form className="space-y-4" onSubmit={addServer}>
                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{labels.serverName}</span>
                          <input
                            required
                            value={serverForm.name}
                            onChange={(event) => updateServerForm('name', event.target.value)}
                            className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={messages.serverNamePlaceholder}
                          />
                        </label>

                        <div className="overflow-hidden rounded-xl border border-border bg-card">
                          <div className="grid grid-cols-2 gap-0">
                            <button
                              type="button"
                              onClick={() => updateServerForm('transport', 'stdio')}
                              className={`h-11 text-sm font-medium transition-colors ${
                                serverForm.transport === 'stdio'
                                  ? 'bg-muted text-foreground'
                                  : 'bg-transparent text-muted-foreground hover:bg-accent/60'
                              }`}
                            >
                              {labels.stdio}
                            </button>
                            <button
                              type="button"
                              onClick={() => updateServerForm('transport', 'http_stream')}
                              className={`h-11 border-l border-border text-sm font-medium transition-colors ${
                                serverForm.transport === 'http_stream'
                                  ? 'bg-muted text-foreground'
                                  : 'bg-transparent text-muted-foreground hover:bg-accent/60'
                              }`}
                            >
                              {labels.httpStreaming}
                            </button>
                          </div>
                        </div>

                        {serverForm.transport === 'stdio' ? (
                          <div className="space-y-4">
                            <label className="block space-y-2">
                              <span className="text-sm text-muted-foreground">{labels.launchCommand}</span>
                              <input
                                required
                                value={serverForm.command}
                                onChange={(event) => updateServerForm('command', event.target.value)}
                                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                placeholder={messages.commandPlaceholder}
                              />
                            </label>

                            <div className="space-y-2 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">{labels.arguments}</div>
                              <div className="space-y-2">
                                {serverForm.args.map((arg, index) => (
                                  <div key={`arg-${index}`} className="flex items-center gap-2">
                                    <input
                                      value={arg}
                                      onChange={(event) =>
                                        updateStringListField('args', index, event.target.value)
                                      }
                                      className="h-10 flex-1 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={messages.argumentPlaceholder}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => removeStringListField('args', index)}
                                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                              <button
                                type="button"
                                onClick={() => addStringListField('args')}
                                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-muted px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                              >
                                <Plus className="h-4 w-4" />
                                {labels.addArgument}
                              </button>
                            </div>

                            <div className="space-y-2 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">{labels.environmentVariables}</div>
                              <div className="space-y-2">
                                {serverForm.env_vars.map((pair, index) => (
                                  <div key={`env-${index}`} className="grid grid-cols-[1fr_1fr_auto] gap-2">
                                    <input
                                      value={pair.key}
                                      onChange={(event) =>
                                        updateKeyValueField('env_vars', index, 'key', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.key}
                                    />
                                    <input
                                      value={pair.value}
                                      onChange={(event) =>
                                        updateKeyValueField('env_vars', index, 'value', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.value}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => removeKeyValueField('env_vars', index)}
                                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                              <button
                                type="button"
                                onClick={() => addKeyValueField('env_vars')}
                                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-muted px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                              >
                                <Plus className="h-4 w-4" />
                                {labels.addEnvironmentVariable}
                              </button>
                            </div>

                            <div className="space-y-2 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">{labels.environmentVariablePassthrough}</div>
                              <div className="space-y-2">
                                {serverForm.env_passthrough.map((value, index) => (
                                  <div key={`env-pass-${index}`} className="flex items-center gap-2">
                                    <input
                                      value={value}
                                      onChange={(event) =>
                                        updateStringListField('env_passthrough', index, event.target.value)
                                      }
                                      className="h-10 flex-1 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={messages.envPassthroughPlaceholder}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => removeStringListField('env_passthrough', index)}
                                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                              <button
                                type="button"
                                onClick={() => addStringListField('env_passthrough')}
                                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-muted px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                              >
                                <Plus className="h-4 w-4" />
                                {labels.addVariable}
                              </button>
                            </div>

                            <label className="block space-y-2">
                              <span className="text-sm text-muted-foreground">{labels.workingDirectory}</span>
                              <input
                                value={serverForm.working_dir}
                                onChange={(event) => updateServerForm('working_dir', event.target.value)}
                                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                placeholder={messages.workingDirectoryPlaceholder}
                              />
                            </label>

                            {editingServerIntegration?.catalog_item_id === 'filesystem' ? (
                              <div className="space-y-2 rounded-xl border border-electric-blue/20 bg-electric-blue/8 p-4">
                                <div className="text-sm font-medium text-electric-blue">Shared Folder</div>
                                <p className="text-sm text-muted-foreground">
                                  Filesystem MCP uses this folder path as its main accessible directory.
                                </p>
                                <input
                                  value={serverForm.args[serverForm.args.length - 1] ?? ''}
                                  onChange={(event) => updateServerLastArg(event.target.value)}
                                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                  placeholder="/Users/artur/Desktop/Projects/my/embedservice"
                                />
                              </div>
                            ) : null}

                            <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm">
                              <input
                                type="checkbox"
                                checked={serverForm.auto_start}
                                onChange={(event) => updateServerForm('auto_start', event.target.checked)}
                                className="h-4 w-4 rounded border-border"
                              />
                              {labels.autoStart}
                            </label>
                          </div>
                        ) : (
                          <div className="space-y-4">
                            <label className="block space-y-2">
                              <span className="text-sm text-muted-foreground">{labels.url}</span>
                              <input
                                required
                                value={serverForm.url}
                                onChange={(event) => updateServerForm('url', event.target.value)}
                                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                placeholder={messages.urlPlaceholder}
                              />
                            </label>

                            <label className="block space-y-2">
                              <span className="text-sm text-muted-foreground">{labels.bearerTokenEnvironmentVariable}</span>
                              <input
                                value={serverForm.bearer_token_env_var}
                                onChange={(event) =>
                                  updateServerForm('bearer_token_env_var', event.target.value)
                                }
                                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                placeholder={messages.bearerTokenPlaceholder}
                              />
                            </label>

                            <div className="space-y-3 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">Authentication</div>
                              <div className="rounded-lg border border-electric-blue/20 bg-electric-blue/8 px-3 py-3 text-sm text-muted-foreground">
                                Custom remote servers now default to direct URL, bearer env, and headers. OAuth-based remote integrations should be added from the Market tab.
                              </div>
                              {serverForm.auth_type === 'oauth2' ? (
                                <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-3 text-sm text-amber-700">
                                  This server already uses OAuth. Save to keep the existing OAuth settings, then manage connection state from the server OAuth panel.
                                </div>
                              ) : null}
                            </div>

                            <div className="space-y-2 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">{labels.headers}</div>
                              <div className="space-y-2">
                                {serverForm.headers.map((pair, index) => (
                                  <div key={`header-${index}`} className="grid grid-cols-[1fr_1fr_auto] gap-2">
                                    <input
                                      value={pair.key}
                                      onChange={(event) =>
                                        updateKeyValueField('headers', index, 'key', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.key}
                                    />
                                    <input
                                      value={pair.value}
                                      onChange={(event) =>
                                        updateKeyValueField('headers', index, 'value', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.value}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => removeKeyValueField('headers', index)}
                                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                              <button
                                type="button"
                                onClick={() => addKeyValueField('headers')}
                                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-muted px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                              >
                                <Plus className="h-4 w-4" />
                                {labels.addHeader}
                              </button>
                            </div>

                            <div className="space-y-2 rounded-xl border border-border bg-background p-4">
                              <div className="text-sm text-muted-foreground">{labels.headersFromEnvironmentVariables}</div>
                              <div className="space-y-2">
                                {serverForm.header_env_vars.map((pair, index) => (
                                  <div key={`header-env-${index}`} className="grid grid-cols-[1fr_1fr_auto] gap-2">
                                    <input
                                      value={pair.key}
                                      onChange={(event) =>
                                        updateKeyValueField('header_env_vars', index, 'key', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.key}
                                    />
                                    <input
                                      value={pair.value}
                                      onChange={(event) =>
                                        updateKeyValueField('header_env_vars', index, 'value', event.target.value)
                                      }
                                      className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                                      placeholder={labels.value}
                                    />
                                    <button
                                      type="button"
                                      onClick={() => removeKeyValueField('header_env_vars', index)}
                                      className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </button>
                                  </div>
                                ))}
                              </div>
                              <button
                                type="button"
                                onClick={() => addKeyValueField('header_env_vars')}
                                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-muted px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                              >
                                <Plus className="h-4 w-4" />
                                {labels.addVariable}
                              </button>
                            </div>
                          </div>
                        )}

                        <button
                          type="submit"
                          disabled={addingServer}
                          className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {addingServer ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Plus className="h-4 w-4" />
                          )}
                          {editingServerId ? 'Save Server' : labels.addServer}
                        </button>
                      </form>
                    </DialogContent>
                  </Dialog>
                </div>

                {selectedProject.servers.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-center text-muted-foreground">
                    {messages.noServers}
                  </div>
                ) : (
                  <div className="space-y-4">
                    {selectedProject.servers.map((server) => {
                      const busy = busyServerId === server.id;

                      return (
                        <div
                          key={server.id}
                          className="rounded-xl border border-border bg-background p-5"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <div className="flex items-center gap-2">
                                <h4 className="font-semibold">{server.name}</h4>
                                <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                  {server.transport === 'http_stream' ? labels.httpStreaming : labels.stdio}
                                </span>
                              </div>

                              <div className="mt-2 flex flex-wrap items-center gap-2">
                                <span
                                  className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium ${statusTone(
                                    server.status,
                                  )}`}
                                >
                                  {statusIcon(server.status)}
                                  {server.status}
                                </span>
                                <span
                                  className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium ${healthTone(
                                    server.health_status,
                                  )}`}
                                >
                                  <CheckCircle2 className="h-3.5 w-3.5" />
                                  {labels.health}: {healthLabel(server.health_status, labels)}
                                </span>

                                {server.transport === 'stdio' && server.auto_start ? (
                                  <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                    {labels.autoStart}
                                  </span>
                                ) : null}
                                {!server.is_enabled ? (
                                  <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-xs font-medium text-amber-600">
                                    {labels.disabled}
                                  </span>
                                ) : null}
                                {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                                  <span
                                    className={`rounded-full border px-2 py-1 text-xs font-medium ${
                                      server.oauth_connected
                                        ? 'border-status-running/30 bg-status-running/12 text-status-running'
                                        : 'border-amber-500/30 bg-amber-500/10 text-amber-700'
                                    }`}
                                  >
                                    {labels.oauth}: {server.oauth_connected ? labels.connected : labels.notConnected}
                                  </span>
                                ) : null}
                              </div>
                            </div>
                          </div>

                          <div className="mt-4 space-y-2 text-sm">
                            <div>
                              <div className="text-muted-foreground">
                                {server.transport === 'http_stream' ? labels.url : labels.launchCommand}
                              </div>
                              <code className="mt-1 block overflow-x-auto rounded-md bg-card px-3 py-2 text-xs text-electric-blue">
                                {server.transport === 'http_stream'
                                  ? server.url
                                  : server.launch_command}
                              </code>
                            </div>
                            {server.transport === 'stdio' ? (
                              <div>
                                <div className="text-muted-foreground">{labels.workingDirectory}</div>
                                <div className="mt-1 text-sm">
                                  {server.working_dir || labels.notSpecified}
                                </div>
                              </div>
                            ) : server.bearer_token_env_var ? (
                              <div>
                                <div className="text-muted-foreground">
                                  {labels.bearerTokenEnvironmentVariable}
                                </div>
                                <div className="mt-1 text-sm">{server.bearer_token_env_var}</div>
                              </div>
                            ) : null}
                            {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                              <div>
                                <div className="text-muted-foreground">OAuth</div>
                                <div className="mt-1 text-sm">
                                  {server.oauth_provider || 'custom'}
                                  {server.oauth_connected_at
                                    ? ` · connected ${new Date(server.oauth_connected_at).toLocaleString()}`
                                    : ''}
                                </div>
                                {server.oauth_last_error ? (
                                  <div className="mt-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                                    {server.oauth_last_error}
                                  </div>
                                ) : null}
                              </div>
                            ) : null}
                            <div>
                              <div className="text-muted-foreground">{labels.lastCheck}</div>
                              <div className="mt-1 text-sm">
                                {server.health_checked_at
                                  ? new Date(server.health_checked_at).toLocaleString()
                                  : labels.notSpecified}
                              </div>
                              {server.health_error ? (
                                <div className="mt-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                                  {server.health_error}
                                </div>
                              ) : null}
                            </div>
                          </div>

                          <div className="mt-5 flex flex-wrap gap-2">
                            {server.transport === 'stdio' ? (
                              <button
                                onClick={() =>
                                  void runServerAction(
                                    server.id,
                                    server.status === 'Running' ? 'stop' : 'start',
                                  )
                                }
                                disabled={busy || !server.is_enabled}
                                className={`inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-70 ${
                                  server.status === 'Running'
                                    ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                                    : 'bg-status-running text-white hover:bg-status-running/90'
                                }`}
                              >
                                {busy ? (
                                  <LoaderCircle className="h-4 w-4 animate-spin" />
                                ) : server.status === 'Running' ? (
                                  <Square className="h-4 w-4" />
                                ) : (
                                  <Play className="h-4 w-4" />
                                )}
                                {server.status === 'Running' ? labels.stop : labels.start}
                              </button>
                            ) : null}

                            <button
                              onClick={() => void checkServerHealth(server.id)}
                              disabled={busy}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              {busy ? (
                                <LoaderCircle className="h-4 w-4 animate-spin" />
                              ) : (
                                <CheckCircle2 className="h-4 w-4" />
                              )}
                              {labels.check}
                            </button>

                            {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                              <button
                                onClick={() => openAuthModal(server.id)}
                                disabled={busy}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                              >
                                {busy ? (
                                  <LoaderCircle className="h-4 w-4 animate-spin" />
                                ) : (
                                  <Info className="h-4 w-4" />
                                )}
                                {labels.oauth}
                              </button>
                            ) : null}

                            {server.transport === 'stdio' ? (
                              <button
                                onClick={() => void inspectServer(server)}
                                disabled={inspectingServerId === server.id}
                                className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                              >
                                {inspectingServerId === server.id ? (
                                  <LoaderCircle className="h-4 w-4 animate-spin" />
                                ) : (
                                  <Info className="h-4 w-4" />
                                )}
                                {labels.info}
                              </button>
                            ) : null}

                            <button
                              onClick={() => startEditServer(server)}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                            >
                              <Pencil className="h-4 w-4" />
                              Edit
                            </button>

                            <button
                              onClick={() => void setServerEnabled(server.id, !server.is_enabled)}
                              disabled={busy}
                              className={`inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-70 ${
                                server.is_enabled
                                  ? 'border border-amber-500/30 bg-amber-500/10 text-amber-700 hover:bg-amber-500/20'
                                  : 'border border-border bg-card text-foreground hover:bg-accent'
                              }`}
                            >
                              {busy ? (
                                <LoaderCircle className="h-4 w-4 animate-spin" />
                              ) : server.is_enabled ? (
                                <Pause className="h-4 w-4" />
                              ) : (
                                <Play className="h-4 w-4" />
                              )}
                              {server.is_enabled ? labels.disableServer : labels.enableServer}
                            </button>

                            <button
                              onClick={() => void deleteServer(server.id)}
                              disabled={busy}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              <Trash2 className="h-4 w-4" />
                              Delete
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </section>

              <Dialog open={inspectOpen} onOpenChange={setInspectOpen}>
                <DialogContent className="sm:max-w-5xl">
                  <DialogHeader>
                    <DialogTitle>
                      {labels.serverInfo}
                      {inspectionServerName ? ` · ${inspectionServerName}` : ''}
                    </DialogTitle>
                    <DialogDescription>{messages.inspectDescription}</DialogDescription>
                  </DialogHeader>

                  {inspectingServerId ? (
                    <div className="flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                      <LoaderCircle className="h-4 w-4 animate-spin" />
                      {labels.inspectServer}
                    </div>
                  ) : inspectionError ? (
                    <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                      {inspectionError}
                    </div>
                  ) : inspection ? (
                    <div className="space-y-5">
                      <section className="grid gap-4 lg:grid-cols-3">
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">{labels.name}</div>
                          <div className="mt-1 font-medium">
                            {inspection.server_info.title || inspection.server_info.name || inspectionServerName}
                          </div>
                        </div>
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">{labels.protocolVersion}</div>
                          <div className="mt-1 font-medium">{inspection.protocol_version || labels.notSpecified}</div>
                        </div>
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">{labels.version}</div>
                          <div className="mt-1 font-medium">{inspection.server_info.version || labels.notSpecified}</div>
                        </div>
                      </section>

                      <section className="rounded-xl border border-border bg-background p-4">
                        <h4 className="font-semibold">{labels.capabilities}</h4>
                        {(inspection.capabilities ?? []).length > 0 ? (
                          <div className="mt-3 flex flex-wrap gap-2">
                            {(inspection.capabilities ?? []).map((capability) => (
                              <span
                                key={capability}
                                className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue"
                              >
                                {capability}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <div className="mt-3 text-sm text-muted-foreground">{labels.noActivity}</div>
                        )}
                        {inspection.instructions ? (
                          <div className="mt-4">
                            <div className="text-sm text-muted-foreground">{labels.instructions}</div>
                            <pre className="mt-2 whitespace-pre-wrap rounded-lg bg-card px-3 py-3 text-sm text-foreground/85">
                              {inspection.instructions}
                            </pre>
                          </div>
                        ) : null}
                      </section>

                      <section className="rounded-xl border border-border bg-background p-4">
                        <h4 className="font-semibold">{labels.tools}</h4>
                        {(inspection.tools ?? []).length === 0 ? (
                          <div className="mt-3 text-sm text-muted-foreground">{labels.noTools}</div>
                        ) : (
                          <div className="mt-3 space-y-3">
                            {(inspection.tools ?? []).map((tool) => (
                              <div key={tool.name} className="rounded-lg border border-border bg-card p-4">
                                <div className="font-medium">{tool.title || tool.name}</div>
                                {tool.description ? (
                                  <div className="mt-1 text-sm text-muted-foreground">{tool.description}</div>
                                ) : null}
                                {tool.input_schema ? (
                                  <div className="mt-3">
                                    <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                                      inputSchema
                                    </div>
                                    <pre className="mt-2 overflow-x-auto rounded-lg bg-background px-3 py-3 text-xs text-foreground/85">
                                      {formatSchema(tool.input_schema)}
                                    </pre>
                                  </div>
                                ) : null}
                                {tool.output_schema ? (
                                  <div className="mt-3">
                                    <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                                      outputSchema
                                    </div>
                                    <pre className="mt-2 overflow-x-auto rounded-lg bg-background px-3 py-3 text-xs text-foreground/85">
                                      {formatSchema(tool.output_schema)}
                                    </pre>
                                  </div>
                                ) : null}
                              </div>
                            ))}
                          </div>
                        )}
                      </section>

                      <section className="grid gap-5 xl:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background p-4">
                          <h4 className="font-semibold">{labels.resources}</h4>
                          {(inspection.resources ?? []).length === 0 ? (
                            <div className="mt-3 text-sm text-muted-foreground">{labels.noResources}</div>
                          ) : (
                            <div className="mt-3 space-y-3">
                              {(inspection.resources ?? []).map((resource) => (
                                <div key={`${resource.uri}-${resource.name}`} className="rounded-lg border border-border bg-card p-4">
                                  <div className="font-medium">{resource.title || resource.name}</div>
                                  {resource.description ? (
                                    <div className="mt-1 text-sm text-muted-foreground">{resource.description}</div>
                                  ) : null}
                                  {resource.uri ? (
                                    <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">
                                      {resource.uri}
                                    </code>
                                  ) : null}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div className="rounded-xl border border-border bg-background p-4">
                          <h4 className="font-semibold">{labels.prompts}</h4>
                          {(inspection.prompts ?? []).length === 0 ? (
                            <div className="mt-3 text-sm text-muted-foreground">{labels.noPrompts}</div>
                          ) : (
                            <div className="mt-3 space-y-3">
                              {(inspection.prompts ?? []).map((prompt) => (
                                <div key={prompt.name} className="rounded-lg border border-border bg-card p-4">
                                  <div className="font-medium">{prompt.title || prompt.name}</div>
                                  {prompt.description ? (
                                    <div className="mt-1 text-sm text-muted-foreground">{prompt.description}</div>
                                  ) : null}
                                  {(prompt.arguments ?? []).length > 0 ? (
                                    <div className="mt-3 space-y-2">
                                      {(prompt.arguments ?? []).map((argument) => (
                                        <div
                                          key={`${prompt.name}-${argument.name}`}
                                          className="rounded-md bg-background px-3 py-2 text-sm"
                                        >
                                          <div className="font-medium">
                                            {argument.name}
                                            {argument.required ? ' *' : ''}
                                          </div>
                                          {argument.description ? (
                                            <div className="mt-1 text-muted-foreground">{argument.description}</div>
                                          ) : null}
                                        </div>
                                      ))}
                                    </div>
                                  ) : null}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </section>

                      <section className="rounded-xl border border-border bg-background p-4">
                        <h4 className="font-semibold">{labels.readme}</h4>
                        {inspection.readme_path ? (
                          <div className="mt-2 text-sm text-muted-foreground">
                            {inspection.readme_path}
                          </div>
                        ) : null}
                        {inspection.readme ? (
                          <pre className="mt-3 overflow-x-auto whitespace-pre-wrap rounded-lg bg-card px-3 py-3 text-sm text-foreground/85">
                            {inspection.readme}
                          </pre>
                        ) : (
                          <div className="mt-3 text-sm text-muted-foreground">{labels.noReadme}</div>
                        )}
                      </section>
                    </div>
                  ) : null}
                </DialogContent>
              </Dialog>

              <Dialog
                open={authOpen}
                onOpenChange={(open) => {
                  setAuthOpen(open);
                  if (!open) {
                    setAuthServerId(null);
                  }
                }}
              >
                <DialogContent className="sm:max-w-3xl">
                  <DialogHeader>
                    <DialogTitle>
                      {labels.oauth}
                      {authServer ? ` · ${authServer.name}` : ''}
                    </DialogTitle>
                    <DialogDescription>
                      Manage OAuth connection settings and current authorization state for this remote MCP server.
                    </DialogDescription>
                  </DialogHeader>

                  {authServer ? (
                    <div className="space-y-5">
                      <section className="grid gap-4 sm:grid-cols-3">
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">Provider</div>
                          <div className="mt-1 font-medium">{authServer.oauth_provider || 'custom'}</div>
                        </div>
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">{labels.oauth}</div>
                          <div className="mt-1 font-medium">
                            {authServer.oauth_connected ? labels.connected : labels.notConnected}
                          </div>
                        </div>
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">{labels.lastCheck}</div>
                          <div className="mt-1 font-medium">
                            {authServer.oauth_connected_at
                              ? new Date(authServer.oauth_connected_at).toLocaleString()
                              : labels.notSpecified}
                          </div>
                        </div>
                      </section>

                      <section className="rounded-xl border border-border bg-background p-4">
                        <div className="text-sm text-muted-foreground">Callback URL</div>
                        <code className="mt-2 block overflow-x-auto rounded-lg bg-card px-3 py-3 text-xs text-electric-blue">
                          {window.location.origin}/oauth/callback
                        </code>
                      </section>

                      <section className="grid gap-5 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">Authorize URL</div>
                          <code className="mt-2 block overflow-x-auto rounded-lg bg-card px-3 py-3 text-xs text-electric-blue">
                            {authServer.oauth_authorize_url || labels.notSpecified}
                          </code>
                        </div>
                        <div className="rounded-xl border border-border bg-background p-4">
                          <div className="text-sm text-muted-foreground">Token URL</div>
                          <code className="mt-2 block overflow-x-auto rounded-lg bg-card px-3 py-3 text-xs text-electric-blue">
                            {authServer.oauth_token_url || labels.notSpecified}
                          </code>
                        </div>
                      </section>

                      <section className="rounded-xl border border-border bg-background p-4">
                        <div className="text-sm text-muted-foreground">Scopes</div>
                        {(authServer.oauth_scopes ?? []).length > 0 ? (
                          <div className="mt-3 flex flex-wrap gap-2">
                            {(authServer.oauth_scopes ?? []).map((scope) => (
                              <span
                                key={scope}
                                className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue"
                              >
                                {scope}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <div className="mt-3 text-sm text-muted-foreground">{labels.notSpecified}</div>
                        )}
                      </section>

                      {authServer.oauth_last_error ? (
                        <section className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                          {authServer.oauth_last_error}
                        </section>
                      ) : null}

                      <section className="flex flex-wrap gap-3">
                        <button
                          onClick={() => void connectOAuth(authServer.id)}
                          disabled={busyServerId === authServer.id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {busyServerId === authServer.id ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4" />
                          )}
                          {authServer.oauth_connected ? `Reconnect ${labels.oauth}` : `Connect ${labels.oauth}`}
                        </button>

                        <button
                          onClick={() => void disconnectOAuth(authServer.id)}
                          disabled={busyServerId === authServer.id || !authServer.oauth_connected}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {busyServerId === authServer.id ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Pause className="h-4 w-4" />
                          )}
                          {`Disconnect ${labels.oauth}`}
                        </button>
                      </section>
                    </div>
                  ) : null}
                </DialogContent>
              </Dialog>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
