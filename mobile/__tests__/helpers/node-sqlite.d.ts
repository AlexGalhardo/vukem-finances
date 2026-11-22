// The tests run on Node's built-in SQLite; @types/node is not a dependency, so the few members used are declared here.
declare module "node:sqlite" {
  export class DatabaseSync {
    constructor(path: string);
    exec(sql: string): void;
    prepare(sql: string): {
      run(...params: (string | number | null)[]): unknown;
      all(...params: (string | number | null)[]): object[];
    };
  }
}
