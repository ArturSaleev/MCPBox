import { useState } from 'react';
import { apiRequest } from '../utils/api';
import type { OllamaStatus } from '../types';

export function useOllama(messages: { requestFailed: string }) {
  const [ollamaStatus, setOllamaStatus] = useState<OllamaStatus | null>(null);
  const [ollamaRefreshing, setOllamaRefreshing] = useState(false);
  const [selectedOllamaModel, setSelectedOllamaModel] = useState('');

  async function loadOllamaStatus(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setOllamaRefreshing(true);
    }

    try {
      const response = await apiRequest<OllamaStatus>('/api/ollama/status', () => messages.requestFailed);
      setOllamaStatus(response);
    } catch (loadError) {
      console.error('Failed to load Ollama status:', loadError);
    } finally {
      setOllamaRefreshing(false);
    }
  }

  return {
    // State
    ollamaStatus,
    setOllamaStatus,
    ollamaRefreshing,
    setOllamaRefreshing,
    selectedOllamaModel,
    setSelectedOllamaModel,
    // Functions
    loadOllamaStatus,
  };
}
