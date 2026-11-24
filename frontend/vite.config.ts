import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// Tests run in UTC so that times printed in assertions are the same on every
// machine, including the Linux machines GitHub Actions uses. This is set here,
// before Vitest starts its test workers, because the workers copy it.
process.env.TZ = 'UTC';

export default defineConfig({
  plugins: [react()],
  server: {
    // During local development, send /api calls to the deployed back end, so
    // front-end work needs no local database. Set VITE_API_PROXY to point
    // somewhere else, such as http://localhost:8080 for a local API.
    proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY ?? 'https://gdb-web-442687404328.us-central1.run.app',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
});
