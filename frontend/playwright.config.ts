import { defineConfig, devices } from '@playwright/test'

// E2E configuration for the Knot export/import round-trip suite.
//
// The webServer boots the real production stack: e2e/run-env.sh builds the
// frontend bundle, embeds it into the Go CLI binary (embedfrontend build
// tag) and serves everything on port 3999 against an isolated SQLite
// database. One shared server + database backs the whole run, so tests run
// sequentially in a single worker.
export default defineConfig({
  testDir: './e2e',
  outputDir: './test-results',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  forbidOnly: !!process.env.CI,
  // Trace + screenshots are kept for every run so green-run evidence is
  // available in the HTML report (playwright-report/).
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:3999',
    acceptDownloads: true,
    trace: 'on',
    screenshot: 'on',
  },
  webServer: {
    command: 'bash e2e/run-env.sh',
    url: 'http://localhost:3999',
    reuseExistingServer: !process.env.CI,
    timeout: 240_000,
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
