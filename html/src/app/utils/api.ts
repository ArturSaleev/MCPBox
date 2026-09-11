type ApiError = {
  error: string;
};

let defaultAuthToken = '';

export function setAPIAuthToken(token: string) {
  defaultAuthToken = token.trim();
}

export async function apiRequest<T>(
  input: RequestInfo,
  requestFailedMessage: (status: number) => string,
  init?: RequestInit,
): Promise<T> {
  const isFormDataBody = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  const headers = new Headers(init?.headers);
  if (!isFormDataBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (defaultAuthToken && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${defaultAuthToken}`);
  }
  const response = await fetch(input, {
    ...init,
    headers,
  });

  if (!response.ok) {
    let message = requestFailedMessage(response.status);
    try {
      const payload = (await response.json()) as ApiError;
      if (payload?.error) {
        message = payload.error;
      }
    } catch {
      // Ignore JSON parsing errors and keep fallback message.
    }

    throw new Error(message);
  }

  return (await response.json()) as T;
}
