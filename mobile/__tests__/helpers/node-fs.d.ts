// @types/node is not a dependency; the few members the tests use are declared here.
declare module "node:fs" {
	export function readFileSync(path: string, encoding: "utf8"): string;
}

declare const __dirname: string;
