import { applyPage } from "../../src/sync/apply";
import { findMissingRefs, integrityPass, listMissingRefs } from "../../src/sync/integrity";
import { getCursor } from "../../src/sync/state";
import { openEnv } from "../helpers/fixtures";
import { FakeTransport } from "./fake-transport";
import { accountRow, page, tagRow, transactionRow, upsert } from "./wire";

const NOW = (): string => "2026-11-24T12:00:00.000Z";
const A1 = "01JD5Z8K3M0000000000000A01";
const T1 = "01JD5Z8K3M0000000000000T01";
const G1 = "01JD5Z8K3M0000000000000G01";

describe("the integrity pass", () => {
  it("finds nothing when every reference is held, a deleted parent included", async () => {
    const env = await openEnv();
    await applyPage(
      env.db,
      page([upsert(1, "account", accountRow(A1)), upsert(2, "transaction", transactionRow(T1, A1))]),
      NOW(),
    );
    expect(await findMissingRefs(env.db)).toEqual([]);
    const transport = new FakeTransport();
    expect(await integrityPass(env.db, transport, NOW)).toEqual({ missing: 0, repulled: false });
    expect(transport.calls).toEqual([]);
  });

  it("pulls once more from the cursor minus one page when a parent is missing, and finds it", async () => {
    const env = await openEnv();
    await applyPage(env.db, page([upsert(600, "transaction", transactionRow(T1, A1, { tag_uids: [G1] }))]), NOW());
    expect((await findMissingRefs(env.db)).map((i) => [i.field, i.missing])).toEqual([
      ["account_uid", A1],
      ["tag_uids", G1],
    ]);

    const asked: number[] = [];
    const transport = new FakeTransport({
      pull: (cursor) => {
        asked.push(cursor);
        return page([upsert(601, "account", accountRow(A1)), upsert(602, "tag", tagRow(G1))], 602);
      },
    });
    expect(await integrityPass(env.db, transport, NOW, 500)).toEqual({ missing: 0, repulled: true });
    expect(asked).toEqual([100]);
    expect(await getCursor(env.db)).toBe(602);
    expect(await listMissingRefs(env.db)).toEqual([]);
  });

  it("lists what is still missing and never deletes it; the cursor does not move back", async () => {
    const env = await openEnv();
    await applyPage(env.db, page([upsert(900, "transaction", transactionRow(T1, A1))]), NOW());
    const transport = new FakeTransport({ pull: () => page([], 400) });

    expect(await integrityPass(env.db, transport, NOW, 500)).toEqual({ missing: 1, repulled: true });
    expect(await listMissingRefs(env.db)).toEqual([
      { entity: "transaction", uid: T1, field: "account_uid", missing: A1 },
    ]);
    expect(await env.repo.get("transaction", T1)).not.toBeNull();
    expect(await getCursor(env.db)).toBe(900);

    // A later pass that finds the parent clears the list.
    await applyPage(env.db, page([upsert(901, "account", accountRow(A1))]), NOW());
    expect(await integrityPass(env.db, transport, NOW)).toEqual({ missing: 0, repulled: false });
    expect(await listMissingRefs(env.db)).toEqual([]);
  });
});
