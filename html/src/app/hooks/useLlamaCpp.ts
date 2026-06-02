import { useState } from 'react';
import { apiRequest } from '../utils/api';
import type { LlamaCppStatus } from '../types';

export function useLlamaCpp(messages: { requestFailed: string }) {
  const [llamaCppStatus, setLlamaCppStatus] = useState<LlamaCppStatus | null>(null);
  const [llamaCppRefreshing, setLlamaCppRefreshing] = useState(false);

  async function loadLlamaCppStatus(options?: { silent?: boolean }) {
    if (!options?.silent) {
      setLlamaCppRefreshing(true);
    }

    try {
      const response = await apiRequest<LlamaCppStatus>('/api/llamacpp/status', () => messages.requestFailed);
      setLlamaCppStatus(response);
    } catch (loadError) {
      console.error('Failed to load llama.cpp status:', loadError);
    } finally {
      setLlamaCppRefreshing(false);
    }
  }

  return {
    llamaCppStatus,
    setLlamaCppStatus,
    llamaCppRefreshing,
    setLlamaCppRefreshing,
    loadLlamaCppStatus,
  };
}
