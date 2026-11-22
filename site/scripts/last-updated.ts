import { spawnSync } from "node:child_process";
import fs from "node:fs";
import { type DocPage, lastUpdatedFile, loadPages, readLastUpdated, repoRoot, sourceHash } from "../lib/docs.ts";
import { nextEntries, staleEntries } from "../lib/last-updated.ts";

// Writes content/last-updated.json from the git history; never edit it by hand. With --check it only verifies.
function git(...args: string[]): string {
  const result = spawnSync("git", args, { cwd: repoRoot, encoding: "utf8" });
  return result.status === 0 ? result.stdout.trim() : "";
}

const today = new Date().toISOString().slice(0, 10);

function dateOf(page: DocPage): string {
  const dirty = git("status", "--porcelain", "--", page.file) !== "";
  const committed = git("log", "-1", "--format=%cs", "--", page.file);
  return !dirty && committed ? committed : today;
}

const pages = loadPages();
const previous = readLastUpdated();

if (process.argv.includes("--check")) {
  const problems = staleEntries(pages, previous, sourceHash);
  if (problems.length > 0) {
    console.error(
      `check:last-updated failed (run "bun run last-updated" and commit):\n${problems.map((p) => `  ${p}`).join("\n")}`,
    );
    process.exit(1);
  }
  console.log(`check:last-updated ok (${pages.length} pages)`);
} else {
  const entries = nextEntries(pages, previous, sourceHash, dateOf);
  fs.writeFileSync(lastUpdatedFile, `${JSON.stringify(entries, null, "\t")}\n`);
  console.log(`last-updated: ${pages.length} pages`);
}
