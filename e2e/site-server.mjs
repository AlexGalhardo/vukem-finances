// A plain static file server for the built site (site/out) and for planted pages, used by site-network.spec.ts.
// Node built-ins live here because the root tsconfig has no Node types (see site-server.d.mts).
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import http from "node:http";
import os from "node:os";
import path from "node:path";

export const OUT = path.resolve("site", "out");

const TYPES = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".webp": "image/webp",
  ".txt": "text/plain",
  ".md": "text/markdown",
  ".xml": "application/xml",
};

/** Serves a directory on a free loopback port; a directory path answers with its index.html. */
export function serve(root) {
  const server = http.createServer((req, res) => {
    const pathname = decodeURIComponent(new URL(req.url ?? "/", "http://x").pathname);
    let file = path.join(root, pathname);
    if (!file.startsWith(root)) {
      res.writeHead(403).end();
      return;
    }
    if (fs.existsSync(file) && fs.statSync(file).isDirectory()) file = path.join(file, "index.html");
    if (!fs.existsSync(file)) {
      res.writeHead(404).end("not found");
      return;
    }
    res.writeHead(200, { "content-type": TYPES[path.extname(file)] ?? "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => {
    server.listen(0, "127.0.0.1", () => {
      resolve({
        origin: `http://127.0.0.1:${server.address().port}`,
        // close() alone waits for a socket the browser opened ahead of time and never used.
        close: () =>
          new Promise((done) => {
            server.close(() => done());
            server.closeAllConnections();
          }),
      });
    });
  });
}

/** Builds the site when site/out is missing (CI installs the site dependencies first). */
export function ensureBuilt() {
  if (!fs.existsSync(path.join(OUT, "index.html"))) {
    execFileSync("bun", ["run", "build"], { cwd: "site", stdio: "inherit", timeout: 240_000 });
  }
}

/** The URL path of every page of the built site. */
export function builtPages() {
  return fs
    .readdirSync(OUT, { recursive: true, withFileTypes: true })
    .filter((e) => e.isFile() && e.name === "index.html")
    .map((e) => {
      const rel = path.relative(OUT, e.parentPath).split(path.sep).join("/");
      return rel ? `/${rel}/` : "/";
    })
    .sort();
}

/** Serves one planted page; close() also removes it. */
export async function servePlanted(html) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vf-site-plant-"));
  fs.writeFileSync(path.join(dir, "index.html"), html);
  const site = await serve(dir);
  return {
    origin: site.origin,
    close: async () => {
      await site.close();
      fs.rmSync(dir, { recursive: true, force: true });
    },
  };
}
