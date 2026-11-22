import { describe, expect, test } from "bun:test";
import type { SharedProps } from "../page-props";
import { tutorialScreens } from "./screens";
import {
  back,
  dismissWelcome,
  helpAction,
  initialState,
  next,
  pause,
  restart,
  resume,
  sequenceFor,
  skipScreen,
  start,
  visibleScreens,
} from "./state";
import { loadState, migrate, saveState, storageKey } from "./store";
import type { TutorialState } from "./types";

function props(overrides: { edition?: string; role?: string; admin?: boolean } = {}): SharedProps {
  return {
    appName: "t",
    edition: overrides.edition ?? "personal",
    errors: {},
    googleEnabled: false,
    locale: "en",
    workspace: {
      admin: overrides.admin ?? false,
      all: [],
      current: { currency: "USD", id: 1, name: "W", timezone: "UTC" },
      edition: overrides.edition ?? "personal",
      plan: "free",
      role: overrides.role ?? "owner",
    },
  };
}

describe("the screen data", () => {
  test("has 35 screens and 80 steps, each target unique within its screen", () => {
    expect(tutorialScreens).toHaveLength(35);
    expect(tutorialScreens.reduce((sum, screen) => sum + screen.steps.length, 0)).toBe(80);
    for (const screen of tutorialScreens) {
      const targets = screen.steps.map((step) => step.target);
      expect(new Set(targets).size, screen.id).toBe(targets.length);
    }
  });

  test("the personal edition never shows billing, revenue or sign-up screens", () => {
    const personal = visibleScreens(props({ edition: "personal", admin: true })).map((screen) => screen.id);
    expect(personal).not.toContain("Billing");
    expect(personal).not.toContain("AdminRevenue");
    expect(personal).toContain("Dashboard");
    expect(visibleScreens(props({ edition: "saas" })).map((screen) => screen.id)).toContain("Billing");
  });

  test("a viewer is not taught the screens that need to record, and admin screens need the panel", () => {
    const viewer = visibleScreens(props({ role: "viewer" })).map((screen) => screen.id);
    expect(viewer).not.toContain("Transactions");
    expect(viewer).not.toContain("Import");
    expect(viewer).not.toContain("AdminDashboard");
    expect(visibleScreens(props({ admin: true })).map((screen) => screen.id)).toContain("AdminDashboard");
  });

  test("the panel's own pages, which carry no workspace, still offer the admin tour", () => {
    const panel = { ...props(), workspace: undefined } as unknown as SharedProps;
    const win = globalThis as { window?: unknown };
    win.window = { location: { pathname: "/admin/revenue" } };
    try {
      expect(visibleScreens(panel).map((screen) => screen.id)).toContain("AdminDashboard");
    } finally {
      win.window = undefined;
    }
    expect(visibleScreens(panel).map((screen) => screen.id)).not.toContain("AdminDashboard");
  });

  test("screens whose route has a parameter cannot be visited, so they stay out of the run", () => {
    expect(visibleScreens(props()).map((screen) => screen.id)).not.toContain("CardStatement");
  });
});

function pick(state: TutorialState) {
  const { status, screenId, stepIndex, welcomeDismissed } = state;
  return { status, screenId, stepIndex, welcomeDismissed };
}

describe("the run", () => {
  const visible = visibleScreens(props());
  const core = visible.filter((screen) => screen.core);

  test("starts at the first core screen and walks the steps then the next screen", () => {
    let state = start(initialState, visible);
    expect(pick(state)).toEqual({ status: "running", screenId: core[0]?.id, stepIndex: 0, welcomeDismissed: true });
    const first = core[0];
    for (let i = 1; i < (first?.steps.length ?? 0); i++) state = next(state, visible);
    expect(state.stepIndex).toBe((first?.steps.length ?? 1) - 1);
    state = next(state, visible);
    expect(state.screenId).toBe(core[1]?.id);
    expect(state.stepIndex).toBe(0);
    expect(state.completed[first?.id ?? ""]).toBe(true);
  });

  test("back goes to the previous step and, from a first step, to the last step of the previous screen", () => {
    let state = start(initialState, visible, core[1]?.id);
    state = back(state, visible);
    expect(state.screenId).toBe(core[0]?.id);
    expect(state.stepIndex).toBe((core[0]?.steps.length ?? 1) - 1);
    state = back(state, visible);
    expect(state.stepIndex).toBe((core[0]?.steps.length ?? 2) - 2);
    expect(back(start(initialState, visible), visible).stepIndex).toBe(0);
  });

  test("skip marks the screen and moves on; the last screen ends the run as done", () => {
    let state = start(initialState, visible);
    state = skipScreen(state, visible);
    expect(state.skipped[core[0]?.id ?? ""]).toBe(true);
    expect(state.screenId).toBe(core[1]?.id);
    const last = core[core.length - 1];
    state = start(initialState, visible, last?.id);
    state = skipScreen(state, visible);
    expect(pick(state)).toEqual({ status: "done", screenId: null, stepIndex: 0, welcomeDismissed: true });
  });

  test("an optional screen runs alone and then ends", () => {
    const optional = visible.find((screen) => !screen.core);
    expect(optional).toBeDefined();
    expect(sequenceFor(visible, optional?.id ?? "")).toHaveLength(1);
    let state = start(initialState, visible, optional?.id);
    for (let i = 0; i < (optional?.steps.length ?? 0); i++) state = next(state, visible);
    expect(state.status).toBe("done");
  });

  test("pause keeps the position, resume continues, and a step that vanished restarts the screen", () => {
    let state = next(start(initialState, visible), visible);
    state = pause(state);
    expect(pick(state).status).toBe("paused");
    expect(state.stepIndex).toBe(1);
    expect(helpAction(state)).toBe("resume");
    expect(pick(resume(state, visible))).toEqual({ ...pick(state), status: "running" });
    expect(resume({ ...state, stepIndex: 99 }, visible).stepIndex).toBe(0);
    expect(helpAction(initialState)).toBe("start");
    expect(helpAction(resume(state, visible))).toBe("none");
  });

  test("restart clears progress only and keeps the dismissed welcome", () => {
    const state = restart(dismissWelcome(skipScreen(start(initialState, visible), visible)));
    expect(state).toEqual({ ...initialState, welcomeDismissed: true });
  });
});

describe("persistence", () => {
  function memoryStorage() {
    const data = new Map<string, string>();
    return {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => void data.set(key, value),
      data,
    };
  }

  test("saves and loads per user, and a running run comes back paused", () => {
    const storage = memoryStorage();
    const state = next(start(initialState, visibleScreens(props())), visibleScreens(props()));
    saveState(storage, 7, state);
    expect([...storage.data.keys()]).toEqual([storageKey(7)]);
    expect(loadState(storage, 7)).toEqual({ ...state, status: "paused" });
    expect(loadState(storage, 8)).toEqual(initialState);
  });

  test("stores only ids, an index and flags", () => {
    const storage = memoryStorage();
    saveState(storage, 1, start(initialState, visibleScreens(props())));
    const keys = Object.keys(JSON.parse(storage.data.get(storageKey(1)) ?? "{}"));
    expect(keys.sort()).toEqual(
      ["completed", "screenId", "skipped", "status", "stepIndex", "version", "welcomeDismissed"].sort(),
    );
  });

  test("a broken or foreign value resets, and a failing storage keeps the run in memory", () => {
    expect(migrate(null)).toEqual(initialState);
    expect(migrate({ version: 2 })).toEqual(initialState);
    expect(migrate({ version: 1, status: "bogus", completed: {}, skipped: {} })).toEqual(initialState);
    const storage = memoryStorage();
    storage.data.set(storageKey(3), "{not json");
    expect(loadState(storage, 3)).toEqual(initialState);
    const broken = {
      getItem: () => {
        throw new Error("blocked");
      },
      setItem: () => {
        throw new Error("blocked");
      },
    };
    const state = start(initialState, visibleScreens(props()));
    saveState(broken, 9, state);
    expect(loadState(broken, 9).screenId).toBe(state.screenId);
  });
});
