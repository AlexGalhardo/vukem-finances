import fs from "node:fs";
import path from "node:path";

function pages(dir: string, sub = ""): string[] {
  return fs.readdirSync(path.join(dir, sub), { withFileTypes: true }).flatMap((entry) => {
    const rel = path.posix.join(sub, entry.name);
    if (entry.isDirectory()) return entry.name === "_astro" ? [] : pages(dir, rel);
    return entry.name === "index.html" ? [sub ? `/${sub}/` : "/"] : [];
  });
}

/** Public addresses of the built pages, from the files of the output directory. */
export function builtAddresses(dir: string): string[] {
  return pages(dir).sort();
}

export function sitemapXml(addresses: string[], origin: string): string {
  const base = new URL(origin);
  const prefix = base.pathname.replace(/\/$/, "");
  const urls = addresses.map((a) => `  <url><loc>${base.origin}${prefix}${a}</loc></url>`).join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`;
}

export function robotsTxt(origin: string): string {
  const base = new URL(origin);
  return `User-agent: *\nAllow: /\n\nSitemap: ${base.origin}${base.pathname.replace(/\/$/, "")}/sitemap.xml\n`;
}
