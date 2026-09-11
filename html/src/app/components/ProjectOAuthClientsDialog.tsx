import { useEffect, useState, type FormEvent } from 'react';

import { Copy, Eye, EyeOff, LoaderCircle, Pause, Play, Plus, Trash2, Users } from 'lucide-react';

import { dictionaries } from '../i18n';
import { apiRequest } from '../utils/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

type OAuthClient = {
  id: number;
  name: string;
  is_enabled: boolean;
  created_at: string;
};

type ProjectOAuthClientsDialogProps = {
  projectId: number;
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
};

export function ProjectOAuthClientsDialog({
  projectId,
  labels,
  messages,
}: ProjectOAuthClientsDialogProps) {
  const [open, setOpen] = useState(false);
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [loading, setLoading] = useState(false);
  const [busyClientId, setBusyClientId] = useState<number | null>(null);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [redirectURI, setRedirectURI] = useState('');
  const [revealedToken, setRevealedToken] = useState('');
  const [revealedClientName, setRevealedClientName] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setRevealedToken('');
      setRevealedClientName('');
      return;
    }
    void loadClients();
  }, [open, projectId]);

  async function loadClients() {
    setLoading(true);
    setError(null);
    try {
      setClients(await apiRequest<OAuthClient[]>(
        `/api/projects/${projectId}/oauth-clients`,
        messages.requestFailed,
      ));
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : messages.loadProjectsError);
    } finally {
      setLoading(false);
    }
  }

  async function createClient(event: FormEvent) {
    event.preventDefault();
    setCreating(true);
    setError(null);
    try {
      await apiRequest<OAuthClient>(
        `/api/projects/${projectId}/oauth-clients`,
        messages.requestFailed,
        {
          method: 'POST',
          body: JSON.stringify({ name, redirect_uri: redirectURI }),
        },
      );
      setName('');
      setRedirectURI('');
      await loadClients();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : messages.createProjectError);
    } finally {
      setCreating(false);
    }
  }

  async function revealToken(client: OAuthClient) {
    if (revealedToken && revealedClientName === client.name) {
      setRevealedToken('');
      setRevealedClientName('');
      return;
    }
    setBusyClientId(client.id);
    setError(null);
    try {
      const payload = await apiRequest<{ token: string }>(
        `/api/projects/${projectId}/oauth-clients/${client.id}/token`,
        messages.requestFailed,
      );
      setRevealedToken(payload.token);
      setRevealedClientName(client.name);
    } catch (revealError) {
      setError(revealError instanceof Error ? revealError.message : messages.loadProjectsError);
    } finally {
      setBusyClientId(null);
    }
  }

  async function setClientEnabled(client: OAuthClient) {
    setBusyClientId(client.id);
    setError(null);
    try {
      await apiRequest<OAuthClient>(
        `/api/projects/${projectId}/oauth-clients/${client.id}/${client.is_enabled ? 'disable' : 'enable'}`,
        messages.requestFailed,
        { method: 'POST' },
      );
      if (revealedClientName === client.name) {
        setRevealedToken('');
        setRevealedClientName('');
      }
      await loadClients();
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : messages.setServerEnabledError);
    } finally {
      setBusyClientId(null);
    }
  }

  async function deleteClient(client: OAuthClient) {
    if (!window.confirm(`${labels.delete}: ${client.name}?`)) {
      return;
    }
    setBusyClientId(client.id);
    setError(null);
    try {
      await apiRequest<{ deleted: boolean }>(
        `/api/projects/${projectId}/oauth-clients/${client.id}`,
        messages.requestFailed,
        { method: 'DELETE' },
      );
      if (revealedClientName === client.name) {
        setRevealedToken('');
        setRevealedClientName('');
      }
      await loadClients();
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : messages.requestFailed(500));
    } finally {
      setBusyClientId(null);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <DialogTrigger asChild>
            <button
              type="button"
              aria-label={labels.oauthClients}
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <Users className="h-4 w-4" />
            </button>
          </DialogTrigger>
        </TooltipTrigger>
        <TooltipContent>{labels.oauthClients}</TooltipContent>
      </Tooltip>

      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{labels.oauthClients}</DialogTitle>
          <DialogDescription>{labels.oauthClientsDescription}</DialogDescription>
        </DialogHeader>

        <form className="grid gap-3 rounded-xl border border-border bg-background p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_auto] md:items-end" onSubmit={createClient}>
          <label className="block space-y-2">
            <span className="text-sm font-medium">{labels.clientName}</span>
            <input
              required
              value={name}
              onChange={(event) => setName(event.target.value)}
              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none focus:border-electric-blue"
              placeholder="chatgpt-user-1"
            />
          </label>
          <label className="block space-y-2">
            <span className="text-sm font-medium">{labels.oauthCallbackURL}</span>
            <input
              required
              type="url"
              value={redirectURI}
              onChange={(event) => setRedirectURI(event.target.value)}
              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none focus:border-electric-blue"
              placeholder="https://chatgpt.com/connector/oauth/..."
            />
          </label>
          <button
            type="submit"
            disabled={creating}
            className="inline-flex h-10 items-center justify-center gap-2 whitespace-nowrap rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:opacity-60"
          >
            {creating ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            {labels.addOAuthClient}
          </button>
          <p className="text-xs text-muted-foreground md:col-span-3">{labels.callbackStoredHidden}</p>
        </form>

        {error ? (
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div>
        ) : null}

        {revealedToken ? (
          <div className="rounded-xl border border-electric-blue/30 bg-electric-blue/5 p-4">
            <div className="mb-2 flex items-center justify-between gap-3">
              <span className="text-sm font-medium">{revealedClientName}</span>
              <button type="button" onClick={() => { setRevealedToken(''); setRevealedClientName(''); }} aria-label={labels.hideToken}>
                <EyeOff className="h-4 w-4" />
              </button>
            </div>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 overflow-x-auto rounded-md border border-border bg-card px-3 py-2 text-xs text-electric-blue">{revealedToken}</code>
              <button
                type="button"
                onClick={() => void navigator.clipboard.writeText(revealedToken)}
                aria-label={labels.copyToken}
                title={labels.copyToken}
                className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border hover:bg-accent"
              >
                <Copy className="h-4 w-4" />
              </button>
            </div>
          </div>
        ) : null}

        <div className="overflow-x-auto rounded-xl border border-border">
          <table className="w-full min-w-[620px] text-left text-sm">
            <thead className="border-b border-border bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground">
              <tr>
                <th className="px-4 py-3">{labels.clientName}</th>
                <th className="px-4 py-3">{labels.status}</th>
                <th className="px-4 py-3">{labels.createdAt}</th>
                <th className="px-4 py-3 text-right">{labels.actions}</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground"><LoaderCircle className="mx-auto h-5 w-5 animate-spin" /></td></tr>
              ) : clients.length === 0 ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">{labels.noOAuthClients}</td></tr>
              ) : clients.map((client) => {
                const busy = busyClientId === client.id;
                return (
                  <tr key={client.id} className="border-b border-border last:border-b-0">
                    <td className="px-4 py-3 font-medium">{client.name}</td>
                    <td className="px-4 py-3">
                      <span className={`rounded-full border px-2 py-1 text-xs ${client.is_enabled ? 'border-status-running/30 bg-status-running/10 text-status-running' : 'border-border bg-muted text-muted-foreground'}`}>
                        {client.is_enabled ? labels.enabled : labels.disabled}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{new Date(client.created_at).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-2">
                        <button type="button" onClick={() => void revealToken(client)} disabled={busy} aria-label={labels.showToken} title={labels.showToken} className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border hover:bg-accent disabled:opacity-60">
                          {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />}
                        </button>
                        <button type="button" onClick={() => void setClientEnabled(client)} disabled={busy} aria-label={client.is_enabled ? labels.disconnect : labels.connect} title={client.is_enabled ? labels.disconnect : labels.connect} className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border hover:bg-accent disabled:opacity-60">
                          {client.is_enabled ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                        </button>
                        <button type="button" onClick={() => void deleteClient(client)} disabled={busy} aria-label={labels.delete} title={labels.delete} className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-destructive/30 text-destructive hover:bg-destructive/10 disabled:opacity-60">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </DialogContent>
    </Dialog>
  );
}
