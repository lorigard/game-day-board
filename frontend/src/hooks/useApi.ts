import { useCallback, useEffect, useState } from 'react';
import { ApiError, fetchApi } from '../api/client';

/** The three states every data-driven part of the page can be in. */
export type ApiState<T> =
  | { status: 'loading' }
  | { status: 'error'; message: string; httpStatus: number }
  | { status: 'success'; data: T; lastUpdated: string | null };

/**
 * Loads a back-end path and tracks its loading, error, and success states.
 *
 * When the path changes (for example, the visitor clicks "Next week" twice
 * quickly), the effect's cleanup marks the older request as stale, so its
 * answer is ignored if it arrives late. Without this, a slow answer for the
 * wrong week could replace the right one on screen.
 */
export function useApi<T>(path: string): { state: ApiState<T>; retry: () => void } {
  const [state, setState] = useState<ApiState<T>>({ status: 'loading' });
  // Bumping this number re-runs the effect, which is how retry works.
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let stale = false;
    setState({ status: 'loading' });

    fetchApi<T>(path)
      .then((env) => {
        if (!stale) setState({ status: 'success', data: env.data, lastUpdated: env.lastUpdated });
      })
      .catch((err: unknown) => {
        if (stale) return;
        const apiErr = err instanceof ApiError ? err : new ApiError(0, 'Something went wrong. Please try again.');
        setState({ status: 'error', message: apiErr.message, httpStatus: apiErr.status });
      });

    return () => {
      stale = true;
    };
  }, [path, attempt]);

  const retry = useCallback(() => setAttempt((n) => n + 1), []);
  return { state, retry };
}
