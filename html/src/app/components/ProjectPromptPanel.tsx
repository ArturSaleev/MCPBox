import { useEffect, useMemo, useState } from 'react';
import { FileText, Plus, Trash2 } from 'lucide-react';

import { dictionaries } from '../i18n';
import type { ProjectPromptProfile } from '../types';

type ProjectPromptPanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  selectedProject: {
    project_id: number;
    name: string;
    prompt: string;
    prompt_profiles?: ProjectPromptProfile[];
  };
  updateProjectPrompt: (prompt: string, promptProfiles: ProjectPromptProfile[]) => void | Promise<void>;
  updatingPrompt: boolean;
};

type PromptProfileDraft = ProjectPromptProfile & {
  local_id: string;
};

function createEmptyProfile(index: number): PromptProfileDraft {
  return {
    local_id: `new-${Date.now()}-${index}`,
    id: '',
    name: '',
    description: '',
    prompt: '',
    response_format: 'text',
    response_schema: '',
    is_default: false,
  };
}

function normalizeProfileDrafts(profiles: PromptProfileDraft[]) {
  let defaultAssigned = false;
  return profiles
    .map((profile, index) => {
      const normalizedName = profile.name.trim();
      const normalizedPrompt = profile.prompt.trim();
      if (!normalizedName && !normalizedPrompt && !profile.description.trim() && !profile.response_schema.trim()) {
        return null;
      }
      const isDefault = profile.is_default && !defaultAssigned;
      if (isDefault) defaultAssigned = true;
      return {
        id: profile.id.trim(),
        name: normalizedName,
        description: profile.description.trim(),
        prompt: normalizedPrompt,
        response_format: (profile.response_format || 'text').trim() || 'text',
        response_schema: profile.response_schema.trim(),
        is_default: isDefault,
        local_id: profile.local_id || `profile-${index}`,
      };
    })
    .filter((profile): profile is PromptProfileDraft => Boolean(profile));
}

export function ProjectPromptPanel({
  labels,
  messages,
  selectedProject,
  updateProjectPrompt,
  updatingPrompt,
}: ProjectPromptPanelProps) {
  const isRussian = labels.save === 'Сохранить';
  const copy = useMemo(
    () =>
      isRussian
        ? {
            promptProfilesTitle: 'Prompt profiles',
            promptProfilesDescription:
              'Здесь можно сохранить несколько named prompt profiles для разных сценариев AgentBox.',
            profileName: 'Название профиля',
            profileDescription: 'Описание',
            profilePrompt: 'Prompt',
            responseFormat: 'Формат ответа',
            responseSchema: 'Schema / JSON contract',
            defaultProfile: 'Default profile',
            addProfile: 'Добавить профиль',
            deleteProfile: 'Удалить',
            emptyProfiles: 'Пока нет отдельных prompt profiles.',
            textFormat: 'Text',
            jsonFormat: 'JSON',
          }
        : {
            promptProfilesTitle: 'Prompt profiles',
            promptProfilesDescription:
              'Store multiple named prompt profiles for different AgentBox flows.',
            profileName: 'Profile name',
            profileDescription: 'Description',
            profilePrompt: 'Prompt',
            responseFormat: 'Response format',
            responseSchema: 'Schema / JSON contract',
            defaultProfile: 'Default profile',
            addProfile: 'Add profile',
            deleteProfile: 'Delete',
            emptyProfiles: 'No prompt profiles yet.',
            textFormat: 'Text',
            jsonFormat: 'JSON',
          },
    [isRussian, labels.save],
  );

  const [basePrompt, setBasePrompt] = useState(selectedProject.prompt || '');
  const [profiles, setProfiles] = useState<PromptProfileDraft[]>([]);

  useEffect(() => {
    setBasePrompt(selectedProject.prompt || '');
    setProfiles(
      (selectedProject.prompt_profiles || []).map((profile, index) => ({
        ...profile,
        local_id: profile.id || `profile-${index}`,
      })),
    );
  }, [selectedProject.project_id, selectedProject.prompt, selectedProject.prompt_profiles]);

  function updateProfile(localID: string, patch: Partial<PromptProfileDraft>) {
    setProfiles((current) =>
      current.map((profile) => {
        if (profile.local_id !== localID) return profile;
        const next = { ...profile, ...patch };
        if (patch.is_default) {
          return { ...next, is_default: true };
        }
        return next;
      }).map((profile) => {
        if (patch.is_default && profile.local_id !== localID) {
          return { ...profile, is_default: false };
        }
        return profile;
      }),
    );
  }

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

      <div className="space-y-5 rounded-xl border border-border bg-background p-4">
        <div>
          <label className="block text-sm font-medium" htmlFor="project-prompt">
            {labels.prompt}
          </label>
          <textarea
            value={basePrompt}
            onChange={(event) => setBasePrompt(event.target.value)}
            disabled={updatingPrompt}
            className="mt-3 min-h-[220px] w-full rounded-xl border border-border bg-card px-4 py-3 text-sm transition-colors focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={labels.prompt}
            id="project-prompt"
          />
        </div>

        <div className="border-t border-border pt-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h4 className="text-sm font-semibold">{copy.promptProfilesTitle}</h4>
              <p className="mt-1 text-xs text-muted-foreground">{copy.promptProfilesDescription}</p>
            </div>
            <button
              type="button"
              disabled={updatingPrompt}
              onClick={() => setProfiles((current) => [...current, createEmptyProfile(current.length + 1)])}
              className="inline-flex h-9 items-center gap-2 rounded-lg border border-border bg-card px-3 text-sm font-medium transition-colors hover:border-electric-blue hover:text-electric-blue disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Plus className="h-4 w-4" />
              {copy.addProfile}
            </button>
          </div>

          <div className="space-y-4">
            {profiles.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border bg-card px-4 py-5 text-sm text-muted-foreground">
                {copy.emptyProfiles}
              </div>
            ) : null}

            {profiles.map((profile, index) => (
              <div key={profile.local_id} className="space-y-4 rounded-xl border border-border bg-card p-4">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-sm font-medium">
                    {copy.profileName} #{index + 1}
                  </div>
                  <button
                    type="button"
                    disabled={updatingPrompt}
                    onClick={() =>
                      setProfiles((current) => current.filter((item) => item.local_id !== profile.local_id))
                    }
                    className="inline-flex h-8 items-center gap-2 rounded-lg border border-border px-3 text-sm text-muted-foreground transition-colors hover:border-red-400 hover:text-red-500 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Trash2 className="h-4 w-4" />
                    {copy.deleteProfile}
                  </button>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <label className="grid gap-2 text-sm">
                    <span className="text-muted-foreground">{copy.profileName}</span>
                    <input
                      value={profile.name}
                      onChange={(event) => updateProfile(profile.local_id, { name: event.target.value })}
                      disabled={updatingPrompt}
                      className="h-10 rounded-lg border border-border bg-background px-3 text-sm focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    />
                  </label>
                  <label className="grid gap-2 text-sm">
                    <span className="text-muted-foreground">ID</span>
                    <input
                      value={profile.id}
                      onChange={(event) => updateProfile(profile.local_id, { id: event.target.value })}
                      disabled={updatingPrompt}
                      className="h-10 rounded-lg border border-border bg-background px-3 text-sm focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    />
                  </label>
                </div>

                <label className="grid gap-2 text-sm">
                  <span className="text-muted-foreground">{copy.profileDescription}</span>
                  <input
                    value={profile.description}
                    onChange={(event) => updateProfile(profile.local_id, { description: event.target.value })}
                    disabled={updatingPrompt}
                    className="h-10 rounded-lg border border-border bg-background px-3 text-sm focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>

                <div className="grid gap-4 md:grid-cols-[180px_minmax(0,1fr)]">
                  <label className="grid gap-2 text-sm">
                    <span className="text-muted-foreground">{copy.responseFormat}</span>
                    <select
                      value={profile.response_format || 'text'}
                      onChange={(event) => updateProfile(profile.local_id, { response_format: event.target.value })}
                      disabled={updatingPrompt}
                      className="h-10 rounded-lg border border-border bg-background px-3 text-sm focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <option value="text">{copy.textFormat}</option>
                      <option value="json">{copy.jsonFormat}</option>
                    </select>
                  </label>
                  <label className="flex items-end gap-3 rounded-lg border border-border bg-background px-3 py-2 text-sm">
                    <input
                      type="checkbox"
                      checked={profile.is_default}
                      onChange={(event) => updateProfile(profile.local_id, { is_default: event.target.checked })}
                      disabled={updatingPrompt}
                    />
                    <span>{copy.defaultProfile}</span>
                  </label>
                </div>

                <label className="grid gap-2 text-sm">
                  <span className="text-muted-foreground">{copy.profilePrompt}</span>
                  <textarea
                    value={profile.prompt}
                    onChange={(event) => updateProfile(profile.local_id, { prompt: event.target.value })}
                    disabled={updatingPrompt}
                    className="min-h-[140px] rounded-xl border border-border bg-background px-4 py-3 text-sm focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>

                <label className="grid gap-2 text-sm">
                  <span className="text-muted-foreground">{copy.responseSchema}</span>
                  <textarea
                    value={profile.response_schema}
                    onChange={(event) => updateProfile(profile.local_id, { response_schema: event.target.value })}
                    disabled={updatingPrompt}
                    className="min-h-[110px] rounded-xl border border-border bg-background px-4 py-3 font-mono text-xs focus:border-electric-blue focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => void updateProjectPrompt(basePrompt, normalizeProfileDrafts(profiles))}
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
