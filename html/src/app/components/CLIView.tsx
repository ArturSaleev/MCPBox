import { useEffect, useState, type FormEvent } from 'react';
import { LoaderCircle, Pencil, Plus, Terminal, Trash2 } from 'lucide-react';
import { apiRequest } from '../utils/api';
import { Button } from './ui/button';
import { Checkbox } from './ui/checkbox';
import { Input } from './ui/input';
import { Textarea } from './ui/textarea';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog';

export type CLIEntry = {
  id: number; name: string; slug: string; description: string; binary: string;
  is_enabled: boolean; operations: Array<{ command: string; argument_sets?: string[][] }>;
  timeout_seconds: number; max_output_bytes: number; binary_available?: boolean; binary_path?: string;
};
function cliPayload(item: CLIEntry) { const { name, slug, description, binary, is_enabled, operations, timeout_seconds, max_output_bytes } = item; return { name, slug, description, binary, is_enabled, operations, timeout_seconds, max_output_bytes }; }
const emptyCLI = (): CLIEntry => ({ id: 0, name: '', slug: '', description: '', binary: '', is_enabled: true, operations: [], timeout_seconds: 30, max_output_bytes: 1048576 });
const copy = {
  en: { title: 'CLI Registry', intro: 'Register approved binaries and exact operations, then connect them to projects.', add: 'Add CLI', edit: 'Edit CLI', name: 'Name', slug: 'CLI ID', binary: 'Binary', description: 'Description', operations: 'Allowed operations (one per line)', enabled: 'Enabled', disabled: 'Disabled', save: 'Save', cancel: 'Cancel', remove: 'Delete', empty: 'No CLI registered yet.', confirm: 'Delete this CLI and disconnect it from all projects?', timeout: 'Timeout (seconds)', limit: 'Maximum output (bytes)', args: 'Approved argument sets (JSON, optional)', argsHint: 'Extra arguments are denied by default. Example: {"log":[["-n","5"]]}. Each array is a complete allowed set.', project: 'Project CLI', projectIntro: 'Choose which active CLI this project can use.', noActive: 'No active CLI. Add or enable a CLI in CLI Registry.', error: 'Request failed', invalidArgs: 'Argument sets must be an object of operation names and arrays of string arrays.', refresh: 'Refresh' },
  ru: { title: 'Реестр CLI', intro: 'Зарегистрируйте binary и разрешённые операции, затем подключите CLI к проектам.', add: 'Добавить CLI', edit: 'Редактировать CLI', name: 'Название', slug: 'ID CLI', binary: 'Binary', description: 'Описание', operations: 'Разрешённые операции (по одной на строку)', enabled: 'Включён', disabled: 'Отключён', save: 'Сохранить', cancel: 'Отмена', remove: 'Удалить', empty: 'CLI пока не зарегистрированы.', confirm: 'Удалить этот CLI и отключить его от всех проектов?', timeout: 'Timeout (секунды)', limit: 'Максимальный вывод (байты)', args: 'Разрешённые наборы аргументов (JSON, необязательно)', argsHint: 'Дополнительные аргументы по умолчанию запрещены. Пример: {"log":[["-n","5"]]}. Каждый массив — полный разрешённый набор.', project: 'CLI проекта', projectIntro: 'Выберите активные CLI, доступные этому проекту.', noActive: 'Нет активных CLI. Добавьте или включите CLI на странице реестра.', error: 'Ошибка запроса', invalidArgs: 'Наборы аргументов должны быть объектом: название операции и массивы строковых массивов.', refresh: 'Обновить' },
};
export function cliRequest<T>(url: string, init?: RequestInit) {
  return apiRequest<T>(url, status => `CLI: HTTP ${status}`, init);
}
export function CLIView({ language = 'en' }: { language?: 'ru' | 'en' }) {
  const t = copy[language];
  const [items, setItems] = useState<CLIEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<CLIEntry>(emptyCLI);
  const [operations, setOperations] = useState('');
  const [argumentSets, setArgumentSets] = useState('{}');
  async function load() {
    setLoading(true);
    try { const data = await cliRequest<{ items: CLIEntry[] }>('/api/cli'); setItems(data.items); setError(null); }
    catch (e) { setError(e instanceof Error ? e.message : t.error); }
    finally { setLoading(false); }
  }
  useEffect(() => { void load(); }, []);
  function edit(item: CLIEntry) {
    setDraft({ ...item }); setOperations(item.operations.map(op => op.command).join('\n'));
    setArgumentSets(JSON.stringify(Object.fromEntries(item.operations.filter(op => op.argument_sets?.length).map(op => [op.command, op.argument_sets])), null, 2));
    setError(null); setOpen(true);
  }
  async function save(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(null);
    try {
      const sets = JSON.parse(argumentSets || '{}');
      const commands = operations.split('\n').map(s => s.trim().split(/\s+/).join(' ')).filter(Boolean);
      if (!sets || typeof sets !== 'object' || Array.isArray(sets) || Object.entries(sets).some(([key, values]) => !commands.includes(key) || !Array.isArray(values) || values.some(v => !Array.isArray(v) || v.some(a => typeof a !== 'string')))) throw new Error(t.invalidArgs);
      const payload = { ...cliPayload(draft), operations: commands.map(command => ({ command, argument_sets: sets[command] || [] })) };
      await cliRequest(draft.id ? `/api/cli/${draft.id}` : '/api/cli', { method: draft.id ? 'PUT' : 'POST', body: JSON.stringify(payload) });
      setOpen(false); await load();
    } catch (e) { setError(e instanceof Error ? e.message : t.error); }
    finally { setBusy(false); }
  }
  async function mutate(item: CLIEntry, remove: boolean) {
    if (remove && !window.confirm(t.confirm)) return;
    setBusy(true); setError(null);
    try { await cliRequest(`/api/cli/${item.id}`, { method: remove ? 'DELETE' : 'PUT', ...(remove ? {} : { body: JSON.stringify({ ...cliPayload(item), is_enabled: !item.is_enabled }) }) }); await load(); }
    catch (e) { setError(e instanceof Error ? e.message : t.error); }
    finally { setBusy(false); }
  }
  const errorView = error ? <p role="alert" className="rounded-xl bg-destructive/10 p-3 text-sm text-destructive">{error}</p> : null;
  return <div className="space-y-6">
    <div className="flex items-center justify-between gap-4"><div><h1 className="flex items-center gap-3 text-2xl font-semibold"><Terminal className="text-electric-blue" />{t.title}</h1><p className="mt-2 text-muted-foreground">{t.intro}</p></div><Button onClick={() => edit(emptyCLI())}><Plus className="mr-2 h-4 w-4" />{t.add}</Button></div>
    <CLICatalogPanel language={language} onImported={load} />
    {!open && errorView}
    {loading ? <LoaderCircle className="animate-spin" /> : !items.length ? <p className="rounded-2xl border border-dashed border-border p-8 text-muted-foreground">{t.empty}</p> : <div className="space-y-3">{items.map(item => <div key={item.id} className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-border bg-card p-5">
      <div className="min-w-0"><h2 className="font-semibold">{item.name} <span className="text-xs text-muted-foreground">{item.slug}</span></h2><p className="text-sm text-muted-foreground">{item.description}</p><code className="text-xs">{item.binary}</code>{item.binary_available === false && <span className="ml-2 text-xs text-muted-foreground">{language === 'ru' ? 'Программа не найдена' : 'Binary not found'}</span>}<p className="mt-2 text-sm">{item.operations.map(op => op.command).join(' · ')}</p></div>
      <div className="flex items-center gap-2"><Button variant="outline" disabled={busy} onClick={() => void mutate(item, false)}>{item.is_enabled ? t.enabled : t.disabled}</Button><Button variant="outline" aria-label={t.edit} disabled={busy} onClick={() => edit(item)}><Pencil className="h-4 w-4" /></Button><Button variant="outline" aria-label={t.remove} disabled={busy} onClick={() => void mutate(item, true)}><Trash2 className="h-4 w-4" /></Button></div>
    </div>)}</div>}
    <Dialog open={open} onOpenChange={value => { if (!busy) setOpen(value); }}><DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl"><DialogHeader><DialogTitle>{draft.id ? t.edit : t.add}</DialogTitle><DialogDescription>{t.intro}</DialogDescription></DialogHeader>
      <form onSubmit={save} className="space-y-4">{errorView}
        {(['name', 'slug', 'binary', 'description'] as const).map(key => <label key={key} className="block space-y-1 text-sm"><span>{t[key]}</span><Input value={draft[key]} required={key !== 'description'} onChange={e => setDraft({ ...draft, [key]: e.target.value })} /></label>)}
        <label className="block space-y-1 text-sm"><span>{t.operations}</span><Textarea required rows={5} value={operations} onChange={e => setOperations(e.target.value)} placeholder={'status\ndiff\nlog\nmcp list'} /></label>
        <label className="block space-y-1 text-sm"><span>{t.args}</span><Textarea rows={3} value={argumentSets} onChange={e => setArgumentSets(e.target.value)} /><p className="text-xs text-muted-foreground">{t.argsHint}</p></label>
        <div className="grid grid-cols-2 gap-3"><label className="space-y-1 text-sm">{t.timeout}<Input type="number" min={1} max={300} value={draft.timeout_seconds} onChange={e => setDraft({ ...draft, timeout_seconds: Number(e.target.value) })} /></label><label className="space-y-1 text-sm">{t.limit}<Input type="number" min={1024} max={4194304} value={draft.max_output_bytes} onChange={e => setDraft({ ...draft, max_output_bytes: Number(e.target.value) })} /></label></div>
        <label className="flex items-center gap-2 text-sm"><Checkbox checked={draft.is_enabled} onCheckedChange={value => setDraft({ ...draft, is_enabled: value === true })} />{t.enabled}</label>
        <DialogFooter><Button type="button" variant="outline" disabled={busy} onClick={() => setOpen(false)}>{t.cancel}</Button><Button disabled={busy}>{busy && <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />}{t.save}</Button></DialogFooter>
      </form></DialogContent></Dialog>
  </div>;
}

export function ProjectCLIPanel({ projectID, language = 'en', onChanged }: { projectID: number; language?: 'ru' | 'en'; onChanged?: () => void | Promise<void> }) {
  const t = copy[language];
  const [items, setItems] = useState<CLIEntry[]>([]);
  const [connected, setConnected] = useState<number[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  async function load() {
    setLoading(true);
    try {
      const [all, links] = await Promise.all([cliRequest<{ items: CLIEntry[] }>('/api/cli'), cliRequest<{ items: CLIEntry[] }>(`/api/projects/${projectID}/cli`)]);
      setItems(all.items.filter(item => item.is_enabled)); setConnected(links.items.map(item => item.id)); setError(null);
    } catch (e) { setError(e instanceof Error ? e.message : t.error); }
    finally { setLoading(false); }
  }
  useEffect(() => { void load(); }, [projectID]);
  async function toggle(item: CLIEntry, enabled: boolean) {
    setBusy(true); setError(null);
    try {
      await cliRequest(`/api/projects/${projectID}/cli/${item.id}`, { method: enabled ? 'PUT' : 'DELETE' });
      setConnected(ids => enabled ? [...ids, item.id] : ids.filter(id => id !== item.id));
      await onChanged?.();
    } catch (e) { setError(e instanceof Error ? e.message : t.error); }
    finally { setBusy(false); }
  }
  return <div className="space-y-4 rounded-2xl border border-border bg-card p-6"><div className="flex items-center justify-between"><div><h2 className="text-lg font-semibold">{t.project}</h2><p className="text-sm text-muted-foreground">{t.projectIntro}</p></div><Button variant="outline" disabled={busy || loading} onClick={() => void load()}>{t.refresh}</Button></div>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    {loading ? <LoaderCircle className="animate-spin" /> : !items.length ? <p className="text-muted-foreground">{t.noActive}</p> : items.map(item => <label key={item.id} className="flex items-center gap-3 rounded-xl border border-border p-4"><Checkbox disabled={busy} checked={connected.includes(item.id)} onCheckedChange={value => void toggle(item, value === true)} /><span><span className="font-medium">{item.name}</span><span className="ml-2 text-xs text-muted-foreground">{item.slug}</span><span className="block text-sm text-muted-foreground">{item.description}</span></span></label>)}
  </div>;
}


type CatalogEntry = Omit<CLIEntry, 'id' | 'is_enabled'> & { platforms: string[] };
type CatalogItem = { entry: CatalogEntry; status: string; current?: CLIEntry; binary_path?: string; revision: string };
type CatalogResult = { url: string; source: string; version: string; warning?: string; items: CatalogItem[]; added: string[] };
function CLICatalogPanel({ language, onImported }: { language: 'ru' | 'en'; onImported: () => Promise<void> }) {
  const ru = language === 'ru';
  const [catalog, setCatalog] = useState<CatalogResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [review, setReview] = useState<CatalogItem | null>(null);
  const [added, setAdded] = useState<string[]>([]);
  const statuses: Record<string, string> = ru
    ? { available: 'Найдена программа — можно добавить', installed: 'Уже добавлен', update_available: 'Есть отличия — доступен просмотр', dismissed: 'Ранее удалён', binary_missing: 'Программа не найдена', unsupported_os: 'Не поддерживается на этой ОС' }
    : { available: 'Binary found — ready to add', installed: 'Registered', update_available: 'Changes available for review', dismissed: 'Previously removed', binary_missing: 'Binary not found', unsupported_os: 'Unsupported OS' };
  async function refresh(sync = false) {
    setBusy(true); setError(null);
    try {
      const result = await cliRequest<CatalogResult>(sync ? '/api/cli/catalog/sync' : '/api/cli/catalog', sync ? { method: 'POST' } : undefined);
      setCatalog(result); setAdded(result.added);
      if (sync) await onImported();
    } catch (e) { setError(e instanceof Error ? e.message : 'CLI catalog error'); }
    finally { setBusy(false); }
  }
  useEffect(() => { void refresh(); }, []);
  async function apply() {
    if (!review) return;
    setBusy(true); setError(null);
    try {
      await cliRequest(`/api/cli/catalog/${encodeURIComponent(review.entry.slug)}/apply`, { method: 'POST', body: JSON.stringify({ revision: review.revision }) });
      setReview(null); await onImported(); await refresh();
    } catch (e) { setError(e instanceof Error ? e.message : 'CLI catalog error'); }
    finally { setBusy(false); }
  }
  const proposed = review ? cliPayload({ ...review.entry, id: review.current?.id || 0, is_enabled: review.current?.is_enabled ?? true }) : null;
  return <section className="space-y-3 rounded-2xl border border-border bg-card p-5">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="font-semibold">{ru ? 'Каталог готовых CLI' : 'CLI starter catalog'}</h2><p className="text-sm text-muted-foreground">{ru ? 'Добавляются только программы, найденные на сервере. Подключение к проектам остаётся ручным.' : 'Only binaries found on the server are added. Project connections remain manual.'}</p></div><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{busy && <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />}{ru ? 'Проверить и добавить найденные' : 'Check and add found binaries'}</Button></div>
    {catalog && <p className="text-xs text-muted-foreground">{catalog.url} · {catalog.version} · {catalog.source === 'bundled' ? (ru ? 'Встроенный каталог: удалённый недоступен' : 'Bundled catalog: remote unavailable') : (ru ? 'Каталог с сервера' : 'Remote catalog')}</p>}
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    {added.length > 0 && <p role="status" className="text-sm">{ru ? 'Добавлены: ' : 'Added: '}{added.join(', ')}</p>}
    <details><summary className="cursor-pointer text-sm">{ru ? 'Доступность и обновления' : 'Availability and updates'}</summary><div className="mt-3 space-y-2">{catalog?.items.map(item => <div key={item.entry.slug} className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border p-3"><div><span className="font-medium">{item.entry.name}</span><p className="text-xs text-muted-foreground">{statuses[item.status] || item.status}</p>{item.binary_path && <code className="text-xs text-muted-foreground">{item.binary_path}</code>}</div>{['available', 'update_available', 'dismissed'].includes(item.status) && <Button variant="outline" disabled={busy} onClick={() => { setError(null); setReview(item); }}>{ru ? 'Посмотреть' : 'Review'}</Button>}</div>)}</div></details>
    <Dialog open={!!review} onOpenChange={open => { if (!open && !busy) setReview(null); }}><DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl"><DialogHeader><DialogTitle>{ru ? 'Изменения CLI' : 'CLI changes'}: {review?.entry.name}</DialogTitle><DialogDescription>{ru ? 'Применение заменит эту конфигурацию. Для подключённых проектов новые операции станут доступны ИИ. Состояние включения и подключения сохранятся.' : 'Applying replaces this configuration. New operations become available to AI in connected projects. Enabled state and project links are preserved.'}</DialogDescription></DialogHeader>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <div className="grid gap-4 sm:grid-cols-2"><div><h3 className="mb-2 font-medium">{ru ? 'Сейчас' : 'Current'}</h3><pre className="max-h-80 overflow-auto rounded-xl bg-muted p-3 text-xs">{review?.current ? JSON.stringify(cliPayload(review.current), null, 2) : (ru ? 'Нет записи' : 'Not registered')}</pre></div><div><h3 className="mb-2 font-medium">{ru ? 'После применения' : 'Proposed'}</h3><pre className="max-h-80 overflow-auto rounded-xl bg-muted p-3 text-xs">{JSON.stringify(proposed, null, 2)}</pre></div></div>
      <DialogFooter><Button variant="outline" disabled={busy} onClick={() => setReview(null)}>{ru ? 'Отмена' : 'Cancel'}</Button><Button disabled={busy} onClick={() => void apply()}>{ru ? 'Применить изменения' : 'Apply changes'}</Button></DialogFooter>
    </DialogContent></Dialog>
  </section>;
}
