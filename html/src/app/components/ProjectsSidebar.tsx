import { useRef, useState } from 'react';
import type { FormEvent } from 'react';

import { AlertCircle, ArrowDown, ArrowUp, Copy, GripVertical, LoaderCircle, Plus, Radio, RefreshCw } from 'lucide-react';

import { moveProject, readProjectOrder, reconcileProjectOrder } from '../utils/project-order';

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
  bearer_auth_enabled: boolean;
  bearer_token: string;
  oauth_redirect_uri: string;
};

type SidebarProject = {
  project_id: number;
  name: string;
  description: string;
  connection_ready: boolean;
  is_paused: boolean;
  bearer_auth_enabled?: boolean;
  bearer_token?: string;
  oauth_redirect_uri?: string;
  servers: Array<{ status: string }>;
};

type ProjectsSidebarProps = {
  orderStorageKey: string;
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
  orderStorageKey,
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
  const [savedOrder, setSavedOrder] = useState<number[]>(() => {
    try { return readProjectOrder(window.localStorage.getItem(orderStorageKey)); } catch { return []; }
  });
  const [draggedId, setDraggedId] = useState<number | null>(null);
  const [dropTarget, setDropTarget] = useState<{ id: number; after: boolean } | null>(null);
  const draggedIdRef = useRef<number | null>(null);
  const [announcement, setAnnouncement] = useState('');
  const [storageError, setStorageError] = useState(false);
  const orderedIds = reconcileProjectOrder(projects.map(project => project.project_id), savedOrder);
  const projectsById = new Map(projects.map(project => [project.project_id, project]));
  const orderedProjects = orderedIds.map(id => projectsById.get(id)!);
  const copy = language === 'ru' ? {
    drag: 'Переместить проект', up: 'Поднять проект', down: 'Опустить проект',
    hint: 'Перетащите проект за ручку. Или используйте стрелки вверх и вниз.',
    moved: (name: string, position: number) => `${name}: позиция ${position} из ${projects.length}.`,
    failed: 'Порядок изменён, но браузер не смог сохранить его для следующего входа.',
  } : {
    drag: 'Move project', up: 'Move project up', down: 'Move project down',
    hint: 'Drag a project by its handle, or use the up and down arrows.',
    moved: (name: string, position: number) => `${name}: position ${position} of ${projects.length}.`,
    failed: 'The order changed, but the browser could not save it for your next visit.',
  };
  function reorder(source: number, target: number, after: boolean) {
    const next = moveProject(orderedIds, source, target, after);
    if (next.every((id, index) => id === orderedIds[index])) return;
    setSavedOrder(next);
    try { window.localStorage.setItem(orderStorageKey, JSON.stringify(next)); setStorageError(false); }
    catch { setStorageError(true); }
    setAnnouncement(copy.moved(projectsById.get(source)?.name ?? '', next.indexOf(source) + 1));
  }
  function finishDrag() {
    draggedIdRef.current = null;
    setDraggedId(null);
    setDropTarget(null);
  }
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
                  <button className="inline-flex h-8 items-center justify-center gap-2 rounded-md bg-brand-solid px-3 text-xs font-medium text-white transition-colors hover:bg-brand-solid/90">
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

                    <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-3">
                      <label className="flex items-start gap-3">
                        <input
                          type="checkbox"
                          checked={projectForm.bearer_auth_enabled}
                          onChange={(event) =>
                            setProjectForm((current) => ({
                              ...current,
                              bearer_auth_enabled: event.target.checked,
                            }))
                          }
                          className="mt-1 h-4 w-4 rounded border-border text-electric-blue focus:ring-electric-blue"
                        />
                        <span>
                          <span className="block text-sm font-medium">{labels.bearerEndpointProtection}</span>
                          <span className="mt-1 block text-xs leading-5 text-muted-foreground">
                            {messages.bearerEndpointProtectionDescription}
                          </span>
                        </span>
                      </label>

                      {projectForm.bearer_auth_enabled ? (
                        <p className="text-xs leading-5 text-muted-foreground">
                          {messages.bearerTokenGeneratedAfterCreate}
                        </p>
                      ) : null}
                    </div>

                    <button
                      type="submit"
                      disabled={creatingProject}
                      className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-brand-solid px-4 text-sm font-medium text-white transition-colors hover:bg-brand-solid/90 disabled:cursor-not-allowed disabled:opacity-70"
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
                      className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-brand-solid px-4 text-sm font-medium text-white transition-colors hover:bg-brand-solid/90 disabled:cursor-not-allowed disabled:opacity-70"
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
              <p id="project-order-help" className="sr-only">{copy.hint}</p>
              <p role="status" aria-live="polite" className="sr-only">{announcement}</p>
              {storageError ? <p role="status" className="rounded-lg border border-warning/30 bg-warning/10 p-2 text-xs">{copy.failed}</p> : null}
              <div className="space-y-2" role="list" aria-label={labels.projects}>
              {orderedProjects.map((project, index) => {
                const runningCount = project.servers.filter(
                  (server) => server.status === 'Running',
                ).length;

                return (
                  <div
                    role="listitem"
                    key={project.project_id}
                    onDragOver={event => {
                      if (draggedIdRef.current === null || draggedIdRef.current === project.project_id) return;
                      event.preventDefault();
                      event.dataTransfer.dropEffect = 'move';
                      const bounds = event.currentTarget.getBoundingClientRect();
                      const after = event.clientY > bounds.top + bounds.height / 2;
                      setDropTarget(current => current?.id === project.project_id && current.after === after ? current : { id: project.project_id, after });
                    }}
                    onDragLeave={event => {
                      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDropTarget(current => current?.id === project.project_id ? null : current);
                    }}
                    onDrop={event => {
                      event.preventDefault();
                      const source = draggedIdRef.current;
                      if (source !== null) {
                        const bounds = event.currentTarget.getBoundingClientRect();
                        reorder(source, project.project_id, event.clientY > bounds.top + bounds.height / 2);
                      }
                      finishDrag();
                    }}
                    className={`relative flex items-center gap-1 rounded-xl border px-1 transition-colors ${draggedId === project.project_id ? 'opacity-50' : ''} ${
                      dropTarget?.id === project.project_id ? (dropTarget.after ? 'after:absolute after:inset-x-0 after:-bottom-1 after:h-0.5 after:bg-electric-blue' : 'before:absolute before:inset-x-0 before:-top-1 before:h-0.5 before:bg-electric-blue') : ''
                    } ${
                      selectedProjectId === project.project_id
                        ? 'border-electric-blue bg-brand-solid/8'
                        : 'border-border bg-card hover:bg-accent/60'
                    }`}
                  >
                    <button type="button" draggable={projects.length > 1} disabled={projects.length < 2}
                      aria-label={`${copy.drag}: ${project.name}`} aria-describedby="project-order-help" title={copy.hint}
                      onDragStart={event => {
                        draggedIdRef.current = project.project_id;
                        event.dataTransfer.effectAllowed = 'move';
                        event.dataTransfer.setData('text/plain', String(project.project_id));
                        setDraggedId(project.project_id);
                      }}
                      onDragEnd={finishDrag}
                      onKeyDown={event => {
                        if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return;
                        event.preventDefault();
                        const targetIndex = index + (event.key === 'ArrowUp' ? -1 : 1);
                        if (orderedIds[targetIndex] !== undefined) reorder(project.project_id, orderedIds[targetIndex], event.key === 'ArrowDown');
                      }}
                      className="flex h-10 w-6 shrink-0 cursor-grab items-center justify-center rounded-md text-muted-foreground hover:bg-accent focus-visible:outline-2 focus-visible:outline-electric-blue active:cursor-grabbing disabled:cursor-default disabled:opacity-30">
                      <GripVertical className="h-4 w-4" />
                    </button>
                    <button type="button" onClick={() => setSelectedProjectId(project.project_id)} aria-current={selectedProjectId === project.project_id ? 'page' : undefined} className="min-w-0 flex-1 rounded-lg px-2 py-4 text-left focus-visible:outline-2 focus-visible:outline-electric-blue">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 break-words font-medium">{project.name}</div>
                      {project.connection_ready ? (
                        <Radio className="mt-0.5 h-4 w-4 text-status-running" />
                      ) : (
                        <AlertCircle className="mt-0.5 h-4 w-4 text-amber-500" />
                      )}
                    </div>
                    <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
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
                    <div className="flex shrink-0 flex-col gap-1">
                      <button type="button" aria-label={`${copy.up}: ${project.name}`} title={copy.up} disabled={index === 0} onClick={() => reorder(project.project_id, orderedIds[index - 1], false)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent focus-visible:outline-2 focus-visible:outline-electric-blue disabled:opacity-25"><ArrowUp className="h-3.5 w-3.5" /></button>
                      <button type="button" aria-label={`${copy.down}: ${project.name}`} title={copy.down} disabled={index === orderedIds.length - 1} onClick={() => reorder(project.project_id, orderedIds[index + 1], true)} className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent focus-visible:outline-2 focus-visible:outline-electric-blue disabled:opacity-25"><ArrowDown className="h-3.5 w-3.5" /></button>
                    </div>
                  </div>
                );
              })}
              </div>
            </div>
          )}
        </section>
      </div>
    </aside>
  );
}
