import { useMemo, useState, type Dispatch, type FormEvent, type SetStateAction } from 'react';

import {
  CheckCircle2,
  Info,
  LoaderCircle,
  Pause,
  Pencil,
  Play,
  Plus,
  Radio,
  Settings2,
  Square,
  Trash2,
} from 'lucide-react';

import { dictionaries } from '../i18n';
import {
  applyProjectDefaultsToInstallValues,
  catalogConfigFields,
  catalogEnvFields,
  normalizeEnvConfig,
  normalizeInstallConfig,
  type CatalogItem,
  type CatalogConfigField,
  type InstalledPackage,
} from '../market';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';
import { Input } from './ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';

type KeyValuePair = {
  key: string;
  value: string;
};

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

type ProjectServersPanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  selectedProject: {
    project_id: number;
    name: string;
    root_path: string;
    servers: ServerStatus[];
    installed_integrations?: Array<{ catalog_item_id: string }>;
  };
  addServerOpen: boolean;
  setAddServerOpen: (open: boolean) => void;
  editingServerId: number | null;
  resetServerEditor: () => void;
  serverForm: ServerFormState;
  updateServerForm: <K extends keyof ServerFormState>(key: K, value: ServerFormState[K]) => void;
  updateStringListField: (
    key: 'args' | 'env_passthrough' | 'oauth_scopes',
    index: number,
    value: string,
  ) => void;
  removeStringListField: (key: 'args' | 'env_passthrough' | 'oauth_scopes', index: number) => void;
  addStringListField: (key: 'args' | 'env_passthrough' | 'oauth_scopes') => void;
  updateKeyValueField: (
    key: 'env_vars' | 'headers' | 'header_env_vars',
    index: number,
    field: keyof KeyValuePair,
    value: string,
  ) => void;
  removeKeyValueField: (key: 'env_vars' | 'headers' | 'header_env_vars', index: number) => void;
  addKeyValueField: (key: 'env_vars' | 'headers' | 'header_env_vars') => void;
  updateServerLastArg: (value: string) => void;
  editingServerIntegrationCatalogItemId: string | null;
  catalogItems: CatalogItem[];
  installedPackages: InstalledPackage[];
  addingCatalogItemId: string | null;
  onPerformCatalogInstall: (
    item: CatalogItem,
    projectId: number,
    config: Record<string, unknown>,
  ) => Promise<boolean>;
  onActionError: (message: string | null) => void;
  addingServer: boolean;
  addServer: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  busyServerId: number | null;
  checkServerHealth: (serverId: number) => void | Promise<void>;
  runServerAction: (serverId: number, action: 'start' | 'stop') => void | Promise<void>;
  openAuthModal: (serverId: number) => void;
  openServerTools: (server: ServerStatus) => void | Promise<void>;
  serverToolsLoadingId: number | null;
  inspectServer: (server: ServerStatus) => void | Promise<void>;
  inspectingServerId: number | null;
  startEditServer: (server: ServerStatus) => void;
  setServerEnabled: (serverId: number, enabled: boolean) => void | Promise<void>;
  deleteServer: (serverId: number) => void | Promise<void>;
};

function isSecretLikeName(name: string) {
  const normalized = name.trim().toLowerCase();
  if (!normalized) {
    return false;
  }

  return (
    normalized.includes('password') ||
    normalized.includes('passwd') ||
    normalized.includes('secret') ||
    normalized.includes('token') ||
    normalized.includes('private_key') ||
    normalized.includes('private-key') ||
    normalized.includes('api_key') ||
    normalized.includes('api-key') ||
    normalized === 'authorization'
  );
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

export function ProjectServersPanel({
  labels,
  messages,
  selectedProject,
  addServerOpen,
  setAddServerOpen,
  editingServerId,
  resetServerEditor,
  serverForm,
  updateServerForm,
  updateStringListField,
  removeStringListField,
  addStringListField,
  updateKeyValueField,
  removeKeyValueField,
  addKeyValueField,
  updateServerLastArg,
  editingServerIntegrationCatalogItemId,
  catalogItems,
  installedPackages,
  addingCatalogItemId,
  onPerformCatalogInstall,
  onActionError,
  addingServer,
  addServer,
  busyServerId,
  checkServerHealth,
  runServerAction,
  openAuthModal,
  openServerTools,
  serverToolsLoadingId,
  inspectServer,
  inspectingServerId,
  startEditServer,
  setServerEnabled,
  deleteServer,
}: ProjectServersPanelProps) {
  const [addServerTab, setAddServerTab] = useState<'stdio_http' | 'installed'>('stdio_http');
  const [installedConfigItem, setInstalledConfigItem] = useState<CatalogItem | null>(null);
  const [installedConfigValues, setInstalledConfigValues] = useState<Record<string, string>>({});
  const [installedEnvValues, setInstalledEnvValues] = useState<Record<string, string>>({});

  const installedCatalogItems = useMemo(() => {
    const installedPackageIDs = new Set(
      installedPackages
        .filter((pkg) => pkg.status === 'installed')
        .map((pkg) => pkg.catalog_item_id),
    );

    return catalogItems
      .filter((item) => installedPackageIDs.has(item.id))
      .sort((left, right) => left.name.localeCompare(right.name));
  }, [catalogItems, installedPackages]);

  const projectCatalogIDs = useMemo(
    () => new Set((selectedProject.installed_integrations ?? []).map((integration) => integration.catalog_item_id)),
    [selectedProject.installed_integrations],
  );

  const installedConfigFields = installedConfigItem ? catalogConfigFields(installedConfigItem) : [];
  const installedEnvFields = installedConfigItem ? catalogEnvFields(installedConfigItem) : [];
  const selectedProjectOption = {
    project_id: selectedProject.project_id,
    name: selectedProject.name,
    root_path: selectedProject.root_path,
  };

  function openInstalledConfig(item: CatalogItem) {
    const fields = catalogConfigFields(item);
    const envFields = catalogEnvFields(item);
    const baseValues = Object.fromEntries(fields.map((field) => [field.key, field.defaultValue]));
    const baseEnvValues = Object.fromEntries(envFields.map((field) => [field.key, field.defaultValue]));

    setInstalledConfigItem(item);
    setInstalledConfigValues(
      applyProjectDefaultsToInstallValues(baseValues, fields, selectedProjectOption),
    );
    setInstalledEnvValues(baseEnvValues);
  }

  async function addInstalledItem(item: CatalogItem) {
    const fields = catalogConfigFields(item);
    const envFields = catalogEnvFields(item);
    if (fields.length > 0 || envFields.length > 0) {
      openInstalledConfig(item);
      return;
    }

    await onPerformCatalogInstall(item, selectedProject.project_id, {});
  }

  async function submitInstalledConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!installedConfigItem) {
      onActionError('Installed MCP server is not selected.');
      return;
    }

    const config = normalizeInstallConfig(installedConfigFields, installedConfigValues);
    const env = {
      ...(config.env && typeof config.env === 'object' ? (config.env as Record<string, string>) : {}),
      ...normalizeEnvConfig(installedEnvFields, installedEnvValues),
    };
    if (Object.keys(env).length > 0) {
      config.env = env;
    }

    const success = await onPerformCatalogInstall(
      installedConfigItem,
      selectedProject.project_id,
      config,
    );
    if (success) {
      setInstalledConfigItem(null);
      setInstalledConfigValues({});
      setInstalledEnvValues({});
      setAddServerOpen(false);
    }
  }

  function renderInstalledConfigField(
    field: CatalogConfigField,
    values: Record<string, string>,
    setValues: Dispatch<SetStateAction<Record<string, string>>>,
    prefix: string,
  ) {
    return (
      <label key={`${prefix}-${field.key}`} className="block space-y-2">
        <span className="text-sm text-muted-foreground">
          {field.label}
          {field.required ? ' *' : ''}
        </span>
        {field.description ? (
          <span className="block text-xs text-muted-foreground">{field.description}</span>
        ) : null}
        {field.type === 'array' ? (
          <textarea
            value={values[field.key] ?? ''}
            onChange={(event) => setValues((current) => ({ ...current, [field.key]: event.target.value }))}
            rows={4}
            className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue"
            placeholder={messages.oneValuePerLine}
            required={field.required}
          />
        ) : (
          <Input
            type={field.secret ? 'password' : 'text'}
            value={values[field.key] ?? ''}
            onChange={(event) => setValues((current) => ({ ...current, [field.key]: event.target.value }))}
            className="h-10"
            required={field.required}
          />
        )}
      </label>
    );
  }

  return (
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
              resetServerEditor();
              setInstalledConfigItem(null);
              setInstalledConfigValues({});
              setInstalledEnvValues({});
              setAddServerTab('stdio_http');
            }
          }}
        >
          <DialogTrigger asChild>
            <button className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
              <Plus className="h-4 w-4" />
              {labels.addServer}
            </button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-3xl">
            <DialogHeader>
              <DialogTitle>{editingServerId ? 'Edit MCP Server' : labels.addServer}</DialogTitle>
              <DialogDescription>{messages.addServerDescription}</DialogDescription>
            </DialogHeader>

            <Tabs value={addServerTab} onValueChange={(value) => setAddServerTab(value as 'stdio_http' | 'installed')} className="space-y-4">
              <TabsList className="grid h-auto w-full grid-cols-2 rounded-xl border border-border bg-background p-1">
                <TabsTrigger
                  value="stdio_http"
                  className="rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-all data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-md data-[state=active]:shadow-electric-blue/20"
                >
                  STDIO / HTTP
                </TabsTrigger>
                <TabsTrigger
                  value="installed"
                  className="rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-all data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-md data-[state=active]:shadow-electric-blue/20"
                >
                  Installed
                </TabsTrigger>
              </TabsList>

              <TabsContent value="stdio_http" className="space-y-4">
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
                            onChange={(event) => updateStringListField('args', index, event.target.value)}
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
                            onChange={(event) => updateKeyValueField('env_vars', index, 'key', event.target.value)}
                            className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={labels.key}
                          />
                          <input
                            type={isSecretLikeName(pair.key) ? 'password' : 'text'}
                            value={pair.value}
                            onChange={(event) => updateKeyValueField('env_vars', index, 'value', event.target.value)}
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
                            onChange={(event) => updateStringListField('env_passthrough', index, event.target.value)}
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

                  {editingServerIntegrationCatalogItemId === 'filesystem' ? (
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
                      onChange={(event) => updateServerForm('bearer_token_env_var', event.target.value)}
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
                            onChange={(event) => updateKeyValueField('headers', index, 'key', event.target.value)}
                            className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={labels.key}
                          />
                          <input
                            value={pair.value}
                            onChange={(event) => updateKeyValueField('headers', index, 'value', event.target.value)}
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
                            onChange={(event) => updateKeyValueField('header_env_vars', index, 'key', event.target.value)}
                            className="h-10 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                            placeholder={labels.key}
                          />
                          <input
                            value={pair.value}
                            onChange={(event) => updateKeyValueField('header_env_vars', index, 'value', event.target.value)}
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
                    {addingServer ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                    {editingServerId ? 'Save Server' : labels.addServer}
                  </button>
                </form>
              </TabsContent>

              <TabsContent value="installed" className="space-y-4">
                {installedCatalogItems.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-center text-sm text-muted-foreground">
                    No installed MCP servers yet. Install a server from Market first.
                  </div>
                ) : installedConfigItem ? (
                  <form className="space-y-4" onSubmit={submitInstalledConfig}>
                    <div className="rounded-xl border border-border bg-background p-4">
                      <div className="font-medium">{installedConfigItem.name}</div>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {installedConfigItem.description || 'Configure this installed server for the current project.'}
                      </p>
                      {selectedProject.root_path ? (
                        <div className="mt-2 text-xs text-muted-foreground">
                          Current project folder: {selectedProject.root_path}
                        </div>
                      ) : null}
                    </div>

                    {installedConfigFields.map((field) =>
                      renderInstalledConfigField(field, installedConfigValues, setInstalledConfigValues, 'installed-config'),
                    )}

                    {installedEnvFields.length > 0 ? (
                      <div className="space-y-4 rounded-xl border border-border bg-background p-4">
                        <div>
                          <div className="text-sm font-medium">{labels.environmentVariables}</div>
                          <div className="mt-1 text-xs text-muted-foreground">{messages.envSchemaDescription}</div>
                        </div>
                        {installedEnvFields.map((field) => (
                          <div key={`installed-env-wrap-${field.key}`} className="space-y-1">
                            <code className="block text-xs text-electric-blue">{field.key}</code>
                            {renderInstalledConfigField(field, installedEnvValues, setInstalledEnvValues, 'installed-env')}
                          </div>
                        ))}
                      </div>
                    ) : null}

                    <div className="flex flex-col gap-2 sm:flex-row">
                      <button
                        type="button"
                        onClick={() => {
                          setInstalledConfigItem(null);
                          setInstalledConfigValues({});
                          setInstalledEnvValues({});
                        }}
                        className="inline-flex h-10 flex-1 items-center justify-center rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                      >
                        Back to installed
                      </button>
                      <button
                        type="submit"
                        disabled={addingCatalogItemId === installedConfigItem.id}
                        className="inline-flex h-10 flex-1 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {addingCatalogItemId === installedConfigItem.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                        {labels.addToProject}
                      </button>
                    </div>
                  </form>
                ) : (
                  <div className="grid gap-3">
                    {installedCatalogItems.map((item) => {
                      const packageInfo = installedPackages.find((pkg) => pkg.catalog_item_id === item.id && pkg.status === 'installed') ?? null;
                      const alreadyInProject = projectCatalogIDs.has(item.id);
                      const busy = addingCatalogItemId === item.id;
                      const primaryValue = item.transport === 'stdio'
                        ? [item.command, ...(item.args ?? [])].filter(Boolean).join(' ')
                        : item.mcp_url || 'No endpoint';

                      return (
                        <div key={`installed-${item.id}`} className="rounded-xl border border-border bg-background p-4">
                          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                            <div className="min-w-0 flex-1">
                              <div className="font-medium">{item.name}</div>
                              <div className="mt-1 text-sm text-muted-foreground">
                                {item.description || 'Installed MCP server from Market.'}
                              </div>
                              <div className="mt-3 flex flex-wrap gap-2">
                                <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                  {item.transport === 'stdio' ? 'STDIO' : item.transport}
                                </span>
                                <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                                  {item.runtime.type || 'runtime'}
                                </span>
                                {alreadyInProject ? (
                                  <span className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue">
                                    Already in project
                                  </span>
                                ) : null}
                              </div>
                              <code className="mt-3 block overflow-x-auto rounded-md bg-card px-3 py-2 text-xs text-electric-blue">
                                {primaryValue || 'No command'}
                              </code>
                              {packageInfo?.install_dir ? (
                                <div className="mt-2 text-xs text-muted-foreground">
                                  Installed at: {packageInfo.install_dir}
                                </div>
                              ) : null}
                            </div>
                            <button
                              type="button"
                              onClick={() => void addInstalledItem(item)}
                              disabled={busy || !item.enabled}
                              className="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                              {labels.addToProject}
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </TabsContent>
            </Tabs>
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
              <div key={server.id} className="rounded-xl border border-border bg-background p-5">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="font-semibold">{server.name}</h4>
                      <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">
                        {server.transport === 'http_stream' ? labels.httpStreaming : labels.stdio}
                      </span>
                    </div>

                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium ${statusTone(server.status)}`}>
                        {statusIcon(server.status)}
                        {server.status}
                      </span>
                      <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium ${healthTone(server.health_status)}`}>
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
                      {(server.disabled_tool_names?.length ?? 0) > 0 ? (
                        <span className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue">
                          {messages.disabledToolsBadge(server.disabled_tool_names.length)}
                        </span>
                      ) : null}
                      {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                        <span className={`rounded-full border px-2 py-1 text-xs font-medium ${
                          server.oauth_connected
                            ? 'border-status-running/30 bg-status-running/12 text-status-running'
                            : 'border-amber-500/30 bg-amber-500/10 text-amber-700'
                        }`}>
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
                      {server.transport === 'http_stream' ? server.url : server.launch_command_display || server.launch_command}
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
                      <div className="text-muted-foreground">{labels.bearerTokenEnvironmentVariable}</div>
                      <div className="mt-1 text-sm">{server.bearer_token_env_var}</div>
                    </div>
                  ) : null}
                  {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                    <div>
                      <div className="text-muted-foreground">OAuth</div>
                      <div className="mt-1 text-sm">
                        {server.oauth_provider || 'custom'}
                        {server.oauth_connected_at ? ` · connected ${new Date(server.oauth_connected_at).toLocaleString()}` : ''}
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
                      {server.health_checked_at ? new Date(server.health_checked_at).toLocaleString() : labels.notSpecified}
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
                      onClick={() => void runServerAction(server.id, server.status === 'Running' ? 'stop' : 'start')}
                      disabled={busy || !server.is_enabled}
                      className={`inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-70 ${
                        server.status === 'Running'
                          ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                          : 'bg-status-running text-white hover:bg-status-running/90'
                      }`}
                    >
                      {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : server.status === 'Running' ? <Square className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                      {server.status === 'Running' ? labels.stop : labels.start}
                    </button>
                  ) : null}

                  <button
                    onClick={() => void checkServerHealth(server.id)}
                    disabled={busy}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                    {labels.check}
                  </button>

                  {server.transport === 'http_stream' && server.auth_type === 'oauth2' ? (
                    <button
                      onClick={() => openAuthModal(server.id)}
                      disabled={busy}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Info className="h-4 w-4" />}
                      {labels.oauth}
                    </button>
                  ) : null}

                  <button
                    onClick={() => void openServerTools(server)}
                    disabled={serverToolsLoadingId === server.id}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {serverToolsLoadingId === server.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Settings2 className="h-4 w-4" />}
                    {labels.manageTools}
                  </button>

                  {server.transport === 'stdio' ? (
                    <button
                      onClick={() => void inspectServer(server)}
                      disabled={inspectingServerId === server.id}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {inspectingServerId === server.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Info className="h-4 w-4" />}
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
                    {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : server.is_enabled ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
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
  );
}
