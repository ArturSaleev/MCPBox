import { FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { apiRequest } from '../utils/api';
import type { ProjectStatus } from '../types';

export type ProjectFormState = {
  name: string;
  description: string;
  root_path: string;
  identity_verification_enabled: boolean;
  bearer_auth_enabled: boolean;
  bearer_token: string;
  oauth_redirect_uri: string;
};

export type ServerFormState = {
  name: string;
  transport: 'stdio' | 'http_stream';
  command: string;
  args: string[];
  env_vars: KeyValuePair[];
  env_passthrough: string[];
  working_dir: string;
  url: string;
  bearer_token_env_var: string;
  headers: KeyValuePair[];
  header_env_vars: KeyValuePair[];
  auth_type: 'none' | 'oauth2';
  oauth_provider: string;
  oauth_authorize_url: string;
  oauth_token_url: string;
  oauth_refresh_url: string;
  oauth_client_id: string;
  oauth_client_secret: string;
  oauth_scopes: string[];
  auto_start: boolean;
};

export type KeyValuePair = {
  key: string;
  value: string;
};

const emptyProjectForm: ProjectFormState = {
  name: '',
  description: '',
  root_path: '',
  identity_verification_enabled: false,
  bearer_auth_enabled: false,
  bearer_token: '',
  oauth_redirect_uri: '',
};

const emptyServerForm: ServerFormState = {
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
};

export function useProjects(messages: { requestFailed: string; projectCreated: string; projectUpdated: string; projectDeleted: string; projectDuplicated: string }) {
  const [projects, setProjects] = useState<ProjectStatus[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [projectForm, setProjectForm] = useState<ProjectFormState>(emptyProjectForm);
  const [serverForm, setServerForm] = useState<ServerFormState>(emptyServerForm);
  const [creatingProject, setCreatingProject] = useState(false);
  const [createProjectOpen, setCreateProjectOpen] = useState(false);
  const [launchProjectOpen, setLaunchProjectOpen] = useState(false);
  const [duplicateProjectOpen, setDuplicateProjectOpen] = useState(false);
  const [editingProjectId, setEditingProjectId] = useState<number | null>(null);
  const [updatingPrompt, setUpdatingPrompt] = useState(false);
  const [duplicatingProjectId, setDuplicatingProjectId] = useState<number | null>(null);
  const [duplicateProjectName, setDuplicateProjectName] = useState('');
  const [busyProjectId, setBusyProjectId] = useState<number | null>(null);
  const [launchingOllamaProjectId, setLaunchingOllamaProjectId] = useState<number | null>(null);
  const [launchingLMStudioProjectId, setLaunchingLMStudioProjectId] = useState<number | null>(null);

  const selectedProject = projects.find((project) => project.project_id === selectedProjectId) ?? null;

  async function loadProjects(initial = false) {
    try {
      const nextProjects = await apiRequest<ProjectStatus[]>('/api/projects', () => messages.requestFailed);
      setProjects(nextProjects);
    } catch (loadError) {
      console.error('Failed to load projects:', loadError);
    }
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatingProject(true);
    try {
      await apiRequest<void>('/api/projects', () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify(projectForm),
      });
      await loadProjects();
      setCreateProjectOpen(false);
      setProjectForm(emptyProjectForm);
      toast.success(messages.projectCreated);
    } catch (createError) {
      toast.error(messages.requestFailed);
    } finally {
      setCreatingProject(false);
    }
  }

  async function updateProjectPrompt(prompt: string) {
    if (!selectedProject) {
      return;
    }

    setUpdatingPrompt(true);
    try {
      await apiRequest<void>(`/api/projects/${selectedProject.project_id}`, () => messages.requestFailed, {
        method: 'PUT',
        body: JSON.stringify({
          name: selectedProject.name,
          description: selectedProject.description,
          root_path: selectedProject.root_path,
          identity_verification_enabled: selectedProject.identity_verification_enabled,
          prompt,
        }),
      });
      await loadProjects();
      toast.success(messages.projectUpdated);
    } catch (updateError) {
      toast.error(messages.requestFailed);
    } finally {
      setUpdatingPrompt(false);
    }
  }

  async function duplicateProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject) {
      return;
    }

    setDuplicatingProjectId(selectedProject.project_id);
    try {
      await apiRequest<void>(`/api/projects/${selectedProject.project_id}/duplicate`, () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify({ name: duplicateProjectName }),
      });
      await loadProjects();
      setDuplicateProjectOpen(false);
      setDuplicateProjectName('');
      toast.success(messages.projectDuplicated);
    } catch (duplicateError) {
      toast.error(messages.requestFailed);
    } finally {
      setDuplicatingProjectId(null);
    }
  }

  async function deleteProject(projectId: number) {
    const confirmed = window.confirm('Delete this project and all its servers?');
    if (!confirmed) {
      return;
    }

    setBusyProjectId(projectId);
    try {
      await apiRequest<void>(`/api/projects/${projectId}`, () => messages.requestFailed, {
        method: 'DELETE',
      });
      await loadProjects();
      if (selectedProjectId === projectId) {
        setSelectedProjectId(null);
      }
      toast.success(messages.projectDeleted);
    } catch (deleteError) {
      toast.error(messages.requestFailed);
    } finally {
      setBusyProjectId(null);
    }
  }

  async function setProjectPaused(projectId: number, paused: boolean) {
    setBusyProjectId(projectId);
    try {
      await apiRequest<void>(`/api/projects/${projectId}/paused`, () => messages.requestFailed, {
        method: 'PUT',
        body: JSON.stringify({ paused }),
      });
      await loadProjects();
    } catch (pauseError) {
      toast.error(messages.requestFailed);
    } finally {
      setBusyProjectId(null);
    }
  }

  async function launchProjectOllama(projectId: number) {
    setLaunchingOllamaProjectId(projectId);
    try {
      await apiRequest<void>(`/api/projects/${projectId}/launch-ollama`, () => messages.requestFailed, {
        method: 'POST',
      });
      await loadProjects();
    } catch (launchError) {
      toast.error(messages.requestFailed);
    } finally {
      setLaunchingOllamaProjectId(null);
    }
  }

  async function launchProjectLMStudio(projectId: number) {
    setLaunchingLMStudioProjectId(projectId);
    try {
      await apiRequest<void>(`/api/projects/${projectId}/launch-lmstudio`, () => messages.requestFailed, {
        method: 'POST',
      });
      await loadProjects();
    } catch (launchError) {
      toast.error(messages.requestFailed);
    } finally {
      setLaunchingLMStudioProjectId(null);
    }
  }

  async function copyConnectURL() {
    if (!selectedProject?.connect_url) {
      return;
    }

    try {
      await navigator.clipboard.writeText(selectedProject.connect_url);
      toast.success('URL copied to clipboard');
    } catch (copyError) {
      toast.error('Failed to copy URL');
    }
  }

  return {
    // State
    projects,
    setProjects,
    selectedProjectId,
    setSelectedProjectId,
    selectedProject,
    projectForm,
    setProjectForm,
    serverForm,
    setServerForm,
    creatingProject,
    createProjectOpen,
    setCreateProjectOpen,
    launchProjectOpen,
    setLaunchProjectOpen,
    duplicateProjectOpen,
    setDuplicateProjectOpen,
    editingProjectId,
    setEditingProjectId,
    updatingPrompt,
    duplicatingProjectId,
    duplicateProjectName,
    setDuplicateProjectName,
    busyProjectId,
    launchingOllamaProjectId,
    launchingLMStudioProjectId,
    // Constants
    emptyProjectForm,
    emptyServerForm,
    // Functions
    loadProjects,
    createProject,
    updateProjectPrompt,
    duplicateProject,
    deleteProject,
    setProjectPaused,
    launchProjectOllama,
    launchProjectLMStudio,
    copyConnectURL,
  };
}
