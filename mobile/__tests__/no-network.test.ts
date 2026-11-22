import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { createElement } from "react";
import type { Sharer } from "../src/backup/share";
import { generateDue } from "../src/domain/recurrence";
import { AddScreen } from "../src/screens/AddScreen";
import { FirstRunScreen } from "../src/screens/FirstRunScreen";
import { HomeScreen } from "../src/screens/HomeScreen";
import { ReportsScreen } from "../src/screens/ReportsScreen";
import { SettingsScreen } from "../src/screens/SettingsScreen";
import { renderApp } from "./helpers/app";
import { openEnv } from "./helpers/fixtures";

// The app makes no network request at all in this version: no analytics, no crash reporter, no remote fonts.

const calls: string[] = [];
const original = {
  fetch: globalThis.fetch,
  xhr: (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest,
  ws: (globalThis as { WebSocket?: unknown }).WebSocket,
};

class Forbidden {
  constructor() {
    calls.push("constructed");
    throw new Error("network use is forbidden");
  }
}

beforeAll(() => {
  globalThis.fetch = jest.fn((input: unknown) => {
    calls.push(`fetch ${String(input)}`);
    return Promise.reject(new Error("network use is forbidden"));
  }) as unknown as typeof fetch;
  (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest = Forbidden;
  (globalThis as { WebSocket?: unknown }).WebSocket = Forbidden;
});

afterAll(() => {
  globalThis.fetch = original.fetch;
  (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest = original.xhr;
  (globalThis as { WebSocket?: unknown }).WebSocket = original.ws;
});

function files(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = `${dir}/${name}`;
    return statSync(path).isDirectory() ? files(path) : [path];
  });
}

describe("no network", () => {
  it("never calls fetch, XMLHttpRequest or WebSocket through the main flows", async () => {
    const env = await openEnv();
    const share: Sharer = { shareText: async () => undefined };
    const settings = {
      services: {
        lockStore: { isEnabled: async () => false, setEnabled: async () => undefined },
        lockPlatform: { available: async () => true, authenticate: async () => true },
        sharer: share,
        version: "0.1.0",
      },
    };

    await renderApp(createElement(FirstRunScreen), { env, settings: { onboarded: false } });
    await fireEvent.changeText(screen.getByLabelText("First account name"), "Cash");
    await fireEvent.press(screen.getByRole("button", { name: "Start" }));
    await waitFor(async () => expect(await env.repo.list("account")).toHaveLength(1));

    await renderApp(createElement(AddScreen), { env });
    await fireEvent.changeText(await screen.findByLabelText("Amount"), "12,50");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    await waitFor(async () => expect(await env.repo.list("transaction")).toHaveLength(1));

    await renderApp(createElement(HomeScreen), { env });
    await renderApp(createElement(ReportsScreen), { env });
    await renderApp(createElement(SettingsScreen, settings), { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Export a backup" }));
    await generateDue(env.db, env.repo, "2027-03-15");

    expect(calls).toEqual([]);
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });

  it("has no network API or web address in the app sources", () => {
    const root = `${__dirname}/..`;
    const sources = [...files(`${root}/src`), ...files(`${root}/app`)].filter((p) => /\.(ts|tsx)$/u.test(p));
    expect(sources.length).toBeGreaterThan(40);
    const forbidden = /\bfetch\s*\(|XMLHttpRequest|\bWebSocket\b|\baxios\b|https?:\/\//u;
    const offenders = sources.filter((p) => forbidden.test(readFileSync(p, "utf8")));
    expect(offenders).toEqual([]);
  });

  it("declares no networking, analytics or crash-reporting dependency", () => {
    const pkg = JSON.parse(readFileSync(`${__dirname}/../package.json`, "utf8")) as {
      dependencies: Record<string, string>;
    };
    const bad = Object.keys(pkg.dependencies).filter((d) =>
      /sentry|analytics|firebase|amplitude|mixpanel|segment|bugsnag|axios|posthog|expo-font|expo-updates/u.test(d),
    );
    expect(bad).toEqual([]);
  });
});
