import { openDatabaseAsync, type SQLiteDatabase } from "expo-sqlite";
import { type SqlValue, TransactionalDatabase } from "./database";

class ExpoDatabase extends TransactionalDatabase {
  constructor(private readonly native: SQLiteDatabase) {
    super();
  }

  execRaw(sql: string): Promise<void> {
    return this.native.execAsync(sql);
  }

  async runRaw(sql: string, params: readonly SqlValue[] = []): Promise<void> {
    await this.native.runAsync(sql, [...params]);
  }

  allRaw<T extends object>(sql: string, params: readonly SqlValue[] = []): Promise<T[]> {
    return this.native.getAllAsync<T>(sql, [...params]);
  }
}

export async function openLocalDatabase(name = "vukem-finances.db"): Promise<TransactionalDatabase> {
  const native = await openDatabaseAsync(name);
  await native.execAsync("PRAGMA journal_mode = WAL");
  return new ExpoDatabase(native);
}
