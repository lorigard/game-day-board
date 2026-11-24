import { timeAgo } from '../lib/format';

/** Shown while data loads. role="status" makes screen readers announce it politely. */
export function Loading({ what }: { what: string }) {
  return (
    <p role="status" className="muted">
      Loading {what}…
    </p>
  );
}

/**
 * Shown when data could not be loaded (FR-12): a clear message and a way to
 * try again, never a blank screen. role="alert" makes screen readers announce it.
 */
export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="error-state">
      <p>{message}</p>
      <button type="button" onClick={onRetry}>
        Try again
      </button>
    </div>
  );
}

/**
 * Says how old the data is (FR-9), so stale data is visible rather than hidden.
 * The exact time is in the title, which shows on hover.
 */
export function LastUpdated({ at }: { at: string | null }) {
  if (!at) return <p className="muted small">Data has not been loaded yet.</p>;
  return (
    <p className="muted small">
      Data updated <time dateTime={at} title={new Date(at).toLocaleString()}>{timeAgo(at)}</time>
    </p>
  );
}
