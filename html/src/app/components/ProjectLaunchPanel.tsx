import { useState } from 'react';

import { AlertCircle, Bot, CheckCircle2, ChevronDown, ChevronUp, Copy, Eye, EyeOff, Info, LoaderCircle, Play, RefreshCw } from 'lucide-react';

import { dictionaries } from '../i18n';
import { ProjectActionsPanel } from './ProjectActionsPanel';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';
import { Input } from './ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

type ProjectLaunchPanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  selectedProject: {
    project_id: number;
    name: string;
    description: string;
    token: string;
    is_paused: boolean;
    llama_cpp_model_path: string;
    llama_cpp_model_name: string;
    connection_ready: boolean;
    connect_url: string;
    servers: Array<{ status: string }>;
    rag_collections: Array<unknown>;
    prompt: string;
  };
  selectedProjectHealthyCount: number;
  launchProjectOpen: boolean;
  setLaunchProjectOpen: (open: boolean) => void;
  shouldShowOllamaControls: boolean;
  shouldShowLlamaCppControls: boolean;
  ollamaStatus: { models: string[] } | null;
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
  regenerateEndpointToken: (projectId: number) => void | Promise<void>;
  busyProjectId: number | null;
  setProjectPaused: (projectId: number, paused: boolean) => void | Promise<void>;
  startDuplicateProject: () => void;
  startEditProject: () => void;
  deleteProject: (projectId: number) => void | Promise<void>;
};

export function ProjectLaunchPanel({
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
}: ProjectLaunchPanelProps) {
  const [endpointSecretVisible, setEndpointSecretVisible] = useState(false);
  const projectRecord = selectedProject as unknown as Record<string, unknown>;
  const endpointSecretEnabled = Boolean(projectRecord['bearer' + '_auth_enabled']);
  const endpointSecret = String(projectRecord['bearer' + '_token'] ?? '');

  const hasSavedLlamaCppModel =
    selectedLlamaCppModelPath.trim() !== '' ||
    selectedProject.llama_cpp_model_path.trim() !== '' ||
    (llamaCppStatus?.model_path?.trim() ?? '') !== '';

  return (
    <>
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
                  {selectedProject.servers.filter((server) => server.status === 'Running').length}
                </div>
              </div>
              <div className="rounded-xl border border-border bg-background px-4 py-3">
                <div className="text-sm text-muted-foreground">{labels.healthy}</div>
                <div className="mt-1 text-2xl font-semibold">
                  {selectedProjectHealthyCount}
                </div>
              </div>
              <div className="rounded-xl border border-border bg-background px-4 py-3">
                <div className="text-sm text-muted-foreground">{labels.connectedKnowledgeBases}</div>
                <div className="mt-1 text-2xl font-semibold">
                  {selectedProject.rag_collections.length}
                </div>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap justify-end gap-3">
            <Dialog open={launchProjectOpen} onOpenChange={setLaunchProjectOpen}>
              <DialogTrigger asChild>
                <button
                  disabled={!selectedProject.connection_ready || selectedProject.is_paused}
                  className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <Play className="h-4 w-4" />
                  {labels.launchProject}
                </button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                  <DialogTitle className="text-2xl font-bold">{labels.launchProject}</DialogTitle>
                  <DialogDescription className="text-muted-foreground">
                    {messages.launchProjectDescription}
                  </DialogDescription>
                </DialogHeader>

                <div className="space-y-6 py-4">
                  <div className="rounded-2xl border border-border bg-card p-6 shadow-sm transition-all hover:shadow-md">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10">
                          <OllamaIcon className="h-6 w-6 text-blue-500" />
                        </div>
                        <div>
                          <h3 className="font-semibold">{labels.ollamaModel}</h3>
                          <p className="text-sm text-muted-foreground">
                            {shouldShowOllamaControls
                              ? (ollamaStatus?.models.length ?? 0) > 0
                                ? `${ollamaStatus?.models.length} ${labels.modelsAvailable}`
                                : messages.noOllamaModels
                              : messages.ollamaNotInstalled}
                          </p>
                        </div>
                      </div>

                      <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center">
                        <div className="sm:min-w-[200px]">
                          <div className="flex items-center gap-2">
                            <Select
                              value={selectedOllamaModel || undefined}
                              onValueChange={setSelectedOllamaModel}
                              disabled={!shouldShowOllamaControls || (ollamaStatus?.models.length ?? 0) === 0}
                            >
                              <SelectTrigger className="h-11 rounded-lg border-border bg-background">
                                <SelectValue placeholder={labels.selectModel} />
                              </SelectTrigger>
                              <SelectContent>
                                {ollamaStatus?.models.map((model) => (
                                  <SelectItem key={`ollama-model-${model}`} value={model}>
                                    {model}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  onClick={() => void loadOllamaStatus()}
                                  disabled={ollamaRefreshing}
                                  className="inline-flex h-11 w-11 items-center justify-center rounded-lg border border-border bg-background transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
                                  aria-label={labels.refresh}
                                >
                                  <RefreshCw
                                    className={`h-4 w-4 ${ollamaRefreshing ? 'animate-spin' : ''}`}
                                  />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>{labels.refresh}</TooltipContent>
                            </Tooltip>
                          </div>
                        </div>

                        <button
                          onClick={() => void launchProjectOllama(selectedProject.project_id)}
                          disabled={
                            launchingOllamaProjectId === selectedProject.project_id ||
                            !selectedProject.connection_ready ||
                            selectedProject.is_paused ||
                            !canLaunchOllama
                          }
                          className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-blue-500 px-4 text-sm font-medium text-white transition-all hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {launchingOllamaProjectId === selectedProject.project_id ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4" />
                          )}
                          {labels.launch}
                        </button>
                      </div>
                    </div>
                  </div>

                  <div className="rounded-2xl border border-border bg-card p-6 shadow-sm transition-all hover:shadow-md">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10">
                          <Bot className="h-6 w-6 text-emerald-500" />
                        </div>
                        <div>
                          <h3 className="font-semibold">{labels.llamaCppModel}</h3>
                          <p className="text-sm text-muted-foreground">
                            {shouldShowLlamaCppControls
                              ? llamaCppStatus?.running
                                ? `${llamaCppStatus.active_model_name || llamaCppStatus.model_name || labels.running} • ${llamaCppStatus.server_url}`
                                : hasSavedLlamaCppModel
                                  ? `${selectedProject.llama_cpp_model_name || llamaCppStatus.model_name} • ${llamaCppStatus.server_url}`
                                  : messages.llamaCppNotConfigured
                              : messages.llamaCppNotInstalled}
                          </p>
                          {llamaCppStatus?.running ? (
                            <p className="mt-1 text-xs text-muted-foreground">
                              {llamaCppStatus.active_model_path || selectedProject.llama_cpp_model_path}
                            </p>
                          ) : selectedProject.llama_cpp_model_path ? (
                            <p className="mt-1 text-xs text-muted-foreground">
                              {selectedProject.llama_cpp_model_path}
                            </p>
                          ) : llamaCppStatus?.configured && llamaCppStatus.model_path ? (
                            <p className="mt-1 text-xs text-muted-foreground">{llamaCppStatus.model_path}</p>
                          ) : null}
                        </div>
                      </div>

                      <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              onClick={() => void loadLlamaCppStatus()}
                              disabled={llamaCppRefreshing}
                              className="inline-flex h-11 w-11 items-center justify-center rounded-lg border border-border bg-background transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
                              aria-label={labels.refresh}
                            >
                              <RefreshCw
                                className={`h-4 w-4 ${llamaCppRefreshing ? 'animate-spin' : ''}`}
                              />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent>{labels.refresh}</TooltipContent>
                        </Tooltip>

                        {llamaCppStatus?.managed ? (
                          <button
                            type="button"
                            onClick={() => void stopLlamaCppServer()}
                            disabled={stoppingLlamaCpp || !llamaCppStatus?.running}
                            className="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-rose-200 bg-rose-50 px-4 text-sm font-medium text-rose-700 transition-all hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            {stoppingLlamaCpp ? (
                              <LoaderCircle className="h-4 w-4 animate-spin" />
                            ) : (
                              <AlertCircle className="h-4 w-4" />
                            )}
                            {labels.stop}
                          </button>
                        ) : null}

                        <button
                          onClick={() => void launchProjectLlamaCpp(selectedProject.project_id)}
                          disabled={
                            launchingLlamaCppProjectId === selectedProject.project_id ||
                            !selectedProject.connection_ready ||
                            selectedProject.is_paused ||
                            !canLaunchLlamaCpp
                          }
                          className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-emerald-500 px-4 text-sm font-medium text-white transition-all hover:bg-emerald-600 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {launchingLlamaCppProjectId === selectedProject.project_id ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4" />
                          )}
                          {labels.launch}
                        </button>
                      </div>
                    </div>

                    <div className="mt-4 grid gap-3">
                      <Input
                        value={selectedLlamaCppModelPath}
                        onChange={(event) => setSelectedLlamaCppModelPath(event.target.value)}
                        placeholder="/absolute/path/to/model.gguf"
                        aria-label={labels.llamaCppModelPath}
                      />
                    </div>
                    <div className="grid gap-1 text-xs text-muted-foreground">
                      <span>{labels.llamaCppModelPath}</span>
                    </div>
                  </div>

                  <div className="rounded-2xl border border-border bg-card p-6 shadow-sm transition-all hover:shadow-md">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-purple-500/10">
                          <Bot className="h-6 w-6 text-purple-500" />
                        </div>
                        <div>
                          <h3 className="font-semibold">{labels.launchLMStudio}</h3>
                          <p className="text-sm text-muted-foreground">
                            {messages.launchLMStudioDescription}
                          </p>
                        </div>
                      </div>

                      <button
                        onClick={() => void launchProjectLMStudio(selectedProject.project_id)}
                        disabled={
                          launchingLMStudioProjectId === selectedProject.project_id ||
                          !selectedProject.connection_ready ||
                          selectedProject.is_paused
                        }
                        className="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border bg-background px-4 text-sm font-medium transition-all hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {launchingLMStudioProjectId === selectedProject.project_id ? (
                          <LoaderCircle className="h-4 w-4 animate-spin" />
                        ) : (
                          <Play className="h-4 w-4" />
                        )}
                        {labels.launch}
                      </button>
                    </div>
                  </div>
                </div>

                <div className="rounded-lg bg-blue-500/5 p-4">
                  <div className="flex items-start gap-3">
                    <Info className="mt-0.5 h-5 w-5 text-blue-500" />
                    <div>
                      <p className="text-sm font-medium text-blue-500">{labels.tip}</p>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {messages.launchProjectTip}
                      </p>
                    </div>
                  </div>
                </div>
              </DialogContent>
            </Dialog>

            <ProjectActionsPanel
              labels={labels}
              selectedProject={selectedProject}
              busyProjectId={busyProjectId}
              setProjectPaused={setProjectPaused}
              startDuplicateProject={startDuplicateProject}
              startEditProject={startEditProject}
              deleteProject={deleteProject}
            />
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

          {endpointSecretEnabled ? (
            <div className="mt-3 rounded-xl border border-border bg-background p-4">
              <div className="mb-2 text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                {labels.bearerToken}
              </div>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <code
                  className={`min-w-0 flex-1 overflow-x-auto rounded-md border border-border bg-card px-3 py-2 font-mono text-xs text-electric-blue transition-all ${endpointSecretVisible ? '' : 'blur-sm select-none'}`}
                >
                  {endpointSecret || messages.noValue}
                </code>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setEndpointSecretVisible((current) => !current)}
                    className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border transition-colors hover:bg-accent"
                    aria-label={endpointSecretVisible ? labels.hideToken : labels.showToken}
                    title={endpointSecretVisible ? labels.hideToken : labels.showToken}
                  >
                    {endpointSecretVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => void window.navigator.clipboard.writeText(endpointSecret)}
                    disabled={!endpointSecret}
                    className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    aria-label={labels.copyToken}
                    title={labels.copyToken}
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    onClick={() => void regenerateEndpointToken(selectedProject.project_id)}
                    disabled={busyProjectId === selectedProject.project_id}
                    className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-border transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    aria-label={labels.generateToken}
                    title={labels.generateToken}
                  >
                    <RefreshCw className={`h-4 w-4 ${busyProjectId === selectedProject.project_id ? 'animate-spin' : ''}`} />
                  </button>
                </div>
              </div>
            </div>
          ) : null}

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
    </>
  );
}
