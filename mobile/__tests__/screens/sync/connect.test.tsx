import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { SyncConnectScreen } from "../../../src/screens/SyncConnectScreen";
import { loadConnection } from "../../../src/sync/state";
import { TransportError } from "../../../src/sync/types";
import { FakeTransport } from "../../sync/fake-transport";
import { renderSync } from "./harness";

const SECRET = ["tok", "en", "-", "from", "-", "pairing"].join("");
const CODE = ["ABCD", "EFGH"].join("-");

const paired = (): FakeTransport =>
  new FakeTransport({
    pair: () => ({
      token: SECRET,
      device_id: "DEVICE9",
      workspace: { uid: "7", name: "Ana household", currency: "USD", timezone: "UTC" },
      server_time: "",
    }),
  });

async function fill(url: string, code: string): Promise<void> {
  await fireEvent.changeText(screen.getByLabelText("Server address"), url);
  await fireEvent.changeText(screen.getByLabelText("Pairing code"), code);
}

describe("the connect screen", () => {
  it("has labelled fields, a default device name and tells where the code comes from", async () => {
    await renderSync(() => <SyncConnectScreen onConnected={() => undefined} />);
    expect(screen.getByLabelText("Server address")).toBeTruthy();
    expect(screen.getByLabelText("Pairing code")).toBeTruthy();
    expect(screen.getByLabelText("Device name").props.value).toBe("Test phone");
    expect(screen.getByText("On the web, open Devices and press Add device.")).toBeTruthy();
  });

  it("pairs over https, keeps the token only in secure storage and clears the code", async () => {
    const connected: string[] = [];
    const ui = await renderSync(() => <SyncConnectScreen onConnected={() => connected.push("done")} />, {
      transport: paired(),
    });
    await fill("https://money.example.com", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));

    await waitFor(() => expect(connected).toEqual(["done"]));
    expect(ui.tokens.value).toBe(SECRET);
    expect(await loadConnection(ui.env.db)).toMatchObject({
      serverUrl: "https://money.example.com",
      deviceId: "DEVICE9",
    });
    expect(JSON.stringify(await ui.env.db.all("SELECT * FROM meta"))).not.toContain(SECRET);
    expect(screen.getByLabelText("Pairing code").props.value).toBe("");
  });

  it("refuses http to a public host with a message and does not call the server", async () => {
    const ui = await renderSync(() => <SyncConnectScreen onConnected={() => undefined} />, { transport: paired() });
    await fill("http://money.example.com", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Plain http is not allowed for this address. Use https.",
    );
    expect(ui.transport.calls).toEqual([]);
  });

  it("needs the plain-traffic confirmation before http to a private address", async () => {
    const connected: string[] = [];
    const ui = await renderSync(() => <SyncConnectScreen onConnected={() => connected.push("done")} />, {
      transport: paired(),
    });
    await fill("http://192.168.1.20:3000", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));

    expect(await screen.findByText("This address is not encrypted")).toBeTruthy();
    expect(screen.getByText(/can be read by anyone on the network/u)).toBeTruthy();
    expect(ui.transport.calls).toEqual([]);

    await fireEvent.press(screen.getByRole("button", { name: "Connect without encryption" }));
    await waitFor(() => expect(connected).toEqual(["done"]));
    expect((await loadConnection(ui.env.db))?.insecure).toBe(true);
  });

  it("can back out of the confirmation", async () => {
    await renderSync(() => <SyncConnectScreen onConnected={() => undefined} />, { transport: paired() });
    await fill("http://nas.local", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Cancel" }));
    expect(screen.getByLabelText("Server address")).toBeTruthy();
  });

  it("asks for the code and for a valid address", async () => {
    await renderSync(() => <SyncConnectScreen onConnected={() => undefined} />, { transport: paired() });
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByText("Enter the pairing code.")).toBeTruthy();
    await fill("", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByText("Enter the server address.")).toBeTruthy();
    await fill("money.example.com", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByText("The address must start with https://")).toBeTruthy();
  });

  it.each([
    [
      new TransportError("http", "no", 422, "invalid_pairing_code"),
      "That code did not work. Check it or make a new one.",
    ],
    [new TransportError("network", "down"), "Could not reach the server."],
    [new TransportError("http", "slow", 429, "rate_limited"), "Too many tries. Wait a minute and try again."],
  ])("explains a failed pairing (%#) and stores nothing", async (error, message) => {
    const failing = new FakeTransport({
      pair: () => {
        throw error;
      },
    });
    const ui = await renderSync(() => <SyncConnectScreen onConnected={() => undefined} />, { transport: failing });
    await fill("https://money.example.com", CODE);
    await fireEvent.press(screen.getByRole("button", { name: "Connect" }));
    expect(await screen.findByText(message)).toBeTruthy();
    expect(ui.tokens.value).toBeNull();
    expect(await loadConnection(ui.env.db)).toBeNull();
  });
});
