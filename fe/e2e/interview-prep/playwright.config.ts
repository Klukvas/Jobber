import { defineConfig, devices } from "@playwright/test";
export default defineConfig({
  testDir: ".",
  testMatch: "flow.spec.ts",
  timeout: 180000,
  workers: 1,
  retries: 0,
  outputDir:
    "../../../evidence/interview-preparation-2026-10-22/tmp/playwright-results",
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:53081",
    trace: "retain-on-failure",
    channel: "chrome",
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    {
      name: "mobile",
      use: { ...devices["iPhone 13"], defaultBrowserType: "chromium" },
    },
  ],
});
