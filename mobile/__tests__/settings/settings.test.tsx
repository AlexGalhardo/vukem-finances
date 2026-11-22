import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import type { Sharer } from "../../src/backup/share";
import type { LockPlatform, LockStore } from "../../src/lock/lock";
import { OUTBOX_SOFT_CAP } from "../../src/db/repository";
import { type SettingsServices, SettingsScreen } from "../../src/screens/SettingsScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv } from "../helpers/fixtures";
import schema from "../../../docs/export-format.schema.json";
import { type Schema, validate } from "../helpers/json-schema";

let env: Env;
let enabled: boolean;
let available: boolean;
let passes: boolean;
let shared: { name: string; content: string }[];
let shareFails: boolean;
let services: SettingsServices;

beforeEach(async () => {
  env = await openEnv();
  enabled = false;
  available = true;
  passes = true;
  shared = [];
  shareFails = false;
  const lockStore: LockStore = {
    isEnabled: async () => enabled,
    setEnabled: async (v) => {
      enabled = v;
    },
  };
  const lockPlatform: LockPlatform = { available: async () => available, authenticate: async () => passes };
  const sharer: Sharer = {
    shareText: async (name, content) => {
      if (shareFails) throw new Error("no share");
      shared.push({ name, content });
    },
  };
  services = { lockStore, lockPlatform, sharer, version: "0.1.0" };
});

const metaValue = async (key: string): Promise<string | undefined> =>
  (await env.db.all<{ value: string }>("SELECT value FROM meta WHERE key = ?", [key]))[0]?.value;

describe("Settings", () => {
  it("changes the language and the text follows, and the choice is kept", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("radio", { name: "Español" }));
    expect(await screen.findByRole("header", { name: "Ajustes" })).toBeTruthy();
    expect(await metaValue("locale")).toBe("es");
  });

  it("changes theme and currency", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("radio", { name: "Dark" }));
    await fireEvent.press(screen.getByRole("radio", { name: "EUR" }));
    await waitFor(async () => expect(await metaValue("theme")).toBe("dark"));
    expect(await metaValue("currency")).toBe("EUR");
  });

  it("saves a valid time zone and refuses an unknown one", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    const field = await screen.findByLabelText("Time zone");
    await fireEvent.changeText(field, "Mars/Olympus");
    await fireEvent.press(screen.getAllByRole("button", { name: "Save" })[0] as never);
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();
    expect(await metaValue("timezone")).toBeUndefined();

    await fireEvent.changeText(field, "America/Sao_Paulo");
    await fireEvent.press(screen.getAllByRole("button", { name: "Save" })[0] as never);
    await waitFor(async () => expect(await metaValue("timezone")).toBe("America/Sao_Paulo"));
  });

  it("states that the lock is an access gate, not disk encryption", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    expect(await screen.findByText(/It is an access gate, not disk encryption/)).toBeTruthy();
  });

  it("turns the lock on after a successful check, keeping the setting in secure storage only", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("checkbox", { name: "App lock" }));
    await waitFor(() => expect(enabled).toBe(true));
    const everything = await env.db.all<{ key: string; value: string }>("SELECT key, value FROM meta");
    expect(JSON.stringify(everything).toLowerCase()).not.toContain("lock");

    await fireEvent.press(screen.getByRole("checkbox", { name: "App lock" }));
    await waitFor(() => expect(enabled).toBe(false));
  });

  it("does not turn the lock on without a device screen lock, nor when the check fails", async () => {
    available = false;
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("checkbox", { name: "App lock" }));
    expect(await screen.findByText("Set a screen lock or biometrics on this device to use the app lock.")).toBeTruthy();
    expect(enabled).toBe(false);

    available = true;
    passes = false;
    await fireEvent.press(screen.getByRole("checkbox", { name: "App lock" }));
    await waitFor(() => expect(enabled).toBe(false));
  });

  it("shares a backup that validates against the export schema", async () => {
    await env.repo.create("account", { ...ACCOUNT, opening_cents: 5000 });
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Export a backup" }));
    await waitFor(() => expect(shared).toHaveLength(1));
    expect(shared[0]?.name).toBe("vukem-finances-backup-2027-03-15.json");
    const doc = JSON.parse(shared[0]?.content ?? "{}") as { accounts: unknown[]; workspace: { currency: string } };
    expect(validate(schema as unknown as Schema, doc)).toEqual([]);
    expect(doc.accounts).toHaveLength(1);
    expect(doc.workspace.currency).toBe("USD");
  });

  it("tells the person when the backup could not be shared", async () => {
    shareFails = true;
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Export a backup" }));
    expect(await screen.findByText("The backup could not be shared.")).toBeTruthy();
  });

  it("shows the version, the license and the server placeholder", async () => {
    await renderApp(<SettingsScreen services={services} />, { env });
    expect(await screen.findByText("Version 0.1.0")).toBeTruthy();
    expect(screen.getByText("Open source license: MIT")).toBeTruthy();
    expect(screen.getByText(/Syncing with a server arrives in a later version/)).toBeTruthy();
  });
});

describe("the sync queue setting", () => {
  it("counts the waiting changes and asks before turning the queue off, then discards it and stops queuing", async () => {
    await env.repo.create("account", ACCOUNT);
    await renderApp(<SettingsScreen services={services} />, { env });
    expect(await screen.findByText("1 changes waiting to sync")).toBeTruthy();

    await fireEvent.press(screen.getByRole("checkbox", { name: "Keep the sync queue" }));
    expect(await screen.findByText("Turn off the sync queue? The 1 waiting changes will be discarded.")).toBeTruthy();
    expect(await env.repo.outboxCount()).toBe(1);

    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await waitFor(async () => expect(await env.repo.outboxCount()).toBe(0));
    await env.repo.create("account", { ...ACCOUNT, name: "Later" });
    expect(await env.repo.outboxCount()).toBe(0);
    expect(await metaValue("keep_sync_queue")).toBe("0");

    await fireEvent.press(screen.getByRole("checkbox", { name: "Keep the sync queue" }));
    await waitFor(async () => expect(await metaValue("keep_sync_queue")).toBe("1"));
    await env.repo.create("account", { ...ACCOUNT, name: "Queued again" });
    expect(await env.repo.outboxCount()).toBe(1);
  });

  it("cancelling keeps the queue", async () => {
    await env.repo.create("account", ACCOUNT);
    await renderApp(<SettingsScreen services={services} />, { env });
    await fireEvent.press(await screen.findByRole("checkbox", { name: "Keep the sync queue" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Cancel" }));
    expect(await env.repo.outboxCount()).toBe(1);
    expect(await metaValue("keep_sync_queue")).toBeUndefined();
  });

  it("offers to drop the oldest entries past the cap, and only after a confirmation", async () => {
    const rows = Array.from(
      { length: OUTBOX_SOFT_CAP + 3 },
      (_, i) => `('op${i}', 'tag', 'u${i}', 'upsert', '{}', 'x')`,
    );
    await env.db.exec(`INSERT INTO outbox (op_id, entity, uid, op, fields, created_at) VALUES ${rows.join(",")}`);
    await renderApp(<SettingsScreen services={services} />, { env });

    expect(await screen.findByText("The sync queue is over 5000 entries. Drop the oldest ones?")).toBeTruthy();
    expect(await env.repo.outboxCount()).toBe(OUTBOX_SOFT_CAP + 3);

    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete" }));
    await waitFor(async () => expect(await env.repo.outboxCount()).toBe(OUTBOX_SOFT_CAP));
  });
});
