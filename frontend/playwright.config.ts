import { defineConfig, devices } from '@playwright/test';

// DOM/E2E tests run against the running edge (Traefik) by default, or any URL
// via PW_BASE_URL. Mobile viewport because the app is mobile-first.
export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'line' : 'list',
  use: {
    baseURL: process.env.PW_BASE_URL ?? 'http://localhost',
    colorScheme: 'light',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'mobile', use: { ...devices['Pixel 7'] } }],
});
