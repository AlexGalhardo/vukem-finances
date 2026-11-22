export type SqlValue = string | number | null;

/**
 * The small surface the repositories need. expo-sqlite implements it on a
 * device (`expo.ts`); the tests implement it over an in-memory SQLite.
 */
export interface Database {
	exec(sql: string): Promise<void>;
	run(sql: string, params?: readonly SqlValue[]): Promise<void>;
	all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]>;
	/** Runs fn in one SQLite transaction: commit on return, rollback on throw. Calls do not interleave. */
	transaction<T>(fn: () => Promise<T>): Promise<T>;
}

export async function getOne<T extends object>(
	db: Database,
	sql: string,
	params: readonly SqlValue[] = [],
): Promise<T | null> {
	const rows = await db.all<T>(sql, params);
	return rows[0] ?? null;
}

/**
 * Serializes transactions on one connection and implements `transaction`
 * over BEGIN/COMMIT/ROLLBACK. A nested call joins the running transaction.
 */
export abstract class TransactionalDatabase implements Database {
	private tail: Promise<unknown> = Promise.resolve();
	private depth = 0;

	abstract exec(sql: string): Promise<void>;
	abstract run(sql: string, params?: readonly SqlValue[]): Promise<void>;
	abstract all<T extends object>(sql: string, params?: readonly SqlValue[]): Promise<T[]>;

	transaction<T>(fn: () => Promise<T>): Promise<T> {
		if (this.depth > 0) {
			return fn();
		}

		const result = this.tail.then(async () => {
			this.depth = 1;
			await this.exec("BEGIN IMMEDIATE");
			try {
				const value = await fn();
				await this.exec("COMMIT");
				return value;
			} catch (error) {
				await this.exec("ROLLBACK");
				throw error;
			} finally {
				this.depth = 0;
			}
		});
		this.tail = result.catch(() => undefined);

		return result;
	}
}
