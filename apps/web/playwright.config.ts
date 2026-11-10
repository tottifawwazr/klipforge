import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 180_000,
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL || "http://127.0.0.1:3002",
    browserName: "chromium",
    channel: process.env.E2E_BROWSER_CHANNEL || undefined,
    viewport: { width: 1280, height: 900 },
    trace: "off",
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    screenshot: "only-on-failure",
  },
  webServer:
    process.env.E2E_START_WEB === "1"
      ? {
          command: "npm run start",
          env: { PORT: "3002", HOSTNAME: "127.0.0.1" },
          url: "http://127.0.0.1:3002/login",
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
        }
      : undefined,
});
