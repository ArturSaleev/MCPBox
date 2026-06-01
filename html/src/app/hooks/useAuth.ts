import { useState } from 'react';
import type { ServerStatus } from '../types';

export function useAuth() {
  const [authOpen, setAuthOpen] = useState(false);
  const [authServerId, setAuthServerId] = useState<number | null>(null);

  const authServer = authServerId !== null ? { id: authServerId } : null;

  function resetAuthServer() {
    setAuthServerId(null);
  }

  return {
    // State
    authOpen,
    setAuthOpen,
    authServerId,
    setAuthServerId,
    authServer,
    // Functions
    resetAuthServer,
  };
}
