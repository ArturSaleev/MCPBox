import { Terminal, Trash2 } from 'lucide-react';
import { useEffect, useRef } from 'react';
import * as Tooltip from '@radix-ui/react-tooltip';

interface LogEntry {
  id: string;
  timestamp: string;
  type: 'request' | 'response' | 'error';
  data: any;
}

interface TerminalLogsProps {
  logs: LogEntry[];
  onClearLogs: () => void;
}

export function TerminalLogs({ logs, onClearLogs }: TerminalLogsProps) {
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  const getLogColor = (type: LogEntry['type']) => {
    switch (type) {
      case 'request':
        return 'text-electric-blue';
      case 'response':
        return 'text-status-running';
      case 'error':
        return 'text-destructive';
      default:
        return 'text-foreground';
    }
  };

  return (
    <Tooltip.Provider>
      <div className="bg-card border border-border rounded-lg flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Terminal className="w-5 h-5 text-electric-blue" />
            <h3>JSON-RPC Logs</h3>
            <div className="ml-2 px-2 py-1 bg-muted rounded-md text-sm">
              {logs.length} {logs.length === 1 ? 'entry' : 'entries'}
            </div>
          </div>

          <Tooltip.Root>
            <Tooltip.Trigger asChild>
              <button
                onClick={onClearLogs}
                className="p-2 hover:bg-destructive/10 hover:text-destructive rounded-md transition-colors"
                aria-label="Clear logs"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                side="left"
                className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
              >
                Clear all logs
                <Tooltip.Arrow className="fill-popover" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>
        </div>

        <div className="flex-1 overflow-y-auto p-4 bg-background/50 font-mono text-sm">
          {logs.length === 0 ? (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              <div className="text-center">
                <Terminal className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>No logs yet</p>
                <p className="text-xs mt-1">JSON-RPC messages will appear here</p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {logs.map((log) => (
                <div key={log.id} className="border-l-2 border-muted pl-3 py-1">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs text-muted-foreground">{log.timestamp}</span>
                    <span className={`text-xs uppercase font-semibold ${getLogColor(log.type)}`}>
                      {log.type}
                    </span>
                  </div>
                  <pre className="text-xs text-foreground/90 whitespace-pre-wrap break-words">
                    {JSON.stringify(log.data, null, 2)}
                  </pre>
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          )}
        </div>
      </div>
    </Tooltip.Provider>
  );
}
