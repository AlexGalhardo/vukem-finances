import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { builtAddresses, robotsTxt, sitemapXml } from "../lib/sitemap.ts";

// Writes sitemap.xml and robots.txt next to the built pages, so the sitemap lists exactly what was built.
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const out = path.resolve(root, process.env.SITE_OUT || "out");
const origin = process.env.SITE_URL || "https://finances.vukem.dev";

fs.writeFileSync(path.join(out, "sitemap.xml"), sitemapXml(builtAddresses(out), origin));
fs.writeFileSync(path.join(out, "robots.txt"), robotsTxt(origin));
console.log(`sitemap: ${builtAddresses(out).length} pages`);
