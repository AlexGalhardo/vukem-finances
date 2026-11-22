// Starts the prepared binary in the foreground so Playwright can stop it.
// `bun e2e/run.mjs` runs the personal edition; `bun e2e/run.mjs saas` runs the saas edition (billing spec) together
// with a fake AbacatePay, once prepare.mjs has written its environment.
import { spawn } from "node:child_process";
import { createWriteStream, existsSync, readFileSync } from "node:fs";
import { setTimeout as sleep } from "node:timers/promises";
import { startFakeAbacatePay } from "./fake-abacatepay.mjs";
import { startFakeTelegram } from "./fake-telegram.mjs";

const saas = process.argv[2] === "saas";
const file = "e2e/.tmp/env.json";

// The saas server is started by its own Playwright web server entry, in parallel with the preparing one.
for (let i = 0; saas && i < 1800 && !(existsSync(file) && readFileSync(file, "utf8").includes("saasEnv")); i++) {
  await sleep(100);
}

const { binary, appEnv, saasEnv } = JSON.parse(readFileSync(file, "utf8"));
let fake;
if (saas) fake = await startFakeAbacatePay(Number(process.env.E2E_FAKE_PORT ?? 3919));
// The personal server runs the Telegram bot in polling mode against a fake Bot API.
if (!saas) fake = await startFakeTelegram(Number(process.env.E2E_TELEGRAM_PORT ?? 3920));

// E2E_LOG_DIR makes the server output also land in a file, for the log scanner (scripts/qa/run-e2e-with-logscan.sh).
const logDir = process.env.E2E_LOG_DIR;
const child = spawn(binary, [], {
  stdio: logDir ? ["ignore", "pipe", "pipe"] : "inherit",
  env: saas ? saasEnv : appEnv,
});
if (logDir) {
  const log = createWriteStream(`${logDir}/server-${saas ? "saas" : "personal"}.log`, { flags: "a" });
  child.stdout.on("data", (chunk) => (log.write(chunk), process.stdout.write(chunk)));
  child.stderr.on("data", (chunk) => (log.write(chunk), process.stderr.write(chunk)));
  child.on("close", () => log.end());
}
const stop = () => {
  child.kill();
  fake?.close();
};
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, stop);
child.on("exit", (code) => {
  fake?.close();
  process.exit(code ?? 0);
});
