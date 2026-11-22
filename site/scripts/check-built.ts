import path from "node:path";
import { fileURLToPath } from "node:url";
import { checkBuilt } from "../lib/built.ts";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const out = path.resolve(root, process.env.SITE_OUT || "out");

const problems = checkBuilt(out, {
  origin: process.env.SITE_URL || "https://finances.vukem.dev",
  termsApproved: process.env.LANDING_TERMS_APPROVED === "true",
});
if (problems.length > 0) {
  console.error(`check:built failed:\n${problems.map((p) => `  ${p}`).join("\n")}`);
  process.exit(1);
}
console.log("check:built ok");
