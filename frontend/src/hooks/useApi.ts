import { useCallback, useEffect, useState } from 'react';
import { ApiError, fetchApi } from '../api/client';

/**
 * The three states every data-driven part of the page can be in.
 *
 * TypeScript note: this is a "discriminated union". The status field says
 * which of the three shapes you have, and after a check like
 * `if (state.status === 'success')` the compiler knows `state.data` exists.
 * It makes "showing data that has not loaded yet" a compile error.
 */
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
 *
 * React note: this is a custom "hook", a function whose name starts with
 * "use" and that calls React's own hooks. Hooks let a component keep data
 * between renders. Any component can call useApi(path) and get back the
 * current state, and React re-renders that component whenever it changes.
 *
 * A request that is already in flight is not canceled, only ignored. The
 * usual tool for canceling, AbortController, was tried first, but the test
 * setup (jsdom plus Node's fetch) rejects jsdom's AbortSignal, which made
 * every page test fail. Ignoring a late answer is just as safe for the page;
 * the cost is only a small request that finishes unseen.
 */
export function useApi<T>(path: string): { state: ApiState<T>; retry: () => void } {
  // useState keeps a value between renders. It returns the value and a
  // setter; calling the setter stores the new value and re-renders.
  const [state, setState] = useState<ApiState<T>>({ status: 'loading' });
  // Bumping this number re-runs the effect, which is how retry works.
  const [attempt, setAttempt] = useState(0);

  // useEffect runs code after React has drawn the page: the place for work
  // that reaches outside React, such as network requests. The list at the end,
  // [path, attempt], says when to run it again: whenever either one changes.
  // The function it returns ("cleanup") runs before the next run.
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

  // useCallback hands back the same function on every render, rather than a
  // new one each time, so components that receive it do not redraw needlessly.
  // setAttempt((n) => n + 1) uses the "updater" form: React passes in the
  // latest value, which is safe even if two updates happen close together.
  const retry = useCallback(() => setAttempt((n) => n + 1), []);
  return { state, retry };
}
