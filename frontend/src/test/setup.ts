// Runs before every test file.
import '@testing-library/jest-dom/vitest'; // adds matchers such as toBeInTheDocument()
import { cleanup } from '@testing-library/react';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { server } from './server';

// Mock Service Worker answers the app's fetch calls at the network level, so
// components are tested exactly as they run, with no real back end.
// "error" makes any request without a handler fail the test loudly.
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  cleanup();
  server.resetHandlers(); // undo per-test overrides
});
afterAll(() => server.close());
