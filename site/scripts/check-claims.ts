import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { checkCatalogs, checkRegister } from "../lib/claims.ts";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");
const brief = fs.readFileSync(path.join(repoRoot, "docs", "landing-brief.md"), "utf8");

const problems = [...checkRegister(brief, repoRoot), ...checkCatalogs()];
if (problems.length > 0) {
  console.error(`check:claims failed:\n${problems.map((p) => `  ${p}`).join("\n")}`);
  process.exit(1);
}
console.log("check:claims ok");
