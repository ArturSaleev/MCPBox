import { FormEvent, Suspense, lazy, useEffect, useRef, useState } from 'react';
import { Toaster, toast } from 'sonner';
import {
  Bot,
  Database,
  FolderKanban,
  LogOut,
  Shield,
  ShoppingBag,
  TextSearch,
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
import { ProjectsSidebar } from './components/ProjectsSidebar';
import type {
  CatalogItem,
  CatalogSettings,
  InstalledPackage,
  ProjectOption,
} from './market';

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

type ServerStatus = {
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
  auto_reindex: boolean;
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

type AuditLog = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  action: string;
  actor: string;
  detail: string;
  created_at: string;
};

type MetricsWindow = '5m' | '1h' | '24h';

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

type PerformanceMetricsResponse = {
  window: MetricsWindow;
  summary: PerformanceSummary;
  trends: PerformanceTrendBucket[];
  top_slow_servers: PerformanceServerMetricRecord[];
  top_error_servers: PerformanceServerMetricRecord[];
  top_traffic_servers: PerformanceServerMetricRecord[];
  recent_failures: PerformanceFailureRecord[];
};

type ApiError = {
  error: string;
};

type EditionMeta = {
  edition_id: string;
  edition_name: string;
  capabilities: string[];
};

type ProPrincipal = {
  name: string;
  scopes: string[];
  roles: string[];
  user_id?: number;
  session_id?: number;
  auth_method?: string;
  is_bootstrap: boolean;
};

type ProTokenRecord = {
  id: number;
  name: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
};

type CreateProTokenResponse = {
  token: string;
  record: ProTokenRecord;
};

type ProUserRecord = {
  id: number;
  email: string;
  display_name: string;
  auth_provider: string;
  external_id: string;
  is_bootstrap: boolean;
  roles: string[];
  scopes: string[];
  last_login_at?: string;
  disabled_at?: string;
  created_at: string;
};

type ProRoleRecord = {
  id: number;
  name: string;
  display_name: string;
  description: string;
  scopes: string[];
  is_system: boolean;
  created_at: string;
};

type ProSessionRecord = {
  id: number;
  user_id: number;
  user_name: string;
  label: string;
  auth_method: string;
  roles: string[];
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
};

type CreateProUserResponse = ProUserRecord;
type UpdateProUserResponse = ProUserRecord;
type DisableProUserResponse = ProUserRecord;
type EnableProUserResponse = ProUserRecord;

type CreateProSessionResponse = {
  token: string;
  record: ProSessionRecord | null;
};

type ProLocalLoginResponse = {
  token: string;
  user: ProUserRecord;
  session: ProSessionRecord | null;
};

type ProSSOConfig = {
  enabled: boolean;
  provider_name?: string;
  issuer_url?: string;
  redirect_url?: string;
  start_url?: string;
  allowed_hosted_domain?: string;
  default_role?: string;
  session_days?: number;
  auto_create_users?: boolean;
  scopes?: string[];
};

type ProScopePreset = {
  id: 'reader' | 'operator' | 'admin';
  label: string;
  scopes: string[];
  description: string;
};

type LogsFilterMode = 'all' | 'pro';

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

type ServerToolStatus = {
  name: string;
  title: string;
  description: string;
  input_schema?: unknown;
  output_schema?: unknown;
  enabled: boolean;
};

const legacyCatalogSourceURL = 'https://webeasy.kz/mcpbox/catalog.json';
const defaultCatalogSourceURL = 'https://mcpbox.sh/catalog.json';
const proAuthStorageKey = 'mcpbox-pro-auth-token';

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

async function proAPIRequest<T>(
  token: string,
  input: RequestInfo,
  requestFailedMessage: (status: number) => string,
  init?: RequestInit,
): Promise<T> {
  const trimmedToken = token.trim();
  if (!trimmedToken) {
    throw new Error('Missing Pro admin token');
  }

  return apiRequest<T>(input, requestFailedMessage, {
    ...init,
    headers: {
      Authorization: `Bearer ${trimmedToken}`,
      ...(init?.headers ?? {}),
    },
  });
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

function hasProScope(principal: ProPrincipal | null, scope: string) {
  if (!principal) {
    return false;
  }
  return principal.scopes.includes('pro:admin') || principal.scopes.includes(scope);
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
  const [view, setView] = useState<'projects' | 'knowledge' | 'market' | 'logs' | 'pro'>('projects');
  const [language, setLanguage] = useState<Language>(detectInitialLanguage);
  const [editionMeta, setEditionMeta] = useState<EditionMeta>({
    edition_id: 'free',
    edition_name: 'MCPBox',
    capabilities: [],
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
  const [duplicatingProjectId, setDuplicatingProjectId] = useState<number | null>(null);
  const [duplicateProjectName, setDuplicateProjectName] = useState('');
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
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [catalogSettings, setCatalogSettings] = useState<CatalogSettings | null>(null);
  const [installedPackages, setInstalledPackages] = useState<InstalledPackage[]>([]);
  const [ollamaStatus, setOllamaStatus] = useState<OllamaStatus | null>(null);
  const [ollamaRefreshing, setOllamaRefreshing] = useState(false);
  const [selectedOllamaModel, setSelectedOllamaModel] = useState('');
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
  const [busyServerId, setBusyServerId] = useState<number | null>(null);
  const [inspectOpen, setInspectOpen] = useState(false);
  const [inspectingServerId, setInspectingServerId] = useState<number | null>(null);
  const [inspection, setInspection] = useState<ServerInspection | null>(null);
  const [inspectionServerName, setInspectionServerName] = useState('');
  const [inspectionError, setInspectionError] = useState<string | null>(null);
  const [serverToolsOpen, setServerToolsOpen] = useState(false);
  const [serverToolsLoadingId, setServerToolsLoadingId] = useState<number | null>(null);
  const [serverToolsSavingName, setServerToolsSavingName] = useState<string | null>(null);
  const [serverToolsServerId, setServerToolsServerId] = useState<number | null>(null);
  const [serverToolsServerName, setServerToolsServerName] = useState('');
  const [serverTools, setServerTools] = useState<ServerToolStatus[]>([]);
  const [serverToolsError, setServerToolsError] = useState<string | null>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [authServerId, setAuthServerId] = useState<number | null>(null);
  const [proAuthToken, setProAuthToken] = useState('');
  const [proPrincipal, setProPrincipal] = useState<ProPrincipal | null>(null);
  const [proTokens, setProTokens] = useState<ProTokenRecord[]>([]);
  const [proUsers, setProUsers] = useState<ProUserRecord[]>([]);
  const [proRoles, setProRoles] = useState<ProRoleRecord[]>([]);
  const [proSessions, setProSessions] = useState<ProSessionRecord[]>([]);
  const [proSSOConfig, setProSSOConfig] = useState<ProSSOConfig | null>(null);
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
  const [proDisablingUserId, setProDisablingUserId] = useState<number | null>(null);
  const [proEnablingUserId, setProEnablingUserId] = useState<number | null>(null);
  const [proDeletingUserId, setProDeletingUserId] = useState<number | null>(null);
  const [proSessionsFilterUserId, setProSessionsFilterUserId] = useState('all');
  const [proEditingUserId, setProEditingUserId] = useState<string>('');
  const [proEditingUserName, setProEditingUserName] = useState('');
  const [proEditingUserRoles, setProEditingUserRoles] = useState('reader');
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
  const [copied, setCopied] = useState(false);
  const [connectionURLsExpanded, setConnectionURLsExpanded] = useState(false);
  const logsViewportRef = useRef<HTMLDivElement | null>(null);
  const marketAutoSyncTriggeredRef = useRef(false);
  const previousViewRef = useRef(view);
  const dictionary = dictionaries[language];
  const { labels, messages } = dictionary;
  const languageOptions: Array<{ value: Language; label: string }> = [
    { value: 'en', label: labels.english },
    { value: 'ru', label: labels.russian },
  ];
  const proCopy =
    language === 'ru'
      ? {
          nav: 'Pro',
          title: 'MCPBox Pro',
          subtitle: 'Токены агентов и защищенный Pro API',
          signInSSO: 'Войти через SSO',
          localLogin: 'Войти по email и паролю',
          password: 'Пароль',
          login: 'Войти',
          loginHint: 'Локальный admin создаётся из консоли командой `admin create`. После входа MCPBox Pro выдаёт внутреннюю сессию.',
          signOut: 'Выйти',
          ssoAvailable: 'SSO доступен',
          ssoIssuer: 'Issuer',
          ssoRedirectURL: 'Redirect URL',
          ssoScopes: 'Scopes',
          ssoDomainHint: 'Разрешённый домен',
          ssoDefaultRole: 'Роль по умолчанию',
          ssoSessionDays: 'Сессия (дней)',
          ssoAutoCreate: 'Автосоздание пользователей',
          ssoEnabledLabel: 'Включено',
          ssoDisabledLabel: 'Выключено',
          connectedAs: 'Подключено как',
          scopes: 'Скоупы',
          createToken: 'Создать токен',
          createTokenHint: 'Operator может создавать Reader/Operator токены. Admin нужен для admin токенов и revoke.',
          tokenName: 'Название токена',
          tokenNamePlaceholder: 'CI agent',
          tokenScopes: 'Scopes',
          tokenScopesPlaceholder: 'pro:read, pro:write',
          expiresDays: 'Срок (дней)',
          expiryPolicy: 'Срок жизни',
          expiryHint: '0 означает бессрочно. Можно выбрать preset или ввести своё число дней.',
          noExpiry: 'Бессрочно',
          oneDay: '1 день',
          sevenDays: '7 дней',
          thirtyDays: '30 дней',
          ninetyDays: '90 дней',
          oneYear: '365 дней',
          activeTokens: 'Активные токены',
          noTokens: 'Токенов пока нет. Создай первый токен ниже.',
          revoke: 'Отозвать',
          adminOnly: 'Только для admin',
          oneTimeSecret: 'Показывается только один раз',
          copyToken: 'Скопировать токен',
          authHint: 'Pro endpoints защищены bearer token. Значение хранится только в localStorage этого браузера.',
          verifyFailed: 'Не удалось проверить Pro token',
          loadTokensFailed: 'Не удалось загрузить токены',
          createFailed: 'Не удалось создать токен',
          revokeFailed: 'Не удалось отозвать токен',
          users: 'Пользователи',
          rolesTitle: 'Роли',
          sessionsTitle: 'Сессии',
          noUsers: 'Пользователей пока нет.',
          noSessions: 'Сессий пока нет.',
          createUser: 'Создать пользователя',
          editUserRoles: 'Изменить роли',
          disableUser: 'Отключить',
          enableUser: 'Включить',
          deleteUser: 'Удалить',
          createSession: 'Создать сессию',
          displayName: 'Имя',
          email: 'Email',
          roleNames: 'Роли',
          sessionLabel: 'Название сессии',
          issueSession: 'Выпустить сессию',
          createdSessionToken: 'Session token',
          currentSession: 'Текущая сессия',
          revokeCurrentSession: 'Завершить текущую сессию',
          authMethod: 'Метод доступа',
          rolesLabel: 'Роли',
          userId: 'Пользователь',
          sessionId: 'Сессия',
          allUsers: 'Все пользователи',
          sessionsFilter: 'Фильтр сессий',
          createUserFailed: 'Не удалось создать пользователя',
          updateUserRolesFailed: 'Не удалось обновить роли пользователя',
          disableUserFailed: 'Не удалось отключить пользователя',
          enableUserFailed: 'Не удалось включить пользователя',
          deleteUserFailed: 'Не удалось удалить пользователя',
          disableUserConfirm: 'Отключить пользователя? Все его активные сессии будут отозваны.',
          enableUserConfirm: 'Включить пользователя снова?',
          deleteUserConfirm: 'Удалить пользователя? Это удалит его роли и отзовёт активные сессии.',
          createSessionFailed: 'Не удалось создать сессию',
          revokeSessionFailed: 'Не удалось отозвать сессию',
          revokeCurrentSessionConfirm: 'Завершить текущую активную сессию? После этого потребуется войти заново.',
          me: 'Текущий доступ',
          statusReady: 'Готово',
          notConnected: 'Не подключено',
        }
      : {
          nav: 'Pro',
          title: 'MCPBox Pro',
          subtitle: 'Agent tokens and protected Pro API',
          signInSSO: 'Sign in with SSO',
          localLogin: 'Sign in with email and password',
          password: 'Password',
          login: 'Sign in',
          loginHint: 'The first local admin is created from the CLI with `admin create`. MCPBox Pro then issues its own internal session.',
          signOut: 'Sign out',
          ssoAvailable: 'SSO available',
          ssoIssuer: 'Issuer',
          ssoRedirectURL: 'Redirect URL',
          ssoScopes: 'Scopes',
          ssoDomainHint: 'Allowed domain',
          ssoDefaultRole: 'Default role',
          ssoSessionDays: 'Session days',
          ssoAutoCreate: 'Auto-create users',
          ssoEnabledLabel: 'Enabled',
          ssoDisabledLabel: 'Disabled',
          connectedAs: 'Connected as',
          scopes: 'Scopes',
          createToken: 'Create token',
          createTokenHint: 'Operator can create Reader/Operator tokens. Admin is required for admin tokens and revoke.',
          tokenName: 'Token name',
          tokenNamePlaceholder: 'CI agent',
          tokenScopes: 'Scopes',
          tokenScopesPlaceholder: 'pro:read, pro:write',
          expiresDays: 'Expires (days)',
          expiryPolicy: 'Lifetime',
          expiryHint: '0 means no expiry. You can choose a preset or enter a custom number of days.',
          noExpiry: 'No expiry',
          oneDay: '1 day',
          sevenDays: '7 days',
          thirtyDays: '30 days',
          ninetyDays: '90 days',
          oneYear: '365 days',
          activeTokens: 'Active tokens',
          noTokens: 'No tokens yet. Create the first token below.',
          revoke: 'Revoke',
          adminOnly: 'Admin only',
          oneTimeSecret: 'Shown only once',
          copyToken: 'Copy token',
          authHint: 'Pro endpoints are protected by bearer token. The value is stored only in this browser localStorage.',
          verifyFailed: 'Failed to verify Pro token',
          loadTokensFailed: 'Failed to load tokens',
          createFailed: 'Failed to create token',
          revokeFailed: 'Failed to revoke token',
          users: 'Users',
          rolesTitle: 'Roles',
          sessionsTitle: 'Sessions',
          noUsers: 'No users yet.',
          noSessions: 'No sessions yet.',
          createUser: 'Create user',
          editUserRoles: 'Edit roles',
          disableUser: 'Disable',
          enableUser: 'Enable',
          deleteUser: 'Delete',
          createSession: 'Create session',
          displayName: 'Display name',
          email: 'Email',
          roleNames: 'Roles',
          sessionLabel: 'Session label',
          issueSession: 'Issue session',
          createdSessionToken: 'Session token',
          currentSession: 'Current session',
          revokeCurrentSession: 'Revoke current session',
          authMethod: 'Auth method',
          rolesLabel: 'Roles',
          userId: 'User',
          sessionId: 'Session',
          allUsers: 'All users',
          sessionsFilter: 'Session filter',
          createUserFailed: 'Failed to create user',
          updateUserRolesFailed: 'Failed to update user roles',
          disableUserFailed: 'Failed to disable user',
          enableUserFailed: 'Failed to enable user',
          deleteUserFailed: 'Failed to delete user',
          disableUserConfirm: 'Disable this user? All active sessions will be revoked.',
          enableUserConfirm: 'Enable this user again?',
          deleteUserConfirm: 'Delete this user? This removes role links and revokes active sessions.',
          createSessionFailed: 'Failed to create session',
          revokeSessionFailed: 'Failed to revoke session',
          revokeCurrentSessionConfirm: 'Revoke the current active session? You will need to sign in again after that.',
          me: 'Current access',
          statusReady: 'Ready',
          notConnected: 'Not connected',
        };
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
    editionMeta.edition_id === 'pro' ||
    editionMeta.capabilities.includes('auth') ||
    editionMeta.capabilities.includes('agent_tokens');
  const isProEdition = editionMeta.edition_id === 'pro';
  const navigationItems = [
    { id: 'projects' as const, label: labels.projects, icon: FolderKanban },
    { id: 'knowledge' as const, label: labels.knowledgeBase, icon: Database },
    { id: 'market' as const, label: labels.market, icon: ShoppingBag },
    { id: 'logs' as const, label: labels.logs, icon: TextSearch },
    ...(hasProAccess ? [{ id: 'pro' as const, label: proCopy.nav, icon: Shield }] : []),
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
  const authServer =
    selectedProject?.servers.find((server) => server.id === authServerId) ?? null;
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
  const projectOptions: ProjectOption[] = projects.map((project) => ({
    project_id: project.project_id,
    name: project.name,
    root_path: project.root_path,
  }));
  const shouldShowOllamaControls = ollamaStatus?.installed ?? false;
  const canLaunchOllama =
    !!selectedOllamaModel &&
    (ollamaStatus?.models.length ?? 0) > 0;
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
  const proActivityCopy =
    language === 'ru'
      ? {
          title: 'Pro activity',
          subtitle: 'События токенов, сессий и защищенного Pro API',
          all: 'Вся активность',
          proOnly: 'Только Pro auth',
          created: 'Создано',
          used: 'Использовано',
          revoked: 'Отозвано',
          empty: 'Pro auth activity пока нет.',
        }
      : {
          title: 'Pro activity',
          subtitle: 'Agent token, session, and protected Pro API events',
          all: 'All activity',
          proOnly: 'Pro auth only',
          created: 'Created',
          used: 'Used',
          revoked: 'Revoked',
          empty: 'No Pro auth activity yet.',
        };
  const proActivityLogs = logs.filter((entry) => proAuditActions.has(entry.action));
  const visibleLogs =
    logsFilterMode === 'pro' ? proActivityLogs : logs;
  const proCreatedCount = proActivityLogs.filter((entry) => entry.action === 'token_created' || entry.action === 'session_created').length;
  const proUsedCount = proActivityLogs.filter((entry) => entry.action === 'token_used' || entry.action === 'session_used' || entry.action === 'local_login' || entry.action === 'sso_login').length;
  const proRevokedCount = proActivityLogs.filter((entry) => entry.action === 'token_revoked' || entry.action === 'session_revoked').length;
  const canWritePro = hasProScope(proPrincipal, 'pro:write');
  const canAdminPro = hasProScope(proPrincipal, 'pro:admin');
  const proAuthLocked = isProEdition && !proPrincipal;
  const activeView = proAuthLocked ? 'pro' : view;
  const visibleNavigationItems = proAuthLocked
    ? navigationItems.filter((item) => item.id === 'pro')
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
      if (targetView === 'pro') {
        setView('pro');
      }
      url.searchParams.delete('pro_session_token');
      url.searchParams.delete('pro_view');
      window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`);
    }
    if (ssoError) {
      if (targetView === 'pro') {
        setView('pro');
      }
      toast.error(ssoError);
      url.searchParams.delete('pro_sso_error');
      url.searchParams.delete('pro_view');
      window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`);
    }
  }, []);

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
    void Promise.all([
      loadEditionMeta(),
      loadProjects(true),
      loadRAGCollections(),
      loadCatalog(true),
      loadInstalledPackages(),
      loadOllamaStatus(),
    ]);
  }, []);

  useEffect(() => {
    if (!hasProAccess || activeView !== 'pro' || !proAuthToken.trim() || proPrincipal) {
      return;
    }

    void loadProSession();
  }, [hasProAccess, activeView, proAuthToken, proPrincipal]);

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
    if (!hasProAccess && view === 'pro') {
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
        edition_id: 'free',
        edition_name: 'MCPBox',
        capabilities: [],
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

  async function loadOllamaStatus(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setOllamaRefreshing(true);
    }

    try {
      const nextStatus = await apiRequest<OllamaStatus>(
        '/api/ollama/status',
        messages.requestFailed,
      );
      setOllamaStatus(nextStatus);
    } catch {
      setOllamaStatus(null);
    } finally {
      setOllamaRefreshing(false);
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

  async function loadProSession(tokenOverride?: string) {
    const activeToken = (tokenOverride ?? proAuthToken).trim();
    if (!activeToken) {
      setProPrincipal(null);
      setProTokens([]);
      setProUsers([]);
      setProRoles([]);
      setProSessions([]);
      return;
    }

    setProLoading(true);
    try {
      const [principal, tokens, users, roles, sessions] = await Promise.all([
        proAPIRequest<ProPrincipal>(activeToken, '/api/pro/auth/me', messages.requestFailed),
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
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(proAuthStorageKey, activeToken);
      }
    } catch (loadError) {
      resetProAccess(false);
      toast.error(loadError instanceof Error ? loadError.message : proCopy.loadTokensFailed);
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
      setView('pro');
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
    url.searchParams.set('return_to', '/?pro_view=pro');
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
      setLaunchProjectOpen(false);
    } catch (submitError) {
      setActionError(submitError instanceof Error ? submitError.message : messages.launchOllamaError);
    } finally {
      setLaunchingOllamaProjectId(null);
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

  async function copyConnectURL() {
    if (!selectedProject?.connect_url) {
      return;
    }

    await copyToClipboard(selectedProject.connect_url);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }

  async function copyToClipboard(value: string) {
    await navigator.clipboard.writeText(value);
  }

  function updateAndPersistProAuthToken(value: string) {
    setProAuthToken(value);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(proAuthStorageKey, value);
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
    });
    setEditingProjectId(selectedProject.project_id);
    setCreateProjectOpen(true);
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
    setRAGCollectionForm({
      name: collection.name,
      source_path: collection.source_path ?? '',
      auto_reindex: collection.auto_reindex ?? false,
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
    <AppShell
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
            labels={labels}
            messages={messages}
            language={language}
            languageOptions={languageOptions}
            setLanguage={setLanguage}
            onRefresh={() => Promise.all([loadProjects(), loadOllamaStatus()])}
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
            resetProjectForm={() => {
              setEditingProjectId(null);
              setProjectForm(emptyProjectForm);
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
              projectNameFromLog={projectNameFromLog}
              serverNameFromLog={serverNameFromLog}
              serverNamesById={serverNamesById}
              projectNamesById={projectNamesById}
            />
          ) : activeView === 'knowledge' ? (
            <KnowledgeView
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
          ) : activeView === 'pro' ? (
            <ProAccessView
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
              proDisablingUserId={proDisablingUserId}
              proEnablingUserId={proEnablingUserId}
              proDeletingUserId={proDeletingUserId}
              proSessionsFilterUserId={proSessionsFilterUserId}
              proEditingUserId={proEditingUserId}
              proEditingUserName={proEditingUserName}
              proEditingUserRoles={proEditingUserRoles}
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
              onSetProSessionsFilterUserId={setProSessionsFilterUserId}
              onSetProNewSessionUserId={setProNewSessionUserId}
              onSetProNewSessionLabel={setProNewSessionLabel}
              onSetProNewSessionExpiresDays={setProNewSessionExpiresDays}
              onLoginProLocal={loginProLocal}
              onStartProSSOSignIn={startProSSOSignIn}
              onCreateProToken={createProToken}
              onCreateProUser={createProUser}
              onStartEditProUser={startEditProUser}
              onUpdateProUserRoles={updateProUserRoles}
              onDisableProUser={disableProUser}
              onEnableProUser={enableProUser}
              onDeleteProUser={deleteProUser}
              onCreateProSession={createProSession}
              onRevokeCurrentProSession={revokeCurrentProSession}
              onCopyToken={copyToClipboard}
              onRevokeProToken={revokeProToken}
              onRevokeProSession={revokeProSession}
            />
          ) : (
            <ProjectsView
              labels={labels}
              messages={messages}
              selectedProject={selectedProject}
              selectedProjectHealthyCount={selectedProjectHealthyCount}
              launchProjectOpen={launchProjectOpen}
              setLaunchProjectOpen={setLaunchProjectOpen}
              shouldShowOllamaControls={shouldShowOllamaControls}
              ollamaStatus={ollamaStatus}
              selectedOllamaModel={selectedOllamaModel}
              setSelectedOllamaModel={setSelectedOllamaModel}
              loadOllamaStatus={loadOllamaStatus}
              ollamaRefreshing={ollamaRefreshing}
              launchProjectOllama={launchProjectOllama}
              launchingOllamaProjectId={launchingOllamaProjectId}
              canLaunchOllama={canLaunchOllama}
              launchProjectLMStudio={launchProjectLMStudio}
              launchingLMStudioProjectId={launchingLMStudioProjectId}
              OllamaIcon={OllamaIcon}
              alternativeConnectURLs={alternativeConnectURLs}
              connectionURLsExpanded={connectionURLsExpanded}
              setConnectionURLsExpanded={setConnectionURLsExpanded}
              copyConnectURL={copyConnectURL}
              copied={copied}
              busyProjectId={busyProjectId}
              setProjectPaused={setProjectPaused}
              startDuplicateProject={startDuplicateProject}
              startEditProject={startEditProject}
              deleteProject={deleteProject}
              connectRAGCollectionOpen={connectRAGCollectionOpen}
              setConnectRAGCollectionOpen={setConnectRAGCollectionOpen}
              availableRAGCollections={availableRAGCollections}
              connectRAGCollectionToProject={connectRAGCollectionToProject}
              linkingCollectionId={linkingCollectionId}
              disconnectRAGCollectionFromProject={disconnectRAGCollectionFromProject}
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
            />
          )}
      </Suspense>
    </AppShell>
  );
}
