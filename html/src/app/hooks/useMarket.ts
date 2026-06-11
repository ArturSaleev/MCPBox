import { FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { apiRequest } from '../utils/api';
import type { CatalogItem, CatalogSettings, InstalledPackage } from '../market';
import type { ProjectStatus } from '../types';

const legacyCatalogSourceURL = 'https://webeasy.kz/mcpbox/catalog.json';
const defaultCatalogSourceURL = 'https://mcpbox.sh/catalog.json';

function normalizeCatalogSourceURL(url: string) {
  const trimmed = url.trim();
  if (trimmed === legacyCatalogSourceURL) {
    return defaultCatalogSourceURL;
  }
  return trimmed;
}

export function useMarket(projects: ProjectStatus[], messages: { requestFailed: string; packageInUseCannotUninstall: string; packageInstalled: string; packageUninstalled: string; catalogSynced: (count: number) => string; localCatalogFileMissing: string }) {
  const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
  const [catalogSettings, setCatalogSettings] = useState<CatalogSettings | null>(null);
  const [installedPackages, setInstalledPackages] = useState<InstalledPackage[]>([]);
  const [catalogURL, setCatalogURL] = useState(defaultCatalogSourceURL);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogSyncing, setCatalogSyncing] = useState(false);
  const [catalogURLVisible, setCatalogURLVisible] = useState(false);
  const [catalogSourceMode, setCatalogSourceMode] = useState<'server' | 'file'>('server');
  const [localCatalogFileName, setLocalCatalogFileName] = useState('');
  const [localCatalogContent, setLocalCatalogContent] = useState('');
  const [installingCatalogItemId, setInstallingCatalogItemId] = useState<string | null>(null);
  const [addingCatalogItemId, setAddingCatalogItemId] = useState<string | null>(null);
  const [uninstallingCatalogItemId, setUninstallingCatalogItemId] = useState<string | null>(null);

  async function loadCatalog(initial = false) {
    if (initial) {
      setCatalogLoading(true);
    }

    try {
      const normalizedURL = normalizeCatalogSourceURL(catalogURL);
      const response = await apiRequest<{ items: CatalogItem[]; settings: CatalogSettings }>(
        '/api/catalog/items',
        () => messages.requestFailed,
      );
      setCatalogItems(response.items);
      setCatalogSettings(response.settings);
      if (initial) {
        setCatalogURL(normalizedURL);
      }
    } catch (loadError) {
      console.error('Failed to load catalog:', loadError);
    } finally {
      setCatalogLoading(false);
    }
  }

  async function loadInstalledPackages() {
    try {
      const response = await apiRequest<{ items: InstalledPackage[] }>('/api/packages', () => messages.requestFailed);
      setInstalledPackages(response.items);
    } catch (loadError) {
      console.error('Failed to load installed packages:', loadError);
    }
  }

  async function syncCatalog() {
    setCatalogSyncing(true);
    try {
      if (catalogSourceMode === 'file' && localCatalogContent.trim() === '') {
        toast.error(messages.localCatalogFileMissing);
        return;
      }
      const normalizedURL = normalizeCatalogSourceURL(catalogURL);
      const response = await apiRequest<{ items: CatalogItem[]; settings: CatalogSettings }>('/api/catalog/sync', () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify(
          catalogSourceMode === 'file'
            ? { manifest_content: localCatalogContent, file_name: localCatalogFileName }
            : { url: normalizedURL || defaultCatalogSourceURL },
        ),
      });
      setCatalogItems(response.items);
      setCatalogSettings(response.settings);
      await loadInstalledPackages();
      toast.success(messages.catalogSynced(response.items.length));
    } catch (syncError) {
      toast.error(messages.requestFailed);
    } finally {
      setCatalogSyncing(false);
    }
  }

  async function pickLocalCatalogFile(file: File | null) {
    if (!file) {
      setLocalCatalogFileName('');
      setLocalCatalogContent('');
      return;
    }

    try {
      const content = await file.text();
      if (content.trim() === '') {
        toast.error(messages.localCatalogFileMissing);
        setLocalCatalogFileName('');
        setLocalCatalogContent('');
        return;
      }
      setLocalCatalogFileName(file.name);
      setLocalCatalogContent(content);
      setCatalogSourceMode('file');
    } catch (readError) {
      toast.error(readError instanceof Error ? readError.message : messages.localCatalogFileMissing);
      setLocalCatalogFileName('');
      setLocalCatalogContent('');
    }
  }

  async function installCatalogPackage(item: CatalogItem) {
    setInstallingCatalogItemId(item.id);
    try {
      await apiRequest<void>(`/api/catalog/items/${item.id}/install`, () => messages.requestFailed, {
        method: 'POST',
      });
      await loadInstalledPackages();
      toast.success(messages.packageInstalled);
    } catch (installError) {
      toast.error(messages.requestFailed);
    } finally {
      setInstallingCatalogItemId(null);
    }
  }

  async function uninstallCatalogPackage(item: CatalogItem, pkg: InstalledPackage) {
    setUninstallingCatalogItemId(item.id);
    try {
      await apiRequest<void>(`/api/packages/${pkg.id}`, () => messages.requestFailed, {
        method: 'DELETE',
      });
      await loadInstalledPackages();
      toast.success(messages.packageUninstalled);
      return true;
    } catch (uninstallError) {
      toast.error(messages.requestFailed);
      return false;
    } finally {
      setUninstallingCatalogItemId(null);
    }
  }

  async function performCatalogInstall(
    item: CatalogItem,
    projectId: number,
    config: Record<string, unknown>,
  ) {
    setAddingCatalogItemId(item.id);
    try {
      await apiRequest<void>(`/api/catalog/items/${item.id}/add-to-project`, () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify({
          project_id: projectId,
          name: item.name,
          config,
        }),
      });
      await loadInstalledPackages();
      toast.success(messages.packageInstalled);
    } catch (installError) {
      toast.error(messages.requestFailed);
    } finally {
      setAddingCatalogItemId(null);
    }
  }

  return {
    // State
    catalogItems,
    setCatalogItems,
    catalogSettings,
    setCatalogSettings,
    installedPackages,
    setInstalledPackages,
    catalogURL,
    setCatalogURL,
    catalogLoading,
    catalogSyncing,
    catalogURLVisible,
    setCatalogURLVisible,
    catalogSourceMode,
    setCatalogSourceMode,
    localCatalogFileName,
    setLocalCatalogFileName,
    localCatalogContent,
    setLocalCatalogContent,
    installingCatalogItemId,
    addingCatalogItemId,
    uninstallingCatalogItemId,
    // Constants
    defaultCatalogSourceURL,
    // Functions
    loadCatalog,
    loadInstalledPackages,
    syncCatalog,
    pickLocalCatalogFile,
    installCatalogPackage,
    uninstallCatalogPackage,
    performCatalogInstall,
  };
}
