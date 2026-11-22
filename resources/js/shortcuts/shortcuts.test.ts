import { describe, expect, test } from "bun:test";
import { createDispatcher, type DispatchEvent } from "./dispatcher";
import { type GuardElement, isTypingTarget } from "./guard";
import { normalizeBinding, parseBinding } from "./keys";
import { formatAriaKeyShortcuts, formatHintLabel } from "./format";
import { createSequenceMatcher, type Timers } from "./matcher";
import { type Action, createRegistry } from "./registry";
import { isReserved } from "./reserved";
import { resolveActionHint } from "./useActionHint";
import { readSingleKey, SINGLE_KEY_STORAGE_KEY } from "./singleKey";

function el(
  tagName: string,
  attrs: Record<string, string> = {},
  extra: { isContentEditable?: boolean; parent?: GuardElement } = {},
): GuardElement {
  return {
    tagName,
    type: attrs.type,
    isContentEditable: extra.isContentEditable,
    parentElement: extra.parent ?? null,
    getAttribute: (name) => attrs[name] ?? null,
  };
}

class FakeTimers implements Timers {
  private next = 1;
  readonly pending = new Map<number, () => void>();
  set(run: () => void): unknown {
    const id = this.next++;
    this.pending.set(id, run);
    return id;
  }
  clear(handle: unknown): void {
    this.pending.delete(handle as number);
  }
  fire(): void {
    for (const [id, run] of [...this.pending]) {
      this.pending.delete(id);
      run();
    }
  }
}

function key(k: string, extra: Partial<DispatchEvent> = {}): DispatchEvent {
  return {
    key: k,
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    shiftKey: false,
    isComposing: false,
    repeat: false,
    defaultPrevented: false,
    target: el("body"),
    preventDefault() {
      this.defaultPrevented = true;
    },
    ...extra,
  };
}

function setup(options: { single?: boolean; dialog?: boolean } = {}) {
  const registry = createRegistry({ strict: true });
  const timers = new FakeTimers();
  const matcher = createSequenceMatcher({ isMac: false, timers });
  const ran: string[] = [];
  const add = (id: string, keys: string): void => {
    registry.register({ id, labelKey: "dashboard", group: "actions", keys, run: () => ran.push(id) });
  };
  add("go-transactions", "g t");
  add("go-dashboard", "g d");
  add("help", "?");
  add("palette", "mod+k");
  const dispatch = createDispatcher(registry, matcher, {
    blockingDialogOpen: () => options.dialog ?? false,
    singleKeys: () => options.single ?? true,
  });
  return { registry, timers, ran, dispatch };
}

describe("isTypingTarget", () => {
  test("text-like inputs, textarea, select and editable content are typing targets", () => {
    for (const type of ["text", "search", "email", "number", "date", "password", "url", "tel"]) {
      expect(isTypingTarget(el("INPUT", { type })), type).toBe(true);
    }
    expect(isTypingTarget(el("INPUT"))).toBe(true);
    expect(isTypingTarget(el("TEXTAREA"))).toBe(true);
    expect(isTypingTarget(el("SELECT"))).toBe(true);
    expect(isTypingTarget(el("DIV", {}, { isContentEditable: true }))).toBe(true);
  });

  test("checkbox, radio and button inputs are not", () => {
    for (const type of ["checkbox", "radio", "button", "submit"]) {
      expect(isTypingTarget(el("INPUT", { type })), type).toBe(false);
    }
    expect(isTypingTarget(el("BUTTON"))).toBe(false);
    expect(isTypingTarget(null)).toBe(false);
  });

  test("roles and data-shortcuts=off count, and so does anything inside", () => {
    for (const role of ["textbox", "combobox", "searchbox"]) {
      expect(isTypingTarget(el("DIV", { role })), role).toBe(true);
    }
    const off = el("DIV", { "data-shortcuts": "off" });
    expect(isTypingTarget(off)).toBe(true);
    expect(isTypingTarget(el("SPAN", {}, { parent: off }))).toBe(true);
    expect(isTypingTarget(el("SPAN", {}, { parent: el("DIV") }))).toBe(false);
  });
});

describe("dispatcher", () => {
  test("runs a sequence and ignores the same keys typed in a field", () => {
    const { ran, dispatch } = setup();
    dispatch(key("g", { target: el("INPUT", { type: "text" }) }));
    dispatch(key("t", { target: el("INPUT", { type: "text" }) }));
    expect(ran).toEqual([]);
    dispatch(key("g"));
    expect(dispatch(key("t"))).toBe(true);
    expect(ran).toEqual(["go-transactions"]);
  });

  test("a key typed in a field in the middle of a sequence cancels it", () => {
    const { ran, dispatch } = setup();
    dispatch(key("g"));
    dispatch(key("x", { target: el("TEXTAREA") }));
    dispatch(key("t"));
    expect(ran).toEqual([]);
  });

  test("IME composition, key repeat and handled events are ignored", () => {
    const { ran, dispatch } = setup();
    dispatch(key("?", { isComposing: true }));
    dispatch(key("?", { repeat: true }));
    dispatch(key("?", { defaultPrevented: true }));
    expect(ran).toEqual([]);
    dispatch(key("?", { shiftKey: true }));
    expect(ran).toEqual(["help"]);
  });

  test("a blocking dialog silences every shortcut", () => {
    const { ran, dispatch } = setup({ dialog: true });
    dispatch(key("k", { ctrlKey: true }));
    expect(ran).toEqual([]);
  });

  test("the sequence expires after the timeout", () => {
    const { ran, timers, dispatch } = setup();
    dispatch(key("g"));
    timers.fire();
    dispatch(key("t"));
    expect(ran).toEqual([]);
    dispatch(key("g"));
    dispatch(key("d"));
    expect(ran).toEqual(["go-dashboard"]);
  });

  test("a key that breaks a sequence can start another", () => {
    const { ran, dispatch } = setup();
    dispatch(key("g"));
    dispatch(key("g"));
    dispatch(key("t"));
    expect(ran).toEqual(["go-transactions"]);
  });

  test("a modifier press alone does not break a sequence", () => {
    const { ran, dispatch } = setup();
    dispatch(key("g"));
    dispatch(key("Shift", { shiftKey: true }));
    dispatch(key("t"));
    expect(ran).toEqual(["go-transactions"]);
  });

  test("letters with Shift or Alt do not match", () => {
    const { ran, dispatch } = setup();
    dispatch(key("G", { shiftKey: true }));
    dispatch(key("t"));
    expect(ran).toEqual([]);
  });

  test("with single-key shortcuts off only chords run", () => {
    const { ran, dispatch } = setup({ single: false });
    dispatch(key("g"));
    dispatch(key("t"));
    dispatch(key("?", { shiftKey: true }));
    expect(ran).toEqual([]);
    dispatch(key("k", { ctrlKey: true }));
    expect(ran).toEqual(["palette"]);
  });

  test("mod is Cmd on macOS", () => {
    const registry = createRegistry({ strict: true });
    const ran: string[] = [];
    registry.register({ id: "p", labelKey: "dashboard", group: "actions", keys: "mod+k", run: () => ran.push("p") });
    const dispatch = createDispatcher(registry, createSequenceMatcher({ isMac: true }), {
      blockingDialogOpen: () => false,
      singleKeys: () => true,
    });
    dispatch(key("k", { ctrlKey: true }));
    expect(ran).toEqual([]);
    dispatch(key("k", { metaKey: true }));
    expect(ran).toEqual(["p"]);
  });

  test("a disabled action does not run", () => {
    const registry = createRegistry({ strict: true });
    const ran: string[] = [];
    registry.register({
      id: "x",
      labelKey: "dashboard",
      group: "actions",
      keys: "x",
      run: () => ran.push("x"),
      enabled: () => false,
    });
    const dispatch = createDispatcher(registry, createSequenceMatcher({ isMac: false }), {
      blockingDialogOpen: () => false,
      singleKeys: () => true,
    });
    dispatch(key("x"));
    expect(ran).toEqual([]);
  });
});

describe("registry", () => {
  const action = (id: string, keys?: string): Action => ({
    id,
    labelKey: "dashboard",
    group: "actions",
    keys,
    run() {},
  });

  test("duplicate ids and duplicate bindings throw", () => {
    const registry = createRegistry({ strict: true });
    registry.register(action("a", "g d"));
    expect(() => registry.register(action("a", "g x"))).toThrow("already registered");
    expect(() => registry.register(action("b", "g d"))).toThrow("collides");
    expect(() => registry.register(action("c", "g"))).toThrow("collides");
    expect(() => registry.register(action("d", "g d x"))).toThrow("collides");
  });

  test("the unregister function frees the id and the keys", () => {
    const registry = createRegistry({ strict: true });
    const off = registry.register(action("a", "g d"));
    expect(registry.list().map((a) => a.id)).toEqual(["a"]);
    off();
    expect(registry.list()).toEqual([]);
    registry.register(action("b", "g d"));
  });

  test("outside strict mode a duplicate is ignored", () => {
    const registry = createRegistry({ strict: false });
    const warn = console.warn;
    console.warn = () => {};
    registry.register(action("a", "g d"));
    registry.register(action("b", "g d"));
    console.warn = warn;
    expect(registry.list().map((a) => a.id)).toEqual(["a"]);
  });

  test("an action without keys is only listed", () => {
    const registry = createRegistry({ strict: true });
    registry.register(action("a"));
    registry.register(action("b"));
    expect(registry.bindings()).toEqual([]);
    expect(registry.list()).toHaveLength(2);
  });

  test("subscribers hear about changes", () => {
    const registry = createRegistry({ strict: true });
    let calls = 0;
    registry.subscribe(() => calls++);
    registry.register(action("a"))();
    expect(calls).toBe(2);
  });
});

describe("reserved combinations", () => {
  const reserved: string[] = [];
  for (const k of ["t", "n", "w", "q", "r", "l", "d", "f", "p", "s", "o", "u", "h", "j", "tab", "pageup", "pagedown"]) {
    reserved.push(`ctrl+${k}`, `meta+${k}`, `mod+${k}`);
  }
  reserved.push("mod+shift+t", "mod+shift+n", "mod+shift+w");
  for (let i = 1; i <= 9; i++) reserved.push(`mod+${i}`);
  for (let i = 1; i <= 12; i++) reserved.push(`f${i}`);
  reserved.push("alt+left", "alt+right", "alt+home");

  test("registerAction refuses every one", () => {
    const registry = createRegistry({ strict: false });
    for (const keys of reserved) {
      expect(isReserved(keys), keys).toBe(true);
      expect(
        () => registry.register({ id: keys, labelKey: "dashboard", group: "actions", keys, run() {} }),
        keys,
      ).toThrow("reserved");
    }
  });

  test("the allocated bindings are not reserved", () => {
    for (const keys of ["mod+k", "?", "[", "g d", "n e", "/", "x", "mod+shift+k"]) {
      expect(isReserved(keys), keys).toBe(false);
    }
  });
});

describe("keys and preference", () => {
  test("bindings normalize to one canonical form", () => {
    expect(normalizeBinding("Mod+K")).toBe("mod+k");
    expect(normalizeBinding("g   d")).toBe("g d");
    expect(parseBinding("g d")).toHaveLength(2);
    expect(() => parseBinding("hyper+k")).toThrow("unknown modifier");
  });

  test("single-key shortcuts default to on and read the stored switch", () => {
    const store = new Map<string, string>();
    const storage = {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
    };
    expect(readSingleKey(storage)).toBe(true);
    storage.setItem(SINGLE_KEY_STORAGE_KEY, "off");
    expect(readSingleKey(storage)).toBe(false);
    expect(readSingleKey(null)).toBe(true);
  });
});

describe("action hints", () => {
  const actions: Action[] = [
    { id: "open-palette", labelKey: "dashboard", group: "actions", keys: "mod+k", run() {} },
    { id: "go-dashboard", labelKey: "dashboard", group: "navigation", keys: "g d", run() {} },
    { id: "toggle-sidebar", labelKey: "dashboard", group: "actions", keys: "[", run() {} },
    { id: "export", labelKey: "dashboard", group: "actions", keys: "x", run() {} },
    { id: "help", labelKey: "dashboard", group: "actions", keys: "?", run() {} },
    { id: "palette-only", labelKey: "dashboard", group: "actions", run() {} },
    { id: "off", labelKey: "dashboard", group: "actions", keys: "n e", run() {}, enabled: () => false },
  ];
  const on = { isMac: false, singleKeys: true };

  test("formats a chord for Windows and for macOS", () => {
    expect(resolveActionHint(actions, "open-palette", on)).toEqual({
      parts: ["Ctrl+K"],
      ariaKeyShortcuts: "Control+K",
    });
    expect(resolveActionHint(actions, "open-palette", { ...on, isMac: true })).toEqual({
      parts: ["⌘+K"],
      ariaKeyShortcuts: "Meta+K",
    });
  });

  test("a sequence has one cap per key and no aria-keyshortcuts", () => {
    expect(resolveActionHint(actions, "go-dashboard", on)).toEqual({ parts: ["G", "D"], ariaKeyShortcuts: undefined });
    expect(formatHintLabel(["G", "D"], "then")).toBe("G then D");
    expect(formatHintLabel(["Ctrl+K"], "then")).toBe("Ctrl+K");
  });

  test("single symbols and letters keep their own aria form", () => {
    expect(formatAriaKeyShortcuts("?", false)).toBe("Shift+?");
    expect(formatAriaKeyShortcuts("[", false)).toBe("[");
    expect(formatAriaKeyShortcuts("x", false)).toBe("X");
    expect(formatAriaKeyShortcuts("g d", false)).toBeUndefined();
  });

  test("nothing is shown for a missing action, no keys or a disabled action", () => {
    expect(resolveActionHint(actions, "nope", on)).toBeNull();
    expect(resolveActionHint(actions, "palette-only", on)).toBeNull();
    expect(resolveActionHint(actions, "off", on)).toBeNull();
  });

  test("with single-key shortcuts off only chords keep a hint", () => {
    const off = { isMac: false, singleKeys: false };
    expect(resolveActionHint(actions, "export", off)).toBeNull();
    expect(resolveActionHint(actions, "go-dashboard", off)).toBeNull();
    expect(resolveActionHint(actions, "open-palette", off)).not.toBeNull();
    expect(resolveActionHint(actions, "export", on)?.parts).toEqual(["X"]);
  });
});
