import { FormEvent, useEffect, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Copy,
  Info,
  LoaderCircle,
  Pause,
  Play,
  Plus,
  Radio,
  RefreshCw,
  Server,
  Square,
  Star,
  Trash2,
} from 'lucide-react';
import {
  defaultLanguage,
  detectInitialLanguage,
  dictionaries,
  languageStorageKey,
  Language,
} from './i18n';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './components/ui/dialog';

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
  auto_start: boolean;
  status: 'Running' | 'Stopped' | 'Remote' | string;
  is_primary: boolean;
  is_enabled: boolean;
};

type ProjectStatus = {
  project_id: number;
  name: string;
  description: string;
  token: string;
  primary_server_id: number | null;
  is_paused: boolean;
  connect_url: string;
  connection_ready: boolean;
  servers: ServerStatus[];
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
  auto_start: boolean;
};

type KeyValuePair = {
  key: string;
  value: string;
};

const emptyProjectForm: ProjectFormState = {
  name: '',
  description: '',
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
  auto_start: false,
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

export default function App() {
  const [view, setView] = useState<'projects' | 'logs'>('projects');
  const [language, setLanguage] = useState<Language>(detectInitialLanguage);
  const [projects, setProjects] = useState<ProjectStatus[]>([]);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [logsCurrentProjectOnly, setLogsCurrentProjectOnly] = useState(false);
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [projectForm, setProjectForm] = useState<ProjectFormState>(emptyProjectForm);
  const [serverForm, setServerForm] = useState<ServerFormState>(emptyServerForm);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [creatingProject, setCreatingProject] = useState(false);
  const [createProjectOpen, setCreateProjectOpen] = useState(false);
  const [addingServer, setAddingServer] = useState(false);
  const [addServerOpen, setAddServerOpen] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [busyProjectId, setBusyProjectId] = useState<number | null>(null);
  const [busyServerId, setBusyServerId] = useState<number | null>(null);
  const [busyPrimaryId, setBusyPrimaryId] = useState<number | null>(null);
  const [inspectOpen, setInspectOpen] = useState(false);
  const [inspectingServerId, setInspectingServerId] = useState<number | null>(null);
  const [inspection, setInspection] = useState<ServerInspection | null>(null);
  const [inspectionServerName, setInspectionServerName] = useState('');
  const [inspectionError, setInspectionError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const dictionary = dictionaries[language];
  const { labels, messages } = dictionary;

  const selectedProject =
    projects.find((project) => project.project_id === selectedProjectId) ?? null;
  const filteredLogsProject =
    logsCurrentProjectOnly ? selectedProject : null;
  const serverNamesById = Object.fromEntries(
    projects.flatMap((project) =>
      project.servers.map((server) => [server.id, server.name] as const),
    ),
  );

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    window.localStorage.setItem(languageStorageKey, language);
    document.documentElement.lang = language;
  }, [language]);

  useEffect(() => {
    void loadProjects(true);
  }, []);

  useEffect(() => {
    if (view === 'logs') {
      void loadLogs();
    }
  }, [view, logsCurrentProjectOnly, selectedProjectId]);

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

  async function loadLogs() {
    setLogsLoading(true);
    try {
      const query =
        logsCurrentProjectOnly && selectedProjectId ? `?project_id=${selectedProjectId}` : '';
      const nextLogs = await apiRequest<AuditLog[]>(
        `/api/logs${query}`,
        messages.requestFailed,
      );
      setLogs(nextLogs);
    } catch (loadError) {
      setActionError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      setLogsLoading(false);
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
      await apiRequest('/api/projects', messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify(projectForm),
      });

      setProjectForm(emptyProjectForm);
      setCreateProjectOpen(false);
      await loadProjects();
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
        auto_start: serverForm.transport === 'stdio' ? serverForm.auto_start : false,
      };

      const updatedProject = await apiRequest<ProjectStatus>(
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
      setServerForm(emptyServerForm);
      setAddServerOpen(false);
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
    } finally {
      setBusyServerId(null);
    }
  }

  async function setPrimaryServer(serverId: number) {
    if (!selectedProject) {
      return;
    }

    setBusyPrimaryId(serverId);
    setActionError(null);

    try {
      const updatedProject = await apiRequest<ProjectStatus>(
        `/api/projects/${selectedProject.project_id}/primary-server`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ server_id: serverId }),
        },
      );

      setProjects((current) =>
        current.map((project) =>
          project.project_id === updatedProject.project_id ? updatedProject : project,
        ),
      );
      await loadLogs();
    } catch (submitError) {
      setActionError(
        submitError instanceof Error ? submitError.message : messages.updatePrimaryError,
      );
    } finally {
      setBusyPrimaryId(null);
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
      setActionError(submitError instanceof Error ? submitError.message : messages.updatePrimaryError);
    } finally {
      setBusyProjectId(null);
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
      setActionError(
        submitError instanceof Error ? submitError.message : messages.updatePrimaryError,
      );
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
      <div className="mx-auto flex min-h-screen max-w-[1600px]">
        <aside className="w-full max-w-sm border-r border-border bg-sidebar/40">
          <div className="border-b border-border px-6 py-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">{labels.appTitle}</p>
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center rounded-md border border-border bg-card p-1 text-xs">
                  <button
                    onClick={() => setLanguage(defaultLanguage)}
                    className={`rounded px-2 py-1 transition-colors ${
                      language === 'en' ? 'bg-electric-blue text-white' : 'text-muted-foreground'
                    }`}
                    aria-label={labels.english}
                  >
                    EN
                  </button>
                  <button
                    onClick={() => setLanguage('ru')}
                    className={`rounded px-2 py-1 transition-colors ${
                      language === 'ru' ? 'bg-electric-blue text-white' : 'text-muted-foreground'
                    }`}
                    aria-label={labels.russian}
                  >
                    RU
                  </button>
                </div>
                <button
                  onClick={() => void loadProjects()}
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
            <div className="grid grid-cols-2 gap-2 rounded-xl border border-border bg-card p-1">
              <button
                onClick={() => setView('projects')}
                className={`h-10 rounded-lg text-sm font-medium transition-colors ${
                  view === 'projects'
                    ? 'bg-electric-blue text-white'
                    : 'text-muted-foreground hover:bg-accent'
                }`}
              >
                {labels.projects}
              </button>
              <button
                onClick={() => setView('logs')}
                className={`h-10 rounded-lg text-sm font-medium transition-colors ${
                  view === 'logs'
                    ? 'bg-electric-blue text-white'
                    : 'text-muted-foreground hover:bg-accent'
                }`}
              >
                {labels.logs}
              </button>
            </div>

            <section className="space-y-3">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-medium text-muted-foreground">{labels.projects}</h2>
                <div className="flex items-center gap-2">
                  <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
                    {projects.length}
                  </span>
                  <Dialog open={createProjectOpen} onOpenChange={setCreateProjectOpen}>
                    <DialogTrigger asChild>
                      <button className="inline-flex h-8 items-center justify-center gap-2 rounded-md bg-electric-blue px-3 text-xs font-medium text-white transition-colors hover:bg-electric-blue/90">
                        <Plus className="h-3.5 w-3.5" />
                        {labels.createProject}
                      </button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                        <DialogTitle>{labels.createProject}</DialogTitle>
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
                          {labels.createProject}
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
                    <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-2 text-sm text-muted-foreground">
                      <input
                        type="checkbox"
                        checked={logsCurrentProjectOnly}
                        onChange={(event) => setLogsCurrentProjectOnly(event.target.checked)}
                        className="h-4 w-4 rounded border-border"
                      />
                      <span>{labels.currentProjectOnly}</span>
                    </label>
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
                    <div className="max-h-[70vh] overflow-y-auto">
                      {logs.map((entry) => (
                        <div
                          key={entry.id}
                          className="border-b border-white/5 px-4 py-3 font-mono text-xs text-slate-200 last:border-b-0"
                        >
                          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                            <span className="text-slate-500">
                              {new Date(entry.created_at).toLocaleTimeString()}
                            </span>
                            <span className="text-electric-blue">{entry.action}</span>
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
                              {entry.detail}
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

                    <div className="grid gap-3 sm:grid-cols-3">
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
                        <div className="text-sm text-muted-foreground">{labels.primary}</div>
                        <div className="mt-1 flex items-center gap-2 text-sm font-medium">
                          {selectedProject.servers.find((server) => server.is_primary)?.name ??
                            labels.notSelected}
                          {selectedProject.is_paused ? (
                            <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-xs font-medium text-amber-600">
                              {labels.paused}
                            </span>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="flex justify-end">
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
                  </div>
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
                      {selectedProject.connection_ready ? labels.ready : labels.primaryRequired}
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
                    <h3 className="text-lg font-semibold">{labels.servers}</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {messages.serverControlDescription}
                    </p>
                  </div>
                  <Dialog open={addServerOpen} onOpenChange={setAddServerOpen}>
                    <DialogTrigger asChild>
                      <button className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
                        <Plus className="h-4 w-4" />
                        {labels.addServer}
                      </button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-xl">
                      <DialogHeader>
                        <DialogTitle>{labels.addServer}</DialogTitle>
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
                          {labels.addServer}
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
                      const primaryBusy = busyPrimaryId === server.id;

                      return (
                        <div
                          key={server.id}
                          className={`rounded-xl border p-5 ${
                            server.is_primary
                              ? 'border-electric-blue/50 bg-electric-blue/6'
                              : 'border-border bg-background'
                          }`}
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <div className="flex items-center gap-2">
                                <h4 className="font-semibold">{server.name}</h4>
                                {server.is_primary ? (
                                  <span className="inline-flex items-center gap-1 rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue">
                                    <Star className="h-3 w-3 fill-current" />
                                    {labels.primary}
                                  </span>
                                ) : null}
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
                              onClick={() => void setPrimaryServer(server.id)}
                              disabled={server.is_primary || primaryBusy || !server.is_enabled}
                              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              {primaryBusy ? (
                                <LoaderCircle className="h-4 w-4 animate-spin" />
                              ) : (
                                <Star className="h-4 w-4" />
                              )}
                              {server.is_primary ? labels.primaryServer : labels.setAsPrimary}
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
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
