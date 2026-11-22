import { act, fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { type AppStateStatus, Text } from "react-native";
import { UiProvider } from "../../src/components/ui";
import { LockGate } from "../../src/lock/LockGate";
import { LOCK_AFTER_MS, type LockPlatform, type LockStore, shouldRelock } from "../../src/lock/lock";

class FakeStore implements LockStore {
  constructor(public enabled: boolean) {}
  async isEnabled(): Promise<boolean> {
    return this.enabled;
  }
  async setEnabled(enabled: boolean): Promise<void> {
    this.enabled = enabled;
  }
}

class FakePlatform implements LockPlatform {
  prompts: string[] = [];
  constructor(public pass: boolean) {}
  async available(): Promise<boolean> {
    return true;
  }
  async authenticate(prompt: string): Promise<boolean> {
    this.prompts.push(prompt);
    return this.pass;
  }
}

class FakeAppState {
  listener: ((state: AppStateStatus) => void) | null = null;
  addEventListener(_type: "change", listener: (state: AppStateStatus) => void): { remove: () => void } {
    this.listener = listener;
    return { remove: () => (this.listener = null) };
  }
  async emit(state: AppStateStatus): Promise<void> {
    await act(async () => this.listener?.(state));
  }
}

async function mount(
  store: LockStore,
  platform: LockPlatform,
  appState: FakeAppState,
  clock: { t: number },
): Promise<void> {
  await render(
    <UiProvider locale="en" scheme="light" currency="USD">
      <LockGate store={store} platform={platform} appState={appState} now={() => clock.t}>
        <Text>secret numbers</Text>
      </LockGate>
    </UiProvider>,
  );
}

describe("shouldRelock", () => {
  it("locks after 60 seconds in the background and not before", () => {
    expect(LOCK_AFTER_MS).toBe(60_000);
    expect(shouldRelock(null, 1_000_000)).toBe(false);
    expect(shouldRelock(1_000, 60_999)).toBe(false);
    expect(shouldRelock(1_000, 61_000)).toBe(true);
  });
});

describe("LockGate", () => {
  it("shows the app straight away when the lock is off", async () => {
    const platform = new FakePlatform(true);
    await mount(new FakeStore(false), platform, new FakeAppState(), { t: 0 });
    expect(await screen.findByText("secret numbers")).toBeTruthy();
    expect(platform.prompts).toEqual([]);
  });

  it("asks at open, hides the app until the check passes, then shows it", async () => {
    const platform = new FakePlatform(false);
    await mount(new FakeStore(true), platform, new FakeAppState(), { t: 0 });
    expect(await screen.findByRole("button", { name: "Unlock" })).toBeTruthy();
    expect(screen.queryByText("secret numbers")).toBeNull();
    await waitFor(() => expect(platform.prompts).toEqual(["Unlock Vukem Finances"]));
    expect(await screen.findByRole("alert")).toBeTruthy();

    platform.pass = true;
    await fireEvent.press(screen.getByRole("button", { name: "Unlock" }));
    expect(await screen.findByText("secret numbers")).toBeTruthy();
  });

  it("locks again after 60 seconds in the background, not after 59", async () => {
    const platform = new FakePlatform(true);
    const appState = new FakeAppState();
    const clock = { t: 0 };
    await mount(new FakeStore(true), platform, appState, clock);
    expect(await screen.findByText("secret numbers")).toBeTruthy();
    const opens = platform.prompts.length;

    await appState.emit("background");
    clock.t = 59_000;
    await appState.emit("active");
    expect(screen.getByText("secret numbers")).toBeTruthy();
    expect(platform.prompts).toHaveLength(opens);

    await appState.emit("background");
    clock.t = 59_000 + 60_000;
    await appState.emit("active");
    await waitFor(() => expect(platform.prompts).toHaveLength(opens + 1));
  });

  it("does not lock after a long background when the setting is off", async () => {
    const appState = new FakeAppState();
    const clock = { t: 0 };
    await mount(new FakeStore(false), new FakePlatform(true), appState, clock);
    await screen.findByText("secret numbers");
    await appState.emit("background");
    clock.t = 10 * LOCK_AFTER_MS;
    await appState.emit("active");
    expect(screen.getByText("secret numbers")).toBeTruthy();
  });
});
