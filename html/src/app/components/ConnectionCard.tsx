import { Copy, CheckCircle2 } from 'lucide-react';
import { useState } from 'react';
import * as Tooltip from '@radix-ui/react-tooltip';

interface ConnectionCardProps {
  url: string;
}

export function ConnectionCard({ url }: ConnectionCardProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Tooltip.Provider>
      <div className="bg-card border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-electric-blue">Connection URL</h3>
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-status-running animate-pulse" />
            <span className="text-sm text-status-running">Active</span>
          </div>
        </div>

        <div className="flex items-center gap-3 bg-muted/30 rounded-md p-3 border border-border">
          <code className="flex-1 text-sm font-mono text-electric-blue">{url}</code>

          <Tooltip.Root>
            <Tooltip.Trigger asChild>
              <button
                onClick={handleCopy}
                className="p-2 hover:bg-accent rounded-md transition-colors flex-shrink-0"
                aria-label="Copy URL"
              >
                {copied ? (
                  <CheckCircle2 className="w-4 h-4 text-status-running" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </button>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                side="top"
                className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
              >
                {copied ? 'Copied!' : 'Copy URL'}
                <Tooltip.Arrow className="fill-popover" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>
        </div>

        <p className="text-sm text-muted-foreground mt-3">
          Use this URL to connect clients to your MCP server
        </p>
      </div>
    </Tooltip.Provider>
  );
}
