import { type Database, DeadlockError, type SqlValue } from "../../src/db/database";
import { migrate } from "../../src/db/migrations";
import { Repository } from "../../src/db/repository";
import { MemoryDatabase } from "../helpers/memory-db";

/** A memory database that logs every statement that reaches SQLite, in order. */
class LoggingDatabase extends MemoryDatabase {
  readonly log: string[] = [];

  execRaw(sql: string): Promise<void> {
    this.log.push(sql);
    return super.execRaw(sql);
  }

  runRaw(sql: string, params: readonly SqlValue[] = []): Promise<void> {
    this.log.push(sql.startsWith("INSERT INTO tags") ? `tag:${String(params[1])}` : sql.slice(0, 30));
    return super.runRaw(sql, params);
  }
}

function deferred(): { promise: Promise<void>; resolve: () => void } {
  let resolve: () => void = () => undefined;
  const promise = new Promise<void>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

const tick = (): Promise<void> => new Promise((resolve) => setImmediate(resolve));
const insertTag = (db: Database, name: string): Promise<void> =>
  db.run("INSERT INTO tags (uid, updated_local_at, name) VALUES (?, 'x', ?)", [`uid-${name}`, name]);
const names = async (db: Database): Promise<string[]> =>
  (await db.all<{ name: string }>("SELECT name FROM tags ORDER BY name")).map((r) => r.name);

async function open(deadlockMs?: number): Promise<LoggingDatabase> {
  const db = new LoggingDatabase(deadlockMs);
  await migrate(db);
  db.log.length = 0;
  return db;
}

describe("isolation of transactions on one connection", () => {
  it("(a) keeps a UI write that started during a sync transaction that then rolls back", async () => {
    const db = await open();
    const repo = new Repository(db);
    const started = deferred();
    const release = deferred();

    const sync = db.transaction(async (tx) => {
      await insertTag(tx, "from-sync");
      started.resolve();
      await release.promise;
      throw new Error("the apply failed");
    });
    const syncFailed = expect(sync).rejects.toThrow("the apply failed");

    await started.promise;
    // The person saves while the sync holds the connection: the UI reported "saved" only when this resolves.
    const saved = repo.create("tag", { name: "from-ui", color: "" });
    await tick();
    release.resolve();
    await syncFailed;
    await saved;

    expect(await names(db)).toEqual(["from-ui"]);
    expect(await repo.outboxCount()).toBe(1);
  });

  it("(b) does not put a UI write inside a sync transaction that commits", async () => {
    const db = await open();
    const started = deferred();
    const release = deferred();

    const sync = db.transaction(async (tx) => {
      await insertTag(tx, "from-sync");
      started.resolve();
      await release.promise;
    });

    await started.promise;
    const ui = insertTag(db, "from-ui");
    await tick();
    release.resolve();
    await Promise.all([sync, ui]);

    const commit = db.log.indexOf("COMMIT");
    expect(db.log.indexOf("tag:from-sync")).toBeGreaterThan(db.log.indexOf("BEGIN IMMEDIATE"));
    expect(db.log.indexOf("tag:from-sync")).toBeLessThan(commit);
    expect(db.log.indexOf("tag:from-ui")).toBeGreaterThan(commit);
    expect(await names(db)).toEqual(["from-sync", "from-ui"]);
  });

  it("(c) never interleaves two concurrent transactions", async () => {
    const db = await open();
    const body =
      (label: string) =>
      async (tx: Database): Promise<void> => {
        for (const n of [1, 2, 3]) {
          await insertTag(tx, `${label}${n}`);
          await tick();
        }
      };
    await Promise.all([db.transaction(body("a")), db.transaction(body("b")), db.transaction(body("c"))]);

    const sequence = db.log.filter((l) => l === "BEGIN IMMEDIATE" || l === "COMMIT" || l.startsWith("tag:"));
    expect(sequence).toHaveLength(15);
    for (let i = 0; i < sequence.length; i += 5) {
      const block = sequence.slice(i, i + 5);
      expect(block[0]).toBe("BEGIN IMMEDIATE");
      expect(block[4]).toBe("COMMIT");
      expect(new Set(block.slice(1, 4).map((l) => l.slice(4, 5))).size).toBe(1);
    }
  });

  it("(d) joins a nested transaction, and a throw rolls the whole thing back", async () => {
    const db = await open();
    await expect(
      db.transaction(async (tx) => {
        await insertTag(tx, "outer");
        await tx.transaction(async (inner) => {
          expect(inner).toBe(tx);
          await insertTag(inner, "inner");
          throw new Error("inner failed");
        });
      }),
    ).rejects.toThrow("inner failed");

    expect(await names(db)).toEqual([]);
    expect(db.log.filter((l) => l === "BEGIN IMMEDIATE")).toHaveLength(1);
    expect(db.log.filter((l) => l === "ROLLBACK")).toHaveLength(1);

    await db.transaction(async (tx) => {
      await tx.transaction(async (inner) => insertTag(inner, "kept"));
    });
    expect(await names(db)).toEqual(["kept"]);
  });

  it("(e) never shows uncommitted rows to a read, committed or rolled back", async () => {
    const db = await open();
    for (const outcome of ["commit", "rollback"] as const) {
      const started = deferred();
      const release = deferred();
      const writer = db.transaction(async (tx) => {
        await insertTag(tx, `pending-${outcome}`);
        started.resolve();
        await release.promise;
        if (outcome === "rollback") throw new Error("no");
      });
      const settled = writer.then(
        () => undefined,
        () => undefined,
      );

      await started.promise;
      let seen: string[] | null = null;
      const read = names(db).then((n) => {
        seen = n;
      });
      await tick();
      expect(seen).toBeNull();
      release.resolve();
      await Promise.all([settled, read]);

      // The commit pass sees its row only after the commit; the rollback pass never sees its row (only the earlier committed one).
      expect(seen).toEqual(["pending-commit"]);
    }
  });

  it("fails loudly, not by hanging, when code inside a transaction uses the root", async () => {
    const db = await open(60);
    await expect(
      db.transaction(async () => {
        await db.all("SELECT 1");
      }),
    ).rejects.toBeInstanceOf(DeadlockError);
    // The connection is usable again afterwards.
    await insertTag(db, "after");
    expect(await names(db)).toEqual(["after"]);

    await expect(
      db.transaction(async () => {
        await db.transaction(async () => undefined);
      }),
    ).rejects.toBeInstanceOf(DeadlockError);
    expect(await names(db)).toEqual(["after"]);
  });

  it("refuses a tx used after its transaction ended", async () => {
    const db = await open();
    let leaked: Database | null = null;
    await db.transaction(async (tx) => {
      leaked = tx;
    });
    await expect((leaked as unknown as Database).run("SELECT 1")).rejects.toThrow("has ended");
  });
});

describe("the old design, for the record", () => {
  /** What the app had: one global depth, so every call joined whichever transaction was open. */
  class GlobalDepth extends MemoryDatabase {
    private depth = 0;

    override async transaction<T>(fn: (tx: Database) => Promise<T>): Promise<T> {
      if (this.depth > 0) return fn(this);
      this.depth = 1;
      await this.execRaw("BEGIN IMMEDIATE");
      try {
        const v = await fn(this);
        await this.execRaw("COMMIT");
        return v;
      } catch (e) {
        await this.execRaw("ROLLBACK");
        throw e;
      } finally {
        this.depth = 0;
      }
    }

    override async run(sql: string, params?: readonly SqlValue[]): Promise<void> {
      return this.runRaw(sql, params);
    }

    override async all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]> {
      return this.allRaw<T>(sql, params);
    }
  }

  it("lost a saved UI write when a sync transaction rolled back, which is what (a) now forbids", async () => {
    const db = new GlobalDepth();
    await migrate(db);
    const started = deferred();
    const release = deferred();
    const sync = db.transaction(async (tx) => {
      await insertTag(tx, "from-sync");
      started.resolve();
      await release.promise;
      throw new Error("the apply failed");
    });
    const failed = expect(sync).rejects.toThrow();
    await started.promise;
    await new Repository(db).create("tag", { name: "from-ui", color: "" }); // reported as saved
    release.resolve();
    await failed;
    expect(await names(db)).toEqual([]);
  });
});
