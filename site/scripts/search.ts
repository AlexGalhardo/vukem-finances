import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Indexes the built documentation with Pagefind (only the pages marked data-pagefind-body) and keeps just the files
// the site's own search box loads: the library, its worker, the WebAssembly and the index. The stock UI files stay out.
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const out = path.resolve(root, process.env.SITE_OUT || "out");

const proc = spawnSync("bun", ["x", "pagefind", "--site", out], { cwd: root, stdio: "inherit" });
if (proc.status !== 0) {
  console.error("pagefind failed");
  process.exit(1);
}

const dir = path.join(out, "pagefind");
for (const name of fs.readdirSync(dir)) {
  if (/^pagefind-(ui|component-ui|modular-ui|highlight)\./.test(name)) fs.rmSync(path.join(dir, name));
}
console.log("search: index written to pagefind/");
