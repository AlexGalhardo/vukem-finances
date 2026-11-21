declare const process: { env: Record<string, string | undefined> };

export const env = process.env;
export const E2E_PORT = Number(env.E2E_PORT ?? 3917);
export const STORAGE_STATE = "e2e/.tmp/state.json";
export const E2E_ADMIN_EMAIL = "admin@example.test";
export const E2E_ADMIN_PASSWORD = "Example-Local-Passw0rd-xyz";
