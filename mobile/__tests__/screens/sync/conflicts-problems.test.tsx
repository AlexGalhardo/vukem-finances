import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { SyncConflictsScreen } from "../../../src/screens/SyncConflictsScreen";
import { SyncProblemsScreen } from "../../../src/screens/SyncProblemsScreen";
import { ACCOUNT, type Env, openEnv } from "../../helpers/fixtures";
import { renderSync } from "./harness";

async function conflict(env: Env, over: Record<string, string>): Promise<void> {
  const c = { entity: "tag", uid: "x", field: "name", kind: "field", lost: '"Old"', applied: '"New"', ...over };
  await env.db.run(
    "INSERT INTO sync_conflicts (entity, uid, field, kind, lost_value, applied_value, created_at) VALUES (?, ?, ?, ?, ?, ?, 'x')",
    [c.entity, c.uid, c.field, c.kind, c.lost, c.applied],
  );
}

async function problem(env: Env, over: Record<string, string>): Promise<void> {
  const p = {
    entity: "account",
    uid: "x",
    op: "upsert",
    code: "unknown_field",
    message: "no such field",
    fields: "{}",
    ...over,
  };
  await env.db.run(
    "INSERT INTO sync_problems (entity, uid, op_id, op, code, message, fields, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'x')",
    [p.entity, p.uid, `op-${p.uid}`, p.op, p.code, p.message, p.fields],
  );
}

describe("the conflicts screen", () => {
  it("says there is nothing to review", async () => {
    await renderSync(() => <SyncConflictsScreen />);
    expect(await screen.findByText("No conflicts.")).toBeTruthy();
  });

  it("shows both values and applies the replaced value again as a normal edit", async () => {
    const env = await openEnv();
    const tag = await env.repo.create("tag", { name: "New", color: "" });
    await env.repo.clearOutbox();
    await conflict(env, { uid: tag.uid });
    await renderSync(() => <SyncConflictsScreen />, { env });

    expect(await screen.findByText("tag name: kept New, replaced Old")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Apply the replaced value again" }));

    await waitFor(async () => expect((await env.repo.get("tag", tag.uid))?.name).toBe("Old"));
    expect(await env.repo.outbox()).toMatchObject([{ entity: "tag", op: "upsert", fields: { name: "Old" } }]);
    expect(await screen.findByText("No conflicts.")).toBeTruthy();
  });

  it("restores a row the server deleted and applies the dropped value", async () => {
    const env = await openEnv();
    const tag = await env.repo.create("tag", { name: "Kept", color: "" });
    await env.repo.remove("tag", tag.uid);
    await env.repo.clearOutbox();
    await conflict(env, { uid: tag.uid, kind: "edited_deleted_row", lost: '"Mine"', applied: "null" });
    await renderSync(() => <SyncConflictsScreen />, { env });

    expect(await screen.findByText(/your change was dropped because the item was deleted on the server/u)).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Restore the item and apply my value" }));
    await waitFor(async () => expect((await env.repo.get("tag", tag.uid))?.name).toBe("Mine"));
    expect((await env.repo.outbox()).map((o) => o.op)).toEqual(["restore", "upsert"]);
  });

  it("dismisses a conflict without changing the data", async () => {
    const env = await openEnv();
    const tag = await env.repo.create("tag", { name: "New", color: "" });
    await conflict(env, { uid: tag.uid });
    await renderSync(() => <SyncConflictsScreen />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Dismiss" }));
    expect(await screen.findByText("No conflicts.")).toBeTruthy();
    expect((await env.repo.get("tag", tag.uid))?.name).toBe("New");
  });
});

describe("the problems screen", () => {
  it("says there is nothing to review", async () => {
    await renderSync(() => <SyncProblemsScreen />);
    expect(await screen.findByText("No problems.")).toBeTruthy();
  });

  it("sends a refused change again as a new queued change", async () => {
    const env = await openEnv();
    const account = await env.repo.create("account", ACCOUNT);
    await env.repo.clearOutbox();
    await problem(env, { uid: account.uid, fields: '{"name":"Wanted"}' });
    await renderSync(() => <SyncProblemsScreen />, { env });

    expect(await screen.findByText("account: no such field")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Try again" }));
    await waitFor(async () => expect(await env.repo.outboxCount()).toBe(1));
    expect((await env.repo.outbox())[0]).toMatchObject({
      entity: "account",
      uid: account.uid,
      fields: { name: "Wanted" },
    });
    expect(await screen.findByText("No problems.")).toBeTruthy();
  });

  it("discards a refused create by removing the row the server never had", async () => {
    const env = await openEnv();
    const account = await env.repo.create("account", ACCOUNT);
    await env.repo.clearOutbox();
    await problem(env, { uid: account.uid });
    await renderSync(() => <SyncProblemsScreen />, { env });

    await fireEvent.press(await screen.findByRole("button", { name: "Discard" }));
    await waitFor(async () => expect(await env.repo.list("account")).toEqual([]));
    expect(await screen.findByText("No problems.")).toBeTruthy();
  });

  it("discards a refused edit of a synced row by asking for a fresh copy on the next sync", async () => {
    const env = await openEnv();
    const account = await env.repo.create("account", ACCOUNT);
    await env.db.run("UPDATE accounts SET server_seq = 4, dirty = 0 WHERE uid = ?", [account.uid]);
    await env.repo.clearOutbox();
    await problem(env, { uid: account.uid });
    await renderSync(() => <SyncProblemsScreen />, { env });

    await fireEvent.press(await screen.findByRole("button", { name: "Discard" }));
    await waitFor(async () =>
      expect(await env.db.all("SELECT value FROM meta WHERE key = 'sync.resync_needed'")).toEqual([{ value: "1" }]),
    );
    expect(await env.repo.list("account")).toHaveLength(1);
  });

  it("lists the references that name nothing in a data check", async () => {
    const env = await openEnv();
    await env.db.run(
      "INSERT INTO sync_ref_missing (entity, uid, field, missing) VALUES ('transaction', 't1', 'account_uid', 'gone')",
    );
    await renderSync(() => <SyncProblemsScreen />, { env });
    expect(await screen.findByRole("header", { name: "Data check" })).toBeTruthy();
    expect(screen.getByText("transaction refers to something this phone does not have (account_uid).")).toBeTruthy();
  });
});
