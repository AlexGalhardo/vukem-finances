import type { Database } from "../db/database";
import { saveConnection } from "./state";
import { checkServerUrl, type ServerUrlProblem, type TokenStore } from "./transport";
import type { DevicePlatform, Transport } from "./types";

export interface ConnectInput {
  serverUrl: string;
  /** The person confirmed that http to a private address is not encrypted. */
  confirmedInsecure: boolean;
  code: string;
  deviceName: string;
  platform: DevicePlatform;
}

export type ConnectResult =
  | { ok: true; workspaceName: string }
  | { ok: false; reason: ServerUrlProblem | "needs_insecure_confirmation" };

export interface ConnectDeps {
  db: Database;
  tokens: TokenStore;
  /** Builds the transport for an address that passed the rules. Pairing needs no token. */
  makeTransport: (baseUrl: string, allowInsecure: boolean) => Transport;
}

/**
 * Pairs with a server: the address is checked first (https, or http to a
 * private address after the person confirmed), the short code is traded for a
 * token, the token goes to secure storage and only the address, the device and
 * the workspace go to SQLite. The code is not kept anywhere.
 */
export async function connectToServer(deps: ConnectDeps, input: ConnectInput): Promise<ConnectResult> {
  const checked = checkServerUrl(input.serverUrl);
  if (!checked.ok) {
    return { ok: false, reason: checked.reason };
  }
  if (!checked.secure && !input.confirmedInsecure) {
    return { ok: false, reason: "needs_insecure_confirmation" };
  }
  const transport = deps.makeTransport(checked.url, !checked.secure);
  const paired = await transport.pair(input.code, input.deviceName.trim(), input.platform);
  await deps.tokens.set(paired.token);
  await saveConnection(deps.db, {
    serverUrl: checked.url,
    insecure: !checked.secure,
    deviceId: paired.device_id,
    deviceName: input.deviceName.trim(),
    workspaceUid: paired.workspace.uid,
    workspaceName: paired.workspace.name,
  });
  return { ok: true, workspaceName: paired.workspace.name };
}
