import { Plus, Trash2, Eye, EyeOff } from 'lucide-react';
import { useState } from 'react';
import * as Tooltip from '@radix-ui/react-tooltip';

interface McpServer {
  id: string;
  name: string;
  command: string;
  env: { key: string; value: string; hidden?: boolean }[];
}

interface ConfigSectionProps {
  servers: McpServer[];
  onAddServer: () => void;
  onDeleteServer: (id: string) => void;
}

export function ConfigSection({ servers, onAddServer, onDeleteServer }: ConfigSectionProps) {
  const [visibleEnvs, setVisibleEnvs] = useState<{ [key: string]: boolean }>({});

  const toggleEnvVisibility = (envId: string) => {
    setVisibleEnvs((prev) => ({ ...prev, [envId]: !prev[envId] }));
  };

  return (
    <Tooltip.Provider>
      <div className="bg-card border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h3>MCP Servers</h3>
          <Tooltip.Root>
            <Tooltip.Trigger asChild>
              <button
                onClick={onAddServer}
                className="px-3 py-2 bg-electric-blue hover:bg-electric-blue/90 text-white rounded-md flex items-center gap-2 transition-colors"
              >
                <Plus className="w-4 h-4" />
                <span>Add Server</span>
              </button>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                side="top"
                className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
              >
                Add new MCP server
                <Tooltip.Arrow className="fill-popover" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>
        </div>

        <div className="space-y-3">
          {servers.map((server) => (
            <div
              key={server.id}
              className="bg-muted/30 border border-border rounded-md p-4 hover:border-electric-blue/50 transition-colors"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1">
                  <h4 className="mb-1">{server.name}</h4>
                  <code className="text-sm text-muted-foreground font-mono">{server.command}</code>
                </div>
                <Tooltip.Root>
                  <Tooltip.Trigger asChild>
                    <button
                      onClick={() => onDeleteServer(server.id)}
                      className="p-2 hover:bg-destructive/10 hover:text-destructive rounded-md transition-colors"
                      aria-label="Delete server"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </Tooltip.Trigger>
                  <Tooltip.Portal>
                    <Tooltip.Content
                      side="left"
                      className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
                    >
                      Delete server
                      <Tooltip.Arrow className="fill-popover" />
                    </Tooltip.Content>
                  </Tooltip.Portal>
                </Tooltip.Root>
              </div>

              {server.env.length > 0 && (
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">Environment Variables:</p>
                  {server.env.map((envVar, idx) => {
                    const envId = `${server.id}-${envVar.key}`;
                    const isVisible = visibleEnvs[envId];
                    const shouldHide = envVar.hidden && !isVisible;

                    return (
                      <div
                        key={idx}
                        className="flex items-center gap-2 text-sm bg-background/50 rounded px-3 py-2 border border-border"
                      >
                        <code className="text-electric-blue font-mono">{envVar.key}=</code>
                        <code className="flex-1 font-mono">
                          {shouldHide ? '••••••••' : envVar.value}
                        </code>
                        {envVar.hidden && (
                          <button
                            onClick={() => toggleEnvVisibility(envId)}
                            className="p-1 hover:bg-accent rounded transition-colors"
                            aria-label={isVisible ? 'Hide value' : 'Show value'}
                          >
                            {isVisible ? (
                              <EyeOff className="w-4 h-4" />
                            ) : (
                              <Eye className="w-4 h-4" />
                            )}
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          ))}

          {servers.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              <p>No MCP servers configured</p>
              <p className="text-sm mt-1">Click "Add Server" to get started</p>
            </div>
          )}
        </div>
      </div>
    </Tooltip.Provider>
  );
}
