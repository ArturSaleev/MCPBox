import { Suspense, lazy, useEffect, useRef, useState } from 'react';
import { Toaster, toast } from 'sonner';
import {
  Bot,
  Database,
  FolderKanban,
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
import type {
  ProjectStatus,
  ServerStatus,
  RAGCollection,
  RAGSearchResult,
  AuditLog,
  PerformanceMetricsResponse,
  MetricsWindow,
  EditionMeta,
  ServerInspection,
  ServerToolStatus,
  KeyValuePair,
  LlamaCppLaunchResponse,
  LlamaCppStatus,
  OllamaStatus,
  OllamaLaunchResponse,
  ProjectFormState,
  ServerFormState,
} from './types';
import { useAuth } from './hooks/useAuth';
import { useLlamaCpp } from './hooks/useLlamaCpp';
import { useOllama } from './hooks/useOllama';
import { useServerTools } from './hooks/useServerTools';
import { apiRequest } from './utils/api';
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
  const [view, setView] = useState<'projects' | 'knowledge' | 'market' | 'logs'>('projects');
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
  const [activeRAGSearchCollectionId, setActiveRAGSearchCollectionId] = useState<string | null>(null);
  const [indexingCollectionId, setIndexingCollectionId] = useState<string | null>(null);
  const [searchingCollectionId, setSearchingCollectionId] = useState<string | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [catalogSettings, setCatalogSettings] = useState<CatalogSettings | null>(null);
  const [installedPackages, setInstalledPackages] = useState<InstalledPackage[]>([]);

  // Ollama hook
  const { ollamaStatus, setOllamaStatus, ollamaRefreshing, setOllamaRefreshing, selectedOllamaModel, setSelectedOllamaModel, loadOllamaStatus: hookLoadOllamaStatus } = useOllama({ requestFailed: 'Request failed' });
  const { llamaCppStatus, setLlamaCppStatus, llamaCppRefreshing, setLlamaCppRefreshing, loadLlamaCppStatus: hookLoadLlamaCppStatus } = useLlamaCpp({ requestFailed: 'Request failed' });

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
  const [launchingLlamaCppProjectId, setLaunchingLlamaCppProjectId] = useState<number | null>(null);
  const [launchingLMStudioProjectId, setLaunchingLMStudioProjectId] = useState<number | null>(null);
  const [stoppingLlamaCpp, setStoppingLlamaCpp] = useState(false);
  const [selectedLlamaCppModelPath, setSelectedLlamaCppModelPath] = useState('');
  const [selectedLlamaCppModelName, setSelectedLlamaCppModelName] = useState('');

  // Auth hook
  const { authOpen, setAuthOpen, authServerId, setAuthServerId, resetAuthServer } = useAuth();

  const authServer: ServerStatus | null = authServerId !== null
    ? projects.flatMap((p) => p.servers).find((s) => s.id === authServerId) ?? null
    : null;

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
  const editingServerIntegration =
    editingServerId !== null ? serverIntegrationsByServerID[editingServerId] ?? null : null;
  const projectOptions: ProjectOption[] = projects.map((project) => ({
    project_id: project.project_id,
    name: project.name,
    root_path: project.root_path,
  }));
  const shouldShowOllamaControls = ollamaStatus?.installed ?? false;
  const shouldShowLlamaCppControls = llamaCppStatus?.installed ?? false;
  const canLaunchOllama =
    !!selectedOllamaModel &&
    (ollamaStatus?.models.length ?? 0) > 0;
  const canLaunchLlamaCpp =
    !!llamaCppStatus?.installed &&
    !!(
      selectedLlamaCppModelPath.trim() ||
      selectedProject?.llama_cpp_model_path?.trim() ||
      llamaCppStatus?.model_path?.trim()
    );
  const requestTrendValues = logMetrics?.trends.map((entry) => entry.request_count) ?? [];
  const errorTrendValues = logMetrics?.trends.map((entry) => entry.error_count) ?? [];
  const avgLatencyTrendValues = logMetrics?.trends.map((entry) => entry.avg_latency_ms) ?? [];
  const p95LatencyTrendValues = logMetrics?.trends.map((entry) => entry.p95_latency_ms) ?? [];
  const visibleLogs = logs;
  const activeView = view;

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
    void Promise.all([
      loadEditionMeta(),
      loadProjects(true),
      loadRAGCollections(),
      loadCatalog(true),
      loadInstalledPackages(),
      loadOllamaStatus(),
      loadLlamaCppStatus(),
    ]);
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

  useEffect(() => {
    if (view === 'logs') {
      void loadLogs();
      void loadLogMetrics();
    }
  }, [view, selectedLogsProjectId, metricsWindow]);

  useEffect(() => {
    if (!launchProjectOpen) {
      return;
    }

    void Promise.all([loadOllamaStatus({ silent: true }), loadLlamaCppStatus({ silent: true })]);
  }, [launchProjectOpen]);

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

  const loadOllamaStatus = hookLoadOllamaStatus;
  const loadLlamaCppStatus = hookLoadLlamaCppStatus;

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

  async function updateProjectPrompt(prompt: string) {
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
            prompt,
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
      navigationItems={navigationItems}
      view={activeView}
      setView={setView}
      immersive={false}
      sidebar={
        activeView === 'projects' ? (
          <ProjectsSidebar
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
              visibleLogs={visibleLogs}
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
          ) : (
            <ProjectsView
              labels={labels}
              messages={messages}
              selectedProject={selectedProject}
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
              updateProjectPrompt={updateProjectPrompt}
              updatingPrompt={updatingPrompt}
            />
          )}
      </Suspense>
    </AppShell>
  );
}
