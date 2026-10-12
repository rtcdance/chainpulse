import { defineConfig } from '@playwright/test';

// Playwright Test reads its configuration from THIS file.
// The equivalent settings previously lived in a "playwright" key in package.json,
// which Playwright does not read — as a result `npm test` ignored testDir and
// globbed the whole repo (picking up the frontend's vitest unit tests) and
// discovered 0 acceptance tests.
//
// The acceptance suite targets an already-running stack. Start it first:
//   docker compose -f docker/docker-compose.yml up -d
// then: npm test
// See test/acceptance/README.md.
export default defineConfig({
  testDir: './test/acceptance',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        browserName: 'chromium',
      },
    },
  ],
});
