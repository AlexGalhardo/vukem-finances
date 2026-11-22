import { type Database, getOne } from "../db/database";

/** What the phone remembers about its server, in `meta`. The token is not here: it lives in secure storage. */
export interface Connection {
  serverUrl: string;
  /** The person confirmed plain http to a private address. */
  insecure: boolean;
  deviceId: string | null;
  deviceName: string;
  workspaceUid: string;
  workspaceName: string;
}

const CONNECTION_KEYS = {
  serverUrl: "sync.server_url",
  insecure: "sync.insecure",
  deviceId: "sync.device_id",
  deviceName: "sync.device_name",
  workspaceUid: "sync.workspace_uid",
  workspaceName: "sync.workspace_name",
} as const;

async function setMeta(db: Database, key: string, value: string): Promise<void> {
  await db.run("INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", [
    key,
    value,
  ]);
}

export async function getMeta(db: Database, key: string): Promise<string | null> {
  return (await getOne<{ value: string }>(db, "SELECT value FROM meta WHERE key = ?", [key]))?.value ?? null;
}

export async function loadConnection(db: Database): Promise<Connection | null> {
  const url = await getMeta(db, CONNECTION_KEYS.serverUrl);
  if (url === null) {
    return null;
  }
  return {
    serverUrl: url,
    insecure: (await getMeta(db, CONNECTION_KEYS.insecure)) === "1",
    deviceId: await getMeta(db, CONNECTION_KEYS.deviceId),
    deviceName: (await getMeta(db, CONNECTION_KEYS.deviceName)) ?? "",
    workspaceUid: (await getMeta(db, CONNECTION_KEYS.workspaceUid)) ?? "",
    workspaceName: (await getMeta(db, CONNECTION_KEYS.workspaceName)) ?? "",
  };
}

export async function saveConnection(db: Database, c: Connection): Promise<void> {
  await db.transaction(async (db) => {
    await setMeta(db, CONNECTION_KEYS.serverUrl, c.serverUrl);
    await setMeta(db, CONNECTION_KEYS.insecure, c.insecure ? "1" : "0");
    await setMeta(db, CONNECTION_KEYS.deviceName, c.deviceName);
    await setMeta(db, CONNECTION_KEYS.workspaceUid, c.workspaceUid);
    await setMeta(db, CONNECTION_KEYS.workspaceName, c.workspaceName);
    if (c.deviceId === null) {
      await db.run("DELETE FROM meta WHERE key = ?", [CONNECTION_KEYS.deviceId]);
    } else {
      await setMeta(db, CONNECTION_KEYS.deviceId, c.deviceId);
    }
  });
}

export async function setDeviceId(db: Database, deviceId: string): Promise<void> {
  await setMeta(db, CONNECTION_KEYS.deviceId, deviceId);
}

/** Forgets the server, the device, the cursor and the pauses. Data and the outbox stay. */
export async function clearConnection(db: Database): Promise<void> {
  await db.run("DELETE FROM meta WHERE key LIKE 'sync.%'");
}

export async function getCursor(db: Database): Promise<number> {
  return Number((await getMeta(db, "sync.cursor")) ?? 0);
}

/** Writes the cursor with the page it belongs to, so call it inside the page's transaction. */
export async function setCursor(db: Database, cursor: number): Promise<void> {
  await setMeta(db, "sync.cursor", String(cursor));
}

export async function lastSyncAt(db: Database): Promise<string | null> {
  return getMeta(db, "sync.last_sync_at");
}

export async function setLastSyncAt(db: Database, iso: string): Promise<void> {
  await setMeta(db, "sync.last_sync_at", iso);
}

/** Entities whose pushes are paused because the plan quota refused them, with the server's message. */
export async function pausedEntities(db: Database): Promise<Map<string, string>> {
  const rows = await db.all<{ key: string; value: string }>(
    "SELECT key, value FROM meta WHERE key LIKE 'sync.paused.%'",
  );
  return new Map(rows.map((r) => [r.key.slice("sync.paused.".length), r.value]));
}

export async function pauseEntity(db: Database, entity: string, message: string): Promise<void> {
  await setMeta(db, `sync.paused.${entity}`, message);
}

export async function resumeEntities(db: Database): Promise<void> {
  await db.run("DELETE FROM meta WHERE key LIKE 'sync.paused.%'");
}

/** A discarded problem may leave this phone different from the server: the next run downloads a snapshot. */
export async function requestResync(db: Database): Promise<void> {
  await setMeta(db, "sync.resync_needed", "1");
}

export async function takeResyncRequest(db: Database): Promise<boolean> {
  const wanted = (await getMeta(db, "sync.resync_needed")) === "1";
  if (wanted) {
    await db.run("DELETE FROM meta WHERE key = 'sync.resync_needed'");
  }
  return wanted;
}
