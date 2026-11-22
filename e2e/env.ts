declare const process: { env: Record<string, string | undefined> };

export const env = process.env;
export const E2E_PORT = Number(env.E2E_PORT ?? 3917);
export const STORAGE_STATE = "e2e/.tmp/state.json";
// The billing spec runs against a second server (saas edition) and a fake AbacatePay, see e2e/prepare.mjs.
export const E2E_SAAS_PORT = Number(env.E2E_SAAS_PORT ?? 3918);
export const E2E_FAKE_PORT = Number(env.E2E_FAKE_PORT ?? 3919);
export const SAAS_STORAGE_STATE = "e2e/.tmp/state-saas.json";
export const PIX_WEBHOOK_SECRET = "e2e-placeholder-webhook-secret";
export const E2E_ADMIN_EMAIL = "admin@example.test";
export const E2E_ADMIN_PASSWORD = "Example-Local-Passw0rd-xyz";
// The Telegram spec drives the bot through a fake Bot API on this port (e2e/fake-telegram.mjs, started by run.mjs).
export const E2E_TELEGRAM_PORT = Number(env.E2E_TELEGRAM_PORT ?? 3920);
