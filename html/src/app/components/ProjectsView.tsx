import type { FormEvent } from 'react';

import { Server } from 'lucide-react';

import { dictionaries } from '../i18n';
import type { CatalogItem, InstalledPackage } from '../market';
import { ProjectKnowledgePanel } from './ProjectKnowledgePanel';
import { ProjectLaunchPanel } from './ProjectLaunchPanel';
import { ProjectPromptPanel } from './ProjectPromptPanel';
import { ProjectServerDialogs } from './ProjectServerDialogs';
import { ProjectServersPanel } from './ProjectServersPanel';
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

type RAGCollection = {
  id?: number;
  collection_id: string;
  name: string;
  data_type: string;
  source_path?: string;
  auto_reindex?: boolean;
  index_path?: string;
};

type RAGCollectionForm = {
  name: string;
  source_path: string;
  auto_reindex: boolean;
};

type ProjectStatus = {
  project_id: number;
  name: string;
  description: string;
  root_path: string;
  token: string;
  is_paused: boolean;
  llama_cpp_model_path: string;
  llama_cpp_model_name: string;
  connection_ready: boolean;
  connect_url: string;
  servers: ServerStatus[];
  rag_collections: RAGCollection[];
  installed_integrations: Array<{ catalog_item_id: string }>;
  prompt: string;
  prompt_profiles: Array<{
    id: string;
    name: string;
    description: string;
    prompt: string;
    response_format: string;
    response_schema: string;
    is_default: boolean;
  }>;
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

type OllamaStatus = {
  models: string[];
} | null;

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

type AuthServer = {
  id: number;
  name: string;
  oauth_provider: string;
  oauth_connected: boolean;
  oauth_connected_at: string;
  oauth_last_error: string;
  oauth_authorize_url: string;
  oauth_token_url: string;
  oauth_scopes: string[];
};

type ProjectsViewProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  selectedProject: ProjectStatus | null;
  selectedProjectHealthyCount: number;
  launchProjectOpen: boolean;
  setLaunchProjectOpen: (open: boolean) => void;
  shouldShowOllamaControls: boolean;
  shouldShowLlamaCppControls: boolean;
  ollamaStatus: OllamaStatus;
  llamaCppStatus: {
    configured: boolean;
    model_name: string;
    model_path: string;
    server_url: string;
    running: boolean;
    managed: boolean;
    active_model_path: string;
    active_model_name: string;
  } | null;
  selectedLlamaCppModelPath: string;
  setSelectedLlamaCppModelPath: (value: string) => void;
  selectedLlamaCppModelName: string;
  setSelectedLlamaCppModelName: (value: string) => void;
  selectedOllamaModel: string;
  setSelectedOllamaModel: (value: string) => void;
  loadOllamaStatus: () => void | Promise<void>;
  loadLlamaCppStatus: () => void | Promise<void>;
  ollamaRefreshing: boolean;
  llamaCppRefreshing: boolean;
  launchProjectOllama: (projectId: number) => void | Promise<void>;
  launchProjectLlamaCpp: (projectId: number) => void | Promise<void>;
  stopLlamaCppServer: () => void | Promise<void>;
  launchingOllamaProjectId: number | null;
  launchingLlamaCppProjectId: number | null;
  stoppingLlamaCpp: boolean;
  canLaunchOllama: boolean;
  canLaunchLlamaCpp: boolean;
  launchProjectLMStudio: (projectId: number) => void | Promise<void>;
  launchingLMStudioProjectId: number | null;
  OllamaIcon: (props: { className?: string }) => JSX.Element;
  alternativeConnectURLs: string[];
  connectionURLsExpanded: boolean;
  setConnectionURLsExpanded: (updater: (current: boolean) => boolean) => void;
  copyConnectURL: () => void | Promise<void>;
  copied: boolean;
  busyProjectId: number | null;
  setProjectPaused: (projectId: number, paused: boolean) => void | Promise<void>;
  startDuplicateProject: () => void;
  startEditProject: () => void;
  deleteProject: (projectId: number) => void | Promise<void>;
  connectRAGCollectionOpen: boolean;
  setConnectRAGCollectionOpen: (open: boolean) => void;
  allRAGCollections: RAGCollection[];
  availableRAGCollections: RAGCollection[];
  connectRAGCollectionToProject: (collectionId: string) => void | Promise<void>;
  linkingCollectionId: string | null;
  disconnectRAGCollectionFromProject: (collectionId: string) => void | Promise<void>;
  ragCollectionForm: RAGCollectionForm;
  setRAGCollectionForm: (updater: (current: RAGCollectionForm) => RAGCollectionForm) => void;
  resetRAGCollectionForm: () => void;
  createAndConnectRAGCollectionToProject: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  creatingRAGCollection: boolean;
  ragIndexPaths: Record<string, string>;
  setRAGIndexPaths: (updater: (current: Record<string, string>) => Record<string, string>) => void;
  indexRAGCollection: (collectionId: string) => void | Promise<void>;
  indexingCollectionId: string | null;
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
  inspectOpen: boolean;
  setInspectOpen: (open: boolean) => void;
  inspectionServerName: string;
  inspectionError: string | null;
  inspection: ServerInspection | null;
  formatSchema: (schema: unknown) => string;
  serverToolsOpen: boolean;
  setServerToolsOpen: (open: boolean) => void;
  resetServerTools: () => void;
  serverToolsServerName: string;
  serverToolsError: string | null;
  serverTools: ServerToolStatus[];
  serverToolsSavingName: string | null;
  setServerToolEnabled: (toolName: string, enabled: boolean) => void | Promise<void>;
  authOpen: boolean;
  setAuthOpen: (open: boolean) => void;
  resetAuthServer: () => void;
  authServer: AuthServer | null;
  connectOAuth: (serverId: number) => void | Promise<void>;
  disconnectOAuth: (serverId: number) => void | Promise<void>;
  updateProjectPrompt: (
    prompt: string,
    promptProfiles: ProjectStatus['prompt_profiles'],
  ) => void | Promise<void>;
  updatingPrompt: boolean;
};

export function ProjectsView({
  labels,
  messages,
  selectedProject,
  selectedProjectHealthyCount,
  launchProjectOpen,
  setLaunchProjectOpen,
  shouldShowOllamaControls,
  shouldShowLlamaCppControls,
  ollamaStatus,
  llamaCppStatus,
  selectedLlamaCppModelPath,
  setSelectedLlamaCppModelPath,
  selectedLlamaCppModelName,
  setSelectedLlamaCppModelName,
  selectedOllamaModel,
  setSelectedOllamaModel,
  loadOllamaStatus,
  loadLlamaCppStatus,
  ollamaRefreshing,
  llamaCppRefreshing,
  launchProjectOllama,
  launchProjectLlamaCpp,
  stopLlamaCppServer,
  launchingOllamaProjectId,
  launchingLlamaCppProjectId,
  stoppingLlamaCpp,
  canLaunchOllama,
  canLaunchLlamaCpp,
  launchProjectLMStudio,
  launchingLMStudioProjectId,
  OllamaIcon,
  alternativeConnectURLs,
  connectionURLsExpanded,
  setConnectionURLsExpanded,
  copyConnectURL,
  copied,
  regenerateEndpointToken,
  busyProjectId,
  setProjectPaused,
  startDuplicateProject,
  startEditProject,
  deleteProject,
  connectRAGCollectionOpen,
  setConnectRAGCollectionOpen,
  allRAGCollections,
  availableRAGCollections,
  connectRAGCollectionToProject,
  linkingCollectionId,
  disconnectRAGCollectionFromProject,
  ragCollectionForm,
  setRAGCollectionForm,
  resetRAGCollectionForm,
  createAndConnectRAGCollectionToProject,
  creatingRAGCollection,
  ragIndexPaths,
  setRAGIndexPaths,
  indexRAGCollection,
  indexingCollectionId,
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
  inspectOpen,
  setInspectOpen,
  inspectionServerName,
  inspectionError,
  inspection,
  formatSchema,
  serverToolsOpen,
  setServerToolsOpen,
  resetServerTools,
  serverToolsServerName,
  serverToolsError,
  serverTools,
  serverToolsSavingName,
  setServerToolEnabled,
  authOpen,
  setAuthOpen,
  resetAuthServer,
  authServer,
  connectOAuth,
  disconnectOAuth,
  updateProjectPrompt,
  updatingPrompt,
}: ProjectsViewProps) {
  if (!selectedProject) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center rounded-2xl border border-dashed border-border bg-card/50">
        <div className="max-w-md text-center">
          <Server className="mx-auto h-12 w-12 text-electric-blue" />
          <h2 className="mt-4 text-2xl font-semibold">{labels.noProjectSelected}</h2>
          <p className="mt-2 text-muted-foreground">{messages.emptySelectionBody}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <ProjectLaunchPanel
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
        regenerateEndpointToken={regenerateEndpointToken}
        busyProjectId={busyProjectId}
        setProjectPaused={setProjectPaused}
        startDuplicateProject={startDuplicateProject}
        startEditProject={startEditProject}
        deleteProject={deleteProject}
      />

      <Tabs defaultValue="mcp" className="gap-5">
        <TabsList className="grid h-auto w-full grid-cols-3 rounded-2xl border border-border bg-card p-1.5">
          <TabsTrigger
            className="h-auto rounded-xl px-4 py-3 text-sm font-semibold text-muted-foreground transition-all data-[state=active]:border-electric-blue data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-lg data-[state=active]:shadow-electric-blue/20"
            value="mcp"
          >
            <span className="flex items-center gap-2">
              <span>MCP</span>
              <span className="rounded-full bg-background/80 px-2 py-0.5 text-xs text-foreground/70 data-[state=active]:bg-white/20 data-[state=active]:text-white">
                {selectedProject.servers.length}
              </span>
            </span>
          </TabsTrigger>
          <TabsTrigger
            className="h-auto rounded-xl px-4 py-3 text-sm font-semibold text-muted-foreground transition-all data-[state=active]:border-electric-blue data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-lg data-[state=active]:shadow-electric-blue/20"
            value="rag"
          >
            <span className="flex items-center gap-2">
              <span>RAG</span>
              <span className="rounded-full bg-background/80 px-2 py-0.5 text-xs text-foreground/70 data-[state=active]:bg-white/20 data-[state=active]:text-white">
                {selectedProject.rag_collections.length}
              </span>
            </span>
          </TabsTrigger>
          <TabsTrigger
            className="h-auto rounded-xl px-4 py-3 text-sm font-semibold text-muted-foreground transition-all data-[state=active]:border-electric-blue data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-lg data-[state=active]:shadow-electric-blue/20"
            value="prompt"
          >
            <span className="flex items-center gap-2">
              <span>Prompt</span>
              <span className="rounded-full bg-background/80 px-2 py-0.5 text-xs text-foreground/70 data-[state=active]:bg-white/20 data-[state=active]:text-white">
                {(selectedProject.prompt.trim() ? 1 : 0) + (selectedProject.prompt_profiles?.length ?? 0)}
              </span>
            </span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="mcp" className="space-y-6">
          <ProjectServersPanel
            labels={labels}
            messages={messages}
            selectedProject={selectedProject}
            addServerOpen={addServerOpen}
            setAddServerOpen={setAddServerOpen}
            editingServerId={editingServerId}
            resetServerEditor={resetServerEditor}
            serverForm={serverForm}
            updateServerForm={updateServerForm}
            updateStringListField={updateStringListField}
            removeStringListField={removeStringListField}
            addStringListField={addStringListField}
            updateKeyValueField={updateKeyValueField}
            removeKeyValueField={removeKeyValueField}
            addKeyValueField={addKeyValueField}
            updateServerLastArg={updateServerLastArg}
            editingServerIntegrationCatalogItemId={editingServerIntegrationCatalogItemId}
            catalogItems={catalogItems}
            installedPackages={installedPackages}
            addingCatalogItemId={addingCatalogItemId}
            onPerformCatalogInstall={onPerformCatalogInstall}
            onActionError={onActionError}
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
          />
        </TabsContent>

        <TabsContent value="rag">
          <ProjectKnowledgePanel
            labels={labels}
            messages={messages}
            connectRAGCollectionOpen={connectRAGCollectionOpen}
            setConnectRAGCollectionOpen={setConnectRAGCollectionOpen}
            allRAGCollections={allRAGCollections}
            availableRAGCollections={availableRAGCollections}
            connectRAGCollectionToProject={connectRAGCollectionToProject}
            linkingCollectionId={linkingCollectionId}
            selectedProject={selectedProject}
            disconnectRAGCollectionFromProject={disconnectRAGCollectionFromProject}
            busyProjectId={busyProjectId}
            ragCollectionForm={ragCollectionForm}
            setRAGCollectionForm={setRAGCollectionForm}
            resetRAGCollectionForm={resetRAGCollectionForm}
            createAndConnectRAGCollectionToProject={createAndConnectRAGCollectionToProject}
            creatingRAGCollection={creatingRAGCollection}
            ragIndexPaths={ragIndexPaths}
            setRAGIndexPaths={setRAGIndexPaths}
            indexRAGCollection={indexRAGCollection}
            indexingCollectionId={indexingCollectionId}
          />
        </TabsContent>

        <TabsContent value="prompt">
          <ProjectPromptPanel
            labels={labels}
            messages={messages}
            selectedProject={selectedProject}
            updateProjectPrompt={updateProjectPrompt}
            updatingPrompt={updatingPrompt}
          />
        </TabsContent>
      </Tabs>

      <ProjectServerDialogs
        labels={labels}
        messages={messages}
        inspectOpen={inspectOpen}
        setInspectOpen={setInspectOpen}
        inspectionServerName={inspectionServerName}
        inspectingServerId={inspectingServerId}
        inspectionError={inspectionError}
        inspection={inspection}
        formatSchema={formatSchema}
        serverToolsOpen={serverToolsOpen}
        setServerToolsOpen={setServerToolsOpen}
        resetServerTools={resetServerTools}
        serverToolsLoadingId={serverToolsLoadingId}
        serverToolsServerName={serverToolsServerName}
        serverToolsError={serverToolsError}
        serverTools={serverTools}
        serverToolsSavingName={serverToolsSavingName}
        setServerToolEnabled={setServerToolEnabled}
        authOpen={authOpen}
        setAuthOpen={setAuthOpen}
        resetAuthServer={resetAuthServer}
        authServer={authServer}
        busyServerId={busyServerId}
        connectOAuth={connectOAuth}
        disconnectOAuth={disconnectOAuth}
      />
    </div>
  );
}
