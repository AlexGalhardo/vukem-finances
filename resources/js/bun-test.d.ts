// Minimal ambient types for the Bun test runner. @types/bun is not an approved dependency yet
// (QUESTIONS.md, Q-006); delete this file when it is installed.
declare module "bun:test" {
  interface Matchers<T> {
    not: Matchers<T>;
    toBe(expected: unknown): void;
    toEqual(expected: unknown): void;
    toBeDefined(): void;
    toBeUndefined(): void;
    toBeNull(): void;
    toBeTruthy(): void;
    toBeFalsy(): void;
    toBeCloseTo(expected: number, digits?: number): void;
    toBeGreaterThan(expected: number): void;
    toBeGreaterThanOrEqual(expected: number): void;
    toBeLessThan(expected: number): void;
    toContain(expected: unknown): void;
    toHaveLength(expected: number): void;
    toMatch(expected: string | RegExp): void;
    toThrow(expected?: string | RegExp): void;
  }
  export function describe(name: string, run: () => void): void;
  export function test(name: string, run: () => void | Promise<void>): void;
  export function beforeEach(run: () => void | Promise<void>): void;
  export function expect<T>(actual: T, message?: string): Matchers<T>;
}

declare module "node:fs" {
  export function readFileSync(path: string, encoding: "utf8"): string;
  export function readdirSync(path: string): string[];
}

declare module "node:path" {
  export function join(...parts: string[]): string;
}

interface ImportMeta {
  readonly dir: string;
}
