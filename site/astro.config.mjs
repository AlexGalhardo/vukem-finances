import path from "node:path";
import { fileURLToPath } from "node:url";
import react from "@astrojs/react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "astro/config";

const root = path.dirname(fileURLToPath(import.meta.url));

// SITE_URL overrides the public address for previews. The default and the production value are the
// custom domain on GitHub Pages, served with the base path "/".
const siteUrl = new URL(process.env.SITE_URL || "https://finances.vukem.dev");
const prefix = siteUrl.pathname.replace(/\/$/, "");

// Fully static HTML: no server entry of any kind. The documentation (epic 23) lives at /docs.
export default defineConfig({
  site: siteUrl.origin,
  base: prefix || "/",
  output: "static",
  outDir: process.env.SITE_OUT || "out",
  trailingSlash: "always",
  build: { format: "directory" },
  integrations: [react()],
  vite: {
    plugins: [tailwindcss()],
    resolve: { alias: { "@": root } },
  },
});
