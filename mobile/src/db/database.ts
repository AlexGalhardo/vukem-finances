export type SqlValue = string | number | null;

/**
 * The small surface the repositories need. expo-sqlite implements it on a
 * device (`expo.ts`); the tests implement it over an in-memory SQLite.
 */
export interface Database {
  exec(sql: string): Promise<void>;
  run(sql: string, params?: readonly SqlValue[]): Promise<void>;
  all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]>;
  /**
   * Runs fn in one SQLite transaction: commit on return, rollback on throw.
   *
   * On the root database transactions queue: they never interleave, and a plain
   * `run`, `all` or `exec` waits for the open one, so no one reads uncommitted
   * rows and no write lands inside a transaction that is not its own. fn gets
   * `tx`, and everything inside the transaction must use `tx`, not the root.
   * `tx.transaction(...)` joins the running transaction: a throw that leaves it
   * rolls the whole thing back.
   */
  transaction<T>(fn: (tx: Database) => Promise<T>): Promise<T>;
}

export async function getOne<T extends object>(
  db: Database,
  sql: string,
  params: readonly SqlValue[] = [],
): Promise<T | null> {
  const rows = await db.all<T>(sql, params);
  return rows[0] ?? null;
}

/** How long the root database waits for a transaction before it says the caller is probably inside it. */
export const DEFAULT_DEADLOCK_MS = 30_000;

export class DeadlockError extends Error {
  constructor(what: string) {
    super(
      `${what} waited for an open transaction for too long: code that runs inside a transaction must use the tx it was given, not the root database`,
    );
    this.name = "DeadlockError";
  }
}

/** The handle a transaction body works with: valid until the transaction ends. */
class Tx implements Database {
  private live = true;

  constructor(private readonly root: TransactionalDatabase) {}

  close(): void {
    this.live = false;
  }

  private check(): void {
    if (!this.live) {
      throw new Error("this transaction has ended: a tx must not be used after its transaction");
    }
  }

  async exec(sql: string): Promise<void> {
    this.check();
    return this.root.execRaw(sql);
  }

  async run(sql: string, params?: readonly SqlValue[]): Promise<void> {
    this.check();
    return this.root.runRaw(sql, params);
  }

  async all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]> {
    this.check();
    return this.root.allRaw<T>(sql, params);
  }

  async transaction<T>(fn: (tx: Database) => Promise<T>): Promise<T> {
    this.check();
    return fn(this);
  }
}

/**
 * One connection, real isolation. A transaction holds the connection from BEGIN
 * to COMMIT: other transactions queue behind it, and plain statements on the root
 * wait for it, so a UI write or read never lands inside a sync transaction and a
 * rollback never takes someone else's write with it.
 */
export abstract class TransactionalDatabase implements Database {
  private tail: Promise<void> = Promise.resolve();
  private open: Promise<void> | null = null;

  constructor(private readonly deadlockMs: number = DEFAULT_DEADLOCK_MS) {}

  abstract execRaw(sql: string): Promise<void>;
  abstract runRaw(sql: string, params?: readonly SqlValue[]): Promise<void>;
  abstract allRaw<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]>;

  /** Waits until no transaction is open, or fails loudly instead of hanging. */
  private async idle(what: string): Promise<void> {
    while (this.open !== null) {
      let timer: ReturnType<typeof setTimeout> | undefined;
      const timeout = new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new DeadlockError(what)), this.deadlockMs);
      });
      try {
        await Promise.race([this.open, timeout]);
      } finally {
        clearTimeout(timer);
      }
    }
  }

  async exec(sql: string): Promise<void> {
    await this.idle("exec");
    return this.execRaw(sql);
  }

  async run(sql: string, params?: readonly SqlValue[]): Promise<void> {
    await this.idle("run");
    return this.runRaw(sql, params);
  }

  async all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]> {
    await this.idle("all");
    return this.allRaw<T>(sql, params);
  }

  transaction<T>(fn: (tx: Database) => Promise<T>): Promise<T> {
    const previous = this.tail;
    let abandoned = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const turn = Promise.race([
      previous,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => {
          abandoned = true;
          reject(new DeadlockError("transaction"));
        }, this.deadlockMs);
      }),
    ]).finally(() => clearTimeout(timer));

    const result = turn.then(async () => {
      if (abandoned) {
        throw new DeadlockError("transaction");
      }
      let finish: () => void = () => undefined;
      this.open = new Promise<void>((resolve) => {
        finish = resolve;
      });
      const tx = new Tx(this);
      try {
        await this.execRaw("BEGIN IMMEDIATE");
        try {
          const value = await fn(tx);
          await this.execRaw("COMMIT");
          return value;
        } catch (error) {
          await this.execRaw("ROLLBACK");
          throw error;
        }
      } finally {
        tx.close();
        this.open = null;
        finish();
      }
    });
    // The next transaction starts when this one is over, whatever its outcome. A timed out turn never started.
    this.tail = result.then(
      () => undefined,
      () => undefined,
    );
    return result;
  }
}
