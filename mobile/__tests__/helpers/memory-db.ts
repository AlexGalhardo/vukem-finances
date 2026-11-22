import { DatabaseSync } from "node:sqlite";
import { type SqlValue, TransactionalDatabase } from "../../src/db/database";
import { migrate, type Migration } from "../../src/db/migrations";

/** An in-memory SQLite behind the same Database interface the app uses on a device. */
export class MemoryDatabase extends TransactionalDatabase {
  private readonly native = new DatabaseSync(":memory:");

  exec(sql: string): Promise<void> {
    try {
      this.native.exec(sql);
      return Promise.resolve();
    } catch (error) {
      return Promise.reject(error);
    }
  }

  run(sql: string, params: readonly SqlValue[] = []): Promise<void> {
    try {
      this.native.prepare(sql).run(...params);
      return Promise.resolve();
    } catch (error) {
      return Promise.reject(error);
    }
  }

  all<T extends object>(sql: string, params: readonly SqlValue[] = []): Promise<T[]> {
    try {
      return Promise.resolve(this.native.prepare(sql).all(...params) as T[]);
    } catch (error) {
      return Promise.reject(error);
    }
  }
}

export async function openMemoryDatabase(migrations?: readonly Migration[]): Promise<MemoryDatabase> {
  const db = new MemoryDatabase();
  await migrate(db, migrations);
  return db;
}
