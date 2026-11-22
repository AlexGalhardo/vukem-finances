import type { ReactElement } from "react";
import { SyncProvider } from "../../../src/sync/context";
import { SyncController } from "../../../src/sync/controller";
import { saveConnection } from "../../../src/sync/state";
import type { TokenStore } from "../../../src/sync/transport";
import type { Clock, Transport } from "../../../src/sync/types";
import { renderApp } from "../../helpers/app";
import { type Env, openEnv } from "../../helpers/fixtures";
import { FakeTransport } from "../../sync/fake-transport";

const clock: Clock = {
  now: () => Date.parse("2026-11-24T12:00:00.000Z"),
  setTimeout: () => 0,
  clearTimeout: () => undefined,
};

export interface SyncUi {
  env: Env;
  controller: SyncController;
  transport: FakeTransport;
  tokens: TokenStore & { value: string | null };
}

export const SERVER_INFO = {
  protocol: { min: 1, max: 1 },
  server_time: "",
  workspace: { uid: "1", currency: "USD", timezone: "UTC" },
  entities: [],
  limits: { max_mutations: 200, pull_max_limit: 1000, snapshot_max_rows: 1000 },
};

export interface SetupOptions {
  /** Starts with a saved connection (a paired phone). */
  connected?: boolean;
  /** The first sync already happened (no merge choice). */
  synced?: boolean;
  transport?: FakeTransport;
  env?: Env;
}

/** A controller over an in-memory database and a scripted transport, rendered inside the app providers. */
export async function renderSync(screen: (ui: SyncUi) => ReactElement, options: SetupOptions = {}): Promise<SyncUi> {
  const env = options.env ?? (await openEnv());
  const transport =
    options.transport ??
    new FakeTransport({
      info: () => SERVER_INFO,
      pull: () => ({ changes: [], next_cursor: 0, has_more: false, server_time: "" }),
      push: () => [],
    });
  const tokens: TokenStore & { value: string | null } = {
    value: options.connected === true ? "stored-token" : null,
    get: async () => tokens.value,
    set: async (t) => {
      tokens.value = t;
    },
    clear: async () => {
      tokens.value = null;
    },
  };
  if (options.connected === true) {
    await saveConnection(env.db, {
      serverUrl: "https://money.example.com",
      insecure: false,
      deviceId: "DEVICE1",
      deviceName: "Ana's phone",
      workspaceUid: "1",
      workspaceName: "Ana household",
    });
    if (options.synced !== false) {
      await env.db.run("INSERT INTO meta (key, value) VALUES ('sync.first_sync_done', '1')");
    }
  }
  const controller = new SyncController({
    db: env.db,
    repo: env.repo,
    tokens,
    makeTransport: (): Transport => transport,
    clock,
    random: { next: () => 0.5 },
    defaultDeviceName: "Test phone",
    platform: "android",
  });
  await controller.engine.start();
  controller.engine.stop();
  const ui: SyncUi = { env, controller, transport, tokens };
  await renderApp(<SyncProvider controller={controller}>{screen(ui)}</SyncProvider>, { env });
  return ui;
}
