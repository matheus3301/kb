import { defineConfig } from '@playwright/test';

const previewURL = 'http://127.0.0.1:4321/kb/';

export default defineConfig({
  testDir: './tests',
  outputDir: '../.build/site/results',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['html', { outputFolder: '../.build/site/report', open: 'never' }],
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
    command: 'npm exec -- astro preview --host 127.0.0.1 --port 4321 --ignore-lock',
    url: previewURL,
    env: { KB_SITE_BASE: '/kb/', ASTRO_TELEMETRY_DISABLED: '1' },
    reuseExistingServer: false,
    timeout: 30_000,
  },
});
