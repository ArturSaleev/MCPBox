import { FileText } from 'lucide-react';

import { dictionaries } from '../i18n';

type ProjectPromptPanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  selectedProject: {
    project_id: number;
    name: string;
    prompt: string;
  };
  updateProjectPrompt: (prompt: string) => void | Promise<void>;
  updatingPrompt: boolean;
};

export function ProjectPromptPanel({
  labels,
  messages,
  selectedProject,
  updateProjectPrompt,
  updatingPrompt,
}: ProjectPromptPanelProps) {
  return (
    <section className="rounded-2xl border border-border bg-card p-6">
      <div className="mb-4 flex items-start gap-3">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-electric-blue/10 text-electric-blue">
          <FileText className="h-5 w-5" />
        </div>
        <div>
          <h3 className="text-lg font-semibold">{labels.prompt}</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {messages.projectPromptDescription}
          </p>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-background p-4">
        <label className="block text-sm font-medium" htmlFor="project-prompt">
          {labels.prompt}
        </label>
        <textarea
          key={selectedProject.project_id}
          defaultValue={selectedProject.prompt || ''}
          disabled={updatingPrompt}
          className="mt-3 min-h-[220px] w-full rounded-xl border border-border bg-card px-4 py-3 text-sm transition-colors focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
          placeholder={labels.prompt}
          id="project-prompt"
        />
        <div className="mt-3 flex justify-end">
          <button
            type="button"
            onClick={() => {
              const textarea = document.getElementById('project-prompt') as HTMLTextAreaElement;
              if (textarea) {
                void updateProjectPrompt(textarea.value);
              }
            }}
            disabled={updatingPrompt}
            className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {updatingPrompt ? labels.saving : labels.save}
          </button>
        </div>
      </div>
    </section>
  );
}
