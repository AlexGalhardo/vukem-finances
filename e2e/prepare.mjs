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
  // The AI spec points the openai_compatible provider at a fake server on this machine (personal edition only).
  AI_ALLOW_PRIVATE_HOSTS: "true",
  // The Telegram spec runs the bot in polling mode (the personal default) against e2e/fake-telegram.mjs. The token
  // is a made-up placeholder; nothing here reaches the real Bot API.
  TELEGRAM_BOT_TOKEN: "123456:e2e-placeholder-token",
  TELEGRAM_BOT_USERNAME: "e2e_finance_bot",
  TELEGRAM_API_BASE: `http://127.0.0.1:${process.env.E2E_TELEGRAM_PORT ?? "3920"}`,
};

function run(cmd, args, env = process.env) {
  const result = spawnSync(cmd, args, { stdio: "inherit", env });
  if (result.status !== 0) process.exit(result.status ?? 1);
}

run("bun", ["run", "build"]);
run("go", ["build", "-o", binary, "./cmd/app"]);
run(binary, ["migrate"], appEnv);
run(binary, ["db:seed", "FinanceAdminSeeder", "FinanceDemoSeeder"], appEnv);

// The billing spec runs against the saas edition on its own port and database, with PIX pointed at a fake
// AbacatePay (e2e/fake-abacatepay.mjs, started by run.mjs). No real provider is ever called and no real
// credential is used: the values below are made up.
const saasPort = process.env.E2E_SAAS_PORT ?? "3918";
export const saasEnv = {
  ...appEnv,
  APP_EDITION: "saas",
  TELEGRAM_BOT_TOKEN: "",
  PORT: saasPort,
  APP_URL: `http://localhost:${saasPort}`,
  DATABASE_URL: `file:${join(dir, "db-saas.sqlite").replaceAll("\\", "/")}`,
  ABACATEPAY_API_KEY: "abc_dev_e2e_placeholder_key",
  ABACATEPAY_WEBHOOK_SECRET: "e2e-placeholder-webhook-secret",
  ABACATEPAY_API_BASE: `http://127.0.0.1:${process.env.E2E_FAKE_PORT ?? "3919"}`,
  PIX_PRICE_PRO_MONTHLY_CENTS: "1990",
};
run(binary, ["migrate"], saasEnv);
run(binary, ["db:seed", "FinanceAdminSeeder"], saasEnv);

// The key and database are shared with run.mjs through this file, so both use the same ones.
import { writeFileSync } from "node:fs";
writeFileSync(join(dir, "env.json"), JSON.stringify({ binary, appEnv, saasEnv }));
