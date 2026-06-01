type ApiError = {
  error: string;
};

export async function apiRequest<T>(
  input: RequestInfo,
  requestFailedMessage: (status: number) => string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(input, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
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
