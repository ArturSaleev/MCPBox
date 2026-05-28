import { Database, LoaderCircle, Plus, Trash2 } from 'lucide-react';

import { dictionaries } from '../i18n';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';

type RAGCollection = {
  collection_id: string;
  name: string;
  data_type: string;
};

type ProjectKnowledgePanelProps = {
  labels: typeof dictionaries.en.labels;
  messages: typeof dictionaries.en.messages;
  connectRAGCollectionOpen: boolean;
  setConnectRAGCollectionOpen: (open: boolean) => void;
  availableRAGCollections: RAGCollection[];
  connectRAGCollectionToProject: (collectionId: string) => void | Promise<void>;
  linkingCollectionId: string | null;
  selectedProject: {
    project_id: number;
    rag_collections: RAGCollection[];
  };
  disconnectRAGCollectionFromProject: (collectionId: string) => void | Promise<void>;
  busyProjectId: number | null;
};

export function ProjectKnowledgePanel({
  labels,
  messages,
  connectRAGCollectionOpen,
  setConnectRAGCollectionOpen,
  availableRAGCollections,
  connectRAGCollectionToProject,
  linkingCollectionId,
  selectedProject,
  disconnectRAGCollectionFromProject,
  busyProjectId,
}: ProjectKnowledgePanelProps) {
  return (
    <section className="rounded-2xl border border-border bg-card p-6">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">{labels.connectedKnowledgeBases}</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {messages.connectedKnowledgeBasesDescription}
          </p>
        </div>
        <Dialog open={connectRAGCollectionOpen} onOpenChange={setConnectRAGCollectionOpen}>
          <DialogTrigger asChild>
            <button className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90">
              <Plus className="h-4 w-4" />
              {messages.connectKnowledgeBaseTitle}
            </button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-xl">
            <DialogHeader>
              <DialogTitle>{messages.connectKnowledgeBaseTitle}</DialogTitle>
              <DialogDescription>
                {messages.connectKnowledgeBaseDescription}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3">
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
            </div>
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
            {selectedProject.rag_collections.map((collection) => (
              <div key={collection.collection_id} className="flex flex-col gap-3 rounded-xl border border-border bg-background p-4 lg:flex-row lg:items-center lg:justify-between">
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
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
