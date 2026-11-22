import { LATEST_VERSION, MIGRATIONS, type Migration, migrate, schemaVersion } from "../../src/db/migrations";
import { MemoryDatabase, openMemoryDatabase } from "../helpers/memory-db";

const NOW = "2026-11-21T10:00:00.000Z";

async function tables(db: MemoryDatabase): Promise<string[]> {
  const rows = await db.all<{ name: string }>("SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name");
  return rows.map((r) => r.name);
}

describe("migrate", () => {
  it("creates every table of the local schema", async () => {
    const db = await openMemoryDatabase();
    expect(await schemaVersion(db)).toBe(LATEST_VERSION);
    expect(await tables(db)).toEqual(
      expect.arrayContaining([
        "accounts",
        "budgets",
        "categories",
        "credit_cards",
        "meta",
        "outbox",
        "recurrences",
        "tags",
        "transaction_tags",
        "transactions",
      ]),
    );
  });

  it("has no foreign keys", async () => {
    const db = await openMemoryDatabase();
    const sql = await db.all<{ sql: string }>("SELECT sql FROM sqlite_master WHERE sql LIKE '%REFERENCES%'");
    expect(sql).toEqual([]);
  });

  it("is idempotent", async () => {
    const db = await openMemoryDatabase();
    expect(await migrate(db)).toBe(LATEST_VERSION);
    expect(await schemaVersion(db)).toBe(LATEST_VERSION);
  });

  it("numbers its migrations 1..n in order", () => {
    expect(MIGRATIONS.map((m) => m.version)).toEqual(MIGRATIONS.map((_, i) => i + 1));
  });

  it.each(MIGRATIONS.map((m) => m.version))("upgrades from schema version %i keeping the data", async (from) => {
    const db = new MemoryDatabase();
    await migrate(
      db,
      MIGRATIONS.filter((m) => m.version <= from),
    );
    expect(await schemaVersion(db)).toBe(from);

    if (from >= 1) {
      await db.run(
        "INSERT INTO accounts (uid, name, kind, opening_cents, updated_local_at) VALUES ('A', 'Cash', 'cash', 500, ?)",
        [NOW],
      );
    }
    await migrate(db);

    expect(await schemaVersion(db)).toBe(LATEST_VERSION);
    const rows = await db.all<{ name: string; opening_cents: number }>("SELECT name, opening_cents FROM accounts");
    expect(rows).toEqual([{ name: "Cash", opening_cents: 500 }]);
  });

  it("applies only newer migrations, each atomically with its version", async () => {
    const db = new MemoryDatabase();
    const extra: Migration = { version: LATEST_VERSION + 1, statements: ["ALTER TABLE tags ADD COLUMN note TEXT"] };
    await migrate(db);
    expect(await migrate(db, [...MIGRATIONS, extra])).toBe(extra.version);
    await db.run("INSERT INTO tags (uid, name, note, updated_local_at) VALUES ('T', 'x', 'n', ?)", [NOW]);
  });

  it("rolls a failing migration back and stays at the last good version", async () => {
    const db = new MemoryDatabase();
    await migrate(db);
    const bad: Migration = {
      version: LATEST_VERSION + 1,
      statements: ["ALTER TABLE tags ADD COLUMN ok TEXT", "THIS IS NOT SQL"],
    };
    await expect(migrate(db, [...MIGRATIONS, bad])).rejects.toThrow();
    expect(await schemaVersion(db)).toBe(LATEST_VERSION);
    await expect(db.all("SELECT ok FROM tags")).rejects.toThrow();
  });
});
