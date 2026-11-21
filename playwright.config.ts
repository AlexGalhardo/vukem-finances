import { defineConfig, devices } from "@playwright/test";
import { E2E_ADMIN_EMAIL, E2E_ADMIN_PASSWORD, E2E_PORT, STORAGE_STATE, env } from "./e2e/env";

// The bundled Chromium is the default (CI). Set E2E_CHANNEL=chrome to use the Chrome installed on the machine.
const channel = env.E2E_CHANNEL;

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.{spec,setup}.ts",
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(env.CI),
  retries: 0,
  reporter: env.CI ? [["github"], ["list"]] : "list",
  use: {
    baseURL: `http://localhost:${E2E_PORT}`,
    trace: "retain-on-failure",
  },
  projects: [
    { name: "setup", testMatch: /auth.setup.ts/ },
    {
      name: "chromium",
      testMatch: /.*.spec.ts/,
      dependencies: ["setup"],
      use: { ...devices["Desktop Chrome"], ...(channel ? { channel } : {}), storageState: STORAGE_STATE },
    },
  ],
  // The app has no `serve` subcommand: prepare.mjs builds ./cmd/app, migrates and seeds a throwaway database,
  // then the binary starts in development with the demo data and a fixed administrator.
  webServer: {
    command: "bun e2e/prepare.mjs && bun e2e/run.mjs",
    url: `http://localhost:${E2E_PORT}/login`,
    reuseExistingServer: false,
    timeout: 180_000,
    env: {
      E2E_PORT: String(E2E_PORT),
      E2E_ADMIN_EMAIL,
      E2E_ADMIN_PASSWORD,
    },
  },
});
