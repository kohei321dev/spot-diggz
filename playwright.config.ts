import { defineConfig, devices } from "@playwright/test";
import os from "node:os";
import path from "node:path";

const port = process.env.E2E_PORT ?? "18080";
const baseURL = process.env.E2E_BASE_URL ?? `http://127.0.0.1:${port}`;
const outputDir = process.env.PLAYWRIGHT_OUTPUT_DIR
  ?? path.join(os.tmpdir(), "spot-diggz-playwright-results");
const reportDir = process.env.PLAYWRIGHT_REPORT_DIR
  ?? path.join(os.tmpdir(), "spot-diggz-playwright-report");
const consoleReporter = process.env.CI ? "github" : "line";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: {
    timeout: 7_500,
  },
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  outputDir,
  reporter: [
    [consoleReporter],
    ["html", { outputFolder: reportDir, open: "never" }],
  ],
  use: {
    baseURL,
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
    colorScheme: "light",
    locale: "ja-JP",
    reducedMotion: "reduce",
    screenshot: "only-on-failure",
    timezoneId: "Asia/Tokyo",
    trace: "retain-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "desktop-chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "mobile-chromium",
      use: { ...devices["Pixel 7"] },
    },
  ],
  webServer: process.env.PLAYWRIGHT_EXTERNAL_SERVER === "1"
    ? undefined
    : {
        command: "bash ./scripts/run-e2e-server.sh",
        env: { PORT: port },
        reuseExistingServer: false,
        stderr: "pipe",
        stdout: "pipe",
        timeout: 120_000,
        url: `${baseURL}/readyz`,
      },
});
