// Generates the product pictures of the landing page from this repository's own app and its demo seed.
// It builds the app, runs it on a free local port with a throwaway SQLite database and the demo seed, signs in
// with Playwright (the one of the repository root, @playwright/test), takes the screenshots and writes a WebP and
// a PNG fallback of each to site/public/landing. Nothing is hosted and nothing from another product is involved.
//
//   cd site && bun run screenshots
import { type ChildProcess, spawn, spawnSync } from "node:child_process";
import { randomBytes } from "node:crypto";
import fs from "node:fs";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");
const outDir = path.join(root, "site", "public", "landing");
export const MAX_BYTES = 150 * 1024;

const email = "demo@example.test";
const password = `Demo-${randomBytes(9).toString("hex")}-Pw1`;

const shots = [
  { name: "dashboard", path: "/dashboard" },
  { name: "transactions", path: "/transactions" },
  { name: "reports", path: "/reports" },
];

function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address() as net.AddressInfo;
      server.close(() => resolve(port));
    });
  });
}

function run(cmd: string, args: string[], env: NodeJS.ProcessEnv, cwd = root): void {
  const result = spawnSync(cmd, args, { cwd, env, stdio: "inherit" });
  if (result.status !== 0) throw new Error(`${cmd} ${args.join(" ")} exited with ${result.status}`);
}

async function waitFor(url: string, server: ChildProcess): Promise<void> {
  for (let i = 0; i < 120; i++) {
    if (server.exitCode !== null) throw new Error("the app exited before it was ready");
    try {
      if ((await fetch(url)).status < 500) return;
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`${url} did not answer`);
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "vf-shots-"));
let server: ChildProcess | undefined;

try {
  const port = await freePort();
  const binary = path.join(tmp, process.platform === "win32" ? "app.exe" : "app");
  const env: NodeJS.ProcessEnv = {
    ...process.env,
    APP_NAME: "vukem-finances",
    APP_ENV: "development",
    APP_KEY: `base64:${randomBytes(32).toString("base64")}`,
    APP_EDITION: "personal",
    PORT: String(port),
    APP_URL: `http://localhost:${port}`,
    DATABASE_URL: `file:${path.join(tmp, "db.sqlite").replaceAll("\\", "/")}`,
    ADMIN_EMAILS: email,
    FINANCE_ADMIN_EMAIL: email,
    FINANCE_ADMIN_PASSWORD: password,
    MAIL_MAILER: "log",
  };

  run("bun", ["run", "build"], process.env);
  run("go", ["build", "-o", binary, "./cmd/app"], process.env);
  run(binary, ["migrate"], env);
  run(binary, ["db:seed", "FinanceAdminSeeder", "FinanceDemoSeeder"], env);

  server = spawn(binary, [], { env, stdio: "ignore" });
  await waitFor(`http://localhost:${port}/login`, server);

  const browser = await chromium.launch();
  try {
    const context = await browser.newContext({
      viewport: { width: 1280, height: 800 },
      colorScheme: "light",
      reducedMotion: "reduce",
    });
    const page = await context.newPage();
    await page.goto(`http://localhost:${port}/login`);
    await page.getByLabel("Email").fill(email);
    await page.locator("input[name=password]").fill(password);
    await page
      .getByRole("button", { name: /log in/i })
      .first()
      .click();
    await page.waitForURL(/\/dashboard/);

    fs.mkdirSync(outDir, { recursive: true });
    const encoder = await context.newPage();
    await encoder.goto("about:blank");

    for (const shot of shots) {
      await page.goto(`http://localhost:${port}${shot.path}`);
      await page.getByRole("heading", { level: 1 }).first().waitFor();
      await page.waitForLoadState("networkidle");
      const png = (await page.screenshot({ type: "png" })).toString("base64");

      // The browser encodes WebP and the resized PNG, so no image library is needed.
      let written = false;
      for (const width of [1000, 900, 800, 700, 600]) {
        const encoded = await encoder.evaluate(
          async ({ data, width }) => {
            const image = new Image();
            image.src = `data:image/png;base64,${data}`;
            await image.decode();
            const height = Math.round((image.height * width) / image.width);
            const canvas = document.createElement("canvas");
            canvas.width = width;
            canvas.height = height;
            const ctx = canvas.getContext("2d");
            if (!ctx) throw new Error("no 2d context");
            ctx.drawImage(image, 0, 0, width, height);
            const read = (type: string, quality: number) => canvas.toDataURL(type, quality).split(",")[1];
            return { webp: read("image/webp", 0.8), png: read("image/png", 1), width, height };
          },
          { data: png, width },
        );
        const webp = Buffer.from(encoded.webp, "base64");
        const fallback = Buffer.from(encoded.png, "base64");
        if (webp.length <= MAX_BYTES && fallback.length <= MAX_BYTES) {
          fs.writeFileSync(path.join(outDir, `${shot.name}.webp`), webp);
          fs.writeFileSync(path.join(outDir, `${shot.name}.png`), fallback);
          console.log(
            `${shot.name}: ${encoded.width}x${encoded.height}, webp ${webp.length} B, png ${fallback.length} B`,
          );
          written = true;
          break;
        }
      }
      if (!written) throw new Error(`${shot.name} does not fit ${MAX_BYTES} bytes at any size`);
    }
  } finally {
    await browser.close();
  }
} finally {
  if (server && server.exitCode === null) {
    const exited = new Promise((resolve) => server?.once("exit", resolve));
    server.kill();
    await exited;
  }
  fs.rmSync(tmp, { recursive: true, force: true, maxRetries: 10, retryDelay: 500 });
}
