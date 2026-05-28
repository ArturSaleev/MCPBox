import { LoaderCircle, Pause, Play } from 'lucide-react';

import { dictionaries } from '../i18n';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';

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

type ProjectServerDialogsProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  inspectOpen: boolean;
  setInspectOpen: (open: boolean) => void;
  inspectionServerName: string;
  inspectingServerId: number | null;
  inspectionError: string | null;
  inspection: ServerInspection | null;
  formatSchema: (schema: unknown) => string;
  serverToolsOpen: boolean;
  setServerToolsOpen: (open: boolean) => void;
  resetServerTools: () => void;
  serverToolsLoadingId: number | null;
  serverToolsServerName: string;
  serverToolsError: string | null;
  serverTools: ServerToolStatus[];
  serverToolsSavingName: string | null;
  setServerToolEnabled: (toolName: string, enabled: boolean) => void | Promise<void>;
  authOpen: boolean;
  setAuthOpen: (open: boolean) => void;
  resetAuthServer: () => void;
  authServer: AuthServer | null;
  busyServerId: number | null;
  connectOAuth: (serverId: number) => void | Promise<void>;
  disconnectOAuth: (serverId: number) => void | Promise<void>;
};

export function ProjectServerDialogs({
  labels,
  messages,
  inspectOpen,
  setInspectOpen,
  inspectionServerName,
  inspectingServerId,
  inspectionError,
  inspection,
  formatSchema,
  serverToolsOpen,
  setServerToolsOpen,
  resetServerTools,
  serverToolsLoadingId,
  serverToolsServerName,
  serverToolsError,
  serverTools,
  serverToolsSavingName,
  setServerToolEnabled,
  authOpen,
  setAuthOpen,
  resetAuthServer,
  authServer,
  busyServerId,
  connectOAuth,
  disconnectOAuth,
}: ProjectServerDialogsProps) {
  return (
    <>
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
        open={serverToolsOpen}
        onOpenChange={(open) => {
          setServerToolsOpen(open);
          if (!open) {
            resetServerTools();
          }
        }}
      >
        <DialogContent className="sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>
              {labels.manageTools}
              {serverToolsServerName ? ` · ${serverToolsServerName}` : ''}
            </DialogTitle>
            <DialogDescription>{messages.manageToolsDescription}</DialogDescription>
          </DialogHeader>

          {serverToolsLoadingId ? (
            <div className="flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-5 text-sm text-muted-foreground">
              <LoaderCircle className="h-4 w-4 animate-spin" />
              {labels.tools}
            </div>
          ) : serverToolsError ? (
            <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
              {serverToolsError}
            </div>
          ) : serverTools.length === 0 ? (
            <div className="rounded-xl border border-border bg-background p-4 text-sm text-muted-foreground">
              {messages.noServerTools}
            </div>
          ) : (
            <div className="space-y-3">
              {serverTools.map((tool) => (
                <div key={tool.name} className="rounded-xl border border-border bg-background p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="font-medium">{tool.title || tool.name}</div>
                      <code className="mt-1 block overflow-x-auto text-xs text-electric-blue">
                        {tool.name}
                      </code>
                      {tool.description ? (
                        <div className="mt-2 text-sm text-muted-foreground">{tool.description}</div>
                      ) : null}
                    </div>
                    <label className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2 text-sm">
                      <input
                        type="checkbox"
                        checked={tool.enabled}
                        disabled={serverToolsSavingName === tool.name}
                        onChange={(event) => void setServerToolEnabled(tool.name, event.target.checked)}
                        className="h-4 w-4 rounded border-border"
                      />
                      {serverToolsSavingName === tool.name ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : null}
                      {tool.enabled ? labels.enabled : labels.disabled}
                    </label>
                  </div>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={authOpen}
        onOpenChange={(open) => {
          setAuthOpen(open);
          if (!open) {
            resetAuthServer();
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
    </>
  );
}
