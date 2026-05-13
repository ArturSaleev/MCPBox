import { useState, useEffect, useRef } from 'react';
import { Terminal, X, Search } from 'lucide-react';
import { BarChart, Bar, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer, Cell } from 'recharts';
import * as Tooltip from '@radix-ui/react-tooltip';

interface LogEntry {
  id: string;
  timestamp: string;
  projectId: string;
  projectName: string;
  serverName: string;
  type: 'request' | 'response' | 'error' | 'info';
  message: string;
  data?: any;
}

interface Project {
  id: string;
  name: string;
}

interface GlobalLogsViewProps {
  logs: LogEntry[];
  projects: Project[];
  onClearLogs: () => void;
}

export function GlobalLogsView({ logs, projects, onClearLogs }: GlobalLogsViewProps) {
  const [selectedProject, setSelectedProject] = useState<string | null>(null);
  const [selectedServer, setSelectedServer] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  // Get unique server names
  const uniqueServers = Array.from(new Set(logs.map(log => log.serverName))).sort();

  const filteredLogs = logs.filter((log) => {
    const matchesProject = !selectedProject || log.projectId === selectedProject;
    const matchesServer = !selectedServer || log.serverName === selectedServer;
    const matchesSearch = !searchQuery ||
      log.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
      log.projectName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      log.serverName.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesProject && matchesServer && matchesSearch;
  });

  // Calculate project usage statistics
  const projectUsage = projects.map((project) => {
    const count = logs.filter((log) => log.projectId === project.id).length;
    return {
      name: project.name,
      count,
    };
  }).sort((a, b) => b.count - a.count);

  // Calculate MCP server usage statistics
  const serverUsageMap = new Map<string, number>();
  logs.forEach((log) => {
    const current = serverUsageMap.get(log.serverName) || 0;
    serverUsageMap.set(log.serverName, current + 1);
  });

  const serverUsage = Array.from(serverUsageMap.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 5);

  const getLogTypeColor = (type: LogEntry['type']) => {
    switch (type) {
      case 'request':
        return 'text-electric-blue';
      case 'response':
        return 'text-status-running';
      case 'error':
        return 'text-destructive';
      case 'info':
        return 'text-muted-foreground';
      default:
        return 'text-foreground';
    }
  };

  const getLogTypePrefix = (type: LogEntry['type']) => {
    switch (type) {
      case 'request':
        return '→';
      case 'response':
        return '←';
      case 'error':
        return '✗';
      case 'info':
        return 'ℹ';
      default:
        return '•';
    }
  };

  const COLORS = ['#0ea5e9', '#10b981', '#8b5cf6', '#f59e0b', '#ec4899'];

  return (
    <Tooltip.Provider>
      <div className="h-full flex gap-6">
        {/* Main Console Area */}
        <div className="flex-1 flex flex-col min-w-0">
          <div className="bg-card border border-border rounded-lg flex flex-col h-full">
            {/* Console Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/20">
              <div className="flex items-center gap-3">
                <Terminal className="w-5 h-5 text-electric-blue" />
                <h3>Global Logs Console</h3>
                <div className="px-2 py-1 bg-background rounded-md text-sm">
                  {filteredLogs.length} / {logs.length}
                </div>
              </div>

              <div className="flex items-center gap-2">
                {/* Search */}
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <input
                    type="text"
                    placeholder="Search logs..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9 pr-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-electric-blue w-48"
                  />
                </div>

                {/* Project Filter */}
                <select
                  value={selectedProject || ''}
                  onChange={(e) => setSelectedProject(e.target.value || null)}
                  className="px-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-electric-blue min-w-[140px]"
                >
                  <option value="">All Projects</option>
                  {projects.map((project) => (
                    <option key={project.id} value={project.id}>
                      {project.name}
                    </option>
                  ))}
                </select>

                {/* MCP Server Filter */}
                <select
                  value={selectedServer || ''}
                  onChange={(e) => setSelectedServer(e.target.value || null)}
                  className="px-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-electric-blue min-w-[160px]"
                >
                  <option value="">All MCP Servers</option>
                  {uniqueServers.map((server) => (
                    <option key={server} value={server}>
                      {server}
                    </option>
                  ))}
                </select>

                {(selectedProject || selectedServer) && (
                  <Tooltip.Root>
                    <Tooltip.Trigger asChild>
                      <button
                        onClick={() => {
                          setSelectedProject(null);
                          setSelectedServer(null);
                        }}
                        className="p-2 hover:bg-accent rounded-md transition-colors"
                        aria-label="Clear all filters"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </Tooltip.Trigger>
                    <Tooltip.Portal>
                      <Tooltip.Content
                        side="bottom"
                        className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
                      >
                        Clear all filters
                        <Tooltip.Arrow className="fill-popover" />
                      </Tooltip.Content>
                    </Tooltip.Portal>
                  </Tooltip.Root>
                )}

                <Tooltip.Root>
                  <Tooltip.Trigger asChild>
                    <button
                      onClick={onClearLogs}
                      className="px-3 py-2 bg-destructive/10 hover:bg-destructive/20 text-destructive rounded-md text-sm transition-colors"
                    >
                      Clear All
                    </button>
                  </Tooltip.Trigger>
                  <Tooltip.Portal>
                    <Tooltip.Content
                      side="bottom"
                      className="bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg border border-border"
                    >
                      Clear all logs
                      <Tooltip.Arrow className="fill-popover" />
                    </Tooltip.Content>
                  </Tooltip.Portal>
                </Tooltip.Root>
              </div>
            </div>

            {/* Console Content */}
            <div className="flex-1 overflow-y-auto p-4 bg-[#0a0a0a] font-mono text-sm">
              {filteredLogs.length === 0 ? (
                <div className="h-full flex items-center justify-center text-muted-foreground">
                  <div className="text-center">
                    <Terminal className="w-12 h-12 mx-auto mb-3 opacity-30" />
                    <p>{logs.length === 0 ? 'No logs yet' : 'No logs match your filters'}</p>
                    <p className="text-xs mt-1">
                      {logs.length === 0 ? 'Activity will appear here' : 'Try adjusting your filters'}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="space-y-1">
                  {filteredLogs.map((log) => (
                    <div key={log.id} className="hover:bg-muted/10 px-2 py-1 rounded transition-colors group">
                      <div className="flex items-start gap-3">
                        <span className="text-muted-foreground text-xs flex-shrink-0 w-20">
                          {log.timestamp}
                        </span>
                        <span className={`flex-shrink-0 w-4 ${getLogTypeColor(log.type)}`}>
                          {getLogTypePrefix(log.type)}
                        </span>
                        <span className="text-electric-blue text-xs flex-shrink-0 min-w-[120px]">
                          [{log.projectName}]
                        </span>
                        <span className="text-status-running text-xs flex-shrink-0 min-w-[140px]">
                          {log.serverName}
                        </span>
                        <span className={`flex-1 ${getLogTypeColor(log.type)}`}>
                          {log.message}
                        </span>
                      </div>
                      {log.data && (
                        <div className="ml-[180px] mt-1 text-xs text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity">
                          <pre className="whitespace-pre-wrap break-words">
                            {JSON.stringify(log.data, null, 2)}
                          </pre>
                        </div>
                      )}
                    </div>
                  ))}
                  <div ref={logsEndRef} />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Statistics Sidebar */}
        <div className="w-80 flex flex-col gap-6">
          {/* Project Usage Chart */}
          <div className="bg-card border border-border rounded-lg p-4">
            <h4 className="mb-4">Most Active Projects</h4>
            {projectUsage.length > 0 && projectUsage.some(p => p.count > 0) ? (
              <>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={projectUsage} layout="vertical">
                    <XAxis type="number" stroke="#717182" fontSize={12} />
                    <YAxis
                      type="category"
                      dataKey="name"
                      stroke="#717182"
                      fontSize={11}
                      width={100}
                    />
                    <RechartsTooltip
                      contentStyle={{
                        backgroundColor: 'oklch(0.145 0 0)',
                        border: '1px solid oklch(0.269 0 0)',
                        borderRadius: '0.375rem',
                        fontSize: '12px',
                      }}
                      labelStyle={{ color: '#0ea5e9' }}
                    />
                    <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                      {projectUsage.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
                <div className="mt-4 space-y-2">
                  {projectUsage.slice(0, 3).map((project, index) => (
                    <div key={index} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-3 h-3 rounded-sm flex-shrink-0"
                          style={{ backgroundColor: COLORS[index % COLORS.length] }}
                        />
                        <span className="text-muted-foreground truncate">{project.name}</span>
                      </div>
                      <span className="font-mono ml-2">{project.count}</span>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <div className="h-48 flex items-center justify-center text-muted-foreground text-sm">
                No activity yet
              </div>
            )}
          </div>

          {/* MCP Server Usage Chart */}
          <div className="bg-card border border-border rounded-lg p-4">
            <h4 className="mb-4">Most Used MCP Servers</h4>
            {serverUsage.length > 0 ? (
              <>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={serverUsage} layout="vertical">
                    <XAxis type="number" stroke="#717182" fontSize={12} />
                    <YAxis
                      type="category"
                      dataKey="name"
                      stroke="#717182"
                      fontSize={11}
                      width={100}
                    />
                    <RechartsTooltip
                      contentStyle={{
                        backgroundColor: 'oklch(0.145 0 0)',
                        border: '1px solid oklch(0.269 0 0)',
                        borderRadius: '0.375rem',
                        fontSize: '12px',
                      }}
                      labelStyle={{ color: '#10b981' }}
                    />
                    <Bar dataKey="count" fill="#10b981" radius={[0, 4, 4, 0]} />
                  </BarChart>
                </ResponsiveContainer>
                <div className="mt-4 space-y-2">
                  {serverUsage.slice(0, 3).map((server, index) => (
                    <div key={index} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-sm bg-status-running flex-shrink-0" />
                        <span className="text-muted-foreground truncate">{server.name}</span>
                      </div>
                      <span className="font-mono ml-2">{server.count}</span>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <div className="h-48 flex items-center justify-center text-muted-foreground text-sm">
                No server activity yet
              </div>
            )}
          </div>
        </div>
      </div>
    </Tooltip.Provider>
  );
}
