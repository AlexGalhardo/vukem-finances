// Declared for the integration suite, which mints pairing codes: @types/node is not a dependency.
declare module "node:crypto" {
  export function createHash(algorithm: "sha256"): { update(data: string): { digest(encoding: "hex"): string } };
  export function randomBytes(size: number): { [index: number]: number; length: number };
}
