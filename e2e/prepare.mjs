// Builds the app and prepares a throwaway SQLite database with the demo seed. Run by playwright.config.ts.
import { spawnSync } from "node:child_process";
import { mkdirSync, rmSync } from "node:fs";
import { join, resolve } from "node:path";
import { randomBytes } from "node:crypto";

const dir = resolve("e2e/.tmp");
rmSync(dir, { recursive: true, force: true });
mkdirSync(dir, { recursive: true });
export const binary = join(dir, process.platform === "win32" ? "app.exe" : "app");
export const appEnv = {
  ...process.env,
  APP_NAME: "vukem-finances",
  APP_ENV: "development",
  APP_KEY: `base64:${randomBytes(32).toString("base64")}`,
  APP_DEBUG: "true",
  APP_EDITION: process.env.E2E_EDITION ?? "personal",
  PORT: process.env.E2E_PORT ?? "3917",
  APP_URL: `http://localhost:${process.env.E2E_PORT ?? "3917"}`,
  DATABASE_URL: `file:${join(dir, "db.sqlite").replaceAll("\\", "/")}`,
  ADMIN_EMAILS: process.env.E2E_ADMIN_EMAIL,
  FINANCE_ADMIN_EMAIL: process.env.E2E_ADMIN_EMAIL,
  FINANCE_ADMIN_PASSWORD: process.env.E2E_ADMIN_PASSWORD,
  MAIL_MAILER: "log",
};

function run(cmd, args, env = process.env) {
  const result = spawnSync(cmd, args, { stdio: "inherit", env });
  if (result.status !== 0) process.exit(result.status ?? 1);
}

run("bun", ["run", "build"]);
run("go", ["build", "-o", binary, "./cmd/app"]);
run(binary, ["migrate"], appEnv);
run(binary, ["db:seed", "FinanceAdminSeeder", "FinanceDemoSeeder"], appEnv);
// The key and database are shared with run.mjs through this file, so both use the same ones.
import { writeFileSync } from "node:fs";
writeFileSync(join(dir, "env.json"), JSON.stringify({ binary, appEnv }));
