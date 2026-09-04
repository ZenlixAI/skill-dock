import { defineConfig } from '@playwright/test';

const catalogCommand = process.env.PLAYWRIGHT_CATALOG_COMMAND?.trim();
const catalogPort = Number(process.env.PLAYWRIGHT_CATALOG_PORT || 0);
const baseURL = process.env.PLAYWRIGHT_BASE_URL?.trim() || 'http://127.0.0.1:4174';
const webServers = process.env.PLAYWRIGHT_BASE_URL?.trim()
  ? []
  : [
      {
        command: 'npm run preview:e2e',
        port: 4174,
        reuseExistingServer: process.env.CI ? false : true,
        timeout: 120_000,
      },
    ];

if (catalogCommand) {
  if (!catalogPort) {
    throw new Error('PLAYWRIGHT_CATALOG_PORT must be set when PLAYWRIGHT_CATALOG_COMMAND is provided.');
  }
  webServers.unshift({
    command: catalogCommand,
    port: catalogPort,
    reuseExistingServer: process.env.CI ? false : true,
    timeout: 120_000,
  });
}

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 45_000,
  workers: 1,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  webServer: webServers,
});
