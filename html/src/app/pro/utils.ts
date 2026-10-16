import { apiRequest } from '../utils/api';
import type { ProPrincipal } from './types';

export async function proAPIRequest<T>(
  token: string,
  input: RequestInfo,
  requestFailedMessage: (status: number) => string,
  init?: RequestInit,
): Promise<T> {
  const trimmedToken = token.trim();
  if (!trimmedToken) {
    throw new Error('Missing Pro admin token');
  }

  return apiRequest<T>(input, requestFailedMessage, {
    ...init,
    headers: {
      Authorization: `Bearer ${trimmedToken}`,
      ...(init?.headers ?? {}),
    },
  });
}

export function hasProScope(principal: ProPrincipal | null, scope: string) {
  if (!principal) {
    return false;
  }
  return principal.scopes.includes('pro:admin') || principal.scopes.includes(scope);
}
