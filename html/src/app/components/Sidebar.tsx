import { Folder, Terminal, Settings, ChevronLeft } from 'lucide-react';
import * as Tooltip from '@radix-ui/react-tooltip';

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
  activeTab: 'projects' | 'logs' | 'settings';
  onTabChange: (tab: 'projects' | 'logs' | 'settings') => void;
}

export function Sidebar({ collapsed, onToggle, activeTab, onTabChange }: SidebarProps) {
  const navItems = [
    { id: 'projects' as const, icon: Folder, label: 'Projects' },
    { id: 'logs' as const, icon: Terminal, label: 'Global Logs' },
    { id: 'settings' as const, icon: Settings, label: 'Settings' },
  ];

  return (
    <Tooltip.Provider>
      <div className={`h-full bg-sidebar border-r border-sidebar-border transition-all duration-300 flex flex-col ${collapsed ? 'w-16' : 'w-64'}`}>
        <div className="p-4 border-b border-sidebar-border flex items-center justify-between">
          {!collapsed && <h1 className="text-electric-blue">MCPBox</h1>}
          <button
            onClick={onToggle}
            className="p-2 hover:bg-sidebar-accent rounded-md transition-colors"
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <ChevronLeft className={`w-5 h-5 transition-transform ${collapsed ? 'rotate-180' : ''}`} />
          </button>
        </div>

        <nav className="flex-1 p-2">
          {navItems.map((item) => (
            <Tooltip.Root key={item.id}>
              <Tooltip.Trigger asChild>
                <button
                  onClick={() => onTabChange(item.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1 ${
                    activeTab === item.id
                      ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                      : 'hover:bg-sidebar-accent/50 text-sidebar-foreground'
                  }`}
                >
                  <item.icon className="w-5 h-5 flex-shrink-0" />
                  {!collapsed && <span>{item.label}</span>}
                </button>
              </Tooltip.Trigger>
              {collapsed && (
                <Tooltip.Portal>
                  <Tooltip.Content
                    side="right"
                    className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
                  >
                    {item.label}
                    <Tooltip.Arrow className="fill-popover" />
                  </Tooltip.Content>
                </Tooltip.Portal>
              )}
            </Tooltip.Root>
          ))}
        </nav>
      </div>
    </Tooltip.Provider>
  );
}
