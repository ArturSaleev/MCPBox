import { Play, Square } from 'lucide-react';

interface Project {
  id: string;
  name: string;
  status: 'running' | 'stopped';
}

interface ProjectListProps {
  projects: Project[];
  selectedProject: string | null;
  onSelectProject: (id: string) => void;
  collapsed: boolean;
}

export function ProjectList({ projects, selectedProject, onSelectProject, collapsed }: ProjectListProps) {
  if (collapsed) return null;

  return (
    <div className="w-64 h-full bg-card border-r border-border flex flex-col">
      <div className="p-4 border-b border-border">
        <h3>Projects</h3>
      </div>

      <div className="flex-1 overflow-y-auto p-2">
        {projects.map((project) => (
          <button
            key={project.id}
            onClick={() => onSelectProject(project.id)}
            className={`w-full text-left p-3 rounded-md mb-2 transition-colors ${
              selectedProject === project.id
                ? 'bg-accent text-accent-foreground'
                : 'hover:bg-accent/50'
            }`}
          >
            <div className="flex items-center justify-between mb-1">
              <span className="font-medium">{project.name}</span>
              {project.status === 'running' ? (
                <Play className="w-4 h-4 text-status-running fill-status-running" />
              ) : (
                <Square className="w-4 h-4 text-status-stopped" />
              )}
            </div>
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full ${
                project.status === 'running' ? 'bg-status-running' : 'bg-status-stopped'
              }`} />
              <span className="text-sm text-muted-foreground capitalize">{project.status}</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
