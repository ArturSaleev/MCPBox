import { Copy, LoaderCircle, MoreHorizontal, Pause, Pencil, Play, Trash2 } from 'lucide-react';

import { dictionaries } from '../i18n';
import {
  Menubar,
  MenubarContent,
  MenubarItem,
  MenubarMenu,
  MenubarTrigger,
} from './ui/menubar';

type ProjectActionsPanelProps = {
  labels: typeof dictionaries.en.labels;
  selectedProject: {
    project_id: number;
    is_paused: boolean;
  };
  busyProjectId: number | null;
  setProjectPaused: (projectId: number, paused: boolean) => void | Promise<void>;
  startDuplicateProject: () => void;
  startEditProject: () => void;
  deleteProject: (projectId: number) => void | Promise<void>;
};

export function ProjectActionsPanel({
  labels,
  selectedProject,
  busyProjectId,
  setProjectPaused,
  startDuplicateProject,
  startEditProject,
  deleteProject,
}: ProjectActionsPanelProps) {
  return (
    <Menubar className="h-auto border-0 bg-transparent p-0 shadow-none">
      <MenubarMenu>
        <MenubarTrigger className="inline-flex h-11 items-center justify-center gap-2 rounded-xl border border-border px-4 text-sm font-medium transition-colors hover:bg-accent">
          <MoreHorizontal className="h-4 w-4" />
          {labels.actions}
        </MenubarTrigger>
        <MenubarContent align="end" className="min-w-[15rem] rounded-xl">
          <MenubarItem
            disabled={busyProjectId === selectedProject.project_id}
            onSelect={() =>
              void setProjectPaused(selectedProject.project_id, !selectedProject.is_paused)
            }
          >
            {busyProjectId === selectedProject.project_id ? (
              <LoaderCircle className="h-4 w-4 animate-spin" />
            ) : selectedProject.is_paused ? (
              <Play className="h-4 w-4" />
            ) : (
              <Pause className="h-4 w-4" />
            )}
            {selectedProject.is_paused ? labels.resumeProject : labels.pauseProject}
          </MenubarItem>
          <MenubarItem onSelect={startDuplicateProject}>
            <Copy className="h-4 w-4" />
            {labels.duplicateProject}
          </MenubarItem>
          <MenubarItem onSelect={startEditProject}>
            <Pencil className="h-4 w-4" />
            {labels.edit}
          </MenubarItem>
          <MenubarItem
            variant="destructive"
            disabled={busyProjectId === selectedProject.project_id}
            onSelect={() => void deleteProject(selectedProject.project_id)}
          >
            <Trash2 className="h-4 w-4" />
            {labels.delete}
          </MenubarItem>
        </MenubarContent>
      </MenubarMenu>
    </Menubar>
  );
}
