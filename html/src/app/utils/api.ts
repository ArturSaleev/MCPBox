export class APIRequestError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = 'APIRequestError';
  }
}

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

    throw new APIRequestError(message, response.status);
  }

  return (await response.json()) as T;
}

export async function downloadAPIFile(input: string, filename: string) {
  const headers = new Headers();
  if (defaultAuthToken) headers.set('Authorization', `Bearer ${defaultAuthToken}`);
  const response = await fetch(input, { headers });
  if (!response.ok) {
    const error = await response.json().catch(() => null);
    throw new APIRequestError(error?.error || `HTTP ${response.status}`, response.status);
  }
  const url = URL.createObjectURL(await response.blob());
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
