import { defineConfig } from '@playwright/test';

const siteBase = process.env.KB_SITE_BASE ?? '/kb/';
const previewURL = `http://127.0.0.1:4321${siteBase}`;
const evidenceBase = siteBase === '/' ? 'root' : 'kb';

export default defineConfig({
  testDir: './tests',
  outputDir: `../.build/site/${evidenceBase}/results`,
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['html', { outputFolder: `../.build/site/${evidenceBase}/report`, open: 'never' }],
  ],
  use: {
    baseURL: previewURL,
    browserName: 'chromium',
    reducedMotion: 'reduce',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1440, height: 1000 } } },
    {
      name: 'mobile',
      use: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true },
    },
  ],
  webServer: {
    command: 'pnpm exec astro preview --host 127.0.0.1 --port 4321 --ignore-lock',
    url: previewURL,
    env: { KB_SITE_BASE: siteBase, ASTRO_TELEMETRY_DISABLED: '1' },
    reuseExistingServer: false,
    timeout: 30_000,
  },
});
