import type { FormEvent } from 'react';

import { AlertCircle, Copy, LoaderCircle, Plus, Radio, RefreshCw } from 'lucide-react';

import type { Language } from '../i18n';
import { dictionaries } from '../i18n';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';

type ProjectFormState = {
  name: string;
  description: string;
  root_path: string;
  identity_verification_enabled: boolean;
};

type SidebarProject = {
  project_id: number;
  name: string;
  description: string;
  connection_ready: boolean;
  is_paused: boolean;
  servers: Array<{ status: string }>;
};

type ProjectsSidebarProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  language: Language;
  languageOptions: Array<{ value: Language; label: string }>;
  setLanguage: (value: Language) => void;
  onRefresh: () => void | Promise<void>;
  refreshing: boolean;
  projects: SidebarProject[];
  loading: boolean;
  selectedProjectId: number | null;
  setSelectedProjectId: (projectId: number) => void;
  createProjectOpen: boolean;
  setCreateProjectOpen: (open: boolean) => void;
  editingProjectId: number | null;
  projectForm: ProjectFormState;
  setProjectForm: (
    updater: (current: ProjectFormState) => ProjectFormState,
  ) => void;
  resetProjectForm: () => void;
  createProject: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  creatingProject: boolean;
  duplicateProjectOpen: boolean;
  setDuplicateProjectOpen: (open: boolean) => void;
  duplicateProjectName: string;
  setDuplicateProjectName: (value: string) => void;
  duplicateProject: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  duplicatingProjectId: number | null;
  selectedProject: SidebarProject | null;
};

export function ProjectsSidebar({
  labels,
  messages,
  language,
  languageOptions,
  setLanguage,
  onRefresh,
  refreshing,
  projects,
  loading,
  selectedProjectId,
  setSelectedProjectId,
  createProjectOpen,
  setCreateProjectOpen,
  editingProjectId,
  projectForm,
  setProjectForm,
  resetProjectForm,
  createProject,
  creatingProject,
  duplicateProjectOpen,
  setDuplicateProjectOpen,
  duplicateProjectName,
  setDuplicateProjectName,
  duplicateProject,
  duplicatingProjectId,
  selectedProject,
}: ProjectsSidebarProps) {
  return (
    <aside className="w-full max-w-sm border-r border-border bg-sidebar/40">
      <div className="border-b border-border px-6 py-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
              {labels.appTitle}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Select value={language} onValueChange={(value) => setLanguage(value as Language)}>
              <SelectTrigger
                className="h-10 w-[150px] rounded-md border-border bg-card text-xs"
                aria-label={labels.language}
              >
                <SelectValue placeholder={labels.language} />
              </SelectTrigger>
              <SelectContent>
                {languageOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <button
              onClick={() => void onRefresh()}
              className="rounded-md border border-border bg-card p-2 transition-colors hover:bg-accent"
              aria-label="Refresh projects"
            >
              <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
        <p className="mt-3 text-sm text-muted-foreground">{labels.appDescription}</p>
      </div>

      <div className="space-y-6 p-6">
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-medium text-muted-foreground">{labels.projects}</h2>
            <div className="flex items-center gap-2">
              <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
                {projects.length}
              </span>
              <Dialog
                open={createProjectOpen}
                onOpenChange={(open) => {
                  setCreateProjectOpen(open);
                  if (!open) {
                    resetProjectForm();
                  }
                }}
              >
                <DialogTrigger asChild>
                  <button className="inline-flex h-8 items-center justify-center gap-2 rounded-md bg-electric-blue px-3 text-xs font-medium text-white transition-colors hover:bg-electric-blue/90">
                    <Plus className="h-3.5 w-3.5" />
                    {labels.createProject}
                  </button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-xl">
                  <DialogHeader>
                    <DialogTitle>
                      {editingProjectId ? 'Edit Project' : labels.createProject}
                    </DialogTitle>
                    <DialogDescription>{messages.projectHelper}</DialogDescription>
                  </DialogHeader>

                  <form className="space-y-3" onSubmit={createProject}>
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{labels.name}</span>
                      <input
                        required
                        value={projectForm.name}
                        onChange={(event) =>
                          setProjectForm((current) => ({ ...current, name: event.target.value }))
                        }
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder={messages.projectNamePlaceholder}
                      />
                    </label>

                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{labels.description}</span>
                      <textarea
                        value={projectForm.description}
                        onChange={(event) =>
                          setProjectForm((current) => ({
                            ...current,
                            description: event.target.value,
                          }))
                        }
                        rows={3}
                        className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder={messages.projectDescriptionPlaceholder}
                      />
                    </label>

                    <button
                      type="submit"
                      disabled={creatingProject}
                      className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {creatingProject ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : (
                        <Plus className="h-4 w-4" />
                      )}
                      {editingProjectId ? 'Save Project' : labels.createProject}
                    </button>
                  </form>
                </DialogContent>
              </Dialog>
              <Dialog
                open={duplicateProjectOpen}
                onOpenChange={(open) => {
                  setDuplicateProjectOpen(open);
                  if (!open) {
                    setDuplicateProjectName('');
                  }
                }}
              >
                <DialogContent className="sm:max-w-xl">
                  <DialogHeader>
                    <DialogTitle>{labels.duplicateProject}</DialogTitle>
                    <DialogDescription>{messages.duplicateProjectDescription}</DialogDescription>
                  </DialogHeader>

                  <form className="space-y-3" onSubmit={duplicateProject}>
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{labels.name}</span>
                      <input
                        required
                        value={duplicateProjectName}
                        onChange={(event) => setDuplicateProjectName(event.target.value)}
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder={messages.duplicateProjectNamePlaceholder}
                      />
                    </label>

                    <button
                      type="submit"
                      disabled={!selectedProject || duplicatingProjectId === selectedProject.project_id}
                      className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {duplicatingProjectId === selectedProject?.project_id ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                      {labels.duplicateProject}
                    </button>
                  </form>
                </DialogContent>
              </Dialog>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-4 py-5 text-sm text-muted-foreground">
              <LoaderCircle className="h-4 w-4 animate-spin" />
              {messages.loadingProjects}
            </div>
          ) : projects.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border bg-card px-4 py-5 text-sm text-muted-foreground">
              {messages.noProjects}
            </div>
          ) : (
            <div className="space-y-2">
              {projects.map((project) => {
                const runningCount = project.servers.filter(
                  (server) => server.status === 'Running',
                ).length;

                return (
                  <button
                    key={project.project_id}
                    onClick={() => setSelectedProjectId(project.project_id)}
                    className={`w-full rounded-xl border p-4 text-left transition-colors ${
                      selectedProjectId === project.project_id
                        ? 'border-electric-blue bg-electric-blue/8'
                        : 'border-border bg-card hover:bg-accent/60'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium">{project.name}</div>
                        <div className="mt-1 text-sm text-muted-foreground">
                          {project.description || messages.workspaceGroupFallback}
                        </div>
                      </div>
                      {project.connection_ready ? (
                        <Radio className="mt-0.5 h-4 w-4 text-status-running" />
                      ) : (
                        <AlertCircle className="mt-0.5 h-4 w-4 text-amber-500" />
                      )}
                    </div>
                    <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{messages.serverCount(project.servers.length)}</span>
                      <span>•</span>
                      <span>{messages.runningCount(runningCount)}</span>
                      {project.is_paused ? (
                        <>
                          <span>•</span>
                          <span className="text-amber-600">{labels.paused}</span>
                        </>
                      ) : null}
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </section>
      </div>
    </aside>
  );
}
