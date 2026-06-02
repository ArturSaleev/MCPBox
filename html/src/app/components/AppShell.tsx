import type { ReactNode } from 'react';

import type { LucideIcon } from 'lucide-react';

import { dictionaries } from '../i18n';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

type ViewId = 'projects' | 'knowledge' | 'market' | 'logs' | 'pro';

type NavigationItem = {
  id: ViewId;
  label: string;
  icon: LucideIcon;
};

type AppShellProps = {
  logoSrc: string;
  labels: typeof dictionaries.en.labels;
  navigationItems: NavigationItem[];
  view: ViewId;
  setView: (view: ViewId) => void;
  immersive?: boolean;
  sidebar?: ReactNode;
  footer?: ReactNode;
  error: string | null;
  actionError: string | null;
  children: ReactNode;
};

export function AppShell({
  logoSrc,
  labels,
  navigationItems,
  view,
  setView,
  immersive = false,
  sidebar,
  footer,
  error,
  actionError,
  children,
}: AppShellProps) {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="flex min-h-screen w-full">
        {!immersive ? (
          <aside className="sticky top-0 flex h-screen w-20 shrink-0 flex-col items-center border-r border-border bg-sidebar/55 px-3 py-6">
            <div className="flex h-full flex-col items-center gap-3">
              <div className="mb-2 flex h-12 w-12 items-center justify-center">
                <img src={logoSrc} alt={labels.appTitle} className="max-h-full w-auto object-contain" />
              </div>
              {navigationItems.map((item) => {
                const Icon = item.icon;
                const isActive = view === item.id;

                return (
                  <Tooltip key={item.id}>
                    <TooltipTrigger asChild>
                      <button
                        onClick={() => setView(item.id)}
                        aria-label={item.label}
                        className={`inline-flex h-12 w-12 items-center justify-center rounded-2xl border transition-colors ${
                          isActive
                            ? 'border-electric-blue/40 bg-electric-blue text-white shadow-[0_12px_30px_rgba(38,132,255,0.22)]'
                            : 'border-transparent bg-card text-muted-foreground hover:border-border hover:bg-accent hover:text-foreground'
                        }`}
                      >
                        <Icon className="h-5 w-5" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" sideOffset={10}>
                      {item.label}
                    </TooltipContent>
                  </Tooltip>
                );
              })}
              {footer ? <div className="mt-auto pt-3">{footer}</div> : null}
            </div>
          </aside>
        ) : null}

        {!immersive ? sidebar : null}

        <main className={`flex-1 ${immersive ? '' : 'p-6 md:p-8'}`}>
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

          {children}
        </main>
      </div>
    </div>
  );
}
