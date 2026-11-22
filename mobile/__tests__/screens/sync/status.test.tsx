import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { SyncStatusScreen } from "../../../src/screens/SyncStatusScreen";
import { TransportError } from "../../../src/sync/types";
import { ACCOUNT } from "../../helpers/fixtures";
import { FakeTransport } from "../../sync/fake-transport";
import { renderSync, SERVER_INFO } from "./harness";

const noop = (): void => undefined;
const statusScreen = (calls: string[] = []) => (
  <SyncStatusScreen
    onConnect={() => calls.push("connect")}
    onChoose={() => calls.push("choose")}
    onConflicts={() => calls.push("conflicts")}
    onProblems={() => calls.push("problems")}
    onBack={noop}
  />
);

describe("the sync status screen", () => {
  it("offers to connect when there is no server and says everything stays on the phone", async () => {
    const calls: string[] = [];
    await renderSync(() => statusScreen(calls));
    expect(await screen.findByText("Not connected. Everything stays on this phone.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Sync now" })).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: "Connect to a server" }));
    expect(calls).toEqual(["connect"]);
    expect(screen.getByText("Sync runs only while the app is open.")).toBeTruthy();
  });

  it("shows the state, the details and a Sync now that runs a sync; the status line is a polite live region", async () => {
    const ui = await renderSync(() => statusScreen(), { connected: true });
    const line = await screen.findByText("Synced");
    expect(line.props.accessibilityLiveRegion).toBe("polite");
    expect(screen.getByText("Server: https://money.example.com")).toBeTruthy();
    expect(screen.getByText("Workspace: Ana household")).toBeTruthy();
    expect(screen.getByText("Device: Ana's phone")).toBeTruthy();
    expect(screen.getByText("Not synced yet")).toBeTruthy();

    await fireEvent.press(screen.getByRole("button", { name: "Sync now" }));
    await waitFor(() => expect(screen.getByText("Last sync: 2026-11-24 12:00 UTC")).toBeTruthy());
    expect(ui.transport.calls).toContain("pull");
  });

  it("announces that it is offline with the number of waiting changes", async () => {
    const transport = new FakeTransport({
      info: () => {
        throw new TransportError("network", "unreachable");
      },
    });
    const ui = await renderSync(() => statusScreen(), { connected: true, transport });
    await ui.env.repo.create("account", ACCOUNT);
    await ui.env.repo.create("tag", { name: "x", color: "" });
    await fireEvent.press(await screen.findByRole("button", { name: "Sync now" }));
    expect(await screen.findByText("Offline, 2 changes waiting")).toBeTruthy();
    expect(screen.getByText("2 changes waiting")).toBeTruthy();
  });

  it("explains a revoked device and offers Pair again and Disconnect, keeping the data", async () => {
    const calls: string[] = [];
    const transport = new FakeTransport({
      info: () => {
        throw new TransportError("http", "revoked", 403, "device_revoked");
      },
    });
    const ui = await renderSync(() => statusScreen(calls), { connected: true, transport });
    await ui.env.repo.create("account", ACCOUNT);
    await fireEvent.press(await screen.findByRole("button", { name: "Sync now" }));

    expect(await screen.findByText("Sign-in needed")).toBeTruthy();
    expect(screen.getByText(/the device or its token was revoked/u)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Sync now" })).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: "Pair again" }));
    expect(calls).toEqual(["connect"]);
    expect(await ui.env.repo.list("account")).toHaveLength(1);
  });

  it("shows the plan quota message as an alert", async () => {
    const ui = await renderSync(() => statusScreen(), { connected: true });
    await ui.env.db.run(
      "INSERT INTO meta (key, value) VALUES ('sync.paused.account', 'The Free plan has one account.')",
    );
    await fireEvent.press(await screen.findByRole("button", { name: "Sync now" }));
    // Sync now tries a paused entity again, so the pause is read before it is lifted: show it once more by pausing again.
    await ui.env.db.run(
      "INSERT OR REPLACE INTO meta (key, value) VALUES ('sync.paused.account', 'The Free plan has one account.')",
    );
    await ui.controller.engine.refresh();
    expect(await screen.findByText("Paused: The Free plan has one account.")).toBeTruthy();
    expect(screen.getByRole("alert", { name: "Paused: The Free plan has one account." })).toBeTruthy();
  });

  it("links to the problems and the conflicts when there are some", async () => {
    const calls: string[] = [];
    const ui = await renderSync(() => statusScreen(calls), { connected: true });
    await ui.env.db.run(
      "INSERT INTO sync_problems (entity, uid, op_id, op, code, created_at) VALUES ('account', 'u', 'o', 'upsert', 'unknown_field', 'x')",
    );
    await ui.env.db.run(
      "INSERT INTO sync_conflicts (entity, uid, field, kind, created_at) VALUES ('tag', 'u', 'name', 'field', 'x')",
    );
    await ui.controller.engine.refresh();
    await fireEvent.press(await screen.findByText("1 problems"));
    await fireEvent.press(screen.getByText("1 conflicts"));
    expect(calls).toEqual(["problems", "conflicts"]);
  });

  it("asks what to do with the waiting changes before disconnecting, and keeps the data", async () => {
    const ui = await renderSync(() => statusScreen(), { connected: true });
    await ui.env.repo.create("account", ACCOUNT);
    await fireEvent.press(await screen.findByRole("button", { name: "Disconnect" }));
    expect(screen.getByText(/Your data stays on this phone/u)).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Disconnect and keep the waiting changes" }));
    expect(await screen.findByText("Not connected. Everything stays on this phone.")).toBeTruthy();
    expect(ui.tokens.value).toBeNull();
    expect(await ui.env.repo.list("account")).toHaveLength(1);
    expect(await ui.env.repo.outboxCount()).toBe(1);
  });

  it("asks for the merge choice when the phone has data and never synced", async () => {
    const calls: string[] = [];
    const transport = new FakeTransport({ info: () => SERVER_INFO });
    const ui = await renderSync(() => statusScreen(calls), { connected: true, synced: false, transport });
    await ui.env.repo.create("account", ACCOUNT);
    await fireEvent.press(await screen.findByRole("button", { name: "Sync now" }));
    expect(await screen.findByText("Choose how to join this server")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Choose how to join" }));
    expect(calls).toEqual(["choose"]);
  });
});
