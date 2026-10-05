// Playwright, configured the way codefall-equip's testing track declares it: specs are collected
// from the testing root's `test-cases/` tree by suffix, so a case's own markdown is never collected,
// and everything a run produces goes under `testing/.artifacts/`, which is git-ignored. Specs run
// against the server `scripts/local.sh start` brought up; nothing is started here.
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: 'testing/test-cases',
  testMatch: '**/*.e2e.ts',
  retries: 0,
  workers: 1,
  outputDir: 'testing/.artifacts/test-results',
  reporter: [['list'], ['html', { outputFolder: 'testing/.artifacts/playwright-report', open: 'never' }]],
  use: { baseURL: process.env.BASE_URL ?? 'http://localhost:3000' },
});
