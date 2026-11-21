// Starts the prepared binary in the foreground so Playwright can stop it.
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";

const { binary, appEnv } = JSON.parse(readFileSync("e2e/.tmp/env.json", "utf8"));
const child = spawn(binary, [], { stdio: "inherit", env: appEnv });
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => child.kill());
child.on("exit", (code) => process.exit(code ?? 0));
