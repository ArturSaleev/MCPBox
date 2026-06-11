import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Boxes, CheckCircle2, LoaderCircle, Plus, RefreshCw, Search, ShoppingBag, Trash2 } from 'lucide-react';

import type { Dictionary } from '../i18n';
import {
  applyProjectDefaultsToInstallValues,
  catalogConfigFields,
  catalogEnvFields,
  type CatalogItem,
  type CatalogSettings,
  type InstalledPackage,
  normalizeEnvConfig,
  normalizeInstallConfig,
  type ProjectOption,
} from '../market';
import { ImageWithFallback } from './figma/ImageWithFallback';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Input } from './ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';

type SelectedProject = {
  project_id: number;
  installed_integrations: Array<{ catalog_item_id: string }>;
};

type MarketViewProps = {
  labels: Dictionary['labels'];
  messages: Dictionary['messages'];
  language: string;
  languageOptions: Array<{ value: string; label: string }>;
  onLanguageChange: (value: string) => void;
  projects: ProjectOption[];
  selectedProject: SelectedProject | null;
  catalogItems: CatalogItem[];
  catalogSettings: CatalogSettings | null;
  installedPackages: InstalledPackage[];
  catalogURL: string;
  setCatalogURL: (value: string) => void;
  catalogSourceMode: 'server' | 'file';
  setCatalogSourceMode: (value: 'server' | 'file') => void;
  localCatalogFileName: string;
  onPickLocalCatalogFile: (file: File | null) => void;
  catalogLoading: boolean;
  catalogSyncing: boolean;
  catalogURLVisible: boolean;
  installingCatalogItemId: string | null;
  addingCatalogItemId: string | null;
  uninstallingCatalogItemId: string | null;
  onSyncCatalog: () => Promise<void>;
  onInstallCatalogPackage: (item: CatalogItem) => Promise<boolean>;
  onUninstallCatalogPackage: (item: CatalogItem, pkg: InstalledPackage) => Promise<boolean>;
  onPerformCatalogInstall: (
    item: CatalogItem,
    projectId: number,
    config: Record<string, unknown>,
  ) => Promise<boolean>;
  onActionError: (message: string | null) => void;
};

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');
}

export function MarketView({
  labels,
  messages,
  language,
  languageOptions,
  onLanguageChange,
  projects,
  selectedProject,
  catalogItems,
  catalogSettings,
  installedPackages,
  catalogURL,
  setCatalogURL,
  catalogSourceMode,
  setCatalogSourceMode,
  localCatalogFileName,
  onPickLocalCatalogFile,
  catalogLoading,
  catalogSyncing,
  catalogURLVisible,
  installingCatalogItemId,
  addingCatalogItemId,
  uninstallingCatalogItemId,
  onSyncCatalog,
  onInstallCatalogPackage,
  onUninstallCatalogPackage,
  onPerformCatalogInstall,
  onActionError,
}: MarketViewProps) {
  const [selectedCatalogCategory, setSelectedCatalogCategory] = useState('all');
  const [catalogSearchQuery, setCatalogSearchQuery] = useState('');
  const [installDialogOpen, setInstallDialogOpen] = useState(false);
  const [installDialogItem, setInstallDialogItem] = useState<CatalogItem | null>(null);
  const [installDialogProjectId, setInstallDialogProjectId] = useState<number | null>(null);
  const [installDialogValues, setInstallDialogValues] = useState<Record<string, string>>({});
  const [installDialogEnvValues, setInstallDialogEnvValues] = useState<Record<string, string>>({});

  const installedCatalogIDs = useMemo(
    () => new Set((selectedProject?.installed_integrations ?? []).map((integration) => integration.catalog_item_id)),
    [selectedProject],
  );
  const installedPackageCatalogIDs = useMemo(
    () =>
      new Set(
        installedPackages
          .filter((pkg) => pkg.status === 'installed')
          .map((pkg) => pkg.catalog_item_id),
      ),
    [installedPackages],
  );

  const catalogCategories = useMemo(
    () => ['all', ...Array.from(new Set(catalogItems.map((item) => item.category || labels.generalCategory))).sort((left, right) => left.localeCompare(right))],
    [catalogItems, labels.generalCategory],
  );

  const filteredCatalogItems = useMemo(() => {
    const normalizedQuery = catalogSearchQuery.trim().toLowerCase();
    return catalogItems.filter((item) => {
      const matchesCategory =
        selectedCatalogCategory === 'all' ||
        (item.category || labels.generalCategory) === selectedCatalogCategory;
      if (!matchesCategory) {
        return false;
      }
      if (!normalizedQuery) {
        return true;
      }

      const haystack = [
        item.name,
        item.description,
        item.category,
        item.transport,
        item.auth_type,
        item.runtime.type,
        item.runtime.version,
        item.source.type,
        item.source.package,
        item.source.version,
        ...(item.tags ?? []),
        ...(item.capabilities ?? []),
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();

      return haystack.includes(normalizedQuery);
    });
  }, [catalogItems, catalogSearchQuery, labels.generalCategory, selectedCatalogCategory]);

  const installDialogFields = installDialogItem ? catalogConfigFields(installDialogItem) : [];
  const installDialogEnvFields = installDialogItem ? catalogEnvFields(installDialogItem) : [];
  const installDialogProject =
    projects.find((project) => project.project_id === installDialogProjectId) ?? null;

  useEffect(() => {
    if (projects.length === 0) {
      setInstallDialogProjectId(null);
      return;
    }

    if (
      installDialogProjectId &&
      projects.some((project) => project.project_id === installDialogProjectId)
    ) {
      return;
    }

    setInstallDialogProjectId(selectedProject?.project_id ?? projects[0].project_id);
  }, [projects, selectedProject, installDialogProjectId]);

  function openInstallDialog(item: CatalogItem, preferredProjectId?: number | null) {
    const fields = catalogConfigFields(item);
    const baseValues = Object.fromEntries(fields.map((field) => [field.key, field.defaultValue]));
    const envFields = catalogEnvFields(item);
    const baseEnvValues = Object.fromEntries(envFields.map((field) => [field.key, field.defaultValue]));
    const resolvedProjectId =
      preferredProjectId && projects.some((project) => project.project_id === preferredProjectId)
        ? preferredProjectId
        : selectedProject?.project_id && projects.some((project) => project.project_id === selectedProject.project_id)
          ? selectedProject.project_id
          : projects[0]?.project_id ?? null;
    const project = projects.find((entry) => entry.project_id === resolvedProjectId) ?? null;

    setInstallDialogItem(item);
    setInstallDialogProjectId(resolvedProjectId);
    setInstallDialogValues(applyProjectDefaultsToInstallValues(baseValues, fields, project));
    setInstallDialogEnvValues(baseEnvValues);
    setInstallDialogOpen(true);
  }

  async function installCatalogItem(item: CatalogItem, preferredProjectId?: number | null) {
    const targetProjectId =
      preferredProjectId && projects.some((project) => project.project_id === preferredProjectId)
        ? preferredProjectId
        : selectedProject?.project_id && projects.some((project) => project.project_id === selectedProject.project_id)
          ? selectedProject.project_id
          : projects[0]?.project_id ?? null;
    if (!targetProjectId) {
      onActionError(messages.selectProjectBeforeInstall);
      return;
    }

    const packageInstalled = installedPackageCatalogIDs.has(item.id);
    if (!packageInstalled) {
      const ok = await onInstallCatalogPackage(item);
      if (!ok) {
        return;
      }
    }

    const fields = catalogConfigFields(item);
    const envFields = catalogEnvFields(item);
    if (fields.length > 0 || envFields.length > 0) {
      openInstallDialog(item, targetProjectId);
      return;
    }

    await onPerformCatalogInstall(item, targetProjectId, {});
  }

  function renderCatalogIcon(item: CatalogItem) {
    if (item.icon_url) {
      return (
        <ImageWithFallback
          src={item.icon_url}
          alt={item.name}
          className="h-7 w-7 object-contain"
        />
      );
    }

    return initials(item.name) || <Boxes className="h-5 w-5" />;
  }

  return (
    <section className="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)] xl:items-start">
      <aside className="rounded-[28px] border border-border bg-card p-4 xl:sticky xl:top-8 xl:h-[calc(100vh-4rem)] xl:overflow-hidden">
        <div className="flex h-full flex-col gap-4">
          <div className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-background px-4 py-3">
            <div>
              <div className="text-sm font-semibold text-foreground">{labels.appTitle}</div>
            </div>
            <div className="w-[140px]">
              <Select value={language} onValueChange={onLanguageChange}>
                <SelectTrigger className="h-10 rounded-xl border-border bg-card text-xs" aria-label={labels.language}>
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
            </div>
          </div>

          {/*<div className="rounded-2xl border border-electric-blue/20 bg-[linear-gradient(180deg,rgba(14,165,233,0.14),rgba(14,165,233,0.03))] p-4">*/}
          {/*  <div className="flex items-center gap-3">*/}
          {/*    <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-electric-blue text-white shadow-[0_10px_30px_rgba(14,165,233,0.3)]">*/}
          {/*      <ShoppingBag className="h-5 w-5" />*/}
          {/*    </div>*/}
          {/*    <div>*/}
          {/*      <div className="text-sm font-medium text-electric-blue">{labels.catalog}</div>*/}
          {/*      <div className="mt-1 text-lg font-semibold">{labels.integrations}</div>*/}
          {/*    </div>*/}
          {/*  </div>*/}
          {/*  <div className="mt-4 grid grid-cols-2 gap-3">*/}
          {/*    <div className="rounded-xl border border-electric-blue/15 bg-background/95 px-3 py-2 shadow-sm">*/}
          {/*      <div className="text-xs text-muted-foreground">{labels.catalogItems}</div>*/}
          {/*      <div className="mt-1 text-xl font-semibold text-foreground">{catalogItems.length}</div>*/}
          {/*    </div>*/}
          {/*    <div className="rounded-xl border border-electric-blue/15 bg-background/95 px-3 py-2 shadow-sm">*/}
          {/*      <div className="text-xs text-muted-foreground">{labels.installed}</div>*/}
          {/*      <div className="mt-1 text-xl font-semibold text-foreground">{selectedProject?.installed_integrations.length ?? 0}</div>*/}
          {/*    </div>*/}
          {/*  </div>*/}
          {/*</div>*/}

          <div className="space-y-3">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={catalogSearchQuery}
                onChange={(event) => setCatalogSearchQuery(event.target.value)}
                className="h-11 rounded-xl pl-9"
                placeholder={messages.searchCatalogPlaceholder}
              />
            </div>
            <div className="rounded-xl border border-border bg-background px-3 py-2 text-xs text-muted-foreground">
              {messages.catalogResultsSummary(filteredCatalogItems.length, catalogItems.length)}
            </div>
          </div>

          <div className="space-y-2 overflow-y-auto pr-1 xl:flex-1">
            {catalogCategories.map((category) => {
              const count =
                category === 'all'
                  ? catalogItems.length
                  : catalogItems.filter((item) => (item.category || labels.generalCategory) === category).length;
              return (
                <button
                  key={`catalog-category-${category}`}
                  onClick={() => setSelectedCatalogCategory(category)}
                  className={`flex w-full items-center justify-between rounded-xl border px-3 py-2.5 text-left text-sm font-medium transition-colors ${
                    selectedCatalogCategory === category
                      ? 'border-electric-blue bg-electric-blue text-white'
                      : 'border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground'
                  }`}
                >
                  <span>{category === 'all' ? labels.allCategories : category}</span>
                  <span className={`rounded-full px-2 py-0.5 text-xs ${selectedCatalogCategory === category ? 'bg-white/20 text-white' : 'bg-muted text-muted-foreground'}`}>
                    {count}
                  </span>
                </button>
              );
            })}
          </div>

          <div className="mt-auto space-y-3 pt-2">
            <button
              onClick={() => void onSyncCatalog()}
              disabled={catalogSyncing}
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
            >
              {catalogSyncing ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
              {labels.syncCatalog}
            </button>
            <div className="rounded-xl border border-border bg-background px-4 py-3">
              <div className="text-xs text-muted-foreground">{labels.lastSync}</div>
              <div className="mt-1 text-sm font-medium">
                {catalogSettings?.last_sync_at ? new Date(catalogSettings.last_sync_at).toLocaleString() : messages.notSynced}
              </div>
            </div>
          </div>
        </div>
      </aside>

      <div className="space-y-6">
        <div className="overflow-hidden rounded-[28px] border border-border bg-card">
          <div className="bg-[radial-gradient(circle_at_top_left,rgba(14,165,233,0.18),transparent_36%),linear-gradient(180deg,rgba(255,255,255,0.04),transparent)] p-6">
            <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
              <div className="max-w-3xl">
                <p className="text-sm font-medium text-electric-blue">{labels.integrations}</p>
                <h2 className="mt-2 text-3xl font-semibold">{labels.market} / {labels.catalog}</h2>
                <p className="mt-3 text-muted-foreground">{messages.marketDescription}</p>
              </div>

              <div className="grid gap-3 sm:grid-cols-3">
                <div className="rounded-2xl border border-border bg-background/90 px-4 py-3 backdrop-blur">
                  <div className="text-sm text-muted-foreground">{labels.catalogItems}</div>
                  <div className="mt-1 text-2xl font-semibold">{catalogItems.length}</div>
                </div>
                <div className="rounded-2xl border border-border bg-background/90 px-4 py-3 backdrop-blur">
                  <div className="text-sm text-muted-foreground">{labels.installed}</div>
                  <div className="mt-1 text-2xl font-semibold">{selectedProject?.installed_integrations.length ?? 0}</div>
                </div>
                <div className="rounded-2xl border border-border bg-background/90 px-4 py-3 backdrop-blur">
                  <div className="text-sm text-muted-foreground">{labels.lastSync}</div>
                  <div className="mt-1 text-sm font-medium">
                    {catalogSettings?.last_sync_at ? new Date(catalogSettings.last_sync_at).toLocaleString() : messages.notSynced}
                  </div>
                </div>
              </div>
            </div>

            {catalogURLVisible ? (
              <div className="mt-5 space-y-4">
                <div>
                  <span className="text-sm text-muted-foreground">{labels.manifestSource}</span>
                  <div className="mt-2 inline-flex rounded-xl border border-border bg-background p-1">
                    <button
                      type="button"
                      onClick={() => setCatalogSourceMode('server')}
                      className={`rounded-lg px-3 py-2 text-sm transition-colors ${catalogSourceMode === 'server' ? 'bg-electric-blue text-white' : 'text-muted-foreground hover:bg-accent hover:text-foreground'}`}
                    >
                      {labels.serverSource}
                    </button>
                    <button
                      type="button"
                      onClick={() => setCatalogSourceMode('file')}
                      className={`rounded-lg px-3 py-2 text-sm transition-colors ${catalogSourceMode === 'file' ? 'bg-electric-blue text-white' : 'text-muted-foreground hover:bg-accent hover:text-foreground'}`}
                    >
                      {labels.localFileSource}
                    </button>
                  </div>
                </div>

                {catalogSourceMode === 'server' ? (
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{labels.externalManifestUrl}</span>
                    <Input
                      value={catalogURL}
                      onChange={(event) => setCatalogURL(event.target.value)}
                      className="h-11 rounded-xl"
                      placeholder="https://mcpbox.sh/catalog.json"
                    />
                  </label>
                ) : (
                  <div className="space-y-2">
                    <span className="text-sm text-muted-foreground">{labels.localFileSource}</span>
                    <label className="inline-flex h-11 cursor-pointer items-center justify-center gap-2 rounded-xl border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent">
                      {labels.chooseFile}
                      <input
                        type="file"
                        accept="application/json,.json"
                        className="hidden"
                        onClick={(event) => {
                          event.currentTarget.value = '';
                        }}
                        onChange={(event) => onPickLocalCatalogFile(event.target.files?.[0] ?? null)}
                      />
                    </label>
                    {localCatalogFileName ? (
                      <div className="text-xs text-muted-foreground">{messages.localCatalogFileSelected(localCatalogFileName)}</div>
                    ) : null}
                  </div>
                )}
              </div>
            ) : null}

            {catalogURLVisible ? (
              <div className="mt-3 text-xs text-muted-foreground">{messages.advancedModeEnabled}</div>
            ) : null}

            {catalogSettings?.last_sync_status === 'failed' && catalogSettings.last_sync_error ? (
              <div className="mt-4 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                {catalogSettings.last_sync_error}
              </div>
            ) : null}
          </div>
        </div>

        {!selectedProject ? (
          <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
            {messages.selectProjectBeforeInstall}
          </div>
        ) : null}

        <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
          {filteredCatalogItems.map((item) => {
            const packageInstalling = installingCatalogItemId === item.id;
            const addingToProject = addingCatalogItemId === item.id;
            const packageInstalled = installedPackageCatalogIDs.has(item.id);
            const addedToProject = installedCatalogIDs.has(item.id);
            const packageInfo = installedPackages.find((pkg) => pkg.catalog_item_id === item.id && pkg.status === 'installed') ?? null;
            const packageInUse = (packageInfo?.project_use_count ?? 0) > 0;
            const transportLabel = item.transport === 'stdio' ? 'STDIO' : item.transport;
            const primaryInfoLabel = item.transport === 'stdio' ? labels.command : labels.endpoint;
            const primaryInfoValue = item.transport === 'stdio'
              ? [item.command, ...(item.args ?? [])].filter(Boolean).join(' ')
              : item.mcp_url || messages.noValue;
            const authLabel = item.auth_type === 'mcp_discovery' ? labels.mcpDiscovery : item.auth_type;
            const sourceValue = item.source.package || item.source.url || item.source.type || messages.noValue;
            const runtimeValue = [item.runtime.type, item.runtime.version].filter(Boolean).join(' ').trim() || messages.noValue;

            return (
              <div key={item.id} className="group rounded-[26px] border border-border bg-card p-5 shadow-[0_12px_32px_rgba(2,6,23,0.04)] transition-transform hover:-translate-y-0.5 hover:shadow-[0_18px_42px_rgba(2,6,23,0.08)]">
                <div className="flex items-start gap-4">
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,rgba(14,165,233,0.18),rgba(16,185,129,0.16))] text-sm font-semibold text-foreground">
                    {renderCatalogIcon(item)}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="min-w-0">
                        <h3 className="truncate text-lg font-semibold">{item.name}</h3>
                        <div className="mt-1 text-sm text-electric-blue">{item.category || labels.generalCategory}</div>
                      </div>
                      {item.enabled ? (
                        <div className="rounded-full border border-status-running/30 bg-status-running/12 px-2.5 py-1 text-[11px] font-medium text-status-running">
                          {labels.connected}
                        </div>
                      ) : (
                        <div className="rounded-full border border-border bg-muted px-2.5 py-1 text-[11px] font-medium text-muted-foreground">
                          {labels.disabled}
                        </div>
                      )}
                    </div>

                    <div className="mt-3 flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">{transportLabel}</span>
                      <span className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground">{authLabel}</span>
                      <span className={`rounded-full border px-2 py-1 text-xs ${packageInstalled ? 'border-status-running/30 bg-status-running/12 text-status-running' : 'border-border bg-muted text-muted-foreground'}`}>
                        {packageInstalled ? labels.packageInstalled : labels.packageNotInstalled}
                      </span>
                      {selectedProject ? (
                        <span className={`rounded-full border px-2 py-1 text-xs ${addedToProject ? 'border-electric-blue/30 bg-electric-blue/12 text-electric-blue' : 'border-border bg-muted text-muted-foreground'}`}>
                          {addedToProject ? labels.addedToProject : labels.notInProject}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>

                <p className="mt-4 text-sm leading-6 text-muted-foreground">
                  {item.description || messages.noDescriptionProvided}
                </p>

                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-border bg-background p-3">
                    <div className="text-xs text-muted-foreground">{labels.runtime}</div>
                    <div className="mt-2 text-sm font-medium">{runtimeValue}</div>
                  </div>
                  <div className="rounded-2xl border border-border bg-background p-3">
                    <div className="text-xs text-muted-foreground">{labels.source}</div>
                    <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">{sourceValue}</code>
                  </div>
                  <div className="rounded-2xl border border-border bg-background p-3">
                    <div className="text-xs text-muted-foreground">{labels.installModel}</div>
                    <div className="mt-2 text-sm font-medium">
                      {item.shared_install ? messages.sharedInstallMode : messages.projectInstallMode}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {item.supports_multi_project ? messages.multiProjectSupported : messages.singleProjectOnly}
                    </div>
                  </div>
                  <div className="rounded-2xl border border-border bg-background p-3">
                    <div className="text-xs text-muted-foreground">{primaryInfoLabel}</div>
                    <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">{primaryInfoValue || messages.noValue}</code>
                    {packageInfo ? (
                      <div className="mt-2 text-xs text-muted-foreground">
                        {messages.packageUsageCount(packageInfo.project_use_count)}
                      </div>
                    ) : null}
                  </div>
                </div>

                {item.auth_type === 'mcp_discovery' ? (
                  <div className="mt-4 rounded-2xl border border-electric-blue/20 bg-electric-blue/8 px-4 py-3 text-sm text-muted-foreground">
                    {messages.upstreamAuthNotice}
                  </div>
                ) : null}

                {(item.system_dependencies?.length ?? 0) > 0 ? (
                  <div className="mt-4 rounded-2xl border border-amber-500/25 bg-amber-500/8 px-4 py-3">
                    <div className="text-xs font-medium text-amber-700 dark:text-amber-300">{labels.systemDependencies}</div>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {item.system_dependencies?.map((dependency) => (
                        <span
                          key={`${item.id}-dependency-${dependency.executable}`}
                          className="rounded-full border border-amber-500/25 bg-background px-2.5 py-1 text-xs text-foreground"
                        >
                          {dependency.min_version
                            ? messages.systemDependencyVersion(dependency.executable, dependency.min_version)
                            : messages.systemDependencyRequired(dependency.executable)}
                        </span>
                      ))}
                    </div>
                  </div>
                ) : null}

                {(item.tags?.length ?? 0) > 0 ? (
                  <div className="mt-4 flex flex-wrap gap-2">
                    {item.tags.map((tag) => (
                      <span
                        key={`${item.id}-${tag}`}
                        className="rounded-full border border-electric-blue/30 bg-electric-blue/12 px-2 py-1 text-xs font-medium text-electric-blue"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                ) : null}

                <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-3 text-sm">
                    {item.docs_url ? (
                      <a className="text-electric-blue underline-offset-4 hover:underline" href={item.docs_url} target="_blank" rel="noreferrer">
                        {labels.docs}
                      </a>
                    ) : null}
                    {item.website ? (
                      <a className="text-electric-blue underline-offset-4 hover:underline" href={item.website} target="_blank" rel="noreferrer">
                        {labels.website}
                      </a>
                    ) : null}
                  </div>

                  <div className="flex flex-wrap justify-end gap-2">
                    {!packageInstalled ? (
                      <button
                        onClick={() => void onInstallCatalogPackage(item)}
                        disabled={packageInstalling || !item.enabled}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-electric-blue bg-electric-blue/10 px-4 text-sm font-medium text-electric-blue transition-colors hover:bg-electric-blue/20 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {packageInstalling ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                        {labels.installPackage}
                      </button>
                    ) : null}
                    {packageInstalled ? (
                      <button
                        onClick={() => void installCatalogItem(item)}
                        disabled={projects.length === 0 || addingToProject || !item.enabled}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {addingToProject ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                        {labels.addToProject}
                      </button>
                    ) : null}
                    {packageInfo ? (
                      <button
                        onClick={() => void onUninstallCatalogPackage(item, packageInfo)}
                        disabled={uninstallingCatalogItemId === item.id || packageInUse}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-60"
                        title={packageInUse ? messages.packageInUseCannotUninstall : undefined}
                      >
                        {uninstallingCatalogItemId === item.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                        {labels.uninstallPackage}
                      </button>
                    ) : null}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {!catalogLoading && catalogItems.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
            {messages.syncManifestToPopulateCatalog}
          </div>
        ) : null}
        {!catalogLoading && catalogItems.length > 0 && filteredCatalogItems.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center text-muted-foreground">
            {messages.noIntegrationsInCategory}
          </div>
        ) : null}

        <Dialog
          open={installDialogOpen}
          onOpenChange={(open) => {
            setInstallDialogOpen(open);
            if (!open) {
              setInstallDialogItem(null);
              setInstallDialogProjectId(null);
              setInstallDialogValues({});
              setInstallDialogEnvValues({});
            }
          }}
        >
          <DialogContent className="sm:max-w-xl">
            <DialogHeader>
              <DialogTitle>
                {installDialogItem
                  ? messages.addPackageDialogTitle(installDialogItem.name)
                  : messages.addPackageDialogFallbackTitle}
              </DialogTitle>
              <DialogDescription>{messages.addPackageDialogDescription}</DialogDescription>
            </DialogHeader>

            {installDialogItem ? (
              <form
                className="space-y-4"
                onSubmit={(event: FormEvent<HTMLFormElement>) => {
                  event.preventDefault();
                  if (!installDialogProjectId) {
                    onActionError(messages.selectProjectBeforeInstall);
                    return;
                  }
                  const config = normalizeInstallConfig(installDialogFields, installDialogValues);
                  const env = {
                    ...(config.env && typeof config.env === 'object' ? (config.env as Record<string, string>) : {}),
                    ...normalizeEnvConfig(installDialogEnvFields, installDialogEnvValues),
                  };
                  if (Object.keys(env).length > 0) {
                    config.env = env;
                  }
                  void onPerformCatalogInstall(installDialogItem, installDialogProjectId, config).then((success) => {
                    if (!success) {
                      return;
                    }
                    setInstallDialogOpen(false);
                    setInstallDialogItem(null);
                    setInstallDialogProjectId(null);
                    setInstallDialogValues({});
                    setInstallDialogEnvValues({});
                  });
                }}
              >
                <label className="block space-y-2">
                  <span className="text-sm text-muted-foreground">{labels.projects}</span>
                  <Select
                    value={installDialogProjectId ? String(installDialogProjectId) : undefined}
                    onValueChange={(value) => {
                      const nextProjectId = Number(value);
                      const nextProject = projects.find((project) => project.project_id === nextProjectId) ?? null;
                      setInstallDialogProjectId(nextProjectId);
                      setInstallDialogValues((current) =>
                        applyProjectDefaultsToInstallValues(current, installDialogFields, nextProject),
                      );
                    }}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder={labels.notSelected} />
                    </SelectTrigger>
                    <SelectContent>
                      {projects.map((project) => (
                        <SelectItem key={`install-project-${project.project_id}`} value={String(project.project_id)}>
                          {project.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {installDialogProject?.root_path ? (
                    <div className="text-xs text-muted-foreground">{messages.workingDirectoryValue(installDialogProject.root_path)}</div>
                  ) : null}
                </label>

                {installDialogFields.map((field) => (
                  <label key={`install-field-${field.key}`} className="block space-y-2">
                    <span className="text-sm text-muted-foreground">
                      {field.label}
                      {field.required ? ' *' : ''}
                    </span>
                    {field.description ? (
                      <span className="block text-xs text-muted-foreground">{field.description}</span>
                    ) : null}
                    {field.type === 'array' ? (
                      <textarea
                        value={installDialogValues[field.key] ?? ''}
                        onChange={(event) =>
                          setInstallDialogValues((current) => ({
                            ...current,
                            [field.key]: event.target.value,
                          }))
                        }
                        rows={4}
                        className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder={messages.oneValuePerLine}
                        required={field.required}
                      />
                    ) : (
                      <Input
                        type={field.secret ? 'password' : 'text'}
                        value={installDialogValues[field.key] ?? ''}
                        onChange={(event) =>
                          setInstallDialogValues((current) => ({
                            ...current,
                            [field.key]: event.target.value,
                          }))
                        }
                        className="h-10"
                        required={field.required}
                      />
                    )}
                  </label>
                ))}

                {installDialogEnvFields.length > 0 ? (
                  <div className="space-y-4 rounded-xl border border-border bg-background p-4">
                    <div>
                      <div className="text-sm font-medium">{labels.environmentVariables}</div>
                      <div className="mt-1 text-xs text-muted-foreground">{messages.envSchemaDescription}</div>
                    </div>

                    {installDialogEnvFields.map((field) => (
                      <label key={`install-env-field-${field.key}`} className="block space-y-2">
                        <span className="text-sm text-muted-foreground">
                          {field.label}
                          {field.required ? ' *' : ''}
                        </span>
                        <code className="block text-xs text-electric-blue">{field.key}</code>
                        {field.description ? (
                          <span className="block text-xs text-muted-foreground">{field.description}</span>
                        ) : null}
                        <Input
                          type={field.secret ? 'password' : 'text'}
                          value={installDialogEnvValues[field.key] ?? ''}
                          onChange={(event) =>
                            setInstallDialogEnvValues((current) => ({
                              ...current,
                              [field.key]: event.target.value,
                            }))
                          }
                          className="h-10"
                          required={field.required}
                        />
                      </label>
                    ))}
                  </div>
                ) : null}

                <button
                  type="submit"
                  disabled={!installDialogItem || !installDialogProjectId || addingCatalogItemId === installDialogItem.id}
                  className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                >
                  {installDialogItem && addingCatalogItemId === installDialogItem.id ? (
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  {labels.addToProject}
                </button>
              </form>
            ) : null}
          </DialogContent>
        </Dialog>
      </div>
    </section>
  );
}
