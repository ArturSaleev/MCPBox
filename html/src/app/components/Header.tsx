import { Play, Square, Settings } from 'lucide-react';
import * as Tooltip from '@radix-ui/react-tooltip';

interface HeaderProps {
  projectName: string;
  isRunning: boolean;
  onToggleStatus: () => void;
  onOpenSettings: () => void;
}

export function Header({ projectName, isRunning, onToggleStatus, onOpenSettings }: HeaderProps) {
  return (
    <Tooltip.Provider>
      <div className="h-16 border-b border-border bg-card px-6 flex items-center justify-between">
        <h2>{projectName}</h2>

        <div className="flex items-center gap-3">
          <Tooltip.Root>
            <Tooltip.Trigger asChild>
              <button
                onClick={onToggleStatus}
                className={`px-4 py-2 rounded-md flex items-center gap-2 transition-colors ${
                  isRunning
                    ? 'bg-destructive hover:bg-destructive/90 text-destructive-foreground'
                    : 'bg-status-running hover:bg-status-running/90 text-white'
                }`}
              >
                {isRunning ? (
                  <>
                    <Square className="w-4 h-4" />
                    <span>Stop</span>
                  </>
                ) : (
                  <>
                    <Play className="w-4 h-4" />
                    <span>Start</span>
                  </>
                )}
              </button>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                side="bottom"
                className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
              >
                {isRunning ? 'Stop project' : 'Start project'}
                <Tooltip.Arrow className="fill-popover" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>

          <Tooltip.Root>
            <Tooltip.Trigger asChild>
              <button
                onClick={onOpenSettings}
                className="p-2 hover:bg-accent rounded-md transition-colors"
                aria-label="Project settings"
              >
                <Settings className="w-5 h-5" />
              </button>
            </Tooltip.Trigger>
            <Tooltip.Portal>
              <Tooltip.Content
                side="bottom"
                className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
              >
                Project Settings
                <Tooltip.Arrow className="fill-popover" />
              </Tooltip.Content>
            </Tooltip.Portal>
          </Tooltip.Root>
        </div>
      </div>
    </Tooltip.Provider>
  );
}
