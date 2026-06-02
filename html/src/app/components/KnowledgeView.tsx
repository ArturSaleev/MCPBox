import type { FormEvent } from 'react';

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
  auto_reindex: boolean;
  index_path: string;
};

type RAGSearchResult = {
  id: string;
  file_path: string;
  section?: string;
  content: string;
};

type RAGCollectionForm = {
  name: string;
  source_path: string;
  auto_reindex: boolean;
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
}: KnowledgeViewProps) {
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
                      required
                      value={ragCollectionForm.source_path}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({ ...current, source_path: event.target.value }))
                      }
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={messages.sourceFolderPlaceholder}
                    />
                  </label>
                  <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm">
                    <input
                      type="checkbox"
                      checked={ragCollectionForm.auto_reindex}
                      onChange={(event) =>
                        setRAGCollectionForm((current) => ({ ...current, auto_reindex: event.target.checked }))
                      }
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
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Database className="h-4 w-4 text-electric-blue" />
                      <div className="font-medium">{collection.name}</div>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                        {messages.supportedFormatsLabel}: {messages.supportedFormatsValue}
                      </span>
                      {collection.auto_reindex ? (
                        <span className="rounded-full border border-status-running/30 bg-status-running/10 px-2.5 py-1 text-[11px] font-medium text-status-running">
                          {messages.autoReindexBadge}
                        </span>
                      ) : null}
                    </div>
                    <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      {collection.collection_id} · {collection.data_type}
                    </div>
                    <code className="mt-2 block overflow-x-auto text-xs text-electric-blue">
                      {collection.index_path}
                    </code>
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

          {activeRAGSearchResults.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border bg-background px-4 py-8 text-sm text-muted-foreground">
              {messages.searchResultsEmpty}
            </div>
          ) : (
            <div className="max-h-[65vh] space-y-3 overflow-y-auto pr-1">
              {activeRAGSearchResults.map((item) => (
                <div key={item.id} className="rounded-lg border border-border bg-background p-3">
                  <code className="block overflow-x-auto text-xs text-electric-blue">{item.file_path}</code>
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
