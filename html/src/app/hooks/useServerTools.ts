import { FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { apiRequest } from '../utils/api';
import type { ServerStatus, ServerInspection, ServerToolStatus } from '../types';

export function useServerTools(messages: { requestFailed: string; serverStarted: string; serverStopped: string; serverEnabled: string; serverDisabled: string }, loadProjects: () => void) {
  const [addingServer, setAddingServer] = useState(false);
  const [addServerOpen, setAddServerOpen] = useState(false);
  const [editingServerId, setEditingServerId] = useState<number | null>(null);
  const [busyServerId, setBusyServerId] = useState<number | null>(null);
  const [inspectOpen, setInspectOpen] = useState(false);
  const [inspectingServerId, setInspectingServerId] = useState<number | null>(null);
  const [inspection, setInspection] = useState<ServerInspection | null>(null);
  const [inspectionServerName, setInspectionServerName] = useState('');
  const [inspectionError, setInspectionError] = useState<string | null>(null);
  const [serverToolsOpen, setServerToolsOpen] = useState(false);
  const [serverToolsLoadingId, setServerToolsLoadingId] = useState<number | null>(null);
  const [serverToolsSavingName, setServerToolsSavingName] = useState<string | null>(null);
  const [serverToolsServerId, setServerToolsServerId] = useState<number | null>(null);
  const [serverToolsServerName, setServerToolsServerName] = useState('');
  const [serverTools, setServerTools] = useState<ServerToolStatus[]>([]);
  const [serverToolsError, setServerToolsError] = useState<string | null>(null);

  async function addServer(event: FormEvent<HTMLFormElement>, serverForm: any, selectedProject: any, setServerForm: any) {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }

    setAddingServer(true);
    try {
      await apiRequest<void>(`/api/projects/${selectedProject.project_id}/servers`, () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify(serverForm),
      });
      loadProjects();
      setAddServerOpen(false);
      setServerForm({
        name: '',
        transport: 'stdio',
        command: '',
        args: [''],
        env_vars: [{ key: '', value: '' }],
        env_passthrough: [''],
        working_dir: '',
        url: '',
        bearer_token_env_var: '',
        headers: [{ key: '', value: '' }],
        header_env_vars: [{ key: '', value: '' }],
        auth_type: 'none',
        oauth_provider: '',
        oauth_authorize_url: '',
        oauth_token_url: '',
        oauth_refresh_url: '',
        oauth_client_id: '',
        oauth_client_secret: '',
        oauth_scopes: [''],
        auto_start: false,
      });
    } catch (addError) {
      toast.error(messages.requestFailed);
    } finally {
      setAddingServer(false);
    }
  }

  async function runServerAction(serverId: number, action: 'start' | 'stop') {
    setBusyServerId(serverId);
    try {
      await apiRequest<void>(`/api/servers/${serverId}/${action}`, () => messages.requestFailed, {
        method: 'POST',
      });
      loadProjects();
      toast.success(action === 'start' ? messages.serverStarted : messages.serverStopped);
    } catch (actionError) {
      toast.error(messages.requestFailed);
    } finally {
      setBusyServerId(null);
    }
  }

  async function setServerEnabled(serverId: number, enabled: boolean) {
    setBusyServerId(serverId);
    try {
      await apiRequest<void>(`/api/servers/${serverId}/enabled`, () => messages.requestFailed, {
        method: 'PUT',
        body: JSON.stringify({ enabled }),
      });
      loadProjects();
      toast.success(enabled ? messages.serverEnabled : messages.serverDisabled);
    } catch (enableError) {
      toast.error(messages.requestFailed);
    } finally {
      setBusyServerId(null);
    }
  }

  async function inspectServer(server: ServerStatus) {
    setInspectOpen(true);
    setInspectingServerId(server.id);
    setInspection(null);
    setInspectionServerName(server.name);
    setInspectionError(null);
    try {
      const response = await apiRequest<ServerInspection>(`/api/servers/${server.id}/inspect`, () => messages.requestFailed);
      setInspection(response);
    } catch (inspectError) {
      setInspectionError(messages.requestFailed);
    }
  }

  async function openServerTools(server: ServerStatus) {
    setServerToolsOpen(true);
    setServerToolsLoadingId(server.id);
    setServerToolsSavingName(null);
    setServerToolsServerId(server.id);
    setServerToolsServerName(server.name);
    setServerToolsError(null);
    try {
      const response = await apiRequest<{ tools: ServerToolStatus[] }>(`/api/servers/${server.id}/tools`, () => messages.requestFailed);
      setServerTools(response.tools);
    } catch (toolsError) {
      setServerToolsError(messages.requestFailed);
    } finally {
      setServerToolsLoadingId(null);
    }
  }

  async function setServerToolEnabled(toolName: string, enabled: boolean, loadProjects: () => void) {
    if (!serverToolsServerId) {
      return;
    }

    setServerToolsSavingName(toolName);
    try {
      await apiRequest<void>(`/api/servers/${serverToolsServerId}/tools/${toolName}/enabled`, () => messages.requestFailed, {
        method: 'PUT',
        body: JSON.stringify({ enabled }),
      });
      const response = await apiRequest<{ tools: ServerToolStatus[] }>(`/api/servers/${serverToolsServerId}/tools`, () => messages.requestFailed);
      setServerTools(response.tools);
      loadProjects();
    } catch (enableError) {
      toast.error(messages.requestFailed);
    } finally {
      setServerToolsSavingName(null);
    }
  }

  return {
    // State
    addingServer,
    setAddingServer,
    addServerOpen,
    setAddServerOpen,
    editingServerId,
    setEditingServerId,
    busyServerId,
    setBusyServerId,
    inspectOpen,
    setInspectOpen,
    inspectingServerId,
    setInspectingServerId,
    inspection,
    setInspection,
    inspectionServerName,
    setInspectionServerName,
    inspectionError,
    setInspectionError,
    serverToolsOpen,
    setServerToolsOpen,
    serverToolsLoadingId,
    setServerToolsLoadingId,
    serverToolsSavingName,
    setServerToolsSavingName,
    serverToolsServerId,
    setServerToolsServerId,
    serverToolsServerName,
    setServerToolsServerName,
    serverTools,
    setServerTools,
    serverToolsError,
    setServerToolsError,
    // Functions
    addServer,
    runServerAction,
    setServerEnabled,
    inspectServer,
    openServerTools,
    setServerToolEnabled,
  };
}
