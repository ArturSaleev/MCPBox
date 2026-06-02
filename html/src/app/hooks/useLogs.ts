import { useState } from 'react';
import { apiRequest } from '../utils/api';
import type { AuditLog, PerformanceMetricsResponse, MetricsWindow, LogsFilterMode, ProjectStatus } from '../types';

export function useLogs(projects: ProjectStatus[], messages: { requestFailed: string }) {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [logMetrics, setLogMetrics] = useState<PerformanceMetricsResponse | null>(null);
  const [selectedLogsProjectId, setSelectedLogsProjectId] = useState<number | null>(null);
  const [metricsWindow, setMetricsWindow] = useState<MetricsWindow>('1h');
  const [logsFilterMode, setLogsFilterMode] = useState<LogsFilterMode>('all');
  const [logsLoading, setLogsLoading] = useState(false);
  const [metricsLoading, setMetricsLoading] = useState(false);

  const filteredLogsProject =
    selectedLogsProjectId !== null
      ? projects.find((project) => project.project_id === selectedLogsProjectId) ?? null
      : null;

  async function loadLogs(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setLogsLoading(true);
    }

    try {
      const url = filteredLogsProject
        ? `/api/logs?project_id=${filteredLogsProject.project_id}`
        : '/api/logs';
      const response = await apiRequest<{ items: AuditLog[] }>(url, () => messages.requestFailed);
      setLogs(response.items);
    } catch (loadError) {
      console.error('Failed to load logs:', loadError);
    } finally {
      setLogsLoading(false);
    }
  }

  async function loadLogMetrics(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setMetricsLoading(true);
    }

    try {
      const url = filteredLogsProject
        ? `/api/logs/metrics?project_id=${filteredLogsProject.project_id}&window=${metricsWindow}`
        : `/api/logs/metrics?window=${metricsWindow}`;
      const response = await apiRequest<PerformanceMetricsResponse>(url, () => messages.requestFailed);
      setLogMetrics(response);
    } catch (loadError) {
      console.error('Failed to load log metrics:', loadError);
    } finally {
      setMetricsLoading(false);
    }
  }

  return {
    // State
    logs,
    setLogs,
    logMetrics,
    setLogMetrics,
    selectedLogsProjectId,
    setSelectedLogsProjectId,
    metricsWindow,
    setMetricsWindow,
    logsFilterMode,
    setLogsFilterMode,
    logsLoading,
    metricsLoading,
    filteredLogsProject,
    // Functions
    loadLogs,
    loadLogMetrics,
  };
}
