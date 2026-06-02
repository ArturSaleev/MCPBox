import { useState } from 'react';
import { apiRequest } from '../utils/api';
import type { RAGCollection, RAGSearchResult, ProjectStatus } from '../types';

const emptyRAGCollectionForm = {
  name: '',
  source_path: '',
  auto_reindex: false,
};

export function useRAG(projects: ProjectStatus[], messages: { requestFailed: string }) {
  const [allRAGCollections, setAllRAGCollections] = useState<RAGCollection[]>([]);
  const [editingRAGCollectionId, setEditingRAGCollectionId] = useState<string | null>(null);
  const [createRAGCollectionOpen, setCreateRAGCollectionOpen] = useState(false);
  const [connectRAGCollectionOpen, setConnectRAGCollectionOpen] = useState(false);
  const [creatingRAGCollection, setCreatingRAGCollection] = useState(false);
  const [linkingCollectionId, setLinkingCollectionId] = useState<string | null>(null);
  const [ragCollectionForm, setRAGCollectionForm] = useState(emptyRAGCollectionForm);
  const [ragIndexPaths, setRAGIndexPaths] = useState<Record<string, string>>({});
  const [ragSearchQueries, setRAGSearchQueries] = useState<Record<string, string>>({});
  const [ragSearchResults, setRAGSearchResults] = useState<Record<string, RAGSearchResult[]>>({});
  const [ragSearchResultsOpen, setRAGSearchResultsOpen] = useState(false);
  const [activeRAGSearchCollectionId, setActiveRAGSearchCollectionId] = useState<string | null>(null);
  const [indexingCollectionId, setIndexingCollectionId] = useState<string | null>(null);
  const [searchingCollectionId, setSearchingCollectionId] = useState<string | null>(null);

  async function loadRAGCollections() {
    try {
      const response = await apiRequest<{ items: RAGCollection[] }>('/api/rag/collections', () => messages.requestFailed);
      setAllRAGCollections(response.items);
    } catch (loadError) {
      console.error('Failed to load RAG collections:', loadError);
    }
  }

  async function createRAGCollection() {
    setCreatingRAGCollection(true);
    try {
      await apiRequest<void>('/api/rag/collections', () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify(ragCollectionForm),
      });
      await loadRAGCollections();
      setCreateRAGCollectionOpen(false);
      setRAGCollectionForm(emptyRAGCollectionForm);
    } catch (createError) {
      console.error('Failed to create RAG collection:', createError);
    } finally {
      setCreatingRAGCollection(false);
    }
  }

  async function linkRAGCollection(projectId: number, collectionId: string) {
    setLinkingCollectionId(collectionId);
    try {
      await apiRequest<void>(`/api/projects/${projectId}/rag/collections/${collectionId}`, () => messages.requestFailed, {
        method: 'POST',
      });
      await loadRAGCollections();
    } catch (linkError) {
      console.error('Failed to link RAG collection:', linkError);
    } finally {
      setLinkingCollectionId(null);
    }
  }

  async function unlinkRAGCollection(projectId: number, collectionId: string) {
    try {
      await apiRequest<void>(`/api/projects/${projectId}/rag/collections/${collectionId}`, () => messages.requestFailed, {
        method: 'DELETE',
      });
      await loadRAGCollections();
    } catch (unlinkError) {
      console.error('Failed to unlink RAG collection:', unlinkError);
    }
  }

  async function indexRAGCollection(collectionId: string) {
    setIndexingCollectionId(collectionId);
    try {
      await apiRequest<void>(`/api/rag/collections/${collectionId}/index`, () => messages.requestFailed, {
        method: 'POST',
      });
    } catch (indexError) {
      console.error('Failed to index RAG collection:', indexError);
    } finally {
      setIndexingCollectionId(null);
    }
  }

  async function searchRAGCollection(collectionId: string, query: string) {
    setSearchingCollectionId(collectionId);
    try {
      const response = await apiRequest<{ results: RAGSearchResult[] }>(`/api/rag/collections/${collectionId}/search`, () => messages.requestFailed, {
        method: 'POST',
        body: JSON.stringify({ query }),
      });
      setRAGSearchResults((prev) => ({
        ...prev,
        [collectionId]: response.results,
      }));
    } catch (searchError) {
      console.error('Failed to search RAG collection:', searchError);
    } finally {
      setSearchingCollectionId(null);
    }
  }

  return {
    // State
    allRAGCollections,
    setAllRAGCollections,
    editingRAGCollectionId,
    setEditingRAGCollectionId,
    createRAGCollectionOpen,
    setCreateRAGCollectionOpen,
    connectRAGCollectionOpen,
    setConnectRAGCollectionOpen,
    creatingRAGCollection,
    linkingCollectionId,
    ragCollectionForm,
    setRAGCollectionForm,
    ragIndexPaths,
    setRAGIndexPaths,
    ragSearchQueries,
    setRAGSearchQueries,
    ragSearchResults,
    setRAGSearchResults,
    ragSearchResultsOpen,
    setRAGSearchResultsOpen,
    activeRAGSearchCollectionId,
    setActiveRAGSearchCollectionId,
    indexingCollectionId,
    searchingCollectionId,
    // Constants
    emptyRAGCollectionForm,
    // Functions
    loadRAGCollections,
    createRAGCollection,
    linkRAGCollection,
    unlinkRAGCollection,
    indexRAGCollection,
    searchRAGCollection,
  };
}
