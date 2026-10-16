import { useRef, useState, type FormEvent } from 'react';

import { Database, LoaderCircle, Pencil, Plus, RefreshCw, TextSearch, Trash2 } from 'lucide-react';

import { dictionaries } from '../i18n';
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

  return (
    <section className="space-y-6">
      <div className="rounded-2xl border border-border bg-card p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
              {labels.knowledgeBase}
            </p>
            <h2 className="mt-2 text-3xl font-semibold">{messages.knowledgeBaseHeroTitle}</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground">
              {messages.knowledgeBaseHeroDescription}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="rounded-xl border border-border bg-background px-4 py-3">
              <div className="text-sm text-muted-foreground">{labels.collections}</div>
              <div className="mt-1 text-2xl font-semibold">{allRAGCollections.length}</div>
            </div>
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
                <button className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
                  <Plus className="h-4 w-4" />
                  {labels.createCollection}
                </button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>{editingRAGCollectionId ? 'Edit Knowledge Base' : messages.createKnowledgeBaseTitle}</DialogTitle>
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
                        <span className="text-sm text-muted-foreground">Existing RagBox workspace</span>
                        <select
                          value={ragCollectionForm.existing_workspace_id}
                          onChange={(event) =>
                            setRAGCollectionForm((current) => ({ ...current, existing_workspace_id: event.target.value }))
                          }
                          className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        >
                          <option value="">Create or use MCPBox-managed workspace</option>
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
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {creatingRAGCollection ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                      {editingRAGCollectionId ? 'Save Knowledge Base' : labels.create}
                    </button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </div>
        </div>
      </div>

      <section className="rounded-2xl border border-border bg-card p-6">
        {allRAGCollections.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
            {messages.noKnowledgeBasesCreated}
          </div>
        ) : (
          <div className="space-y-3">
            {allRAGCollections.map((collection) => (
              <div key={collection.collection_id} className="rounded-xl border border-border bg-background p-4">
                {(() => {
                  const status = proRagBoxKnowledgeBaseStatus[collection.collection_id];
                  const usesRagBox = collection.service_mode !== 'bleve_only';
                  if (!status || !usesRagBox) {
                    return null;
                  }

                  return (
                    <div className="mb-4 rounded-xl border border-electric-blue/20 bg-electric-blue/5 p-4 text-sm">
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
                            {collection.source_path || 'not set'}
                          </code>
                        </div>
                        <div>
                          {proCopy.ragboxKBManagedFolder}:{' '}
                          <code className="text-xs">
                            {collection.managed_source_path || status.shadow_source_path || 'not set'}
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
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="min-w-0">
                    {(() => {
                      const status = proRagBoxKnowledgeBaseStatus[collection.collection_id];
                      return (
                        <>
                    <div className="flex items-center gap-2">
                      <Database className="h-4 w-4 text-electric-blue" />
                      <div className="font-medium">{collection.name}</div>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                        {messages.supportedFormatsLabel}: {messages.supportedFormatsValue}
                      </span>
                      <span className="rounded-full border border-border px-2.5 py-1 text-[11px] font-medium text-foreground">
                        {proCopy.ragboxKBServiceMode}: {serviceModeLabel(collection.service_mode)}
                      </span>
                      {collection.auto_reindex && collection.service_mode === 'bleve_only' ? (
                        <span className="rounded-full border border-status-running/30 bg-status-running/10 px-2.5 py-1 text-[11px] font-medium text-status-running">
                          {messages.autoReindexBadge}
                        </span>
                      ) : null}
                    </div>
                    <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      {collection.collection_id} · {collection.data_type}
                    </div>
                    {status?.requested_vector_connection_id ? (
                      <div className="mt-2 text-xs text-muted-foreground">
                        Workspace vector connection request: <code>{status.requested_vector_connection_id}</code>
                      </div>
                    ) : null}
                    <div className="mt-2 text-xs text-muted-foreground">
                      {proCopy.ragboxKBManagedFolder}: <code>{collection.managed_source_path}</code>
                    </div>
                    <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">
                      {collection.index_path}
                    </code>
                        </>
                      );
                    })()}
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <button
                      onClick={() => startEditRAGCollection(collection)}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
                    >
                      <Pencil className="h-4 w-4" />
                      Edit
                    </button>
                    <button
                      onClick={() => void deleteRAGCollection(collection.collection_id)}
                      disabled={linkingCollectionId === collection.collection_id}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      <Trash2 className="h-4 w-4" />
                      {labels.delete}
                    </button>
                  </div>
                </div>

                <div className="mt-4 grid gap-4 xl:grid-cols-2">
                  <div className="rounded-xl border border-border bg-card p-4">
                    <div className="text-sm font-medium">{proCopy.ragboxKBUploadTitle}</div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {proCopy.ragboxKBUploadHint}
                    </p>
                    <input
                      ref={(element) => {
                        uploadInputRefs.current[collection.collection_id] = element;
                      }}
                      type="file"
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        if (!file) {
                          return;
                        }
                        void uploadKnowledgeBaseFile(
                          collection.collection_id,
                          file,
                          uploadModes[collection.collection_id] ?? 'keep',
                        ).finally(() => {
                          event.target.value = '';
                        });
                      }}
                      className="hidden"
                    />
                    <label className="mt-3 block space-y-2">
                      <span className="text-sm text-muted-foreground">{proCopy.ragboxKBUploadMode}</span>
                      <select
                        value={uploadModes[collection.collection_id] ?? 'keep'}
                        onChange={(event) =>
                          setUploadModes((current) => ({
                            ...current,
                            [collection.collection_id]: event.target.value as 'keep' | 'transient',
                          }))
                        }
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      >
                        <option value="keep">{proCopy.ragboxKBUploadModeKeep}</option>
                        <option value="transient">{proCopy.ragboxKBUploadModeTransient}</option>
                      </select>
                    </label>
                    <button
                      onClick={() => uploadInputRefs.current[collection.collection_id]?.click()}
                      disabled={uploadingKnowledgeBaseId === collection.collection_id}
                      className="mt-3 inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {uploadingKnowledgeBaseId === collection.collection_id ? (
                        <LoaderCircle className="h-4 w-4 animate-spin" />
                      ) : (
                        <Plus className="h-4 w-4" />
                      )}
                      {proCopy.ragboxKBUploadButton}
                    </button>
                  </div>
                  <div className="rounded-xl border border-border bg-card p-4">
                    <div className="text-sm font-medium">{messages.indexFolderTitle}</div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {messages.indexFolderDescription}
                    </p>
                    <input
                      value={ragIndexPaths[collection.collection_id] ?? ''}
                      onChange={(event) =>
                        setRAGIndexPaths((current) => ({
                          ...current,
                          [collection.collection_id]: event.target.value,
                        }))
                      }
                      className="mt-3 h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={messages.indexFolderPlaceholder}
                    />
                    <button
                      onClick={() => void indexRAGCollection(collection.collection_id)}
                      disabled={indexingCollectionId === collection.collection_id}
                      className="mt-3 inline-flex h-10 items-center justify-center gap-2 rounded-md bg-foreground px-4 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {indexingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                      {labels.index}
                    </button>
                  </div>

                  <div className="rounded-xl border border-border bg-card p-4">
                    <div className="text-sm font-medium">{messages.searchCollectionTitle}</div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {messages.searchCollectionDescription}
                    </p>
                    <div className="mt-3 flex gap-3">
                      <input
                        value={ragSearchQueries[collection.collection_id] ?? ''}
                        onChange={(event) =>
                          setRAGSearchQueries((current) => ({
                            ...current,
                            [collection.collection_id]: event.target.value,
                          }))
                        }
                        className="h-10 min-w-0 flex-1 rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder={messages.searchCollectionPlaceholder}
                      />
                      <button
                        onClick={() => void searchRAGCollection(collection.collection_id)}
                        disabled={searchingCollectionId === collection.collection_id}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {searchingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <TextSearch className="h-4 w-4" />}
                        {labels.search}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <Dialog
        open={ragSearchResultsOpen}
        onOpenChange={(open) => {
          setRAGSearchResultsOpen(open);
          if (!open) {
            setActiveRAGSearchCollectionId(null);
          }
        }}
      >
        <DialogContent className="sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{messages.searchResultsTitle}</DialogTitle>
            <DialogDescription>
              {messages.searchResultsDescription(activeRAGSearchCollection?.name ?? labels.knowledgeBase)}
            </DialogDescription>
          </DialogHeader>

          {activeRAGSearchCollection ? (
            <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
              <div className="font-medium">{activeRAGSearchCollection.name}</div>
              <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {activeRAGSearchCollection.collection_id} · {activeRAGSearchCollection.data_type}
              </div>
            </div>
          ) : null}

          {ragSearchDiagnostics ? (
            <div className="rounded-xl border border-electric-blue/20 bg-electric-blue/5 px-4 py-3 text-sm text-muted-foreground">
              {ragSearchDiagnostics}
            </div>
          ) : null}

          {activeRAGSearchResults.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-sm text-muted-foreground">
              {messages.searchResultsEmpty}
            </div>
          ) : (
            <div className="max-h-[65vh] space-y-3 overflow-y-auto pr-1">
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
                    <div className="mt-2 inline-flex rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                      {item.section}
                    </div>
                  ) : null}
                  <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-foreground/85">{item.content}</pre>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </section>
  );
}
