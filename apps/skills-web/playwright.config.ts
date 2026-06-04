import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 45_000,
  workers: 1,
  use: {
    baseURL: 'http://127.0.0.1:4174',
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'npm run preview:e2e',
    port: 4174,
    reuseExistingServer: process.env.CI ? false : true,
    timeout: 120_000,
  },
});
