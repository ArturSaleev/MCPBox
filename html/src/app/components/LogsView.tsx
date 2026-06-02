import type { RefObject } from 'react';

import { LoaderCircle, RefreshCw } from 'lucide-react';

import { dictionaries } from '../i18n';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';

type AuditLog = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  action: string;
  actor: string;
  detail: string;
  created_at: string;
};

type MetricsWindow = '5m' | '1h' | '24h';

type PerformanceSummary = {
  request_count: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  traffic_in: number;
  traffic_out: number;
};

type PerformanceTrendBucket = {
  timestamp: string;
  request_count: number;
  error_count: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  traffic_in: number;
  traffic_out: number;
};

type PerformanceServerMetricRecord = {
  server_id: number;
  request_count: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  request_bytes: number;
  response_bytes: number;
  total_traffic: number;
};

type PerformanceFailureRecord = {
  id: number;
  project_id: number | null;
  server_id: number | null;
  operation: string;
  transport: string;
  latency_ms: number;
  request_bytes: number;
  response_bytes: number;
  error_detail: string;
  created_at: string;
};

type PerformanceMetricsResponse = {
  window: MetricsWindow;
  summary: PerformanceSummary;
  trends: PerformanceTrendBucket[];
  top_slow_servers: PerformanceServerMetricRecord[];
  top_error_servers: PerformanceServerMetricRecord[];
  top_traffic_servers: PerformanceServerMetricRecord[];
  recent_failures: PerformanceFailureRecord[];
};

type KnowledgeSearchAuditDetail = {
  collections?: string[];
  query?: string;
  results?: number;
};

type ProjectOption = {
  project_id: number;
  name: string;
};

type LogsViewProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  projects: ProjectOption[];
  selectedLogsProjectId: number | null;
  setSelectedLogsProjectId: (value: number | null) => void;
  metricsWindow: MetricsWindow;
  setMetricsWindow: (value: MetricsWindow) => void;
  metricsWindowOptions: Array<{ value: MetricsWindow; label: string }>;
  logsLoading: boolean;
  metricsLoading: boolean;
  onRefresh: () => void;
  logMetrics: PerformanceMetricsResponse | null;
  filteredLogsProjectName: string | null;
  requestTrendValues: number[];
  errorTrendValues: number[];
  avgLatencyTrendValues: number[];
  p95LatencyTrendValues: number[];
  visibleLogs: AuditLog[];
  logsViewportRef: RefObject<HTMLDivElement | null>;
  projectNameFromLog: (projectID: number) => string;
  serverNameFromLog: (serverID: number) => string;
  serverNamesById: Record<number, string>;
  projectNamesById: Record<number, string>;
};

function formatAuditAction(action: string) {
  if (action === 'tool_call_project_knowledge_search') {
    return 'tool_call -> search_project_knowledge';
  }
  return action;
}

function formatAuditDetail(entry: AuditLog) {
  if (entry.action !== 'tool_call_project_knowledge_search') {
    return entry.detail;
  }

  try {
    const parsed = JSON.parse(entry.detail) as KnowledgeSearchAuditDetail;
    const collections = parsed.collections?.length ? `[${parsed.collections.join(', ')}]` : '[all connected collections]';
    const query = parsed.query ? `query="${parsed.query}"` : '';
    const results =
      typeof parsed.results === 'number' ? `results=${parsed.results}` : '';
    return [collections, query, results]
      .filter((part) => part && part.trim() !== '')
      .join(' ');
  } catch {
    return entry.detail;
  }
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB'];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size >= 100 || unitIndex === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unitIndex]}`;
}

function formatLatency(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 ms';
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(2)} s`;
  }
  return `${Math.round(value)} ms`;
}

function formatPercent(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return '0%';
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)}%`;
}

function buildChartPath(values: number[], width: number, height: number) {
  if (values.length === 0) {
    return '';
  }
  const max = Math.max(...values, 1);
  const step = values.length > 1 ? width / (values.length - 1) : width;
  return values
    .map((value, index) => {
      const x = index * step;
      const y = height - (value / max) * height;
      return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`;
    })
    .join(' ');
}

function TrendChart({
  title,
  subtitle,
  primaryValues,
  secondaryValues,
  primaryColor,
  secondaryColor,
  labels,
}: {
  title: string;
  subtitle: string;
  primaryValues: number[];
  secondaryValues: number[];
  primaryColor: string;
  secondaryColor: string;
  labels: { primary: string; secondary: string };
}) {
  const width = 320;
  const height = 120;
  const hasData =
    primaryValues.some((value) => value > 0) || secondaryValues.some((value) => value > 0);
  const primaryPath = buildChartPath(primaryValues, width, height);
  const secondaryPath = buildChartPath(secondaryValues, width, height);

  return (
    <div className="rounded-2xl border border-border bg-card p-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">{title}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: primaryColor }} />
            {labels.primary}
          </span>
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: secondaryColor }} />
            {labels.secondary}
          </span>
        </div>
      </div>
      <div className="mt-4">
        {hasData ? (
          <svg viewBox={`0 0 ${width} ${height}`} className="h-32 w-full">
            <path d={secondaryPath} fill="none" stroke={secondaryColor} strokeWidth="3" strokeLinecap="round" />
            <path d={primaryPath} fill="none" stroke={primaryColor} strokeWidth="3" strokeLinecap="round" />
          </svg>
        ) : (
          <div className="flex h-32 items-center justify-center rounded-xl bg-background text-sm text-muted-foreground">
            {subtitle}
          </div>
        )}
      </div>
    </div>
  );
}

export function LogsView({
  labels,
  messages,
  projects,
  selectedLogsProjectId,
  setSelectedLogsProjectId,
  metricsWindow,
  setMetricsWindow,
  metricsWindowOptions,
  logsLoading,
  metricsLoading,
  onRefresh,
  logMetrics,
  filteredLogsProjectName,
  requestTrendValues,
  errorTrendValues,
  avgLatencyTrendValues,
  p95LatencyTrendValues,
  visibleLogs,
  logsViewportRef,
  projectNameFromLog,
  serverNameFromLog,
  serverNamesById,
  projectNamesById,
}: LogsViewProps) {
  const logsSubtitle = `${filteredLogsProjectName ?? labels.allProjects} · ${metricsWindowOptions.find((option) => option.value === metricsWindow)?.label ?? ''}`;

  return (
    <section className="space-y-6">
      <div className="rounded-2xl border border-border bg-card p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <h2 className="text-2xl font-semibold">{labels.performance}</h2>
            <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
              {messages.performanceDescription}
            </p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="min-w-[220px]">
              <Select
                value={selectedLogsProjectId === null ? 'all' : String(selectedLogsProjectId)}
                onValueChange={(value) =>
                  setSelectedLogsProjectId(value === 'all' ? null : Number(value))
                }
              >
                <SelectTrigger className="h-10 rounded-md border-border bg-background text-sm">
                  <SelectValue placeholder={labels.filterByProject} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{labels.allProjects}</SelectItem>
                  {projects.map((project) => (
                    <SelectItem key={`logs-project-${project.project_id}`} value={String(project.project_id)}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="min-w-[200px]">
              <Select value={metricsWindow} onValueChange={(value) => setMetricsWindow(value as MetricsWindow)}>
                <SelectTrigger className="h-10 rounded-md border-border bg-background text-sm">
                  <SelectValue placeholder={labels.timeWindow} />
                </SelectTrigger>
                <SelectContent>
                  {metricsWindowOptions.map((option) => (
                    <SelectItem key={`metrics-window-${option.value}`} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <button
              onClick={onRefresh}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
            >
              <RefreshCw className={`h-4 w-4 ${(logsLoading || metricsLoading) ? 'animate-spin' : ''}`} />
              {labels.refresh}
            </button>
          </div>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.requests}</div>
          <div className="mt-2 text-2xl font-semibold">{logMetrics?.summary.request_count ?? 0}</div>
        </div>
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.errors}</div>
          <div className="mt-2 text-2xl font-semibold">{logMetrics?.summary.error_count ?? 0}</div>
          <div className="mt-1 text-sm text-muted-foreground">
            {formatPercent(logMetrics?.summary.error_rate ?? 0)} {labels.errorRate.toLowerCase()}
          </div>
        </div>
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.avgLatency}</div>
          <div className="mt-2 text-2xl font-semibold">{formatLatency(logMetrics?.summary.avg_latency_ms ?? 0)}</div>
        </div>
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.p95Latency}</div>
          <div className="mt-2 text-2xl font-semibold">{formatLatency(logMetrics?.summary.p95_latency_ms ?? 0)}</div>
        </div>
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.inputTraffic}</div>
          <div className="mt-2 text-2xl font-semibold">{formatBytes(logMetrics?.summary.traffic_in ?? 0)}</div>
        </div>
        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="text-sm text-muted-foreground">{labels.outputTraffic}</div>
          <div className="mt-2 text-2xl font-semibold">{formatBytes(logMetrics?.summary.traffic_out ?? 0)}</div>
        </div>
      </div>

      {metricsLoading && !logMetrics ? (
        <div className="flex items-center gap-2 rounded-xl border border-border bg-card px-4 py-5 text-sm text-muted-foreground">
          <LoaderCircle className="h-4 w-4 animate-spin" />
          {messages.loadingMetrics}
        </div>
      ) : null}

      {logMetrics && logMetrics.trends.length > 0 ? (
        <div className="grid gap-4 xl:grid-cols-2">
          <TrendChart
            title={labels.requestVolume}
            subtitle={logsSubtitle}
            primaryValues={requestTrendValues}
            secondaryValues={errorTrendValues}
            primaryColor="#10b981"
            secondaryColor="#f97316"
            labels={{ primary: labels.requests, secondary: labels.errors }}
          />
          <TrendChart
            title={labels.latencyTrend}
            subtitle={logsSubtitle}
            primaryValues={avgLatencyTrendValues}
            secondaryValues={p95LatencyTrendValues}
            primaryColor="#0ea5e9"
            secondaryColor="#a855f7"
            labels={{ primary: labels.avgLatency, secondary: labels.p95Latency }}
          />
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-border bg-card px-4 py-8 text-center text-muted-foreground">
          {messages.noMetrics}
        </div>
      )}

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="rounded-2xl border border-border bg-card p-6">
          <div className="mb-4 flex items-center justify-between gap-3 rounded-xl border border-border bg-background px-4 py-3 text-sm">
            <div>
              <div className="font-medium">{labels.auditLogs}</div>
              <div className="text-muted-foreground">
                {filteredLogsProjectName ?? labels.allProjects}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <div className="rounded-full border border-border bg-card px-3 py-1 text-xs text-muted-foreground">
                {visibleLogs.length} {labels.requests}
              </div>
            </div>
          </div>

          {logsLoading ? (
            <div className="flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-5 text-sm text-muted-foreground">
              <LoaderCircle className="h-4 w-4 animate-spin" />
              {messages.loadingLogs}
            </div>
          ) : visibleLogs.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-center text-muted-foreground">
              {messages.noLogs}
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl border border-border bg-[#0b0f14]">
              <div ref={logsViewportRef} className="max-h-[70vh] overflow-y-auto">
                {visibleLogs.map((entry) => (
                  <div
                    key={entry.id}
                    className="border-b border-white/5 px-4 py-3 font-mono text-xs text-slate-200 last:border-b-0"
                  >
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <span className="text-slate-500">
                        {new Date(entry.created_at).toLocaleTimeString()}
                      </span>
                      <span className="text-electric-blue">{formatAuditAction(entry.action)}</span>
                      <span className="text-slate-400">
                        {entry.actor || labels.unknownActor}
                      </span>
                      {entry.project_id ? (
                        <span className="text-emerald-400">
                          {projectNameFromLog(entry.project_id)}
                        </span>
                      ) : null}
                      {entry.server_id ? (
                        <span className="text-amber-300">
                          {serverNameFromLog(entry.server_id)}
                        </span>
                      ) : null}
                    </div>
                    {entry.detail ? (
                      <div className="mt-1 break-words text-slate-300/85">
                        {formatAuditDetail(entry)}
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <aside className="space-y-4">
          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-lg font-semibold">{labels.topSlowServers}</h3>
            <div className="mt-3 space-y-2">
              {(logMetrics?.top_slow_servers ?? []).map((entry) => (
                <div
                  key={`slow-server-${entry.server_id}`}
                  className="rounded-lg bg-background px-3 py-3 text-sm"
                >
                  <div className="font-medium">
                    {serverNamesById[entry.server_id] ?? messages.serverTag(entry.server_id)}
                  </div>
                  <div className="mt-1 flex items-center justify-between gap-3 text-muted-foreground">
                    <span>{entry.request_count} {labels.requests}</span>
                    <span>{formatLatency(entry.p95_latency_ms)}</span>
                  </div>
                </div>
              ))}
              {(logMetrics?.top_slow_servers.length ?? 0) === 0 ? (
                <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                  {messages.noMetrics}
                </div>
              ) : null}
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-lg font-semibold">{labels.topErrorServers}</h3>
            <div className="mt-3 space-y-2">
              {(logMetrics?.top_error_servers ?? []).map((entry) => (
                <div
                  key={`error-server-${entry.server_id}`}
                  className="rounded-lg bg-background px-3 py-3 text-sm"
                >
                  <div className="font-medium">
                    {serverNamesById[entry.server_id] ?? messages.serverTag(entry.server_id)}
                  </div>
                  <div className="mt-1 flex items-center justify-between gap-3 text-muted-foreground">
                    <span>{entry.error_count} {labels.errors.toLowerCase()}</span>
                    <span>{formatPercent(entry.error_rate)}</span>
                  </div>
                </div>
              ))}
              {(logMetrics?.top_error_servers.length ?? 0) === 0 ? (
                <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                  {messages.noMetrics}
                </div>
              ) : null}
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-lg font-semibold">{labels.topTrafficServers}</h3>
            <div className="mt-3 space-y-2">
              {(logMetrics?.top_traffic_servers ?? []).map((entry) => (
                <div
                  key={`traffic-server-${entry.server_id}`}
                  className="rounded-lg bg-background px-3 py-3 text-sm"
                >
                  <div className="font-medium">
                    {serverNamesById[entry.server_id] ?? messages.serverTag(entry.server_id)}
                  </div>
                  <div className="mt-1 flex items-center justify-between gap-3 text-muted-foreground">
                    <span>{formatBytes(entry.total_traffic)}</span>
                    <span>{entry.request_count} {labels.requests}</span>
                  </div>
                </div>
              ))}
              {(logMetrics?.top_traffic_servers.length ?? 0) === 0 ? (
                <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                  {messages.noMetrics}
                </div>
              ) : null}
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-lg font-semibold">{labels.recentFailures}</h3>
            <div className="mt-3 space-y-2">
              {(logMetrics?.recent_failures ?? []).map((entry) => (
                <div
                  key={`failure-${entry.id}`}
                  className="rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-3 text-sm"
                >
                  <div className="font-medium">
                    {entry.server_id
                      ? serverNamesById[entry.server_id] ?? messages.serverTag(entry.server_id)
                      : entry.project_id
                        ? projectNamesById[entry.project_id] ?? messages.projectTag(entry.project_id)
                        : labels.performance}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {entry.operation} · {formatLatency(entry.latency_ms)} · {new Date(entry.created_at).toLocaleTimeString()}
                  </div>
                  <div className="mt-2 break-words text-xs text-foreground/80">
                    {entry.error_detail}
                  </div>
                </div>
              ))}
              {(logMetrics?.recent_failures.length ?? 0) === 0 ? (
                <div className="rounded-lg bg-background px-3 py-2 text-sm text-muted-foreground">
                  {messages.noMetrics}
                </div>
              ) : null}
            </div>
          </div>
        </aside>
      </div>
    </section>
  );
}
