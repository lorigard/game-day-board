import type { Envelope } from './types';

/**
 * An error the page can show to a visitor. `status` is the HTTP status, or 0
 * when the request never reached the server (for example, the network is down).
 */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/**
 * Fetches a back-end path and returns its envelope. Every non-success answer
 * becomes an ApiError with a message fit to show on the page, so components
 * need only one way of handling failures (FR-12).
 */
//
// TypeScript note: <T> is a type parameter, like a C# generic. The caller says
// what the data is, for example fetchApi<Team[]>('/api/teams'), and gets a
// typed result back. The type is a promise to the compiler, not a check: the
// JSON is not inspected at run time, which is why the Go side and types.ts
// must agree.
//
// "async" lets the function use "await", which pauses until a Promise (a
// value that arrives later, like a C# Task) is ready, without blocking the page.
export async function fetchApi<T>(path: string): Promise<Envelope<T>> {
  // An absolute URL works the same in the browser and in tests.
  const url = new URL(path, window.location.origin);
  let response: Response;
  try {
    response = await fetch(url, { headers: { Accept: 'application/json' } });
  } catch {
    throw new ApiError(0, 'Could not reach the server. Check your connection and try again.');
  }

  // fetch only throws when the request could not be sent at all. A 404 or a
  // 503 still "succeeds" as far as fetch is concerned, so we check ok (true
  // for status 200 to 299) ourselves.
  if (!response.ok) {
    throw new ApiError(response.status, await errorMessage(response));
  }
  return (await response.json()) as Envelope<T>;
}

/** Reads the shared error shape, {"error": {"code", "message"}}, falling back to a general message. */
async function errorMessage(response: Response): Promise<string> {
  try {
    const body = (await response.json()) as { error?: { message?: string } };
    if (body.error?.message) return body.error.message;
  } catch {
    // The body was not JSON, for example an error page from a proxy.
  }
  return `The server answered with an error (${response.status}). Please try again.`;
}
