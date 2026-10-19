import { FormEvent, Suspense, lazy, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { Toaster } from './components/ui/sonner';
import {
  Bot,
  Database,
  FolderKanban,
  LogOut,
  Shield,
  ShoppingBag,
  TextSearch,
  Terminal,
} from 'lucide-react';
import {
  defaultLanguage,
  detectInitialLanguage,
  dictionaries,
  languageStorageKey,
  Language,
} from './i18n';
import logo from '../styles/logo.png';
import { AppShell } from './components/AppShell';
import type {
  ProjectStatus,
  ProjectPromptProfile,
  ServerStatus,
  RAGCollection,
  RAGSearchResult,
  AuditLog,
  PerformanceMetricsResponse,
  MetricsWindow,
  EditionMeta,
  LogsFilterMode,
  ServerInspection,
  ServerToolStatus,
  KeyValuePair,
  LlamaCppStatus,
  LlamaCppLaunchResponse,
  OllamaStatus,
  OllamaLaunchResponse,
  ProjectFormState,
  ServerFormState,
} from './types';
import { useAuth } from './hooks/useAuth';
import { useLlamaCpp } from './hooks/useLlamaCpp';
import { useOllama } from './hooks/useOllama';
import { useServerTools } from './hooks/useServerTools';
import type { ProPrincipal, ProTokenRecord, ProUserRecord, ProRoleRecord, ProSessionRecord, CreateProTokenResponse, CreateProUserResponse, UpdateProUserResponse, UpdateProUserPasswordResponse, DisableProUserResponse, EnableProUserResponse, CreateProSessionResponse, ProLocalLoginResponse, ProSSOConfig, ProScopePreset } from './pro/types';
import type {
  ProRagBoxConfig,
  ProRagBoxKnowledgeBaseStatus,
  ProRagBoxSearchResponse,
  ProRagBoxVectorConnection,
  ProRagBoxVectorConnectionListResponse,
  ProRagBoxWorkspace,
  ProRagBoxWorkspaceListResponse,
  ProRagBoxWriteResponse,
} from './pro/ragbox';
import type {
  ProStorageConfig,
  ProStorageRuntimeStatus,
  ProStorageShadowStatus,
} from './pro/storage';
import { proAPIRequest, hasProScope } from './pro/utils';
import { getProCopy } from './pro/i18n';
import { proAuthStorageKey } from './pro/constants';
import { APIRequestError, apiRequest, setAPIAuthToken } from './utils/api';
import { ProjectsSidebar } from './components/ProjectsSidebar';
import { readProjectOrder, reconcileProjectOrder } from './utils/project-order';
import type {
  CatalogItem,
  CatalogSettings,
  InstalledPackage,
  ProjectOption,
} from './market';

const SkillsView = lazy(async () => { const module = await import('./components/SkillsView'); return { default: module.SkillsView }; });
const CLIView = lazy(async () => { const module = await import('./components/CLIView'); return { default: module.CLIView }; });

const KnowledgeView = lazy(async () => {
  const module = await import('./components/KnowledgeView');
  return { default: module.KnowledgeView };
});

const LogsView = lazy(async () => {
  const module = await import('./components/LogsView');
  return { default: module.LogsView };
});

const MarketView = lazy(async () => {
  const module = await import('./components/MarketView');
  return { default: module.MarketView };
});

const ProjectsView = lazy(async () => {
  const module = await import('./components/ProjectsView');
  return { default: module.ProjectsView };
});

const ProAccessView = lazy(async () => {
  const module = await import('./components/ProAccessView');
  return { default: module.ProAccessView };
});

type CatalogResponse = {
  settings: CatalogSettings;
  items: CatalogItem[];
};

type InstalledPackageListResponse = {
  items: InstalledPackage[];
};

type InstallPackageResponse = {
  package: InstalledPackage;
};

type ServerActionResponse = {
  server_id: number;
  status: string;
  health_status?: string;
  health_error?: string;
  health_checked_at?: string;
};

type LMStudioLaunchResponse = {
  project_id: number;
  server_name: string;
  deeplink: string;
};

const legacyCatalogSourceURL = 'https://webeasy.kz/mcpbox/catalog.json';
const defaultCatalogSourceURL = 'https://mcpbox.sh/catalog.json';


function normalizeCatalogSourceURL(url: string) {
  const trimmed = url.trim();
  if (trimmed === legacyCatalogSourceURL) {
    return defaultCatalogSourceURL;
  }
  return trimmed;
}

const emptyProjectForm: ProjectFormState = {
  name: '',
  description: '',
  root_path: '',
  bearer_auth_enabled: false,
  bearer_token: '',
  oauth_redirect_uri: '',
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
  source_path: '',
  auto_reindex: false,
  service_mode: 'bleve_only' as 'bleve_only' | 'ragbox_only' | 'dual',
  vector_connection_id: '',
  existing_workspace_id: '',
};

const emptyProRagBoxVectorConnectionForm = {
  name: '',
  backend_type: 'sqlite' as 'sqlite' | 'qdrant' | 'pgvector' | 'milvus' | 'weaviate' | 'opensearch' | 'elasticsearch' | 'pinecone' | 'chroma',
  description: '',
  is_default: false,
  qdrant_scheme: 'http' as 'http' | 'https',
  qdrant_host: '',
  qdrant_port: '6333',
  qdrant_api_key: '',
  qdrant_collection_prefix: '',
  pgvector_host: '',
  pgvector_port: '5432',
  pgvector_username: '',
  pgvector_password: '',
  pgvector_database_name: '',
  pgvector_schema: '',
  pgvector_table_prefix: '',
  backend_config_json: '',
};

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

function OllamaIcon({ className }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center justify-center rounded-md border border-current/20 bg-current/10 ${className ?? ''}`}
    >
      <Bot className="h-[0.9em] w-[0.9em]" />
    </span>
  );
}

function modelNameFromPath(modelPath: string) {
  const normalized = modelPath.trim().replace(/\\/g, '/');
  const fileName = normalized.split('/').pop() ?? normalized;
  return fileName.replace(/\.gguf$/i, '');
}

export default function App() {
  const [view, setView] = useState<'projects' | 'skills' | 'cli' | 'knowledge' | 'market' | 'logs' | 'settings'>('projects');
  const [language, setLanguage] = useState<Language>(detectInitialLanguage);
  const [editionMeta, setEditionMeta] = useState<EditionMeta>({
    edition_id: 'mcpbox',
    edition_name: 'MCPBox',
    capabilities: ['auth'],
  });
  const [projects, setProjects] = useState<ProjectStatus[]>([]);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [logMetrics, setLogMetrics] = useState<PerformanceMetricsResponse | null>(null);
  const [selectedLogsProjectId, setSelectedLogsProjectId] = useState<number | null>(null);
  const [metricsWindow, setMetricsWindow] = useState<MetricsWindow>('1h');
  const [logsFilterMode, setLogsFilterMode] = useState<LogsFilterMode>('all');
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
  const [launchProjectOpen, setLaunchProjectOpen] = useState(false);
  const [duplicateProjectOpen, setDuplicateProjectOpen] = useState(false);
  const [editingProjectId, setEditingProjectId] = useState<number | null>(null);
  const [updatingPrompt, setUpdatingPrompt] = useState(false);
  const [duplicatingProjectId, setDuplicatingProjectId] = useState<number | null>(null);
  const [duplicateProjectName, setDuplicateProjectName] = useState('');
  const [, setOAuthAdvancedOpen] = useState(false);

  // Server tools hook
  const {
    addingServer,
    setAddingServer,
    addServerOpen,
    setAddServerOpen,
    editingServerId,
    setEditingServerId,
    busyServerId,
    setBusyServerId,
    inspectOpen,
    setInspectOpen,
    inspectingServerId,
    setInspectingServerId,
    inspection,
    setInspection,
    inspectionServerName,
    setInspectionServerName,
    inspectionError,
    setInspectionError,
    serverToolsOpen,
    setServerToolsOpen,
    serverToolsLoadingId,
    setServerToolsLoadingId,
    serverToolsSavingName,
    setServerToolsSavingName,
    serverToolsServerId,
    setServerToolsServerId,
    serverToolsServerName,
    setServerToolsServerName,
    serverTools,
    setServerTools,
    serverToolsError,
    setServerToolsError,
    addServer: hookAddServer,
    runServerAction: hookRunServerAction,
    setServerEnabled: hookSetServerEnabled,
    inspectServer: hookInspectServer,
    openServerTools: hookOpenServerTools,
    setServerToolEnabled: hookSetServerToolEnabled,
  } = useServerTools(
    { requestFailed: 'Request failed', serverStarted: 'Server started', serverStopped: 'Server stopped', serverEnabled: 'Server enabled', serverDisabled: 'Server disabled' },
    () => loadProjects()
  );
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
  const [uploadingKnowledgeBaseId, setUploadingKnowledgeBaseId] = useState<string | null>(null);
  const [activeRAGSearchCollectionId, setActiveRAGSearchCollectionId] = useState<string | null>(null);
  const [indexingCollectionId, setIndexingCollectionId] = useState<string | null>(null);
  const [searchingCollectionId, setSearchingCollectionId] = useState<string | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [catalogSettings, setCatalogSettings] = useState<CatalogSettings | null>(null);
  const [installedPackages, setInstalledPackages] = useState<InstalledPackage[]>([]);

  const { llamaCppStatus, llamaCppRefreshing, loadLlamaCppStatus } = useLlamaCpp({ requestFailed: 'Request failed' });
  const [launchingLlamaCppProjectId, setLaunchingLlamaCppProjectId] = useState<number | null>(null);
  const [stoppingLlamaCpp, setStoppingLlamaCpp] = useState(false);
  const [selectedLlamaCppModelPath, setSelectedLlamaCppModelPath] = useState('');
  const [selectedLlamaCppModelName, setSelectedLlamaCppModelName] = useState('');
  // Ollama hook
  const { ollamaStatus, setOllamaStatus, ollamaRefreshing, setOllamaRefreshing, selectedOllamaModel, setSelectedOllamaModel, loadOllamaStatus: hookLoadOllamaStatus } = useOllama({ requestFailed: 'Request failed' });

  const [catalogURL, setCatalogURL] = useState(defaultCatalogSourceURL);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogSyncing, setCatalogSyncing] = useState(false);
  const [catalogURLVisible, setCatalogURLVisible] = useState(false);
  const [catalogSourceMode, setCatalogSourceMode] = useState<'server' | 'file'>('server');
  const [localCatalogFileName, setLocalCatalogFileName] = useState('');
  const [localCatalogContent, setLocalCatalogContent] = useState('');
  const [installingCatalogItemId, setInstallingCatalogItemId] = useState<string | null>(null);
  const [addingCatalogItemId, setAddingCatalogItemId] = useState<string | null>(null);
  const [uninstallingCatalogItemId, setUninstallingCatalogItemId] = useState<string | null>(null);
  const [busyProjectId, setBusyProjectId] = useState<number | null>(null);
  const [launchingOllamaProjectId, setLaunchingOllamaProjectId] = useState<number | null>(null);
  const [launchingLMStudioProjectId, setLaunchingLMStudioProjectId] = useState<number | null>(null);

  // Auth hook
  const { authOpen, setAuthOpen, authServerId, setAuthServerId, resetAuthServer } = useAuth();

  const authServer: ServerStatus | null = authServerId !== null
    ? projects.flatMap((p) => p.servers).find((s) => s.id === authServerId) ?? null
    : null;

  const [proAuthToken, setProAuthToken] = useState(() => {
    if (typeof window === 'undefined') {
      return '';
    }
    const storedToken = window.localStorage.getItem(proAuthStorageKey) ?? '';
    setAPIAuthToken(storedToken);
    return storedToken;
  });
  const [proPrincipal, setProPrincipal] = useState<ProPrincipal | null>(null);
  const projectOrderStorageKey = `mcpbox-project-order:${proPrincipal?.user_id ?? 'local'}`;
  const [proTokens, setProTokens] = useState<ProTokenRecord[]>([]);
  const [proUsers, setProUsers] = useState<ProUserRecord[]>([]);
  const [proRoles, setProRoles] = useState<ProRoleRecord[]>([]);
  const [proSessions, setProSessions] = useState<ProSessionRecord[]>([]);
  const [proSSOConfig, setProSSOConfig] = useState<ProSSOConfig | null>(null);
  const [proStorageConfig, setProStorageConfig] = useState<ProStorageConfig | null>(null);
  const [proStorageRuntimeStatus, setProStorageRuntimeStatus] = useState<ProStorageRuntimeStatus | null>(null);
  const [proStorageShadowStatus, setProStorageShadowStatus] = useState<ProStorageShadowStatus | null>(null);
  const [proRagBoxConfig, setProRagBoxConfig] = useState<ProRagBoxConfig | null>(null);
  const [proLoading, setProLoading] = useState(false);
  const [proLocalLoginLoading, setProLocalLoginLoading] = useState(false);
  const [proAuthBootstrapping, setProAuthBootstrapping] = useState(false);
  const [proCreateOpen, setProCreateOpen] = useState(false);
  const [proCreatingToken, setProCreatingToken] = useState(false);
  const [proRevokingTokenId, setProRevokingTokenId] = useState<number | null>(null);
  const [proCreateUserOpen, setProCreateUserOpen] = useState(false);
  const [proCreatingUser, setProCreatingUser] = useState(false);
  const [proEditUserOpen, setProEditUserOpen] = useState(false);
  const [proUpdatingUserRoles, setProUpdatingUserRoles] = useState(false);
  const [proPasswordUserOpen, setProPasswordUserOpen] = useState(false);
  const [proUpdatingUserPassword, setProUpdatingUserPassword] = useState(false);
  const [proDisablingUserId, setProDisablingUserId] = useState<number | null>(null);
  const [proEnablingUserId, setProEnablingUserId] = useState<number | null>(null);
  const [proDeletingUserId, setProDeletingUserId] = useState<number | null>(null);
  const [proSessionsFilterUserId, setProSessionsFilterUserId] = useState('all');
  const [proEditingUserId, setProEditingUserId] = useState<string>('');
  const [proEditingUserName, setProEditingUserName] = useState('');
  const [proEditingUserRoles, setProEditingUserRoles] = useState('reader');
  const [proPasswordUserId, setProPasswordUserId] = useState('');
  const [proPasswordUserName, setProPasswordUserName] = useState('');
  const [proPasswordUserPassword, setProPasswordUserPassword] = useState('');
  const [proPasswordUserPasswordConfirm, setProPasswordUserPasswordConfirm] = useState('');
  const [proCreateSessionOpen, setProCreateSessionOpen] = useState(false);
  const [proCreatingSession, setProCreatingSession] = useState(false);
  const [proRevokingSessionId, setProRevokingSessionId] = useState<number | null>(null);
  const [proNewTokenName, setProNewTokenName] = useState('');
  const [proLoginEmail, setProLoginEmail] = useState('');
  const [proLoginPassword, setProLoginPassword] = useState('');
  const [proNewTokenScopes, setProNewTokenScopes] = useState('pro:read, pro:write');
  const [proNewTokenExpiresDays, setProNewTokenExpiresDays] = useState('30');
  const [proCreatedTokenValue, setProCreatedTokenValue] = useState<string | null>(null);
  const [proNewUserName, setProNewUserName] = useState('');
  const [proNewUserEmail, setProNewUserEmail] = useState('');
  const [proNewUserRoles, setProNewUserRoles] = useState('reader');
  const [proNewSessionUserId, setProNewSessionUserId] = useState('');
  const [proNewSessionLabel, setProNewSessionLabel] = useState('');
  const [proNewSessionExpiresDays, setProNewSessionExpiresDays] = useState('30');
  const [proCreatedSessionToken, setProCreatedSessionToken] = useState<string | null>(null);
  const [proStorageOperationalBackend, setProStorageOperationalBackend] = useState<'sqlite' | 'postgres'>('sqlite');
  const [proStoragePostgresHost, setProStoragePostgresHost] = useState('');
  const [proStoragePostgresPort, setProStoragePostgresPort] = useState('5432');
  const [proStoragePostgresDatabase, setProStoragePostgresDatabase] = useState('');
  const [proStoragePostgresSchema, setProStoragePostgresSchema] = useState('public');
  const [proStoragePostgresUsername, setProStoragePostgresUsername] = useState('');
  const [proStoragePostgresPassword, setProStoragePostgresPassword] = useState('');
  const [proStorageClearPostgresDSN, setProStorageClearPostgresDSN] = useState(false);
  const [proStorageLogsBackend, setProStorageLogsBackend] = useState<'operational' | 'clickhouse' | 'elasticsearch'>('operational');
  const [proStorageClickHouseURL, setProStorageClickHouseURL] = useState('');
  const [proStorageClickHouseDatabase, setProStorageClickHouseDatabase] = useState('');
  const [proStorageClickHouseUsername, setProStorageClickHouseUsername] = useState('');
  const [proStorageClickHousePassword, setProStorageClickHousePassword] = useState('');
  const [proStorageClearClickHousePassword, setProStorageClearClickHousePassword] = useState(false);
  const [proStorageElasticsearchURL, setProStorageElasticsearchURL] = useState('');
  const [proStorageElasticsearchUsername, setProStorageElasticsearchUsername] = useState('');
  const [proStorageElasticsearchPassword, setProStorageElasticsearchPassword] = useState('');
  const [proStorageElasticsearchIndex, setProStorageElasticsearchIndex] = useState('');
  const [proStorageClearElasticsearchPassword, setProStorageClearElasticsearchPassword] = useState(false);
  const [proStorageSaving, setProStorageSaving] = useState(false);
  const [proRagBoxBaseURL, setProRagBoxBaseURL] = useState('');
  const [proRagBoxAdminToken, setProRagBoxAdminToken] = useState('');
  const [proRagBoxClearAdminToken, setProRagBoxClearAdminToken] = useState(false);
  const [proRagBoxTimeoutSeconds, setProRagBoxTimeoutSeconds] = useState('30');
  const [proRagBoxWriteMode, setProRagBoxWriteMode] = useState<'bleve_only' | 'dual_write'>('dual_write');
  const [proRagBoxSearchMode, setProRagBoxSearchMode] = useState<'bleve_only' | 'dual_search' | 'hybrid'>('hybrid');
  const [proRagBoxSaving, setProRagBoxSaving] = useState(false);
  const [proRagBoxVectorConnections, setProRagBoxVectorConnections] = useState<ProRagBoxVectorConnection[]>([]);
  const [proRagBoxWorkspaces, setProRagBoxWorkspaces] = useState<ProRagBoxWorkspace[]>([]);
  const [proRagBoxBuiltinFallbackBackend, setProRagBoxBuiltinFallbackBackend] = useState('sqlite');
  const [activeSettingsTab, setActiveSettingsTab] = useState<'storage' | 'ragbox' | 'access' | 'users' | 'sessions'>('storage');
  const [proRagBoxConnectionError, setProRagBoxConnectionError] = useState<string | null>(null);
  const [proRagBoxLoadingVectorConnections, setProRagBoxLoadingVectorConnections] = useState(false);
  const [proRagBoxLoadingWorkspaces, setProRagBoxLoadingWorkspaces] = useState(false);
  const [proRagBoxCreatingVectorConnection, setProRagBoxCreatingVectorConnection] = useState(false);
  const [proRagBoxDefaultingVectorConnectionId, setProRagBoxDefaultingVectorConnectionId] = useState<string | null>(null);
  const [proRagBoxVectorConnectionForm, setProRagBoxVectorConnectionForm] = useState(emptyProRagBoxVectorConnectionForm);
  const [proRagBoxBindingProjectId, setProRagBoxBindingProjectId] = useState('');
  const [proRagBoxBindingVectorConnectionId, setProRagBoxBindingVectorConnectionId] = useState('');
  const [proRagBoxBindingExistingWorkspaceId, setProRagBoxBindingExistingWorkspaceId] = useState('');
  const [proRagBoxBindingCollectionProjectId, setProRagBoxBindingCollectionProjectId] = useState('');
  const [proRagBoxBindingCollectionId, setProRagBoxBindingCollectionId] = useState('');
  const [proRagBoxEnsuringWorkspace, setProRagBoxEnsuringWorkspace] = useState(false);
  const [proRagBoxEnsuringCollection, setProRagBoxEnsuringCollection] = useState(false);
  const [proRagBoxUploadProjectId, setProRagBoxUploadProjectId] = useState('');
  const [proRagBoxUploadCollectionId, setProRagBoxUploadCollectionId] = useState('');
  const [proRagBoxUploadFile, setProRagBoxUploadFile] = useState<File | null>(null);
  const [proRagBoxUploadSourceName, setProRagBoxUploadSourceName] = useState('');
  const [proRagBoxUploadExternalId, setProRagBoxUploadExternalId] = useState('');
  const [proRagBoxUploadTitle, setProRagBoxUploadTitle] = useState('');
  const [proRagBoxUploadMetadata, setProRagBoxUploadMetadata] = useState('');
  const [proRagBoxUploadingFile, setProRagBoxUploadingFile] = useState(false);
  const [proRagBoxRecordsProjectId, setProRagBoxRecordsProjectId] = useState('');
  const [proRagBoxRecordsCollectionId, setProRagBoxRecordsCollectionId] = useState('');
  const [proRagBoxRecordsPayload, setProRagBoxRecordsPayload] = useState(
    '{\n  "source": { "type": "api_push", "name": "mcpboxpro" },\n  "records": []\n}',
  );
  const [proRagBoxUpsertingRecords, setProRagBoxUpsertingRecords] = useState(false);
  const [proRagBoxLastOperation, setProRagBoxLastOperation] = useState<ProRagBoxWriteResponse | null>(null);
  const [proRagBoxKnowledgeBaseStatus, setProRagBoxKnowledgeBaseStatus] = useState<Record<string, ProRagBoxKnowledgeBaseStatus>>({});
  const [ragSearchDiagnosticsByCollection, setRAGSearchDiagnosticsByCollection] = useState<Record<string, string | null>>({});
  const [ragSearchErrors, setRAGSearchErrors] = useState<Record<string, string | null>>({});
  const logsViewportRef = useRef<HTMLDivElement | null>(null);
  const logsStickToBottomRef = useRef(true);
  const marketAutoSyncTriggeredRef = useRef(false);
  const previousViewRef = useRef(view);
  const dictionary = dictionaries[language];
  const { labels, messages } = dictionary;
  const languageOptions: Array<{ value: Language; label: string }> = [
    { value: 'en', label: labels.english },
    { value: 'ru', label: labels.russian },
  ];
  const proCopy = getProCopy(language);
  const proScopePresets: ProScopePreset[] =
    language === 'ru'
      ? [
          {
            id: 'reader',
            label: 'Reader',
            scopes: ['pro:read'],
            description: 'Только просмотр Pro API и token activity',
          },
          {
            id: 'operator',
            label: 'Operator',
            scopes: ['pro:read', 'pro:write'],
            description: 'Рабочие операции без полного admin-доступа',
          },
          {
            id: 'admin',
            label: 'Admin',
            scopes: ['pro:admin'],
            description: 'Полный доступ ко всем Pro операциям',
          },
        ]
      : [
          {
            id: 'reader',
            label: 'Reader',
            scopes: ['pro:read'],
            description: 'Read-only access to Pro API and token activity',
          },
          {
            id: 'operator',
            label: 'Operator',
            scopes: ['pro:read', 'pro:write'],
            description: 'Operational access without full admin rights',
          },
          {
            id: 'admin',
            label: 'Admin',
            scopes: ['pro:admin'],
            description: 'Full access to all Pro operations',
          },
        ];

  function defaultProjectForm(): ProjectFormState {
    return {
      ...emptyProjectForm,
    };
  }
  const viewLoadingFallback = (
    <div className="flex min-h-[40vh] items-center justify-center rounded-2xl border border-border bg-card/50">
      <div className="text-sm text-muted-foreground">{messages.loadingProjects}</div>
    </div>
  );
  const metricsWindowOptions: Array<{ value: MetricsWindow; label: string }> = [
    { value: '5m', label: labels.last5Minutes },
    { value: '1h', label: labels.lastHour },
    { value: '24h', label: labels.last24Hours },
  ];
  const hasProAccess =
    editionMeta.capabilities.includes('auth') ||
    editionMeta.capabilities.includes('agent_tokens');
  const isProEdition = editionMeta.capabilities.includes('auth');
  const navigationItems = [
    { id: 'projects' as const, label: labels.projects, icon: FolderKanban },
    { id: 'cli' as const, label: 'CLI', icon: Terminal },
    { id: 'knowledge' as const, label: labels.knowledgeBase, icon: Database },
    { id: 'market' as const, label: labels.market, icon: ShoppingBag },
    { id: 'skills' as const, label: 'Skills', icon: Bot },
    { id: 'logs' as const, label: labels.logs, icon: TextSearch },
    ...(hasProAccess ? [{ id: 'settings' as const, label: proCopy.nav, icon: Shield }] : []),
  ];

  const selectedProject =
    projects.find((project) => project.project_id === selectedProjectId) ?? null;
  const selectedProjectForProjectsView = selectedProject
    ? {
        ...selectedProject,
        bearer_auth_enabled: Boolean((selectedProject as ProjectStatus & Record<string, unknown>).bearer_auth_enabled),
        bearer_token: String((selectedProject as ProjectStatus & Record<string, unknown>).bearer_token ?? ''),
        oauth_redirect_uri: String((selectedProject as ProjectStatus & Record<string, unknown>).oauth_redirect_uri ?? ''),
        llama_cpp_model_path: String((selectedProject as ProjectStatus & Record<string, unknown>).llama_cpp_model_path ?? ''),
        llama_cpp_model_name: String((selectedProject as ProjectStatus & Record<string, unknown>).llama_cpp_model_name ?? ''),
      }
    : null;
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
  const projectNamesById = Object.fromEntries(
    projects.map((project) => [project.project_id, project.name] as const),
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
  const ragSearchDiagnostics = activeRAGSearchCollectionId ? ragSearchDiagnosticsByCollection[activeRAGSearchCollectionId] ?? null : null;
  const ragSearchError = activeRAGSearchCollectionId ? ragSearchErrors[activeRAGSearchCollectionId] ?? null : null;
  const editingServerIntegration =
    editingServerId !== null ? serverIntegrationsByServerID[editingServerId] ?? null : null;
  const projectOptions: ProjectOption[] = projects.map((project) => ({
    project_id: project.project_id,
    name: project.name,
    root_path: project.root_path,
  }));
  const ragBoxProjects = projects.map((project) => ({
    project_id: project.project_id,
    name: project.name,
  }));
  const ragBoxProjectKnowledgeBases = projects.flatMap((project) =>
    (project.rag_collections ?? []).map((collection) => ({
      project_id: project.project_id,
      collection_id: collection.collection_id,
      name: collection.name,
    })),
  );
  const shouldShowOllamaControls = ollamaStatus?.installed ?? false;
  const canLaunchOllama =
    !!selectedOllamaModel &&
    (ollamaStatus?.models.length ?? 0) > 0;
  const shouldShowLlamaCppControls = llamaCppStatus?.installed ?? false;
  const canLaunchLlamaCpp = !!llamaCppStatus?.installed && !!(selectedLlamaCppModelPath.trim() || selectedProject?.llama_cpp_model_path?.trim() || llamaCppStatus?.model_path?.trim());
  const requestTrendValues = logMetrics?.trends.map((entry) => entry.request_count) ?? [];
  const errorTrendValues = logMetrics?.trends.map((entry) => entry.error_count) ?? [];
  const avgLatencyTrendValues = logMetrics?.trends.map((entry) => entry.avg_latency_ms) ?? [];
  const p95LatencyTrendValues = logMetrics?.trends.map((entry) => entry.p95_latency_ms) ?? [];
  const proAuditActions = new Set([
    'token_created',
    'token_revoked',
    'token_used',
    'session_created',
    'session_used',
    'session_revoked',
    'user_created',
    'user_roles_updated',
    'user_enabled',
    'user_disabled',
    'user_deleted',
    'local_login',
    'sso_login',
  ]);
  const isProAuditLog = (entry: AuditLog) =>
    proAuditActions.has(entry.action);
  const proActivityCopy =
    language === 'ru'
      ? {
          title: 'Pro activity',
          subtitle: 'События токенов, сессий, защищенного Pro API',
          all: 'Вся активность',
          proOnly: 'Pro',
          created: 'Создано',
          used: 'Использовано',
          revoked: 'Отозвано',
          empty: 'Pro activity пока нет.',
          logFeedTab: 'Логи',
          analyticsTab: 'Аналитика',
          exportLogs: 'Выгрузить лог',
          clearLogs: 'Очистить историю',
          clearLogsConfirm: filteredLogsProject?.name
            ? `Удалить всю историю проекта «${filteredLogsProject.name}»? Другие проекты не будут затронуты.`
            : 'Удалить всю историю логов? Это действие нельзя отменить.',
          actor: 'Кто',
          requestIP: 'IP',
          autoScrollPinned: 'Автопрокрутка включена',
          autoScrollDetached: 'Просмотр истории',
        }
      : {
          title: 'Pro activity',
          subtitle: 'Agent token, session, protected Pro API events',
          all: 'All activity',
          proOnly: 'Pro',
          created: 'Created',
          used: 'Used',
          revoked: 'Revoked',
          empty: 'No Pro activity yet.',
          logFeedTab: 'Logs',
          analyticsTab: 'Analytics',
          exportLogs: 'Export log',
          clearLogs: 'Clear history',
          clearLogsConfirm: filteredLogsProject?.name
            ? `Delete the complete history for “${filteredLogsProject.name}”? Other projects will not be affected.`
            : 'Delete the complete log history? This cannot be undone.',
          actor: 'Actor',
          requestIP: 'IP',
          autoScrollPinned: 'Auto-scroll on',
          autoScrollDetached: 'History mode',
        };
  const proActivityLogs = logs.filter(isProAuditLog);
  const visibleLogs =
    logsFilterMode === 'pro' ? proActivityLogs : logs;
  const proCreatedCount = proActivityLogs.filter((entry) => entry.action === 'token_created' || entry.action === 'session_created').length;
  const proUsedCount = proActivityLogs.filter((entry) => entry.action === 'token_used' || entry.action === 'session_used' || entry.action === 'local_login' || entry.action === 'sso_login').length;
  const proRevokedCount = proActivityLogs.filter((entry) => entry.action === 'token_revoked' || entry.action === 'session_revoked').length;
  const canWritePro = hasProScope(proPrincipal, 'pro:write');
  const canAdminPro = hasProScope(proPrincipal, 'pro:admin');
  const proAuthLocked = isProEdition && !proPrincipal;
  const activeView = proAuthLocked ? 'settings' : view;
  const visibleNavigationItems = proAuthLocked
    ? navigationItems.filter((item) => item.id === 'settings')
    : navigationItems;

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    window.localStorage.setItem(languageStorageKey, language);
    document.documentElement.lang = language;
  }, [language]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    setProAuthToken(window.localStorage.getItem(proAuthStorageKey) ?? '');

    const url = new URL(window.location.href);
    const sessionToken = url.searchParams.get('pro_session_token');
    const ssoError = url.searchParams.get('pro_sso_error');
    const targetView = url.searchParams.get('pro_view');
    if (sessionToken) {
      updateAndPersistProAuthToken(sessionToken);
      if (targetView === 'pro' || targetView === 'settings') {
        setView('settings');
      }
      url.searchParams.delete('pro_session_token');
      url.searchParams.delete('pro_view');
      window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`);
    }
    if (ssoError) {
      if (targetView === 'pro' || targetView === 'settings') {
        setView('settings');
      }
      toast.error(ssoError);
      url.searchParams.delete('pro_sso_error');
      url.searchParams.delete('pro_view');
      window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`);
    }
  }, []);

  useEffect(() => {
    setAPIAuthToken(proAuthToken);
  }, [proAuthToken]);

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
    void loadEditionMeta();
  }, []);

  useEffect(() => {
    if (!proAuthToken.trim() || !proPrincipal) return;
    void Promise.all([
      loadProjects(true),
      loadRAGCollections(),
      loadCatalog(true),
      loadInstalledPackages(),
      loadOllamaStatus(),
      loadLlamaCppStatus(),
    ]);
  }, [proAuthToken, proPrincipal]);

  useEffect(() => {
    if (!hasProAccess || activeView !== 'settings' || !proAuthToken.trim() || proPrincipal) {
      return;
    }

    void loadProSession(undefined, { restoring: true });
  }, [hasProAccess, activeView, proAuthToken, proPrincipal]);

  useEffect(() => {
    if (!proStorageConfig) {
      return;
    }

    setProStorageOperationalBackend(proStorageConfig.operational_backend ?? 'sqlite');
    setProStoragePostgresHost(proStorageConfig.operational_postgres_host ?? '');
    setProStoragePostgresPort(String(proStorageConfig.operational_postgres_port ?? 5432));
    setProStoragePostgresDatabase(proStorageConfig.operational_postgres_database ?? '');
    setProStoragePostgresSchema(proStorageConfig.operational_postgres_schema ?? 'public');
    setProStoragePostgresUsername(proStorageConfig.operational_postgres_username ?? '');
    setProStoragePostgresPassword('');
    setProStorageClearPostgresDSN(false);
    setProStorageLogsBackend(proStorageConfig.logs_backend ?? 'operational');
    setProStorageClickHouseURL(proStorageConfig.clickhouse_url ?? '');
    setProStorageClickHouseDatabase(proStorageConfig.clickhouse_database ?? '');
    setProStorageClickHouseUsername(proStorageConfig.clickhouse_username ?? '');
    setProStorageClickHousePassword('');
    setProStorageClearClickHousePassword(false);
    setProStorageElasticsearchURL(proStorageConfig.elasticsearch_url ?? '');
    setProStorageElasticsearchUsername(proStorageConfig.elasticsearch_username ?? '');
    setProStorageElasticsearchPassword('');
    setProStorageElasticsearchIndex(proStorageConfig.elasticsearch_index ?? '');
    setProStorageClearElasticsearchPassword(false);
  }, [proStorageConfig]);



  useEffect(() => {
    if (!proRagBoxConfig) {
      return;
    }

    setProRagBoxBaseURL(proRagBoxConfig.base_url ?? '');
    setProRagBoxAdminToken('');
    setProRagBoxClearAdminToken(false);
    setProRagBoxTimeoutSeconds(String(proRagBoxConfig.request_timeout_seconds || 30));
    setProRagBoxWriteMode(proRagBoxConfig.write_mode ?? 'dual_write');
    setProRagBoxSearchMode(proRagBoxConfig.search_mode ?? 'hybrid');
  }, [proRagBoxConfig]);

  useEffect(() => {
    if (!hasProAccess || activeView !== 'settings' || !proAuthToken.trim() || !proPrincipal) {
      return;
    }

    void loadProStorageContext().catch(() => {});
  }, [hasProAccess, activeView, proAuthToken, proPrincipal]);

  useEffect(() => {
    if (!hasProAccess || !((activeView === 'settings' && activeSettingsTab === 'ragbox') || ((activeView === 'knowledge' || activeView === 'projects') && createRAGCollectionOpen)) || !proAuthToken.trim() || !proPrincipal || !proRagBoxConfig?.enabled || !proRagBoxConfig?.admin_configured) {
      return;
    }

    void loadProRagBoxVectorConnections();
    void loadProRagBoxWorkspaces();
  }, [hasProAccess, activeView, activeSettingsTab, createRAGCollectionOpen, proAuthToken, proPrincipal, proRagBoxConfig]);

  useEffect(() => {
    if (!hasProAccess) {
      setProSSOConfig(null);
      return;
    }

    void (async () => {
      try {
        const config = await apiRequest<ProSSOConfig>('/api/pro/sso/config', messages.requestFailed);
        setProSSOConfig(config);
      } catch {
        setProSSOConfig(null);
      }
    })();
  }, [hasProAccess]);

  useEffect(() => {
    if (!hasProAccess && view === 'settings') {
      setView('projects');
    }
  }, [hasProAccess, view]);

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
      void loadLogMetrics();
    }
  }, [view, selectedLogsProjectId, metricsWindow]);

  useEffect(() => {
    const previousView = previousViewRef.current;
    previousViewRef.current = view;

    if (view !== 'market') {
      marketAutoSyncTriggeredRef.current = false;
      return;
    }

    if (previousView !== 'market') {
      marketAutoSyncTriggeredRef.current = false;
    }

    if (marketAutoSyncTriggeredRef.current || catalogLoading || catalogSyncing) {
      return;
    }

    if (catalogSourceMode === 'file' && localCatalogContent.trim() === '') {
      return;
    }

    marketAutoSyncTriggeredRef.current = true;
    void syncCatalog();
  }, [view, catalogLoading, catalogSyncing, catalogSourceMode, localCatalogContent]);

  useEffect(() => {
    if (view !== 'logs') {
      return;
    }

    const intervalID = window.setInterval(() => {
      void loadLogs({ silent: true });
      void loadLogMetrics({ silent: true });
    }, 5000);

    return () => window.clearInterval(intervalID);
  }, [view, selectedLogsProjectId, metricsWindow]);

  useEffect(() => {
    if (view !== 'logs' || !logsViewportRef.current) {
      return;
    }

    if (!logsStickToBottomRef.current) {
      return;
    }

    logsViewportRef.current.scrollTop = logsViewportRef.current.scrollHeight;
  }, [logs, view]);

  useEffect(() => {
    if (view === 'logs') {
      logsStickToBottomRef.current = true;
    }
  }, [view, selectedLogsProjectId, logsFilterMode]);

  function handleLogsViewportScroll() {
    const viewport = logsViewportRef.current;
    if (!viewport) {
      return;
    }
    const distanceToBottom =
      viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    logsStickToBottomRef.current = distanceToBottom <= 32;
  }

  useEffect(() => {
    if (projects.length === 0) {
      setSelectedProjectId(null);
      return;
    }

    if (
      !selectedProjectId ||
      !projects.some((project) => project.project_id === selectedProjectId)
    ) {
      let savedOrder: number[] = [];
      try {
        savedOrder = readProjectOrder(window.localStorage.getItem(projectOrderStorageKey));
      } catch {
        // Use API order when browser storage is unavailable.
      }
      const orderedIds = reconcileProjectOrder(projects.map(project => project.project_id), savedOrder);
      setSelectedProjectId(orderedIds[0]);
    }
  }, [projects, selectedProjectId, projectOrderStorageKey]);

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

  async function loadEditionMeta() {
    try {
      const meta = await apiRequest<EditionMeta>('/api/meta', messages.requestFailed);
      setEditionMeta(meta);
    } catch {
      setEditionMeta({
        edition_id: 'mcpbox',
        edition_name: 'MCPBox',
        capabilities: ['auth'],
      });
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
      if (hasProAccess && proAuthToken.trim()) {
        void loadProRagBoxKnowledgeBaseStatus();
      }
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    }
  }

  async function loadProRagBoxKnowledgeBaseStatus() {
    if (!hasProAccess || !proAuthToken.trim()) {
      setProRagBoxKnowledgeBaseStatus({});
      return;
    }
    try {
      const response = await proAPIRequest<{ items: ProRagBoxKnowledgeBaseStatus[] }>(
        proAuthToken,
        '/api/pro/ragbox/knowledge-bases',
        messages.requestFailed,
      );
      setProRagBoxKnowledgeBaseStatus(
        Object.fromEntries(
          response.items.map((item) => [item.collection.collection_id, item] as const),
        ),
      );
    } catch {
      setProRagBoxKnowledgeBaseStatus({});
    }
  }

  async function loadProRagBoxVectorConnections(config = proRagBoxConfig) {
    if (!hasProAccess || !proAuthToken.trim() || !config?.enabled || !config?.admin_configured) {
      setProRagBoxVectorConnections([]);
      return;
    }

    setProRagBoxLoadingVectorConnections(true);
    setProRagBoxConnectionError(null);
    try {
      const response = await proAPIRequest<ProRagBoxVectorConnectionListResponse>(
        proAuthToken,
        '/api/pro/ragbox/admin/vector-connections',
        messages.requestFailed,
      );
      setProRagBoxBuiltinFallbackBackend(response.builtin_fallback_backend || 'sqlite');
      setProRagBoxVectorConnections(response.items ?? []);
    } catch (loadError) {
      setProRagBoxVectorConnections([]);
      setProRagBoxConnectionError(loadError instanceof Error ? loadError.message : proCopy.ragboxVectorConnectionsLoadFailed);
    } finally {
      setProRagBoxLoadingVectorConnections(false);
    }
  }

  async function loadProRagBoxWorkspaces(config = proRagBoxConfig) {
    if (!hasProAccess || !proAuthToken.trim() || !config?.enabled || !config?.admin_configured) {
      setProRagBoxWorkspaces([]);
      return;
    }

    setProRagBoxLoadingWorkspaces(true);
    try {
      const response = await proAPIRequest<ProRagBoxWorkspaceListResponse>(
        proAuthToken,
        '/api/pro/ragbox/admin/workspaces',
        messages.requestFailed,
      );
      setProRagBoxWorkspaces(response.items ?? []);
    } catch (loadError) {
      setProRagBoxWorkspaces([]);
      setProRagBoxConnectionError(current => current ?? (loadError instanceof Error ? loadError.message : proCopy.ragboxVectorConnectionsLoadFailed));
    } finally {
      setProRagBoxLoadingWorkspaces(false);
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
      setLogs([...nextLogs].sort(
        (left, right) => new Date(left.created_at).getTime() - new Date(right.created_at).getTime(),
      ));
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      if (!options?.silent) {
        setLogsLoading(false);
      }
    }
  }

  async function clearLogs() {
    setLogsLoading(true);
    setActionError(null);
    try {
      const query = selectedLogsProjectId !== null ? `?project_id=${selectedLogsProjectId}` : '';
      await apiRequest<{ deleted: number }>(`/api/logs${query}`, messages.requestFailed, {
        method: 'DELETE',
      });
      await loadLogs({ silent: true });
      await loadLogMetrics({ silent: true });
      toast.success(language === 'ru' ? 'История очищена' : 'History cleared');
    } catch (clearError) {
      setActionError(clearError instanceof Error ? clearError.message : messages.requestFailed);
    } finally {
      setLogsLoading(false);
    }
  }

  async function loadLogMetrics(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setMetricsLoading(true);
    }
    try {
      const params = new URLSearchParams();
      params.set('window', metricsWindow);
      if (selectedLogsProjectId !== null) {
        params.set('project_id', String(selectedLogsProjectId));
      }
      const nextMetrics = await apiRequest<PerformanceMetricsResponse>(
        `/api/logs/metrics?${params.toString()}`,
        messages.requestFailed,
      );
      setLogMetrics(nextMetrics);
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      if (!options?.silent) {
        setMetricsLoading(false);
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
        if (response.settings.catalog_source_url.startsWith('local-file://')) {
          setCatalogSourceMode('file');
          setLocalCatalogFileName(response.settings.catalog_source_url.replace('local-file://', ''));
        } else {
          setCatalogSourceMode('server');
          setCatalogURL(normalizeCatalogSourceURL(response.settings.catalog_source_url));
        }
      }
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : 'Failed to load catalog');
    } finally {
      if (initial) {
        setCatalogLoading(false);
      }
    }
  }

  useEffect(() => {
    if (!selectedProject) {
      return;
    }

    setSelectedLlamaCppModelPath(selectedProject.llama_cpp_model_path || llamaCppStatus?.model_path || '');
    setSelectedLlamaCppModelName(
      selectedProject.llama_cpp_model_name ||
      modelNameFromPath(selectedProject.llama_cpp_model_path || '') ||
      llamaCppStatus?.model_name ||
      modelNameFromPath(llamaCppStatus?.model_path || ''),
    );
  }, [selectedProject?.project_id, selectedProject?.llama_cpp_model_path, selectedProject?.llama_cpp_model_name, llamaCppStatus?.model_path, llamaCppStatus?.model_name]);

  const loadOllamaStatus = hookLoadOllamaStatus;

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

  async function loadProStorageContext(tokenOverride?: string) {
    const activeToken = (tokenOverride ?? proAuthToken).trim();
    if (!activeToken) {
      setProStorageConfig(null);
      setProStorageRuntimeStatus(null);
      setProStorageShadowStatus(null);
      return;
    }

    try {
      const [config, runtimeStatus, shadowStatus] = await Promise.all([
        proAPIRequest<ProStorageConfig>(activeToken, '/api/pro/storage/config', messages.requestFailed),
        proAPIRequest<ProStorageRuntimeStatus>(activeToken, '/api/pro/storage/runtime-status', messages.requestFailed),
        proAPIRequest<ProStorageShadowStatus>(activeToken, '/api/pro/storage/shadow-status', messages.requestFailed),
      ]);
      setProStorageConfig(config);
      setProStorageRuntimeStatus(runtimeStatus);
      setProStorageShadowStatus(shadowStatus);
    } catch (loadError) {
      setProStorageConfig(null);
      setProStorageRuntimeStatus(null);
      setProStorageShadowStatus(null);
      throw loadError;
    }
  }

  async function loadProSession(tokenOverride?: string, options?: { restoring?: boolean }) {
    const activeToken = (tokenOverride ?? proAuthToken).trim();
    if (!activeToken) {
      setProPrincipal(null);
      setProTokens([]);
      setProUsers([]);
      setProRoles([]);
      setProSessions([]);
      setProStorageConfig(null);
      setProStorageRuntimeStatus(null);
      setProStorageShadowStatus(null);
      setProRagBoxConfig(null);
      return;
    }

    setProLoading(true);
    try {
      // Validate a saved session before loading any protected application data.
      const principal = await proAPIRequest<ProPrincipal>(activeToken, '/api/pro/auth/me', messages.requestFailed);
      const [tokens, users, roles, sessions] = await Promise.all([
        proAPIRequest<ProTokenRecord[]>(activeToken, '/api/pro/tokens', messages.requestFailed),
        proAPIRequest<ProUserRecord[]>(activeToken, '/api/pro/users', messages.requestFailed),
        proAPIRequest<ProRoleRecord[]>(activeToken, '/api/pro/roles', messages.requestFailed),
        proAPIRequest<ProSessionRecord[]>(activeToken, '/api/pro/sessions', messages.requestFailed),
      ]);
      setProPrincipal(principal);
      setProTokens(tokens);
      setProUsers(users);
      setProRoles(roles);
      setProSessions(sessions);

      const [ragboxResult] = await Promise.allSettled([
        proAPIRequest<ProRagBoxConfig>(activeToken, '/api/pro/ragbox/config', messages.requestFailed),
      ]);

      if (ragboxResult.status === 'fulfilled') {
        setProRagBoxConfig(ragboxResult.value);
        void loadProRagBoxKnowledgeBaseStatus();
        if (!ragboxResult.value.admin_configured) {
          setProRagBoxVectorConnections([]);
          setProRagBoxWorkspaces([]);
          setProRagBoxConnectionError(null);
        }
      } else {
        setProRagBoxConfig(null);
        setProRagBoxVectorConnections([]);
        setProRagBoxWorkspaces([]);
        setActionError(ragboxResult.reason instanceof Error ? ragboxResult.reason.message : proCopy.ragboxSaveFailed);
      }

      if (typeof window !== 'undefined') {
        window.localStorage.setItem(proAuthStorageKey, activeToken);
      }
    } catch (loadError) {
      const unauthorized = loadError instanceof APIRequestError && loadError.status === 401;
      resetProAccess(unauthorized);
      if (unauthorized) {
        setError(null);
        setActionError(null);
      }
      if (!options?.restoring || !unauthorized) {
        toast.error(loadError instanceof Error ? loadError.message : proCopy.loadTokensFailed);
      }
    } finally {
      setProLoading(false);
    }
  }

  function resetProAccess(clearToken = false) {
    setProPrincipal(null);
    setProTokens([]);
    setProUsers([]);
    setProRoles([]);
    setProSessions([]);
    setProStorageConfig(null);
    setProStorageRuntimeStatus(null);
    setProStorageShadowStatus(null);
    setProStoragePostgresHost('');
    setProStoragePostgresPort('5432');
    setProStoragePostgresDatabase('');
    setProStoragePostgresSchema('public');
    setProStoragePostgresUsername('');
    setProStoragePostgresPassword('');
    setProStorageClearPostgresDSN(false);
    setProStorageClickHousePassword('');
    setProStorageClearClickHousePassword(false);
    setProStorageElasticsearchPassword('');
    setProStorageClearElasticsearchPassword(false);
    setProRagBoxConfig(null);
    setProRagBoxAdminToken('');
    setProRagBoxClearAdminToken(false);
    setProRagBoxVectorConnections([]);
    setProRagBoxWorkspaces([]);
    setProRagBoxVectorConnectionForm(emptyProRagBoxVectorConnectionForm);
    setProRagBoxKnowledgeBaseStatus({});
    setProCreatedTokenValue(null);
    setProCreatedSessionToken(null);
    setProLoginPassword('');
    setProAuthBootstrapping(false);
    if (clearToken) {
      updateAndPersistProAuthToken('');
    }
  }

  async function connectProToken() {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProLoading(true);
    try {
      const principal = await proAPIRequest<ProPrincipal>(
        proAuthToken,
        '/api/pro/auth/me',
        messages.requestFailed,
      );
      setProPrincipal(principal);
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(proAuthStorageKey, proAuthToken.trim());
      }
      await loadProSession(proAuthToken);
    } catch (verifyError) {
      toast.error(verifyError instanceof Error ? verifyError.message : proCopy.verifyFailed);
    } finally {
      setProLoading(false);
    }
  }

  async function loginProLocal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setProLocalLoginLoading(true);
    setProAuthBootstrapping(true);
    try {
      const payload = await apiRequest<ProLocalLoginResponse>('/api/pro/login', messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify({
          email: proLoginEmail,
          password: proLoginPassword,
          expires_in_days: 30,
        }),
      });
      setProPrincipal({
        name: payload.user.display_name,
        scopes: payload.user.scopes,
        roles: payload.user.roles,
        user_id: payload.user.id,
        session_id: payload.session?.id,
        auth_method: payload.session?.auth_method ?? 'local_password',
        is_bootstrap: payload.user.is_bootstrap,
      });
      updateAndPersistProAuthToken(payload.token);
      setProLoginPassword('');
      setView('settings');
      await loadProSession(payload.token);
    } catch (loginError) {
      setProPrincipal(null);
      toast.error(loginError instanceof Error ? loginError.message : proCopy.verifyFailed);
    } finally {
      setProAuthBootstrapping(false);
      setProLocalLoginLoading(false);
    }
  }

  function startProSSOSignIn() {
    if (typeof window === 'undefined' || !proSSOConfig?.enabled || !proSSOConfig.start_url) {
      return;
    }
    const url = new URL(proSSOConfig.start_url, window.location.origin);
    url.searchParams.set('return_to', '/?pro_view=settings');
    window.location.assign(url.toString());
  }

  async function createProToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingToken(true);
    try {
      const payload = await proAPIRequest<CreateProTokenResponse>(
        proAuthToken,
        '/api/pro/tokens',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            name: proNewTokenName,
            scopes: proNewTokenScopes
              .split(',')
              .map((scope) => scope.trim())
              .filter(Boolean),
            expires_in_days: Number(proNewTokenExpiresDays || '0'),
          }),
        },
      );
      setProCreatedTokenValue(payload.token);
      setProCreateOpen(false);
      setProNewTokenName('');
      setProNewTokenScopes('pro:read, pro:write');
      setProNewTokenExpiresDays('30');
      await loadProSession();
    } catch (createError) {
      toast.error(createError instanceof Error ? createError.message : proCopy.createFailed);
    } finally {
      setProCreatingToken(false);
    }
  }

  async function revokeProToken(tokenId: number) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRevokingTokenId(tokenId);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/tokens/${tokenId}`,
        messages.requestFailed,
        { method: 'DELETE' },
      );
      await loadProSession();
    } catch (revokeError) {
      toast.error(revokeError instanceof Error ? revokeError.message : proCopy.revokeFailed);
    } finally {
      setProRevokingTokenId(null);
    }
  }

  async function createProUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingUser(true);
    try {
      await proAPIRequest<CreateProUserResponse>(
        proAuthToken,
        '/api/pro/users',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            display_name: proNewUserName,
            email: proNewUserEmail,
            role_names: proNewUserRoles
              .split(',')
              .map((role) => role.trim())
              .filter(Boolean),
          }),
        },
      );
      setProCreateUserOpen(false);
      setProNewUserName('');
      setProNewUserEmail('');
      setProNewUserRoles('reader');
      await loadProSession();
    } catch (createError) {
      toast.error(createError instanceof Error ? createError.message : proCopy.createUserFailed);
    } finally {
      setProCreatingUser(false);
    }
  }

  function startEditProUser(user: ProUserRecord) {
    setProEditingUserId(String(user.id));
    setProEditingUserName(user.display_name);
    setProEditingUserRoles(user.roles.join(', '));
    setProEditUserOpen(true);
  }

  function startProPasswordChange(user: ProUserRecord) {
    setProPasswordUserId(String(user.id));
    setProPasswordUserName(user.display_name);
    setProPasswordUserPassword('');
    setProPasswordUserPasswordConfirm('');
    setProPasswordUserOpen(true);
  }

  async function updateProUserRoles(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProUpdatingUserRoles(true);
    try {
      await proAPIRequest<UpdateProUserResponse>(
        proAuthToken,
        `/api/pro/users/${Number(proEditingUserId || '0')}`,
        messages.requestFailed,
        {
          method: 'PUT',
          body: JSON.stringify({
            role_names: proEditingUserRoles
              .split(',')
              .map((role) => role.trim())
              .filter(Boolean),
          }),
        },
      );
      setProEditUserOpen(false);
      setProEditingUserId('');
      setProEditingUserName('');
      setProEditingUserRoles('reader');
      await loadProSession();
    } catch (updateError) {
      toast.error(updateError instanceof Error ? updateError.message : proCopy.updateUserRolesFailed);
    } finally {
      setProUpdatingUserRoles(false);
    }
  }

  async function updateProUserPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }
    if (proPasswordUserPassword !== proPasswordUserPasswordConfirm) {
      toast.error(proCopy.passwordsDoNotMatch);
      return;
    }

    setProUpdatingUserPassword(true);
    try {
      await proAPIRequest<UpdateProUserPasswordResponse>(
        proAuthToken,
        `/api/pro/users/${Number(proPasswordUserId || '0')}/password`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            password: proPasswordUserPassword,
          }),
        },
      );
      setProPasswordUserOpen(false);
      setProPasswordUserId('');
      setProPasswordUserName('');
      setProPasswordUserPassword('');
      setProPasswordUserPasswordConfirm('');
      toast.success(proCopy.passwordChanged);
      await loadProSession();
    } catch (updateError) {
      toast.error(updateError instanceof Error ? updateError.message : proCopy.updateUserPasswordFailed);
    } finally {
      setProUpdatingUserPassword(false);
    }
  }

  async function disableProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }
    if (!window.confirm(proCopy.disableUserConfirm)) {
      return;
    }

    setProDisablingUserId(user.id);
    try {
      await proAPIRequest<DisableProUserResponse>(
        proAuthToken,
        `/api/pro/users/${user.id}/disable`,
        messages.requestFailed,
        { method: 'POST' },
      );
      if (proPrincipal?.user_id === user.id) {
        setProCreatedSessionToken(null);
      }
      await loadProSession();
    } catch (disableError) {
      toast.error(disableError instanceof Error ? disableError.message : proCopy.disableUserFailed);
    } finally {
      setProDisablingUserId(null);
    }
  }

  async function enableProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }
    if (!window.confirm(proCopy.enableUserConfirm)) {
      return;
    }

    setProEnablingUserId(user.id);
    try {
      await proAPIRequest<EnableProUserResponse>(
        proAuthToken,
        `/api/pro/users/${user.id}/enable`,
        messages.requestFailed,
        { method: 'POST' },
      );
      await loadProSession();
    } catch (enableError) {
      toast.error(enableError instanceof Error ? enableError.message : proCopy.enableUserFailed);
    } finally {
      setProEnablingUserId(null);
    }
  }

  async function deleteProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }
    if (!window.confirm(proCopy.deleteUserConfirm)) {
      return;
    }

    setProDeletingUserId(user.id);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/users/${user.id}`,
        messages.requestFailed,
        { method: 'DELETE' },
      );
      if (proEditingUserId === String(user.id)) {
        setProEditUserOpen(false);
        setProEditingUserId('');
        setProEditingUserName('');
        setProEditingUserRoles('reader');
      }
      if (proSessionsFilterUserId === String(user.id)) {
        setProSessionsFilterUserId('all');
      }
      await loadProSession();
    } catch (deleteError) {
      toast.error(deleteError instanceof Error ? deleteError.message : proCopy.deleteUserFailed);
    } finally {
      setProDeletingUserId(null);
    }
  }

  async function createProSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingSession(true);
    try {
      const payload = await proAPIRequest<CreateProSessionResponse>(
        proAuthToken,
        '/api/pro/sessions',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            user_id: Number(proNewSessionUserId || '0'),
            label: proNewSessionLabel,
            auth_method: 'local',
            expires_in_days: Number(proNewSessionExpiresDays || '0'),
          }),
        },
      );
      setProCreatedSessionToken(payload.token);
      setProCreateSessionOpen(false);
      setProNewSessionUserId('');
      setProNewSessionLabel('');
      setProNewSessionExpiresDays('30');
      await loadProSession();
    } catch (createError) {
      toast.error(createError instanceof Error ? createError.message : proCopy.createSessionFailed);
    } finally {
      setProCreatingSession(false);
    }
  }

  async function revokeProSession(sessionId: number) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    const isCurrentSession = proPrincipal?.session_id === sessionId;
    setProRevokingSessionId(sessionId);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/sessions/${sessionId}`,
        messages.requestFailed,
        { method: 'DELETE' },
      );
      if (isCurrentSession) {
        resetProAccess(true);
        return;
      }
      await loadProSession();
    } catch (revokeError) {
      toast.error(revokeError instanceof Error ? revokeError.message : proCopy.revokeSessionFailed);
    } finally {
      setProRevokingSessionId(null);
    }
  }

  async function revokeCurrentProSession() {
    if (!proPrincipal?.session_id) {
      return;
    }
    if (!window.confirm(proCopy.revokeCurrentSessionConfirm)) {
      return;
    }
    await revokeProSession(proPrincipal.session_id);
  }

  async function saveProStorageConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    const postgresFieldsChanged = (
      proStoragePostgresHost.trim() !== (proStorageConfig?.operational_postgres_host ?? '') ||
      (proStoragePostgresPort.trim() || '5432') !== String(proStorageConfig?.operational_postgres_port ?? 5432) ||
      proStoragePostgresDatabase.trim() !== (proStorageConfig?.operational_postgres_database ?? '') ||
      proStoragePostgresSchema.trim() !== (proStorageConfig?.operational_postgres_schema ?? 'public') ||
      proStoragePostgresUsername.trim() !== (proStorageConfig?.operational_postgres_username ?? '')
    );
    const shouldUpdatePostgresDSN = (
      proStorageOperationalBackend === 'postgres' && (
        !proStorageConfig?.operational_postgres_configured ||
        postgresFieldsChanged ||
        proStoragePostgresPassword.trim() !== ''
      )
    );

    if (shouldUpdatePostgresDSN && !proStoragePostgresPassword.trim()) {
      toast.error(proCopy.storagePostgresPasswordRequired);
      return;
    }
    setProStorageSaving(true);
    try {
      const config = await proAPIRequest<ProStorageConfig>(
        proAuthToken,
        '/api/pro/storage/config',
        messages.requestFailed,
        {
          method: 'PUT',
          body: JSON.stringify({
            operational_backend: proStorageOperationalBackend,
            postgres_host: shouldUpdatePostgresDSN ? proStoragePostgresHost : '',
            postgres_port: shouldUpdatePostgresDSN ? (Number.parseInt(proStoragePostgresPort, 10) || 5432) : 0,
            postgres_database: shouldUpdatePostgresDSN ? proStoragePostgresDatabase : '',
            postgres_schema: shouldUpdatePostgresDSN ? proStoragePostgresSchema : '',
            postgres_username: shouldUpdatePostgresDSN ? proStoragePostgresUsername : '',
            postgres_password: shouldUpdatePostgresDSN ? proStoragePostgresPassword : '',
            clear_postgres_dsn: proStorageClearPostgresDSN,
            logs_backend: proStorageLogsBackend,
            clickhouse_url: proStorageClickHouseURL,
            clickhouse_database: proStorageClickHouseDatabase,
            clickhouse_username: proStorageClickHouseUsername,
            clickhouse_password: proStorageClickHousePassword,
            clear_clickhouse_password: proStorageClearClickHousePassword,
            elasticsearch_url: proStorageElasticsearchURL,
            elasticsearch_username: proStorageElasticsearchUsername,
            elasticsearch_password: proStorageElasticsearchPassword,
            elasticsearch_index: proStorageElasticsearchIndex,
            clear_elasticsearch_password: proStorageClearElasticsearchPassword,
          }),
        },
      );
      setProStorageConfig(config);
      await loadProStorageContext(proAuthToken);
      toast.success(proCopy.storageSaved);
    } catch (saveError) {
      toast.error(saveError instanceof Error ? saveError.message : proCopy.storageSaveFailed);
    } finally {
      setProStorageSaving(false);
    }
  }

  async function saveProRagBoxConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRagBoxSaving(true);
    try {
      const config = await proAPIRequest<ProRagBoxConfig>(
        proAuthToken,
        '/api/pro/ragbox/config',
        messages.requestFailed,
        {
          method: 'PUT',
          body: JSON.stringify({
            base_url: proRagBoxBaseURL,
            admin_token: proRagBoxAdminToken,
            clear_admin_token: proRagBoxClearAdminToken,
            request_timeout_seconds: Number.parseInt(proRagBoxTimeoutSeconds, 10) || 30,
            write_mode: proRagBoxWriteMode,
            search_mode: proRagBoxSearchMode,
          }),
        },
      );
      setProRagBoxConfig(config);
      setProRagBoxAdminToken('');
      setProRagBoxClearAdminToken(false);
      if (config.configured) {
        await proAPIRequest(
          proAuthToken,
          '/api/pro/ragbox/knowledge-bases/sync',
          messages.requestFailed,
          { method: 'POST' },
        );
      }
      if (!config.admin_configured) {
        setProRagBoxConnectionError(null);
        setProRagBoxVectorConnections([]);
        setProRagBoxWorkspaces([]);
      }
      toast.success(proCopy.ragboxSaved);
    } catch (saveError) {
      toast.error(saveError instanceof Error ? saveError.message : proCopy.ragboxSaveFailed);
    } finally {
      setProRagBoxSaving(false);
    }
  }

  async function createProRagBoxVectorConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRagBoxCreatingVectorConnection(true);
    try {
      const backendType =
        proRagBoxVectorConnectionForm.backend_type === 'pgvector'
          ? 'pg_vector'
          : proRagBoxVectorConnectionForm.backend_type === 'opensearch'
            ? 'open_search'
            : proRagBoxVectorConnectionForm.backend_type;
      const payload: Record<string, unknown> = {
        name: proRagBoxVectorConnectionForm.name.trim(),
        backend_type: backendType,
        description: proRagBoxVectorConnectionForm.description.trim(),
        is_default: proRagBoxVectorConnectionForm.is_default,
      };
      if (proRagBoxVectorConnectionForm.backend_type === 'qdrant') {
        const qdrantURL = `${proRagBoxVectorConnectionForm.qdrant_scheme}://${proRagBoxVectorConnectionForm.qdrant_host.trim()}:${proRagBoxVectorConnectionForm.qdrant_port.trim()}`;
        payload.qdrant = {
          url: qdrantURL,
          api_key: proRagBoxVectorConnectionForm.qdrant_api_key.trim() || undefined,
          collection_prefix: proRagBoxVectorConnectionForm.qdrant_collection_prefix.trim(),
        };
      }
      if (proRagBoxVectorConnectionForm.backend_type === 'pgvector') {
        payload.pgvector = {
          host: proRagBoxVectorConnectionForm.pgvector_host.trim(),
          port: Number.parseInt(proRagBoxVectorConnectionForm.pgvector_port.trim(), 10) || 5432,
          database: proRagBoxVectorConnectionForm.pgvector_database_name.trim(),
          username: proRagBoxVectorConnectionForm.pgvector_username.trim(),
          password: proRagBoxVectorConnectionForm.pgvector_password,
          schema: proRagBoxVectorConnectionForm.pgvector_schema.trim(),
          table_prefix:
            proRagBoxVectorConnectionForm.pgvector_table_prefix.trim() ||
            proRagBoxVectorConnectionForm.name.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '') ||
            'ragbox',
        };
      }
      if (!['sqlite', 'qdrant', 'pgvector'].includes(proRagBoxVectorConnectionForm.backend_type)) {
        const rawConfig = proRagBoxVectorConnectionForm.backend_config_json.trim();
        if (rawConfig) {
          payload.backend_config = JSON.parse(rawConfig) as Record<string, unknown>;
        }
      }

      await proAPIRequest(
        proAuthToken,
        '/api/pro/ragbox/admin/vector-connections',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify(payload),
        },
      );
      setProRagBoxVectorConnectionForm(emptyProRagBoxVectorConnectionForm);
      await loadProRagBoxVectorConnections();
      await loadProRagBoxWorkspaces();
      toast.success(proCopy.ragboxVectorConnectionCreated);
    } catch (createError) {
      toast.error(createError instanceof Error ? createError.message : proCopy.ragboxVectorConnectionCreateFailed);
    } finally {
      setProRagBoxCreatingVectorConnection(false);
    }
  }

  async function setDefaultProRagBoxVectorConnection(connectionId: string) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRagBoxDefaultingVectorConnectionId(connectionId);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/ragbox/admin/vector-connections/${connectionId}/set-default`,
        messages.requestFailed,
        { method: 'POST' },
      );
      await loadProRagBoxVectorConnections();
      await loadProRagBoxWorkspaces();
      toast.success(proCopy.ragboxVectorConnectionDefaultSet);
    } catch (saveError) {
      toast.error(saveError instanceof Error ? saveError.message : proCopy.ragboxVectorConnectionDefaultFailed);
    } finally {
      setProRagBoxDefaultingVectorConnectionId(null);
    }
  }

  async function migratePolicyBoxProject() {
    if (!selectedProject || !canAdminPro) return;
    const question = language === 'ru'
      ? 'Заменить авторизацию PolicyBox на bearer-токен MCPBox? Токен будет заменён, проект останется на паузе. Проверьте доступ и настройте клиентов перед возобновлением.'
      : 'Replace PolicyBox authorization with an MCPBox bearer token? The credential will rotate and the project will stay paused. Review access and configure clients before resuming.';
    if (!window.confirm(question)) return;
    setBusyProjectId(selectedProject.project_id);
    try {
      const updated = await apiRequest<ProjectStatus>(`/api/projects/${selectedProject.project_id}/migrate-policybox`, messages.requestFailed, { method: 'POST' });
      setProjects((current) => current.map((project) => project.project_id === updated.project_id ? updated : project));
      await loadLogs();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : messages.requestFailed);
    } finally { setBusyProjectId(null); }
  }

  async function ensureRagBoxWorkspace() {
    if (!proAuthToken.trim() || !proRagBoxBindingProjectId) {
      return;
    }

    setProRagBoxEnsuringWorkspace(true);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/ragbox/projects/${proRagBoxBindingProjectId}/workspace/ensure`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            vector_connection_id: proRagBoxBindingVectorConnectionId,
            workspace_id: proRagBoxBindingExistingWorkspaceId,
          }),
        },
      );
      toast.success(proCopy.ragboxWorkspaceEnsured);
    } catch (ensureError) {
      toast.error(ensureError instanceof Error ? ensureError.message : proCopy.ragboxEnsureWorkspaceFailed);
    } finally {
      setProRagBoxEnsuringWorkspace(false);
    }
  }

  async function ensureRagBoxCollection() {
    if (!proAuthToken.trim() || !proRagBoxBindingCollectionProjectId || !proRagBoxBindingCollectionId) {
      return;
    }

    setProRagBoxEnsuringCollection(true);
    try {
      await proAPIRequest(
        proAuthToken,
        `/api/pro/ragbox/projects/${proRagBoxBindingCollectionProjectId}/collections/${proRagBoxBindingCollectionId}/ensure`,
        messages.requestFailed,
        { method: 'POST' },
      );
      toast.success(proCopy.ragboxCollectionEnsured);
    } catch (ensureError) {
      toast.error(ensureError instanceof Error ? ensureError.message : proCopy.ragboxEnsureCollectionFailed);
    } finally {
      setProRagBoxEnsuringCollection(false);
    }
  }

  async function uploadRagBoxFile() {
    if (!proAuthToken.trim() || !proRagBoxUploadProjectId || !proRagBoxUploadCollectionId || !proRagBoxUploadFile) {
      return;
    }

    setProRagBoxUploadingFile(true);
    try {
      const formData = new FormData();
      formData.append('file', proRagBoxUploadFile);
      if (proRagBoxUploadSourceName.trim()) {
        formData.append('source_name', proRagBoxUploadSourceName.trim());
      }
      if (proRagBoxUploadExternalId.trim()) {
        formData.append('external_id', proRagBoxUploadExternalId.trim());
      }
      if (proRagBoxUploadTitle.trim()) {
        formData.append('title', proRagBoxUploadTitle.trim());
      }
      if (proRagBoxUploadMetadata.trim()) {
        formData.append('metadata', proRagBoxUploadMetadata.trim());
      }

      const response = await fetch(
        `/api/pro/ragbox/projects/${proRagBoxUploadProjectId}/collections/${proRagBoxUploadCollectionId}/files:upload`,
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${proAuthToken.trim()}`,
          },
          body: formData,
        },
      );

      if (!response.ok) {
        let message = proCopy.ragboxUploadFileFailed;
        try {
          const payload = (await response.json()) as { error?: string };
          if (payload?.error) {
            message = payload.error;
          }
        } catch {}
        throw new Error(message);
      }

      const operation = (await response.json()) as ProRagBoxWriteResponse;
      setProRagBoxLastOperation(operation);

      setProRagBoxUploadFile(null);
      setProRagBoxUploadSourceName('');
      setProRagBoxUploadExternalId('');
      setProRagBoxUploadTitle('');
      setProRagBoxUploadMetadata('');
      toast.success(proCopy.ragboxFileUploaded);
    } catch (uploadError) {
      toast.error(uploadError instanceof Error ? uploadError.message : proCopy.ragboxUploadFileFailed);
    } finally {
      setProRagBoxUploadingFile(false);
    }
  }

  async function upsertRagBoxRecords() {
    if (!proAuthToken.trim() || !proRagBoxRecordsProjectId || !proRagBoxRecordsCollectionId || !proRagBoxRecordsPayload.trim()) {
      return;
    }

    setProRagBoxUpsertingRecords(true);
    try {
      const parsed = JSON.parse(proRagBoxRecordsPayload);
      const operation = await proAPIRequest<ProRagBoxWriteResponse>(
        proAuthToken,
        `/api/pro/ragbox/projects/${proRagBoxRecordsProjectId}/collections/${proRagBoxRecordsCollectionId}/records:upsert`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify(parsed),
        },
      );
      setProRagBoxLastOperation(operation);
      toast.success(proCopy.ragboxRecordsUpserted);
    } catch (upsertError) {
      toast.error(upsertError instanceof Error ? upsertError.message : proCopy.ragboxUpsertRecordsFailed);
    } finally {
      setProRagBoxUpsertingRecords(false);
    }
  }

  async function signOutPro() {
    if (proPrincipal?.session_id) {
      await revokeCurrentProSession();
      return;
    }
    resetProAccess(true);
    setView('projects');
  }

  async function syncCatalog() {
    setCatalogSyncing(true);
    setActionError(null);

    try {
      if (catalogSourceMode === 'file' && localCatalogContent.trim() === '') {
        toast.error(messages.localCatalogFileMissing);
        return;
      }
      const response = await apiRequest<CatalogResponse>(
        '/api/catalog/sync',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify(
            catalogSourceMode === 'file'
              ? { manifest_content: localCatalogContent, file_name: localCatalogFileName }
              : { url: normalizeCatalogSourceURL(catalogURL) || defaultCatalogSourceURL },
          ),
        },
      );
      setCatalogItems(response.items);
      setCatalogSettings(response.settings);
      await loadInstalledPackages();
      await loadLogs({ silent: true });
    } catch (syncError) {
      toast.error(syncError instanceof Error ? syncError.message : 'Failed to sync catalog');
    } finally {
      setCatalogSyncing(false);
    }
  }

  async function pickLocalCatalogFile(file: File | null) {
    if (!file) {
      setLocalCatalogFileName('');
      setLocalCatalogContent('');
      return;
    }
    const text = await file.text();
    setLocalCatalogFileName(file.name);
    setLocalCatalogContent(text);
    setCatalogSourceMode('file');
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
      toast.error(
        installError instanceof Error ? installError.message : messages.installPackageError,
      );
      return false;
    } finally {
      setInstallingCatalogItemId(null);
    }
  }

  async function uninstallCatalogPackage(item: CatalogItem, pkg: InstalledPackage) {
    if (pkg.project_use_count > 0) {
      toast.error(messages.packageInUseCannotUninstall);
      return false;
    }

    const confirmed = window.confirm(`${labels.uninstallPackage}: ${item.name}?`);
    if (!confirmed) {
      return false;
    }

    setUninstallingCatalogItemId(item.id);
    setActionError(null);

    try {
      await apiRequest<{ deleted: boolean }>(
        `/api/packages/${pkg.id}`,
        messages.requestFailed,
        {
          method: 'DELETE',
        },
      );
      await loadInstalledPackages();
      await loadCatalog();
      await loadLogs({ silent: true });
      return true;
    } catch (uninstallError) {
      toast.error(
        uninstallError instanceof Error ? uninstallError.message : messages.uninstallPackageError,
      );
      return false;
    } finally {
      setUninstallingCatalogItemId(null);
    }
  }

  async function performCatalogInstall(
    item: CatalogItem,
    projectId: number,
    config: Record<string, unknown>,
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
      const installedServerId =
        updatedProject.installed_integrations.find(
          (integration) => integration.catalog_item_id === item.id && integration.server_id,
        )?.server_id ?? null;
      const installedServer =
        (installedServerId
          ? updatedProject.servers.find((server) => server.id === installedServerId)
          : updatedProject.servers.find((server) => server.name === item.name)) ?? null;
      if (installedServer?.health_status === 'healthy') {
        toast.success(messages.catalogHealthCheckPassed(item.name));
      } else if (installedServer?.health_status === 'failed') {
        toast.warning(
          installedServer.health_error
            ? messages.catalogHealthCheckFailedWithReason(item.name, installedServer.health_error)
            : messages.catalogHealthCheckFailed(item.name),
        );
      } else {
        toast.success(messages.catalogInstallAdded(item.name));
      }
      await loadInstalledPackages();
      await loadLogs({ silent: true });
      return true;
    } catch (installError) {
      toast.error(
        installError instanceof Error ? installError.message : messages.addPackageToProjectError,
      );
      return false;
    } finally {
      setAddingCatalogItemId(null);
    }
  }

  function projectNameFromLog(projectId: number | null): string {
    if (!projectId) {
      return messages.projectTag(0);
    }

    return (
      projects.find((project) => project.project_id === projectId)?.name ??
      messages.projectTag(projectId)
    );
  }

  function serverNameFromLog(serverId: number | null): string {
    if (!serverId) {
      return messages.serverTag(0);
    }

    return serverNamesById[serverId] ?? messages.serverTag(serverId);
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatingProject(true);
    setActionError(null);

    try {
      const projectPayload = {
        name: projectForm.name,
        description: projectForm.description,
        root_path: projectForm.root_path,
        bearer_auth_enabled: projectForm.bearer_auth_enabled,
        oauth_redirect_uri: projectForm.oauth_redirect_uri,
      };

      if (editingProjectId) {
        const updatedProject = await apiRequest<ProjectStatus>(
          `/api/projects/${editingProjectId}`,
          messages.requestFailed,
          {
            method: 'PUT',
            body: JSON.stringify(projectPayload),
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
          body: JSON.stringify(projectPayload),
        });
        await loadProjects();
      }

      setProjectForm(defaultProjectForm());
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

  async function updateProjectPromptConfig(prompt: string, promptProfiles: ProjectPromptProfile[]) {
    if (!selectedProject) {
      return;
    }
    setUpdatingPrompt(true);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}`,
        messages.requestFailed,
        {
          method: 'PUT',
          body: JSON.stringify({
            name: selectedProject.name,
            description: selectedProject.description,
            root_path: selectedProject.root_path,
            bearer_auth_enabled: selectedProject.bearer_auth_enabled,
            oauth_redirect_uri: selectedProject.oauth_redirect_uri,
            prompt,
            prompt_profiles: promptProfiles,
          }),
        },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.createProjectError,
      );
    } finally {
      setUpdatingPrompt(false);
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

  async function regenerateProjectBearerToken(projectId: number) {
    setBusyProjectId(projectId);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${projectId}/bearer-token`,
        messages.requestFailed,
        { method: 'POST' },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      setProjectForm((current) => ({
        ...current,
        bearer_auth_enabled: updatedProject.bearer_auth_enabled,
        bearer_token: updatedProject.bearer_token,
        oauth_redirect_uri: updatedProject.oauth_redirect_uri,
      }));
      toast.success(messages.bearerTokenRegenerated);
      await loadLogs({ silent: true });
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.createProjectError);
    } finally {
      setBusyProjectId(null);
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
      setLaunchProjectOpen(false);
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.launchOllamaError);
    } finally {
      setLaunchingOllamaProjectId(null);
    }
  }

  async function launchProjectLlamaCpp(projectId: number) {
    setLaunchingLlamaCppProjectId(projectId);
    setActionError(null);

    try {
      await apiRequest<LlamaCppLaunchResponse>(
        `/api/projects/${projectId}/launch-llamacpp`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            model_path: selectedLlamaCppModelPath.trim(),
            model_name: selectedLlamaCppModelName.trim(),
          }),
        },
      );
      await Promise.all([loadProjects(), loadLlamaCppStatus()]);
      setLaunchProjectOpen(false);
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.launchLlamaCppError,
      );
    } finally {
      setLaunchingLlamaCppProjectId(null);
    }
  }

  async function stopLlamaCppServer() {
    setStoppingLlamaCpp(true);
    setActionError(null);

    try {
      await apiRequest<LlamaCppStatus>(
        '/api/llamacpp/stop',
        messages.requestFailed,
        { method: 'POST' },
      );
      await loadLlamaCppStatus();
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.launchLlamaCppError,
      );
    } finally {
      setStoppingLlamaCpp(false);
    }
  }

  async function launchProjectLMStudio(projectId: number) {
    setLaunchingLMStudioProjectId(projectId);
    setActionError(null);

    try {
      await apiRequest<LMStudioLaunchResponse>(
        `/api/projects/${projectId}/launch-lmstudio`,
        messages.requestFailed,
        {
          method: 'POST',
        },
      );
      setLaunchProjectOpen(false);
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.launchLMStudioError,
      );
    } finally {
      setLaunchingLMStudioProjectId(null);
    }
  }

  async function copyConnectURL(url?: string) {
    const value = url ?? selectedProject?.connect_url;
    if (!value) {
      return;
    }

    await copyToClipboard(value);
    toast.success(labels.copied);
  }

  async function copyToClipboard(value: string) {
    await navigator.clipboard.writeText(value);
  }

  function updateAndPersistProAuthToken(value: string) {
    setAPIAuthToken(value);
    setProAuthToken(value);
    if (typeof window !== 'undefined') {
      if (value.trim()) {
        window.localStorage.setItem(proAuthStorageKey, value);
      } else {
        window.localStorage.removeItem(proAuthStorageKey);
      }
    }
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
    setInspection(null);
    setInspectionError(null);
    setInspectionServerName(server.name);

    if (server.transport !== 'stdio') {
      setInspectingServerId(null);
      return;
    }

    setInspectingServerId(server.id);

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

  async function openServerTools(server: ServerStatus) {
    setServerToolsOpen(true);
    setServerToolsLoadingId(server.id);
    setServerToolsSavingName(null);
    setServerToolsServerId(server.id);
    setServerToolsServerName(server.name);
    setServerTools([]);
    setServerToolsError(null);

    try {
      const payload = await apiRequest<ServerToolStatus[]>(
        `/api/servers/${server.id}/tools`,
        messages.requestFailed,
      );
      setServerTools(payload);
    } catch (loadError) {
      setServerToolsError(
        loadError instanceof Error ? loadError.message : messages.loadServerToolsError,
      );
    } finally {
      setServerToolsLoadingId(null);
    }
  }

  async function setServerToolEnabled(toolName: string, enabled: boolean) {
    if (!serverToolsServerId) {
      return;
    }

    const nextTools = serverTools.map((tool) =>
      tool.name === toolName ? { ...tool, enabled } : tool,
    );
    const disabledTools = nextTools.filter((tool) => !tool.enabled).map((tool) => tool.name);

    setServerToolsSavingName(toolName);
    setServerToolsError(null);

    try {
      const payload = await apiRequest<ServerToolStatus[]>(
        `/api/servers/${serverToolsServerId}/tools`,
        messages.requestFailed,
        {
          method: 'PUT',
          body: JSON.stringify({ disabled_tools: disabledTools }),
        },
      );
      setServerTools(payload);
      await loadProjects();
      await loadLogs({ silent: true });
    } catch (submitError) {
      setServerToolsError(
        submitError instanceof Error ? submitError.message : messages.updateServerToolsError,
      );
    } finally {
      setServerToolsSavingName(null);
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
      bearer_auth_enabled: selectedProject.bearer_auth_enabled,
      bearer_token: selectedProject.bearer_token,
      oauth_redirect_uri: selectedProject.oauth_redirect_uri,
    });
    setEditingProjectId(selectedProject.project_id);
    setCreateProjectOpen(true);
  }

  function prepareCreateProject() {
    setEditingProjectId(null);
    setProjectForm(defaultProjectForm());
  }

  function startDuplicateProject() {
    if (!selectedProject) {
      return;
    }

    setDuplicateProjectName(`${selectedProject.name} Copy`);
    setDuplicateProjectOpen(true);
  }

  async function duplicateProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }

    setDuplicatingProjectId(selectedProject.project_id);
    setActionError(null);

    try {
      const duplicatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}/duplicate`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ name: duplicateProjectName }),
        },
      );
      setProjects((current) => [...current, duplicatedProject]);
      setSelectedProjectId(duplicatedProject.project_id);
      setDuplicateProjectOpen(false);
      setDuplicateProjectName('');
      await loadInstalledPackages();
      await loadLogs({ silent: true });
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.duplicateProjectError,
      );
    } finally {
      setDuplicatingProjectId(null);
    }
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
      const response = await apiRequest<ServerActionResponse>(
        `/api/servers/${serverId}/check`,
        messages.requestFailed,
        {
          method: 'POST',
        },
      );

      const serverName =
        projects.flatMap((project) => project.servers).find((server) => server.id === serverId)?.name ??
        `Server ${serverId}`;

      if (response.health_status === 'healthy') {
        toast.success(messages.checkServerHealthy(serverName));
      } else {
        toast.error(
          messages.checkServerFailed(
            serverName,
            response.health_error || messages.checkServerError,
          ),
        );
      }

      await loadProjects();
      await loadLogs();
    } catch (submitError) {
      toast.error(
        submitError instanceof Error ? submitError.message : messages.checkServerError,
      );
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
    setActionError(null);
    setLinkingCollectionId(collectionId);

    try {
      const preview = await proAPIRequest<{
        collection_id: string;
        managed_source_path: string;
        has_uploaded_files: boolean;
        uploaded_files_count: number;
      }>(
        proAuthToken,
        `/api/pro/ragbox/collections/${collectionId}/delete-preview`,
        messages.requestFailed,
      );

      const confirmed = window.confirm(messages.deleteKnowledgeBaseConfirm);
      if (!confirmed) {
        return;
      }

      let removeUploadedFiles = false;
      if (preview.has_uploaded_files) {
        const deleteFilesMessage = language === 'ru'
          ? `У этой базы знаний есть загруженные файлы: ${preview.uploaded_files_count}.\n\nУдалить их вместе с базой знаний?\n\n${preview.managed_source_path}`
          : `This knowledge base has uploaded files: ${preview.uploaded_files_count}.\n\nDelete them together with the knowledge base?\n\n${preview.managed_source_path}`;
        removeUploadedFiles = window.confirm(deleteFilesMessage);
      }

      await proAPIRequest<{ deleted: boolean }>(
        proAuthToken,
        `/api/pro/ragbox/collections/${collectionId}?remove_uploaded_files=${removeUploadedFiles ? 'true' : 'false'}`,
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
    setRAGCollectionForm({
      name: collection.name,
      source_path: collection.source_path ?? '',
      auto_reindex: collection.auto_reindex ?? false,
      service_mode: collection.service_mode ?? 'bleve_only',
      vector_connection_id: collection.vector_connection_id ?? '',
      existing_workspace_id: proRagBoxKnowledgeBaseStatus[collection.collection_id]?.binding?.ragbox_workspace_id ?? '',
    });
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
            source_path: ragCollectionForm.source_path,
            auto_reindex: ragCollectionForm.auto_reindex,
            service_mode: ragCollectionForm.service_mode,
            vector_connection_id: ragCollectionForm.vector_connection_id,
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
      setRAGIndexPaths((current) => ({
        ...current,
        [savedCollection.collection_id]: savedCollection.source_path,
      }));
      if (
        hasProAccess &&
        proAuthToken.trim() &&
        proRagBoxConfig?.enabled &&
        proRagBoxConfig?.configured &&
        ragCollectionForm.service_mode !== 'bleve_only'
      ) {
        try {
          await proAPIRequest(
            proAuthToken,
            `/api/pro/ragbox/collections/${savedCollection.collection_id}/ensure`,
            messages.requestFailed,
            {
              method: 'POST',
              body: JSON.stringify({
                workspace_id: ragCollectionForm.existing_workspace_id,
              }),
            },
          );
          if (ragCollectionForm.source_path.trim()) {
            await proAPIRequest(
              proAuthToken,
              `/api/pro/ragbox/collections/${savedCollection.collection_id}/sync-source`,
              messages.requestFailed,
              { method: 'POST' },
            );
          }
        } catch (ensureError) {
          toast.error(ensureError instanceof Error ? ensureError.message : proCopy.ragboxEnsureCollectionFailed);
        }
      }
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

  async function createAndConnectRAGCollectionToProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }

    setCreatingRAGCollection(true);
    setActionError(null);

    try {
      const savedCollection = await apiRequest<RAGCollection>(
        '/api/rag/collections',
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({
            name: ragCollectionForm.name,
            source_path: ragCollectionForm.source_path,
            auto_reindex: ragCollectionForm.auto_reindex,
            service_mode: ragCollectionForm.service_mode,
            vector_connection_id: ragCollectionForm.vector_connection_id,
          }),
        },
      );

      setAllRAGCollections((current) => [...current, savedCollection]);
      setRAGIndexPaths((current) => ({
        ...current,
        [savedCollection.collection_id]: savedCollection.source_path,
      }));

      if (
        hasProAccess &&
        proAuthToken.trim() &&
        proRagBoxConfig?.enabled &&
        proRagBoxConfig?.configured &&
        ragCollectionForm.service_mode !== 'bleve_only'
      ) {
        try {
          await proAPIRequest(
            proAuthToken,
            `/api/pro/ragbox/collections/${savedCollection.collection_id}/ensure`,
            messages.requestFailed,
            {
              method: 'POST',
              body: JSON.stringify({
                workspace_id: ragCollectionForm.existing_workspace_id,
              }),
            },
          );
          if (ragCollectionForm.source_path.trim()) {
            await proAPIRequest(
              proAuthToken,
              `/api/pro/ragbox/collections/${savedCollection.collection_id}/sync-source`,
              messages.requestFailed,
              { method: 'POST' },
            );
          }
        } catch (ensureError) {
          toast.error(ensureError instanceof Error ? ensureError.message : proCopy.ragboxEnsureCollectionFailed);
        }
      }

      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}/rag-collections`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ collection_id: savedCollection.collection_id }),
        },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      setRAGCollectionForm(emptyRAGCollectionForm);
      setConnectRAGCollectionOpen(false);
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
    const targetCollection = allRAGCollections.find((collection) => collection.collection_id === collectionId);

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
      if (
        hasProAccess &&
        proAuthToken.trim() &&
        proRagBoxConfig?.enabled &&
        proRagBoxConfig?.configured &&
        targetCollection?.service_mode !== 'bleve_only'
      ) {
        try {
          await proAPIRequest(
            proAuthToken,
            `/api/pro/ragbox/collections/${collectionId}/sync-source`,
            messages.requestFailed,
            { method: 'POST' },
          );
        } catch (syncError) {
          toast.error(syncError instanceof Error ? syncError.message : proCopy.ragboxUploadFileFailed);
        }
      }
      await loadLogs();
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.loadProjectsError);
    } finally {
      setIndexingCollectionId(null);
    }
  }

  async function uploadKnowledgeBaseFile(
    collectionId: string,
    file: File,
    storageMode: 'keep' | 'transient',
  ) {
    if (!hasProAccess || !proAuthToken.trim()) {
      setActionError(proCopy.verifyFailed);
      return;
    }

    setUploadingKnowledgeBaseId(collectionId);
    setActionError(null);

    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('storage_mode', storageMode);

      await proAPIRequest<ProRagBoxWriteResponse>(
        proAuthToken,
        `/api/pro/ragbox/collections/${collectionId}/files:upload`,
        messages.requestFailed,
        {
          method: 'POST',
          body: formData,
        },
      );

      await Promise.all([
        loadRAGCollections(),
        loadProRagBoxKnowledgeBaseStatus(),
        loadLogs(),
      ]);
      toast.success(proCopy.ragboxFileUploaded);
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : proCopy.ragboxUploadFileFailed;
      setActionError(message);
      toast.error(message);
    } finally {
      setUploadingKnowledgeBaseId(null);
    }
  }

  function canUseProRagBoxSearch(collectionId: string) {
    const targetCollection = allRAGCollections.find((collection) => collection.collection_id === collectionId);
    const serviceMode = targetCollection?.service_mode ?? 'bleve_only';
    return (
      hasProAccess &&
      proAuthToken.trim() !== '' &&
      !!proRagBoxConfig?.enabled &&
      !!proRagBoxConfig?.configured &&
      (serviceMode === 'ragbox_only' || proRagBoxConfig.search_mode !== 'bleve_only')
    );
  }

  function mapProSearchHits(items: ProRagBoxSearchResponse['items']): RAGSearchResult[] {
    return items.map((item) => ({
      id: item.id,
      file_path: item.file_path || item.title || item.id,
      section: item.section,
      title: item.title,
      content: item.content,
      source: item.source,
      score: item.score,
    }));
  }

  function formatRagSearchDiagnostics(diagnostics: ProRagBoxSearchResponse['diagnostics']) {
    return proCopy.ragboxSearchDiagnostics
      .replace('{bleve}', String(diagnostics.bleve_count))
      .replace('{ragbox}', String(diagnostics.ragbox_count))
      .replace('{merged}', String(diagnostics.merged_count))
      .replace('{mode}', diagnostics.mode);
  }

  async function searchRAGCollection(collectionId: string) {
    const query = (ragSearchQueries[collectionId] ?? '').trim();
    if (!query) {
      setRAGSearchErrors(current => ({ ...current, [collectionId]: messages.searchQueryRequired }));
      return;
    }

    setSearchingCollectionId(collectionId);
    setActionError(null);
    setRAGSearchDiagnosticsByCollection(current => ({ ...current, [collectionId]: null }));
    setRAGSearchErrors(current => ({ ...current, [collectionId]: null }));

    try {
      if (canUseProRagBoxSearch(collectionId)) {
        const response = await proAPIRequest<ProRagBoxSearchResponse>(
          proAuthToken,
          `/api/pro/ragbox/collections/${collectionId}/search`,
          messages.requestFailed,
          {
            method: 'POST',
            body: JSON.stringify({ query, limit: 5 }),
          },
        );
        setRAGSearchResults((current) => ({
          ...current,
          [collectionId]: mapProSearchHits(response.items),
        }));
        setRAGSearchDiagnosticsByCollection(current => ({ ...current, [collectionId]: formatRagSearchDiagnostics(response.diagnostics) }));
      } else {
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
      }

    } catch (submitError) {
      setRAGSearchErrors(current => ({ ...current, [collectionId]: submitError instanceof Error ? submitError.message : messages.loadProjectsError }));
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
    <AppShell
      language={language}
      logoSrc={logo}
      labels={labels}
      navigationItems={visibleNavigationItems}
      view={activeView}
      setView={setView}
      immersive={proAuthLocked}
      footer={
        !proAuthLocked && proPrincipal ? (
          <button
            onClick={() => void signOutPro()}
            aria-label={proCopy.signOut}
            className="inline-flex h-12 w-12 items-center justify-center rounded-2xl border border-transparent bg-card text-muted-foreground transition-colors hover:border-border hover:bg-accent hover:text-destructive"
          >
            <LogOut className="h-5 w-5" />
          </button>
        ) : null
      }
      sidebar={
        activeView === 'projects' && !proAuthLocked ? (
          <ProjectsSidebar
            key={`project-order:${proPrincipal?.user_id ?? 'local'}`}
            orderStorageKey={projectOrderStorageKey}
            labels={labels}
            messages={messages}
            language={language}
            languageOptions={languageOptions}
            setLanguage={setLanguage}
            onRefresh={async () => { await Promise.all([loadProjects(), loadOllamaStatus(), loadLlamaCppStatus()]); }}
            refreshing={refreshing}
            projects={projects}
            loading={loading}
            selectedProjectId={selectedProjectId}
            setSelectedProjectId={setSelectedProjectId}
            createProjectOpen={createProjectOpen}
            setCreateProjectOpen={setCreateProjectOpen}
            editingProjectId={editingProjectId}
            projectForm={projectForm}
            setProjectForm={setProjectForm}
            prepareCreateProject={prepareCreateProject}
            resetProjectForm={() => {
              setEditingProjectId(null);
              setProjectForm(defaultProjectForm());
            }}
            createProject={createProject}
            creatingProject={creatingProject}
            duplicateProjectOpen={duplicateProjectOpen}
            setDuplicateProjectOpen={setDuplicateProjectOpen}
            duplicateProjectName={duplicateProjectName}
            setDuplicateProjectName={setDuplicateProjectName}
            duplicateProject={duplicateProject}
            duplicatingProjectId={duplicatingProjectId}
            selectedProject={selectedProject}

          />
        ) : null
      }
      error={error}
      actionError={actionError}
    >
      <Toaster position="top-right" richColors />
      {activeView === 'projects' && selectedProject?.identity_verification_enabled ? (
        <div role="alert" className="mb-4 rounded-xl border border-amber-500/40 bg-amber-500/10 p-4">
          <p className="font-semibold">{language === 'ru' ? 'Требуется переход с PolicyBox' : 'PolicyBox migration required'}</p>
          <p className="mt-2 text-sm">{language === 'ru'
            ? 'Подключения к этому проекту заблокированы. Перейдите на bearer-токен MCPBox, проверьте разрешения серверов и клиентов, затем снимите проект с паузы. Правила PolicyBox не переносятся автоматически.'
            : 'Connections to this project are blocked. Switch to an MCPBox bearer token, review server and client permissions, then resume the project. PolicyBox rules are not converted automatically.'}</p>
          {canAdminPro ? <button className="mt-3 rounded-md bg-brand-solid px-4 py-2 text-sm text-white disabled:opacity-50" disabled={busyProjectId === selectedProject.project_id} onClick={() => void migratePolicyBoxProject()}>
            {language === 'ru' ? 'Перейти на bearer и поставить на паузу' : 'Migrate to bearer and pause'}
          </button> : <p className="mt-2 text-sm">{language === 'ru' ? 'Обратитесь к администратору MCPBox.' : 'Contact your MCPBox administrator.'}</p>}
        </div>
      ) : null}
      <Suspense fallback={viewLoadingFallback}>
        {activeView === 'logs' ? (
            <LogsView
              labels={labels}
              messages={messages}
              projects={projects}
              selectedLogsProjectId={selectedLogsProjectId}
              setSelectedLogsProjectId={setSelectedLogsProjectId}
              metricsWindow={metricsWindow}
              setMetricsWindow={setMetricsWindow}
              metricsWindowOptions={metricsWindowOptions}
              logsLoading={logsLoading}
              metricsLoading={metricsLoading}
              onRefresh={() => {
                void loadLogs();
                void loadLogMetrics();
              }}
              onClearLogs={() => {
                void clearLogs();
              }}
              logMetrics={logMetrics}
              filteredLogsProjectName={filteredLogsProject?.name ?? null}
              requestTrendValues={requestTrendValues}
              errorTrendValues={errorTrendValues}
              avgLatencyTrendValues={avgLatencyTrendValues}
              p95LatencyTrendValues={p95LatencyTrendValues}
              hasProAccess={hasProAccess}
              logsFilterMode={logsFilterMode}
              setLogsFilterMode={setLogsFilterMode}
              proActivityCopy={proActivityCopy}
              visibleLogs={visibleLogs}
              proCreatedCount={proCreatedCount}
              proUsedCount={proUsedCount}
              proRevokedCount={proRevokedCount}
              logsViewportRef={logsViewportRef}
              onLogsViewportScroll={handleLogsViewportScroll}
              projectNameFromLog={projectNameFromLog}
              serverNameFromLog={serverNameFromLog}
              serverNamesById={serverNamesById}
              projectNamesById={projectNamesById}
            />
          ) : activeView === 'skills' ? (
 <SkillsView language={language} />
 ) : activeView === 'cli' ? (
            <CLIView language={language} />
          ) : activeView === 'knowledge' ? (
            <KnowledgeView
              language={language}
              labels={labels}
              messages={messages}
              allRAGCollections={allRAGCollections}
              createRAGCollectionOpen={createRAGCollectionOpen}
              setCreateRAGCollectionOpen={setCreateRAGCollectionOpen}
              editingRAGCollectionId={editingRAGCollectionId}
              ragCollectionForm={ragCollectionForm}
              setRAGCollectionForm={setRAGCollectionForm}
              resetRAGCollectionForm={() => {
                setRAGCollectionForm(emptyRAGCollectionForm);
                setEditingRAGCollectionId(null);
              }}
              createRAGCollection={createRAGCollection}
              creatingRAGCollection={creatingRAGCollection}
              startEditRAGCollection={startEditRAGCollection}
              deleteRAGCollection={deleteRAGCollection}
              linkingCollectionId={linkingCollectionId}
              ragIndexPaths={ragIndexPaths}
              setRAGIndexPaths={setRAGIndexPaths}
              indexRAGCollection={indexRAGCollection}
              indexingCollectionId={indexingCollectionId}
              ragSearchQueries={ragSearchQueries}
              setRAGSearchQueries={setRAGSearchQueries}
              searchRAGCollection={searchRAGCollection}
              searchingCollectionId={searchingCollectionId}
              ragSearchResultsOpen={ragSearchResultsOpen}
              setRAGSearchResultsOpen={setRAGSearchResultsOpen}
              activeRAGSearchCollection={activeRAGSearchCollection}
              setActiveRAGSearchCollectionId={setActiveRAGSearchCollectionId}
              activeRAGSearchResults={activeRAGSearchResults}
              ragSearchDiagnostics={ragSearchDiagnostics}
              ragSearchError={ragSearchError}
              uploadKnowledgeBaseFile={uploadKnowledgeBaseFile}
              uploadingKnowledgeBaseId={uploadingKnowledgeBaseId}
              proCopy={proCopy}
              proRagBoxVectorConnections={proRagBoxVectorConnections}
              proRagBoxBuiltinFallbackBackend={proRagBoxBuiltinFallbackBackend}
              proRagBoxWorkspaces={proRagBoxWorkspaces}
              proRagBoxKnowledgeBaseStatus={proRagBoxKnowledgeBaseStatus}
            />
          ) : activeView === 'market' ? (
            <MarketView
              labels={labels}
              messages={messages}
              language={language}
              languageOptions={languageOptions}
              onLanguageChange={(value) => setLanguage(value as Language)}
              projects={projectOptions}
              selectedProject={selectedProject}
              catalogItems={catalogItems}
              catalogSettings={catalogSettings}
              installedPackages={installedPackages}
              catalogURL={catalogURL}
              setCatalogURL={setCatalogURL}
              catalogSourceMode={catalogSourceMode}
              setCatalogSourceMode={setCatalogSourceMode}
              localCatalogFileName={localCatalogFileName}
              onPickLocalCatalogFile={pickLocalCatalogFile}
              catalogLoading={catalogLoading}
              catalogSyncing={catalogSyncing}
              catalogURLVisible={catalogURLVisible}
              installingCatalogItemId={installingCatalogItemId}
              addingCatalogItemId={addingCatalogItemId}
              uninstallingCatalogItemId={uninstallingCatalogItemId}
              onSyncCatalog={syncCatalog}
              onInstallCatalogPackage={installCatalogPackage}
              onUninstallCatalogPackage={uninstallCatalogPackage}
              onPerformCatalogInstall={performCatalogInstall}
              onActionError={setActionError}
            />
          ) : activeView === 'settings' ? (
            <ProAccessView
              language={language}
              activeSettingsTab={activeSettingsTab}
              onSettingsTabChange={setActiveSettingsTab}
              proRagBoxConnectionError={proRagBoxConnectionError}
              proRagBoxEndpoint={proRagBoxConfig?.base_url ?? ''}
              onRetryRagBox={() => { void loadProRagBoxVectorConnections(); void loadProRagBoxWorkspaces(); }}
              logoSrc={logo}
              editionName={editionMeta.edition_name}
              proCopy={proCopy}
              proPrincipal={proPrincipal}
              proLoading={proLoading}
              proLocalLoginLoading={proLocalLoginLoading}
              proLoginEmail={proLoginEmail}
              proLoginPassword={proLoginPassword}
              proSSOEnabled={!!proSSOConfig?.enabled}
              proSSOProviderName={proSSOConfig?.provider_name}
              proSSOIssuerURL={proSSOConfig?.issuer_url}
              proSSORedirectURL={proSSOConfig?.redirect_url}
              proSSOScopes={proSSOConfig?.scopes}
              proSSOHostedDomain={proSSOConfig?.allowed_hosted_domain}
              proSSODefaultRole={proSSOConfig?.default_role}
              proSSOSessionDays={proSSOConfig?.session_days}
              proSSOAutoCreateUsers={proSSOConfig?.auto_create_users}
              proCreateOpen={proCreateOpen}
              proCreatingToken={proCreatingToken}
              proRevokingTokenId={proRevokingTokenId}
              proNewTokenName={proNewTokenName}
              proNewTokenScopes={proNewTokenScopes}
              proNewTokenExpiresDays={proNewTokenExpiresDays}
              proCreatedTokenValue={proCreatedTokenValue}
              proCreatedSessionToken={proCreatedSessionToken}
              proStorageConfig={proStorageConfig}
              proStorageRuntimeStatus={proStorageRuntimeStatus}
              proStorageShadowStatus={proStorageShadowStatus}
              proStorageOperationalBackend={proStorageOperationalBackend}
              proStoragePostgresHost={proStoragePostgresHost}
              proStoragePostgresPort={proStoragePostgresPort}
              proStoragePostgresDatabase={proStoragePostgresDatabase}
              proStoragePostgresSchema={proStoragePostgresSchema}
              proStoragePostgresUsername={proStoragePostgresUsername}
              proStoragePostgresPassword={proStoragePostgresPassword}
              proStorageClearPostgresDSN={proStorageClearPostgresDSN}
              proStorageLogsBackend={proStorageLogsBackend}
              proStorageClickHouseURL={proStorageClickHouseURL}
              proStorageClickHouseDatabase={proStorageClickHouseDatabase}
              proStorageClickHouseUsername={proStorageClickHouseUsername}
              proStorageClickHousePassword={proStorageClickHousePassword}
              proStorageClearClickHousePassword={proStorageClearClickHousePassword}
              proStorageElasticsearchURL={proStorageElasticsearchURL}
              proStorageElasticsearchUsername={proStorageElasticsearchUsername}
              proStorageElasticsearchPassword={proStorageElasticsearchPassword}
              proStorageElasticsearchIndex={proStorageElasticsearchIndex}
              proStorageClearElasticsearchPassword={proStorageClearElasticsearchPassword}
              proStorageSaving={proStorageSaving}
              proRagBoxBaseURL={proRagBoxBaseURL}
              proRagBoxAdminToken={proRagBoxAdminToken}
              proRagBoxAdminTokenConfigured={!!proRagBoxConfig?.admin_token_configured}
              proRagBoxAdminTokenPreview={proRagBoxConfig?.admin_token_preview}
              proRagBoxClearAdminToken={proRagBoxClearAdminToken}
              proRagBoxTimeoutSeconds={proRagBoxTimeoutSeconds}
              proRagBoxWriteMode={proRagBoxWriteMode}
              proRagBoxSearchMode={proRagBoxSearchMode}
              proRagBoxSaving={proRagBoxSaving}
              proRagBoxAdminConfigured={!!proRagBoxConfig?.admin_configured}
              proRagBoxVectorConnections={proRagBoxVectorConnections}
              proRagBoxWorkspaces={proRagBoxWorkspaces}
              proRagBoxBuiltinFallbackBackend={proRagBoxBuiltinFallbackBackend}
              proRagBoxLoadingVectorConnections={proRagBoxLoadingVectorConnections}
              proRagBoxLoadingWorkspaces={proRagBoxLoadingWorkspaces}
              proRagBoxCreatingVectorConnection={proRagBoxCreatingVectorConnection}
              proRagBoxDefaultingVectorConnectionId={proRagBoxDefaultingVectorConnectionId}
              proRagBoxVectorConnectionForm={proRagBoxVectorConnectionForm}
              proRagBoxConfigured={!!proRagBoxConfig?.configured}
              proRagBoxBindingProjectId={proRagBoxBindingProjectId}
              proRagBoxBindingVectorConnectionId={proRagBoxBindingVectorConnectionId}
              proRagBoxBindingExistingWorkspaceId={proRagBoxBindingExistingWorkspaceId}
              proRagBoxBindingCollectionProjectId={proRagBoxBindingCollectionProjectId}
              proRagBoxBindingCollectionId={proRagBoxBindingCollectionId}
              proRagBoxEnsuringWorkspace={proRagBoxEnsuringWorkspace}
              proRagBoxEnsuringCollection={proRagBoxEnsuringCollection}
              proRagBoxUploadProjectId={proRagBoxUploadProjectId}
              proRagBoxUploadCollectionId={proRagBoxUploadCollectionId}
              proRagBoxUploadFileName={proRagBoxUploadFile?.name ?? ''}
              proRagBoxUploadSourceName={proRagBoxUploadSourceName}
              proRagBoxUploadExternalId={proRagBoxUploadExternalId}
              proRagBoxUploadTitle={proRagBoxUploadTitle}
              proRagBoxUploadMetadata={proRagBoxUploadMetadata}
              proRagBoxUploadingFile={proRagBoxUploadingFile}
              proRagBoxRecordsProjectId={proRagBoxRecordsProjectId}
              proRagBoxRecordsCollectionId={proRagBoxRecordsCollectionId}
              proRagBoxRecordsPayload={proRagBoxRecordsPayload}
              proRagBoxUpsertingRecords={proRagBoxUpsertingRecords}
              proRagBoxLastOperation={proRagBoxLastOperation}
              ragBoxProjects={ragBoxProjects}
              ragBoxProjectKnowledgeBases={ragBoxProjectKnowledgeBases}
              proTokens={proTokens}
              proUsers={proUsers}
              proRoles={proRoles}
              proSessions={proSessions}
              proScopePresets={proScopePresets}
              canWritePro={canWritePro}
              canAdminPro={canAdminPro}
              proCreateUserOpen={proCreateUserOpen}
              proCreatingUser={proCreatingUser}
              proEditUserOpen={proEditUserOpen}
              proUpdatingUserRoles={proUpdatingUserRoles}
              proPasswordUserOpen={proPasswordUserOpen}
              proUpdatingUserPassword={proUpdatingUserPassword}
              proDisablingUserId={proDisablingUserId}
              proEnablingUserId={proEnablingUserId}
              proDeletingUserId={proDeletingUserId}
              proSessionsFilterUserId={proSessionsFilterUserId}
              proEditingUserId={proEditingUserId}
              proEditingUserName={proEditingUserName}
              proEditingUserRoles={proEditingUserRoles}
              proPasswordUserId={proPasswordUserId}
              proPasswordUserName={proPasswordUserName}
              proPasswordUserPassword={proPasswordUserPassword}
              proPasswordUserPasswordConfirm={proPasswordUserPasswordConfirm}
              proCreateSessionOpen={proCreateSessionOpen}
              proCreatingSession={proCreatingSession}
              proRevokingSessionId={proRevokingSessionId}
              proNewUserName={proNewUserName}
              proNewUserEmail={proNewUserEmail}
              proNewUserRoles={proNewUserRoles}
              proNewSessionUserId={proNewSessionUserId}
              proNewSessionLabel={proNewSessionLabel}
              proNewSessionExpiresDays={proNewSessionExpiresDays}
              onSetProCreateOpen={(open) => {
                setProCreateOpen(open);
                if (!open) {
                  setProCreatedTokenValue(null);
                }
              }}
              onSetProCreateUserOpen={(open) => {
                setProCreateUserOpen(open);
                if (!open) {
                  setProNewUserName('');
                  setProNewUserEmail('');
                  setProNewUserRoles('reader');
                }
              }}
              onSetProEditUserOpen={(open) => {
                setProEditUserOpen(open);
                if (!open) {
                  setProEditingUserId('');
                  setProEditingUserName('');
                  setProEditingUserRoles('reader');
                }
              }}
              onSetProPasswordUserOpen={(open) => {
                setProPasswordUserOpen(open);
                if (!open) {
                  setProPasswordUserId('');
                  setProPasswordUserName('');
                  setProPasswordUserPassword('');
                  setProPasswordUserPasswordConfirm('');
                }
              }}
              onSetProCreateSessionOpen={(open) => {
                setProCreateSessionOpen(open);
                if (!open) {
                  setProCreatedSessionToken(null);
                  setProNewSessionUserId('');
                  setProNewSessionLabel('');
                  setProNewSessionExpiresDays('30');
                }
              }}
              onSetProLoginEmail={setProLoginEmail}
              onSetProLoginPassword={setProLoginPassword}
              onSetProNewTokenName={setProNewTokenName}
              onSetProNewTokenScopes={setProNewTokenScopes}
              onSetProNewTokenExpiresDays={setProNewTokenExpiresDays}
              onSetProNewUserName={setProNewUserName}
              onSetProNewUserEmail={setProNewUserEmail}
              onSetProNewUserRoles={setProNewUserRoles}
              onSetProEditingUserRoles={setProEditingUserRoles}
              onSetProPasswordUserPassword={setProPasswordUserPassword}
              onSetProPasswordUserPasswordConfirm={setProPasswordUserPasswordConfirm}
              onSetProSessionsFilterUserId={setProSessionsFilterUserId}
              onSetProNewSessionUserId={setProNewSessionUserId}
              onSetProNewSessionLabel={setProNewSessionLabel}
              onSetProNewSessionExpiresDays={setProNewSessionExpiresDays}
              onSetProStorageOperationalBackend={setProStorageOperationalBackend}
              onSetProStoragePostgresHost={setProStoragePostgresHost}
              onSetProStoragePostgresPort={setProStoragePostgresPort}
              onSetProStoragePostgresDatabase={setProStoragePostgresDatabase}
              onSetProStoragePostgresSchema={setProStoragePostgresSchema}
              onSetProStoragePostgresUsername={setProStoragePostgresUsername}
              onSetProStoragePostgresPassword={setProStoragePostgresPassword}
              onSetProStorageClearPostgresDSN={setProStorageClearPostgresDSN}
              onSetProStorageLogsBackend={setProStorageLogsBackend}
              onSetProStorageClickHouseURL={setProStorageClickHouseURL}
              onSetProStorageClickHouseDatabase={setProStorageClickHouseDatabase}
              onSetProStorageClickHouseUsername={setProStorageClickHouseUsername}
              onSetProStorageClickHousePassword={setProStorageClickHousePassword}
              onSetProStorageClearClickHousePassword={setProStorageClearClickHousePassword}
              onSetProStorageElasticsearchURL={setProStorageElasticsearchURL}
              onSetProStorageElasticsearchUsername={setProStorageElasticsearchUsername}
              onSetProStorageElasticsearchPassword={setProStorageElasticsearchPassword}
              onSetProStorageElasticsearchIndex={setProStorageElasticsearchIndex}
              onSetProStorageClearElasticsearchPassword={setProStorageClearElasticsearchPassword}
              onSetProRagBoxBaseURL={setProRagBoxBaseURL}
              onSetProRagBoxAdminToken={setProRagBoxAdminToken}
              onSetProRagBoxClearAdminToken={setProRagBoxClearAdminToken}
              onSetProRagBoxTimeoutSeconds={setProRagBoxTimeoutSeconds}
              onSetProRagBoxWriteMode={setProRagBoxWriteMode}
              onSetProRagBoxSearchMode={setProRagBoxSearchMode}
              onSetProRagBoxVectorConnectionForm={setProRagBoxVectorConnectionForm}
              onSetProRagBoxBindingProjectId={setProRagBoxBindingProjectId}
              onSetProRagBoxBindingVectorConnectionId={setProRagBoxBindingVectorConnectionId}
              onSetProRagBoxBindingExistingWorkspaceId={setProRagBoxBindingExistingWorkspaceId}
              onSetProRagBoxBindingCollectionProjectId={setProRagBoxBindingCollectionProjectId}
              onSetProRagBoxBindingCollectionId={setProRagBoxBindingCollectionId}
              onSetProRagBoxUploadProjectId={setProRagBoxUploadProjectId}
              onSetProRagBoxUploadCollectionId={setProRagBoxUploadCollectionId}
              onSetProRagBoxUploadFile={setProRagBoxUploadFile}
              onSetProRagBoxUploadSourceName={setProRagBoxUploadSourceName}
              onSetProRagBoxUploadExternalId={setProRagBoxUploadExternalId}
              onSetProRagBoxUploadTitle={setProRagBoxUploadTitle}
              onSetProRagBoxUploadMetadata={setProRagBoxUploadMetadata}
              onSetProRagBoxRecordsProjectId={setProRagBoxRecordsProjectId}
              onSetProRagBoxRecordsCollectionId={setProRagBoxRecordsCollectionId}
              onSetProRagBoxRecordsPayload={setProRagBoxRecordsPayload}
              onLoginProLocal={loginProLocal}
              onStartProSSOSignIn={startProSSOSignIn}
              onCreateProToken={createProToken}
              onCreateProUser={createProUser}
              onStartEditProUser={startEditProUser}
              onUpdateProUserRoles={updateProUserRoles}
              onStartProPasswordChange={startProPasswordChange}
              onUpdateProUserPassword={updateProUserPassword}
              onDisableProUser={disableProUser}
              onEnableProUser={enableProUser}
              onDeleteProUser={deleteProUser}
              onCreateProSession={createProSession}
              onRevokeCurrentProSession={revokeCurrentProSession}
              onSaveProStorageConfig={saveProStorageConfig}
              onSaveProRagBoxConfig={saveProRagBoxConfig}
              onCreateProRagBoxVectorConnection={createProRagBoxVectorConnection}
              onSetDefaultProRagBoxVectorConnection={setDefaultProRagBoxVectorConnection}
              onEnsureRagBoxWorkspace={ensureRagBoxWorkspace}
              onEnsureRagBoxCollection={ensureRagBoxCollection}
              onUploadRagBoxFile={uploadRagBoxFile}
              onUpsertRagBoxRecords={upsertRagBoxRecords}
              onCopyToken={copyToClipboard}
              onRevokeProToken={revokeProToken}
              onRevokeProSession={revokeProSession}
            />
          ) : (
            <ProjectsView
              language={language}
              onCLIChanged={loadProjects}
              labels={labels}
              messages={messages}
              selectedProject={selectedProjectForProjectsView}
              selectedProjectHealthyCount={selectedProjectHealthyCount}
              launchProjectOpen={launchProjectOpen}
              setLaunchProjectOpen={setLaunchProjectOpen}
              shouldShowOllamaControls={shouldShowOllamaControls}
              shouldShowLlamaCppControls={shouldShowLlamaCppControls}
              ollamaStatus={ollamaStatus}
              llamaCppStatus={llamaCppStatus}
              selectedLlamaCppModelPath={selectedLlamaCppModelPath}
              setSelectedLlamaCppModelPath={setSelectedLlamaCppModelPath}
              selectedLlamaCppModelName={selectedLlamaCppModelName}
              setSelectedLlamaCppModelName={setSelectedLlamaCppModelName}
              selectedOllamaModel={selectedOllamaModel}
              setSelectedOllamaModel={setSelectedOllamaModel}
              loadOllamaStatus={loadOllamaStatus}
              loadLlamaCppStatus={loadLlamaCppStatus}
              ollamaRefreshing={ollamaRefreshing}
              llamaCppRefreshing={llamaCppRefreshing}
              launchProjectOllama={launchProjectOllama}
              launchProjectLlamaCpp={launchProjectLlamaCpp}
              stopLlamaCppServer={stopLlamaCppServer}
              launchingOllamaProjectId={launchingOllamaProjectId}
              launchingLlamaCppProjectId={launchingLlamaCppProjectId}
              stoppingLlamaCpp={stoppingLlamaCpp}
              canLaunchOllama={canLaunchOllama}
              canLaunchLlamaCpp={canLaunchLlamaCpp}
              launchProjectLMStudio={launchProjectLMStudio}
              launchingLMStudioProjectId={launchingLMStudioProjectId}
              OllamaIcon={OllamaIcon}
              alternativeConnectURLs={alternativeConnectURLs}
              oauthClientManagementEnabled={isProEdition}
              copyConnectURL={copyConnectURL}
              regenerateEndpointToken={regenerateProjectBearerToken}
              busyProjectId={busyProjectId}
              setProjectPaused={setProjectPaused}
              startDuplicateProject={startDuplicateProject}
              startEditProject={startEditProject}
              deleteProject={deleteProject}
              connectRAGCollectionOpen={connectRAGCollectionOpen}
              setConnectRAGCollectionOpen={setConnectRAGCollectionOpen}
              allRAGCollections={allRAGCollections}
              availableRAGCollections={availableRAGCollections}
              connectRAGCollectionToProject={connectRAGCollectionToProject}
              linkingCollectionId={linkingCollectionId}
              disconnectRAGCollectionFromProject={disconnectRAGCollectionFromProject}
              ragCollectionForm={ragCollectionForm}
              setRAGCollectionForm={setRAGCollectionForm}
              resetRAGCollectionForm={() => {
                setRAGCollectionForm(emptyRAGCollectionForm);
                setEditingRAGCollectionId(null);
              }}
              createAndConnectRAGCollectionToProject={createAndConnectRAGCollectionToProject}
              creatingRAGCollection={creatingRAGCollection}
              ragIndexPaths={ragIndexPaths}
              setRAGIndexPaths={setRAGIndexPaths}
              indexRAGCollection={indexRAGCollection}
              indexingCollectionId={indexingCollectionId}
              addServerOpen={addServerOpen}
              setAddServerOpen={setAddServerOpen}
              editingServerId={editingServerId}
              resetServerEditor={() => {
                setEditingServerId(null);
                setServerForm(emptyServerForm);
                setOAuthAdvancedOpen(false);
              }}
              serverForm={serverForm}
              updateServerForm={updateServerForm}
              updateStringListField={updateStringListField}
              removeStringListField={removeStringListField}
              addStringListField={addStringListField}
              updateKeyValueField={updateKeyValueField}
              removeKeyValueField={removeKeyValueField}
              addKeyValueField={addKeyValueField}
              updateServerLastArg={updateServerLastArg}
              editingServerIntegrationCatalogItemId={editingServerIntegration?.catalog_item_id ?? null}
              catalogItems={catalogItems}
              installedPackages={installedPackages}
              addingCatalogItemId={addingCatalogItemId}
              onPerformCatalogInstall={performCatalogInstall}
              onActionError={setActionError}
              addingServer={addingServer}
              addServer={addServer}
              busyServerId={busyServerId}
              checkServerHealth={checkServerHealth}
              runServerAction={runServerAction}
              openAuthModal={openAuthModal}
              openServerTools={openServerTools}
              serverToolsLoadingId={serverToolsLoadingId}
              inspectServer={inspectServer}
              inspectingServerId={inspectingServerId}
              startEditServer={startEditServer}
              setServerEnabled={setServerEnabled}
              deleteServer={deleteServer}
              inspectOpen={inspectOpen}
              setInspectOpen={setInspectOpen}
              inspectionServerName={inspectionServerName}
              inspectionError={inspectionError}
              inspection={inspection}
              formatSchema={formatSchema}
              serverToolsOpen={serverToolsOpen}
              setServerToolsOpen={setServerToolsOpen}
              resetServerTools={() => {
                setServerToolsLoadingId(null);
                setServerToolsSavingName(null);
                setServerToolsServerId(null);
                setServerToolsServerName('');
                setServerTools([]);
                setServerToolsError(null);
              }}
              serverToolsServerName={serverToolsServerName}
              serverToolsError={serverToolsError}
              serverTools={serverTools}
              serverToolsSavingName={serverToolsSavingName}
              setServerToolEnabled={setServerToolEnabled}
              authOpen={authOpen}
              setAuthOpen={setAuthOpen}
              resetAuthServer={() => setAuthServerId(null)}
              authServer={authServer}
              connectOAuth={connectOAuth}
              disconnectOAuth={disconnectOAuth}
              updateProjectPrompt={updateProjectPromptConfig}
              updatingPrompt={updatingPrompt}
            />
          )}
      </Suspense>
    </AppShell>
  );
}
