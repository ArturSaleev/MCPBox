import { useRef, useState, type FormEvent } from 'react';

import { CircleAlert, Database, LoaderCircle, Pencil, Plus, RefreshCw, TextSearch, Trash2, Upload } from 'lucide-react';

import { dictionaries, type Language } from '../i18n';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';

type RAGCollection = {
  id: number;
  collection_id: string;
  name: string;
  data_type: string;
  source_path: string;
  managed_source_path: string;
  auto_reindex: boolean;
  service_mode: 'bleve_only' | 'ragbox_only' | 'dual' | string;
  vector_connection_id: string;
  index_path: string;
};

type RAGSearchResult = {
  id: string;
  file_path: string;
  section?: string;
  title?: string;
  content: string;
  source?: 'bleve' | 'ragbox' | 'hybrid' | string;
  score?: number;
};

type RAGCollectionForm = {
  name: string;
  source_path: string;
  auto_reindex: boolean;
  service_mode: 'bleve_only' | 'ragbox_only' | 'dual';
  vector_connection_id: string;
  existing_workspace_id: string;
};

type ProRagBoxVectorConnection = {
  id: string;
  name: string;
  backend_type: string;
  is_default: boolean;
};

type ProRagBoxKnowledgeBaseStatus = {
  binding?: {
    ragbox_collection_id: string;
    sync_status: string;
    ragbox_workspace_id: string;
    ragbox_workspace_vector_connection_id?: string;
  };
  token_public_id?: string;
  token_secret?: string;
  token_configured: boolean;
  ingest_url_path?: string;
  shadow_source_path?: string;
  uses_shadow_storage: boolean;
  requested_vector_connection_id?: string;
};

type ProRagBoxWorkspaceOption = {
  id: string;
  name: string;
  vector_connection_id: string;
  access_mode: string;
  status: string;
};

type KnowledgeViewProps = {
  language: Language;
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  allRAGCollections: RAGCollection[];
  createRAGCollectionOpen: boolean;
  setCreateRAGCollectionOpen: (open: boolean) => void;
  editingRAGCollectionId: string | null;
  ragCollectionForm: RAGCollectionForm;
  setRAGCollectionForm: (updater: (current: RAGCollectionForm) => RAGCollectionForm) => void;
  resetRAGCollectionForm: () => void;
  createRAGCollection: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  creatingRAGCollection: boolean;
  startEditRAGCollection: (collection: RAGCollection) => void;
  deleteRAGCollection: (collectionId: string) => void | Promise<void>;
  linkingCollectionId: string | null;
  ragIndexPaths: Record<string, string>;
  setRAGIndexPaths: (updater: (current: Record<string, string>) => Record<string, string>) => void;
  indexRAGCollection: (collectionId: string) => void | Promise<void>;
  indexingCollectionId: string | null;
  ragSearchQueries: Record<string, string>;
  setRAGSearchQueries: (updater: (current: Record<string, string>) => Record<string, string>) => void;
  searchRAGCollection: (collectionId: string) => void | Promise<void>;
  searchingCollectionId: string | null;
  ragSearchResultsOpen: boolean;
  setRAGSearchResultsOpen: (open: boolean) => void;
  activeRAGSearchCollection: RAGCollection | null;
  setActiveRAGSearchCollectionId: (collectionId: string | null) => void;
  activeRAGSearchResults: RAGSearchResult[];
  ragSearchDiagnostics: string | null;
  ragSearchError: string | null;
  uploadKnowledgeBaseFile: (collectionId: string, file: File, storageMode: 'keep' | 'transient') => void | Promise<void>;
  uploadingKnowledgeBaseId: string | null;
  proCopy: {
    ragboxSearchSourceBleve: string;
    ragboxSearchSourceRagbox: string;
    ragboxSearchSourceHybrid: string;
    ragboxKBEndpointTitle: string;
    ragboxKBEndpointPath: string;
    ragboxKBEndpointToken: string;
    ragboxKBEndpointShadow: string;
    ragboxKBEndpointLocalFolder: string;
    ragboxKBEndpointStatus: string;
    ragboxKBManagedFolder: string;
    ragboxKBUploadTitle: string;
    ragboxKBUploadMode: string;
    ragboxKBUploadModeKeep: string;
    ragboxKBUploadModeTransient: string;
    ragboxKBUploadHint: string;
    ragboxKBUploadButton: string;
    ragboxKBExistingWorkspaceHint: string;
    ragboxKBVectorConnection: string;
    ragboxKBVectorConnectionDefault: string;
    ragboxKBVectorConnectionHint: string;
    ragboxKBExistingWorkspaceUsesOwnVectorConnection: string;
    ragboxKBServiceMode: string;
    ragboxKBServiceModeHint: string;
    ragboxKBServiceModeBleveOnly: string;
    ragboxKBServiceModeRagBoxOnly: string;
    ragboxKBServiceModeDual: string;
  };
  proRagBoxVectorConnections: ProRagBoxVectorConnection[];
  proRagBoxBuiltinFallbackBackend: string;
  proRagBoxWorkspaces: ProRagBoxWorkspaceOption[];
  proRagBoxKnowledgeBaseStatus: Record<string, ProRagBoxKnowledgeBaseStatus>;
};

export function KnowledgeView({
  language,
  labels,
  messages,
  allRAGCollections,
  createRAGCollectionOpen,
  setCreateRAGCollectionOpen,
  editingRAGCollectionId,
  ragCollectionForm,
  setRAGCollectionForm,
  resetRAGCollectionForm,
  createRAGCollection,
  creatingRAGCollection,
  startEditRAGCollection,
  deleteRAGCollection,
  linkingCollectionId,
  ragIndexPaths,
  setRAGIndexPaths,
  indexRAGCollection,
  indexingCollectionId,
  ragSearchQueries,
  setRAGSearchQueries,
  searchRAGCollection,
  searchingCollectionId,
  ragSearchResultsOpen,
  setRAGSearchResultsOpen,
  activeRAGSearchCollection,
  setActiveRAGSearchCollectionId,
  activeRAGSearchResults,
  ragSearchDiagnostics,
  ragSearchError,
  uploadKnowledgeBaseFile,
  uploadingKnowledgeBaseId,
  proCopy,
  proRagBoxVectorConnections,
  proRagBoxBuiltinFallbackBackend,
  proRagBoxWorkspaces,
  proRagBoxKnowledgeBaseStatus,
}: KnowledgeViewProps) {
  const [uploadModes, setUploadModes] = useState<Record<string, 'keep' | 'transient'>>({});
  const uploadInputRefs = useRef<Record<string, HTMLInputElement | null>>({});
  const autoReindexAllowed = ragCollectionForm.service_mode === 'bleve_only';

  function serviceModeLabel(mode?: string) {
    switch (mode) {
      case 'ragbox_only':
        return proCopy.ragboxKBServiceModeRagBoxOnly;
      case 'dual':
        return proCopy.ragboxKBServiceModeDual;
      case 'bleve_only':
      default:
        return proCopy.ragboxKBServiceModeBleveOnly;
    }
  }

  function searchSourceLabel(source?: string) {
    switch (source) {
      case 'ragbox':
        return proCopy.ragboxSearchSourceRagbox;
      case 'hybrid':
        return proCopy.ragboxSearchSourceHybrid;
      case 'bleve':
      default:
        return proCopy.ragboxSearchSourceBleve;
    }
  }

  const [infoOpen, setInfoOpen] = useState(false);
  const [infoCollectionId, setInfoCollectionId] = useState<string | null>(null);
  const [submittedSearches, setSubmittedSearches] = useState<Record<string, boolean>>({});
  const infoCollection = allRAGCollections.find(v => v.collection_id === infoCollectionId);
  const infoStatus = infoCollection ? proRagBoxKnowledgeBaseStatus[infoCollection.collection_id] : undefined;
  const copy = language === 'ru' ? {
    info: 'О базе знаний', details: 'Информация о коллекции', help: 'Как пользоваться',
    edit: 'Редактировать', editTitle: 'Редактировать базу знаний', save: 'Сохранить',
    search: 'Поиск', searchStart: 'Введите запрос, чтобы найти фрагменты в этой коллекции.',
    searching: 'Ищем в коллекции…', upload: 'Загрузить файл', folder: 'Индексировать папку',
    dataType: 'Тип данных', indexPath: 'Путь индекса', source: 'Исходная папка', notSet: 'Не задано',
    vectorRequest: 'Запрошенное векторное подключение', existingWorkspace: 'Существующий workspace RagBox',
    managedWorkspace: 'Создать или использовать workspace MCPBox',
  } : {
    info: 'About knowledge bases', details: 'Collection information', help: 'How to use',
    edit: 'Edit', editTitle: 'Edit Knowledge Base', save: 'Save',
    search: 'Search', searchStart: 'Enter a query to find relevant passages in this collection.',
    searching: 'Searching collection…', upload: 'Upload file', folder: 'Index a folder',
    dataType: 'Data type', indexPath: 'Index path', source: 'Source folder', notSet: 'Not set',
    vectorRequest: 'Requested vector connection', existingWorkspace: 'Existing RagBox workspace',
    managedWorkspace: 'Create or use MCPBox-managed workspace',
  };
  const actionClass = 'inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-lg border border-border bg-card px-3 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60';
  const primaryActionClass = actionClass.replace('border-border bg-card', 'border-transparent bg-brand-solid text-white').replace('hover:bg-accent', 'hover:bg-brand-solid/90');
  const fieldClass = 'h-10 min-w-0 w-full rounded-lg border border-border bg-input-background px-3 text-sm outline-none focus:border-electric-blue';
  function openInfo(collectionId: string | null = null) {
    setInfoCollectionId(collectionId);
    setInfoOpen(true);
  }
  function openSearch(collectionId: string) {
    setActiveRAGSearchCollectionId(collectionId);
    setRAGSearchResultsOpen(true);
  }
  const searchBusy = activeRAGSearchCollection?.collection_id === searchingCollectionId;
  const showSearchResults = !!activeRAGSearchCollection &&
    (submittedSearches[activeRAGSearchCollection.collection_id] || activeRAGSearchResults.length > 0);

  return (
    <section className="space-y-5">
      <header className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-border bg-card p-5 md:p-6">
        <div className="min-w-0">
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-electric-blue">{labels.knowledgeBase}</p>
          <div className="mt-1 flex items-center gap-3">
            <h2 className="text-2xl font-semibold">{messages.knowledgeBaseHeroTitle}</h2>
            <span aria-label={labels.collections} className="rounded-full border border-border bg-muted px-2.5 py-0.5 text-sm tabular-nums">{allRAGCollections.length}</span>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button type="button" className={actionClass} aria-label={copy.info} title={copy.info} onClick={() => openInfo()}><CircleAlert className="h-5 w-5" /></button>
            <Dialog
              open={createRAGCollectionOpen}
              onOpenChange={(open) => {
                setCreateRAGCollectionOpen(open);
                if (!open) {
                  resetRAGCollectionForm();
                }
              }}
            >
              <DialogTrigger asChild>
                <button className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-brand-solid px-4 text-sm font-medium text-white transition-colors hover:bg-brand-solid/90">
                  <Plus className="h-4 w-4" />
                  {labels.createCollection}
                </button>
              </DialogTrigger>
              <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>{editingRAGCollectionId ? copy.editTitle : messages.createKnowledgeBaseTitle}</DialogTitle>
                  <DialogDescription>
                    {editingRAGCollectionId ? messages.editKnowledgeBaseDescription : messages.createKnowledgeBaseDescription}
                  </DialogDescription>
                </DialogHeader>
                <form className="space-y-4" onSubmit={createRAGCollection}>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{labels.name}</span>
                    <input
                      required
                      value={ragCollectionForm.name}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({ ...current, name: event.target.value }))
                      }
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={messages.collectionNamePlaceholder}
                    />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{messages.sourceFolderTitle}</span>
                    <input
                      value={ragCollectionForm.source_path}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({ ...current, source_path: event.target.value }))
                      }
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={messages.sourceFolderPlaceholder}
                    />
                    <p className="text-xs text-muted-foreground">{messages.createKnowledgeBaseDescription}</p>
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.ragboxKBServiceMode}</span>
                    <select
                      value={ragCollectionForm.service_mode}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({
                          ...current,
                          service_mode: event.target.value as 'bleve_only' | 'ragbox_only' | 'dual',
                          auto_reindex:
                            event.target.value === 'bleve_only'
                              ? current.auto_reindex
                              : false,
                        }))
                      }
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    >
                      <option value="bleve_only">{proCopy.ragboxKBServiceModeBleveOnly}</option>
                      <option value="ragbox_only">{proCopy.ragboxKBServiceModeRagBoxOnly}</option>
                      <option value="dual">{proCopy.ragboxKBServiceModeDual}</option>
                    </select>
                    <p className="text-xs text-muted-foreground">{proCopy.ragboxKBServiceModeHint}</p>
                  </label>
                  {ragCollectionForm.service_mode !== 'bleve_only' ? (
                    <>
                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{copy.existingWorkspace}</span>
                        <select
                          value={ragCollectionForm.existing_workspace_id}
                          onChange={(event) =>
                            setRAGCollectionForm((current) => ({ ...current, existing_workspace_id: event.target.value }))
                          }
                          className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        >
                          <option value="">{copy.managedWorkspace}</option>
                          {proRagBoxWorkspaces.map((workspace) => (
                            <option key={workspace.id} value={workspace.id}>
                              {workspace.name} · {workspace.vector_connection_id} · {workspace.access_mode}
                            </option>
                          ))}
                        </select>
                        <p className="text-xs text-muted-foreground">{proCopy.ragboxKBExistingWorkspaceHint}</p>
                      </label>
                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{proCopy.ragboxKBVectorConnection}</span>
                        <select
                          value={ragCollectionForm.vector_connection_id}
                          onChange={(event) =>
                            setRAGCollectionForm((current) => ({ ...current, vector_connection_id: event.target.value }))
                          }
                          disabled={!!ragCollectionForm.existing_workspace_id}
                          className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          <option value="">
                            {proCopy.ragboxKBVectorConnectionDefault}: {proRagBoxBuiltinFallbackBackend}
                          </option>
                          {proRagBoxVectorConnections.map((connection) => (
                            <option key={connection.id} value={connection.id}>
                              {connection.name} · {connection.backend_type}{connection.is_default ? ' · default' : ''}
                            </option>
                          ))}
                        </select>
                        <p className="text-xs text-muted-foreground">
                          {ragCollectionForm.existing_workspace_id
                            ? proCopy.ragboxKBExistingWorkspaceUsesOwnVectorConnection
                            : proCopy.ragboxKBVectorConnectionHint}
                        </p>
                      </label>
                    </>
                  ) : null}
                  <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm">
                    <input
                      type="checkbox"
                      checked={ragCollectionForm.auto_reindex}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({ ...current, auto_reindex: event.target.checked }))
                      }
                      disabled={!autoReindexAllowed}
                      className="h-4 w-4 rounded border-border"
                    />
                    <div>
                      <div className="font-medium">{messages.autoReindexTitle}</div>
                      <div className="text-muted-foreground">{messages.autoReindexDescription}</div>
                    </div>
                  </label>
                  <DialogFooter>
                    <button
                      type="submit"
                      disabled={creatingRAGCollection}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-brand-solid px-4 text-sm font-medium text-white transition-colors hover:bg-brand-solid/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {creatingRAGCollection ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                      {editingRAGCollectionId ? copy.save : labels.create}
                    </button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
        </div>
      </header>

      {allRAGCollections.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card px-6 py-12 text-center">
          <Database className="h-8 w-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">{messages.noKnowledgeBasesCreated}</p>
        </div>
      ) : (
        <div className="space-y-4">
          {allRAGCollections.map(collection => {
            const status = proRagBoxKnowledgeBaseStatus[collection.collection_id];
            return (
              <article key={collection.collection_id} className="min-w-0 rounded-2xl border border-border bg-card p-4 md:p-5">
                <header className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-electric-blue/20 bg-brand-solid/10 text-electric-blue"><Database className="h-5 w-5" /></div>
                    <div className="min-w-0">
                      <h3 className="break-words font-semibold">{collection.name}</h3>
                      <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs">
                        <span className="rounded-full border border-border bg-muted px-2 py-0.5">{serviceModeLabel(collection.service_mode)}</span>
                        {collection.auto_reindex && collection.service_mode === 'bleve_only' ? <span className="rounded-full border border-status-running/30 bg-status-running/10 px-2 py-0.5 text-status-running">{messages.autoReindexBadge}</span> : null}
                        {collection.service_mode !== 'bleve_only' && status?.binding?.sync_status ? <span className="rounded-full border border-border px-2 py-0.5">RagBox: {status.binding.sync_status}</span> : null}
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <button type="button" className={actionClass} aria-label={`${copy.details}: ${collection.name}`} title={copy.details} onClick={() => openInfo(collection.collection_id)}><CircleAlert className="h-4 w-4" /></button>
                    <button type="button" className={actionClass} onClick={() => openSearch(collection.collection_id)} disabled={!!searchingCollectionId && searchingCollectionId !== collection.collection_id} aria-label={`${copy.search}: ${collection.name}`}>
                      {searchingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <TextSearch className="h-4 w-4" />}{labels.search}
                    </button>
                    <button type="button" className={actionClass} onClick={() => startEditRAGCollection(collection)} aria-label={`${copy.edit}: ${collection.name}`} title={copy.edit}><Pencil className="h-4 w-4" /></button>
                    <button type="button" className={`${actionClass} border-destructive/30 text-destructive hover:bg-destructive/10`} onClick={() => void deleteRAGCollection(collection.collection_id)} disabled={linkingCollectionId === collection.collection_id} aria-label={`${labels.delete}: ${collection.name}`} title={labels.delete}><Trash2 className="h-4 w-4" /></button>
                  </div>
                </header>

                <div className="mt-5 grid gap-4 border-t border-border pt-4 md:grid-cols-2">
                  <div className="min-w-0 space-y-2">
                    <p className="text-sm font-medium">{proCopy.ragboxKBUploadTitle}</p>
                    <input ref={element => { uploadInputRefs.current[collection.collection_id] = element; }} type="file" className="hidden"
                      onChange={event => {
                        const input = event.currentTarget;
                        const file = input.files?.[0];
                        if (!file) return;
                        void Promise.resolve(uploadKnowledgeBaseFile(collection.collection_id, file, uploadModes[collection.collection_id] ?? 'keep')).finally(() => { input.value = ''; });
                      }} />
                    <div className="flex flex-wrap gap-2">
                      <select aria-label={`${proCopy.ragboxKBUploadMode}: ${collection.name}`} value={uploadModes[collection.collection_id] ?? 'keep'}
                        onChange={event => setUploadModes(current => ({ ...current, [collection.collection_id]: event.target.value as 'keep' | 'transient' }))}
                        className={`${fieldClass} flex-1 basis-40`} disabled={uploadingKnowledgeBaseId === collection.collection_id}>
                        <option value="keep">{proCopy.ragboxKBUploadModeKeep}</option>
                        <option value="transient">{proCopy.ragboxKBUploadModeTransient}</option>
                      </select>
                      <button type="button" className={primaryActionClass} onClick={() => uploadInputRefs.current[collection.collection_id]?.click()} disabled={uploadingKnowledgeBaseId === collection.collection_id}>
                        {uploadingKnowledgeBaseId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}{copy.upload}
                      </button>
                    </div>
                  </div>
                  <form className="min-w-0 space-y-2" onSubmit={event => { event.preventDefault(); void indexRAGCollection(collection.collection_id); }}>
                    <label className="text-sm font-medium" htmlFor={`index-folder-${collection.collection_id}`}>{messages.indexFolderTitle}</label>
                    <div className="flex flex-wrap gap-2">
                      <input id={`index-folder-${collection.collection_id}`} value={ragIndexPaths[collection.collection_id] ?? ''}
                        onChange={event => setRAGIndexPaths(current => ({ ...current, [collection.collection_id]: event.target.value }))}
                        className={`${fieldClass} flex-1 basis-40`} placeholder={messages.indexFolderPlaceholder} disabled={indexingCollectionId === collection.collection_id} />
                      <button type="submit" className={actionClass} disabled={indexingCollectionId === collection.collection_id}>
                        {indexingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}{labels.index}
                      </button>
                    </div>
                  </form>
                </div>
              </article>
            );
          })}
        </div>
      )}

      <Dialog open={infoOpen} onOpenChange={setInfoOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{infoCollection ? `${copy.details}: ${infoCollection.name}` : copy.info}</DialogTitle>
            <DialogDescription>{messages.knowledgeBaseHeroDescription}</DialogDescription>
          </DialogHeader>
          {infoCollection ? <div className="space-y-4">
            <dl className="grid gap-3 rounded-xl border border-border bg-background p-4 text-sm sm:grid-cols-2">
              {[
                ['ID', infoCollection.collection_id],
                [labels.name, infoCollection.name],
                [proCopy.ragboxKBServiceMode, serviceModeLabel(infoCollection.service_mode)],
                [copy.dataType, infoCollection.data_type],
                [copy.source, infoCollection.source_path || copy.notSet],
                [proCopy.ragboxKBManagedFolder, infoCollection.managed_source_path || copy.notSet],
                [copy.indexPath, infoCollection.index_path || copy.notSet],
                ...(infoStatus?.requested_vector_connection_id ? [[copy.vectorRequest, infoStatus.requested_vector_connection_id]] : []),
              ].map(([label, value]) => <div key={label} className="min-w-0"><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-1 break-all">{value}</dd></div>)}
            </dl>
                {(() => {
                  const status = proRagBoxKnowledgeBaseStatus[infoCollection.collection_id];
                  const usesRagBox = infoCollection.service_mode !== 'bleve_only';
                  if (!status || !usesRagBox) {
                    return null;
                  }

                  return (
                    <div className="mb-4 rounded-xl border border-electric-blue/20 bg-brand-solid/5 p-4 text-sm">
                      <div className="font-medium text-foreground">{proCopy.ragboxKBEndpointTitle}</div>
                      <div className="mt-2 grid gap-2 text-muted-foreground lg:grid-cols-2">
                        <div>
                          {proCopy.ragboxKBEndpointStatus}: {status.binding?.sync_status ?? 'pending'}
                        </div>
                        {status.binding?.ragbox_workspace_id ? (
                          <div>
                            RagBox workspace: <code className="text-xs">{status.binding.ragbox_workspace_id}</code>
                          </div>
                        ) : null}
                        {status.binding?.ragbox_workspace_vector_connection_id ? (
                          <div>
                            Effective workspace vector connection: <code className="text-xs">{status.binding.ragbox_workspace_vector_connection_id}</code>
                          </div>
                        ) : null}
                        <div>
                          {proCopy.ragboxKBEndpointLocalFolder}:{' '}
                          <code className="text-xs">
                            {infoCollection.source_path || copy.notSet}
                          </code>
                        </div>
                        <div>
                          {proCopy.ragboxKBManagedFolder}:{' '}
                          <code className="text-xs">
                            {infoCollection.managed_source_path || status.shadow_source_path || copy.notSet}
                          </code>
                        </div>
                        {status.ingest_url_path ? (
                          <div className="lg:col-span-2">
                            {proCopy.ragboxKBEndpointPath}: <code className="text-xs">{status.ingest_url_path}</code>
                          </div>
                        ) : null}
                        {status.token_secret ? (
                          <div className="lg:col-span-2">
                            {proCopy.ragboxKBEndpointToken}: <code className="text-xs">{status.token_secret}</code>
                          </div>
                        ) : null}
                        {status.shadow_source_path ? (
                          <div className="lg:col-span-2">
                            {proCopy.ragboxKBEndpointShadow}: <code className="text-xs">{status.shadow_source_path}</code>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  );
                })()}

          </div> : null}
          <div className="space-y-4 text-sm">
            <section><h3 className="font-medium">{messages.supportedFormatsLabel}</h3><p className="mt-1 text-muted-foreground">{messages.supportedFormatsValue}</p></section>
            <section><h3 className="font-medium">{proCopy.ragboxKBUploadTitle}</h3><p className="mt-1 text-muted-foreground">{proCopy.ragboxKBUploadHint}</p><p className="mt-1 text-muted-foreground">{proCopy.ragboxKBUploadMode}: {proCopy.ragboxKBUploadModeKeep} / {proCopy.ragboxKBUploadModeTransient}</p></section>
            <section><h3 className="font-medium">{messages.indexFolderTitle}</h3><p className="mt-1 text-muted-foreground">{messages.indexFolderDescription}</p></section>
            <section><h3 className="font-medium">{messages.searchCollectionTitle}</h3><p className="mt-1 text-muted-foreground">{messages.searchCollectionDescription}</p></section>
            <section><h3 className="font-medium">{messages.autoReindexTitle}</h3><p className="mt-1 text-muted-foreground">{messages.autoReindexDescription}</p></section>
            <section><h3 className="font-medium">{proCopy.ragboxKBServiceMode}</h3><p className="mt-1 text-muted-foreground">{proCopy.ragboxKBServiceModeHint}</p></section>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={ragSearchResultsOpen} onOpenChange={open => { setRAGSearchResultsOpen(open); if (!open) setActiveRAGSearchCollectionId(null); }}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{messages.searchCollectionTitle}</DialogTitle>
            <DialogDescription>{messages.searchResultsDescription(activeRAGSearchCollection?.name ?? labels.knowledgeBase)}</DialogDescription>
          </DialogHeader>
          {activeRAGSearchCollection ? <form className="flex flex-wrap gap-2" onSubmit={event => {
            event.preventDefault();
            const id = activeRAGSearchCollection.collection_id;
            setSubmittedSearches(current => ({ ...current, [id]: true }));
            void searchRAGCollection(id);
          }}>
            <input required aria-label={messages.searchCollectionTitle} placeholder={messages.searchCollectionPlaceholder}
              value={ragSearchQueries[activeRAGSearchCollection.collection_id] ?? ''}
              onChange={event => setRAGSearchQueries(current => ({ ...current, [activeRAGSearchCollection.collection_id]: event.target.value }))}
              className={`${fieldClass} flex-1 basis-48`} disabled={searchBusy} />
            <button type="submit" className={primaryActionClass} disabled={!!searchingCollectionId}>
              {searchBusy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <TextSearch className="h-4 w-4" />}{labels.search}
            </button>
          </form> : null}
          <div aria-live="polite" role="status">
            {searchBusy ? <p className="flex items-center gap-2 py-5 text-sm text-muted-foreground"><LoaderCircle className="h-4 w-4 animate-spin" />{copy.searching}</p> : null}
            {ragSearchError ? <p role="alert" className="rounded-xl border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{ragSearchError}</p> : null}
          </div>
          {!searchBusy && ragSearchDiagnostics ? <div className="rounded-xl border border-electric-blue/20 bg-brand-solid/5 p-3 text-sm text-muted-foreground">{ragSearchDiagnostics}</div> : null}
          {!searchBusy && !ragSearchError ? (!showSearchResults ? <p className="py-5 text-sm text-muted-foreground">{copy.searchStart}</p> : activeRAGSearchResults.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border bg-background p-6 text-sm text-muted-foreground">{messages.searchResultsEmpty}</div>
          ) : (
            <div className="space-y-3">
              {activeRAGSearchResults.map((item) => (
                <div key={`${item.id}-${item.source ?? 'bleve'}`} className="rounded-lg border border-border bg-background p-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <code className="block overflow-x-auto text-xs text-electric-blue">{item.file_path}</code>
                    {item.source ? (
                      <span className="inline-flex rounded-full border border-border px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
                        {searchSourceLabel(item.source)}
                      </span>
                    ) : null}
                    {typeof item.score === 'number' ? (
                      <span className="text-[11px] text-muted-foreground">score {item.score.toFixed(2)}</span>
                    ) : null}
                  </div>
                  {item.section ? (
                    <div className="mt-2 inline-flex rounded-full border border-electric-blue/20 bg-brand-solid/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                      {item.section}
                    </div>
                  ) : null}
                  <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-foreground/85">{item.content}</pre>
                </div>
              ))}
            </div>
          )) : null}
        </DialogContent>
      </Dialog>
    </section>
  );
}
