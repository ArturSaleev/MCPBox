import type { FormEvent } from 'react';

import { Database, LoaderCircle, Plus, RefreshCw, Trash2 } from 'lucide-react';

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
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';

type RAGCollection = {
  id?: number;
  collection_id: string;
  name: string;
  data_type: string;
  source_path?: string;
  auto_reindex?: boolean;
  index_path?: string;
};

type RAGCollectionForm = {
  name: string;
  source_path: string;
  auto_reindex: boolean;
};

type ProjectKnowledgePanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  connectRAGCollectionOpen: boolean;
  setConnectRAGCollectionOpen: (open: boolean) => void;
  allRAGCollections: RAGCollection[];
  availableRAGCollections: RAGCollection[];
  connectRAGCollectionToProject: (collectionId: string) => void | Promise<void>;
  linkingCollectionId: string | null;
  selectedProject: {
    project_id: number;
    root_path: string;
    rag_collections: RAGCollection[];
  };
  disconnectRAGCollectionFromProject: (collectionId: string) => void | Promise<void>;
  busyProjectId: number | null;
  ragCollectionForm: RAGCollectionForm;
  setRAGCollectionForm: (updater: (current: RAGCollectionForm) => RAGCollectionForm) => void;
  resetRAGCollectionForm: () => void;
  createAndConnectRAGCollectionToProject: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  creatingRAGCollection: boolean;
  ragIndexPaths: Record<string, string>;
  setRAGIndexPaths: (updater: (current: Record<string, string>) => Record<string, string>) => void;
  indexRAGCollection: (collectionId: string) => void | Promise<void>;
  indexingCollectionId: string | null;
};

export function ProjectKnowledgePanel({
  labels,
  messages,
  connectRAGCollectionOpen,
  setConnectRAGCollectionOpen,
  allRAGCollections,
  availableRAGCollections,
  connectRAGCollectionToProject,
  linkingCollectionId,
  selectedProject,
  disconnectRAGCollectionFromProject,
  busyProjectId,
  ragCollectionForm,
  setRAGCollectionForm,
  resetRAGCollectionForm,
  createAndConnectRAGCollectionToProject,
  creatingRAGCollection,
  ragIndexPaths,
  setRAGIndexPaths,
  indexRAGCollection,
  indexingCollectionId,
}: ProjectKnowledgePanelProps) {
  const connectedCollections = selectedProject.rag_collections.map(
    (collection) =>
      allRAGCollections.find((item) => item.collection_id === collection.collection_id) ?? collection,
  );

  return (
    <section className="rounded-2xl border border-border bg-card p-6">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">{labels.connectedKnowledgeBases}</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {messages.connectedKnowledgeBasesDescription}
          </p>
        </div>
        <Dialog
          open={connectRAGCollectionOpen}
          onOpenChange={(open) => {
            setConnectRAGCollectionOpen(open);
            if (!open) {
              resetRAGCollectionForm();
            }
          }}
        >
          <DialogTrigger asChild>
            <button
              onClick={() => {
                if (!ragCollectionForm.source_path.trim() && selectedProject.root_path.trim()) {
                  setRAGCollectionForm((current) => ({ ...current, source_path: selectedProject.root_path }));
                }
              }}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90"
            >
              <Plus className="h-4 w-4" />
              {messages.connectKnowledgeBaseTitle}
            </button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>{messages.connectKnowledgeBaseTitle}</DialogTitle>
              <DialogDescription>
                {messages.connectKnowledgeBaseDescription}
              </DialogDescription>
            </DialogHeader>

            <Tabs defaultValue="select" className="space-y-4">
              <TabsList className="grid h-auto w-full grid-cols-2 rounded-xl border border-border bg-background p-1">
                <TabsTrigger
                  value="select"
                  className="rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-all data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-md data-[state=active]:shadow-electric-blue/20"
                >
                  {labels.connect}
                </TabsTrigger>
                <TabsTrigger
                  value="create"
                  className="rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-all data-[state=active]:bg-electric-blue data-[state=active]:text-white data-[state=active]:shadow-md data-[state=active]:shadow-electric-blue/20"
                >
                  {labels.createCollection}
                </TabsTrigger>
              </TabsList>

              <TabsContent value="select" className="space-y-3">
                {availableRAGCollections.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
                    {messages.noAvailableCollections}
                  </div>
                ) : (
                  availableRAGCollections.map((collection) => (
                    <div key={collection.collection_id} className="flex items-center justify-between gap-3 rounded-xl border border-border bg-background p-4">
                      <div className="min-w-0">
                        <div className="font-medium">{collection.name}</div>
                        <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                          {collection.collection_id} · {collection.data_type}
                        </div>
                      </div>
                      <button
                        onClick={() => void connectRAGCollectionToProject(collection.collection_id)}
                        disabled={linkingCollectionId === collection.collection_id}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {linkingCollectionId === collection.collection_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                        {labels.connect}
                      </button>
                    </div>
                  ))
                )}
              </TabsContent>

              <TabsContent value="create">
                <form className="space-y-4" onSubmit={createAndConnectRAGCollectionToProject}>
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
                      placeholder={selectedProject.root_path || messages.sourceFolderPlaceholder}
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
                      {labels.create}
                    </button>
                  </DialogFooter>
                </form>
              </TabsContent>
            </Tabs>
          </DialogContent>
        </Dialog>
      </div>

      {selectedProject.rag_collections.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-background px-4 py-6 text-sm text-muted-foreground">
          {messages.noKnowledgeBasesConnected}
        </div>
      ) : (
        <div className="space-y-4">
          <div className="rounded-xl border border-electric-blue/20 bg-electric-blue/8 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-electric-blue">
              <Database className="h-4 w-4" />
              {labels.mcpToolReady}
            </div>
            <p className="mt-2 text-sm text-foreground/85">
              {messages.mcpToolReadyIntro}<code>search_project_knowledge</code>{messages.mcpToolReadyOutro}
            </p>
            <div className="mt-3 rounded-lg border border-border bg-background p-3">
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {labels.toolContract}
              </div>
              <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-foreground/85">{`search_project_knowledge({
  query: "payment gateway",
  limit: 5,
  collections: ["crm_gym"]
})`}</pre>
            </div>
          </div>

          <div className="space-y-3">
            {connectedCollections.map((collection) => (
              <div key={collection.collection_id} className="rounded-xl border border-border bg-background p-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Database className="h-4 w-4 text-electric-blue" />
                      <div className="font-medium">{collection.name}</div>
                    </div>
                    <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      {collection.collection_id} · {collection.data_type}
                    </div>
                  </div>
                  <button
                    onClick={() => void disconnectRAGCollectionFromProject(collection.collection_id)}
                    disabled={busyProjectId === selectedProject.project_id}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <Trash2 className="h-4 w-4" />
                    {labels.disconnect}
                  </button>
                </div>

                <div className="mt-4 rounded-xl border border-border bg-card p-4">
                  <div className="text-sm font-medium">{messages.indexFolderTitle}</div>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {messages.indexFolderDescription}
                  </p>
                  <input
                    value={ragIndexPaths[collection.collection_id] ?? collection.source_path ?? selectedProject.root_path ?? ''}
                    onChange={(event) =>
                      setRAGIndexPaths((current) => ({
                        ...current,
                        [collection.collection_id]: event.target.value,
                      }))
                    }
                    className="mt-3 h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    placeholder={selectedProject.root_path || messages.indexFolderPlaceholder}
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
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
