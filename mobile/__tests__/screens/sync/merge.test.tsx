import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import type { Sharer } from "../../../src/backup/share";
import { SyncMergeScreen } from "../../../src/screens/SyncMergeScreen";
import type { SnapshotPage } from "../../../src/sync/types";
import { ACCOUNT } from "../../helpers/fixtures";
import { FakeTransport } from "../../sync/fake-transport";
import { categoryRow } from "../../sync/wire";
import { renderSync, SERVER_INFO } from "./harness";

const CATEGORY = { name: "Food", kind: "expense", parent_uid: null, color: "", icon: "", archived_at: null };

function server(): FakeTransport {
  return new FakeTransport({
    info: () => SERVER_INFO,
    pull: () => ({ changes: [], next_cursor: 5, has_more: false, server_time: "" }),
    push: (m) =>
      m.map((x) => ({
        op_id: x.op_id,
        status: "applied" as const,
        entity: x.entity,
        uid: x.uid,
        row: null,
        version: 1,
        seq: 1,
        conflicts: [],
        effects: [],
        replaced_by_uid: "",
        error: null,
      })),
    snapshot: (q): SnapshotPage => ({
      snapshot_id: "s",
      snapshot_cursor: 5,
      entity: q.entity,
      rows: q.entity === "category" ? [categoryRow("01JD5Z8K3M0000000000000C01")] : [],
      next_after: "",
      has_more: false,
    }),
  });
}

let shared: { name: string; content: string }[];
const sharer: Sharer = {
  shareText: async (name, content) => {
    shared.push({ name, content });
  },
};

beforeEach(() => {
  shared = [];
});

describe("the merge choice screen", () => {
  it("merges equal names and reports how many were merged", async () => {
    const done: string[] = [];
    const ui = await renderSync(() => <SyncMergeScreen sharer={sharer} onDone={() => done.push("done")} />, {
      connected: true,
      synced: false,
      transport: server(),
    });
    await ui.env.repo.create("category", CATEGORY);
    await ui.env.repo.create("account", ACCOUNT);

    expect(screen.getByText("This phone already has data. Choose what to do with it.")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Merge (recommended)" }));

    expect(await screen.findByText("Merged 1 items with the same name.")).toBeTruthy();
    expect((await ui.env.repo.list("category")).map((c) => c.uid)).toEqual(["01JD5Z8K3M0000000000000C01"]);
    expect(await ui.env.repo.list("account")).toHaveLength(1);
    await fireEvent.press(screen.getByRole("button", { name: "Done" }));
    expect(done).toEqual(["done"]);
  });

  it("asks for a confirmation that says local data will be deleted, and offers an export first", async () => {
    const ui = await renderSync(() => <SyncMergeScreen sharer={sharer} onDone={() => undefined} />, {
      connected: true,
      synced: false,
      transport: server(),
    });
    await ui.env.repo.create("account", ACCOUNT);
    await ui.env.repo.create("category", CATEGORY);

    await fireEvent.press(screen.getByRole("button", { name: "Replace this phone's data with the server's" }));
    expect(await screen.findByText("Replace the data on this phone?")).toBeTruthy();
    expect(screen.getByText(/2 items on this phone will be deleted and 2 waiting changes will be lost/u)).toBeTruthy();
    // Nothing is deleted by asking.
    expect(await ui.env.repo.list("account")).toHaveLength(1);

    await fireEvent.press(screen.getByRole("button", { name: "Export a backup first" }));
    await waitFor(() => expect(shared).toHaveLength(1));
    expect(shared[0]?.name).toMatch(/\.json$/u);
    expect(shared[0]?.content).toContain('"name": "Checking"');
    expect(await ui.env.repo.list("account")).toHaveLength(1);

    await fireEvent.press(screen.getByRole("button", { name: "Delete local data and replace" }));
    await waitFor(async () => expect(await ui.env.repo.list("account")).toEqual([]));
    expect((await ui.env.repo.list("category")).map((c) => c.uid)).toEqual(["01JD5Z8K3M0000000000000C01"]);
    expect(await ui.env.repo.outboxCount()).toBe(0);
  });

  it("can step back from the confirmation without deleting anything", async () => {
    const ui = await renderSync(() => <SyncMergeScreen sharer={sharer} onDone={() => undefined} />, {
      connected: true,
      synced: false,
      transport: server(),
    });
    await ui.env.repo.create("account", ACCOUNT);
    await fireEvent.press(screen.getByRole("button", { name: "Replace this phone's data with the server's" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Merge (recommended)" })).toBeTruthy();
    expect(await ui.env.repo.list("account")).toHaveLength(1);
  });

  it("cancels by disconnecting and keeps the data", async () => {
    const done: string[] = [];
    const ui = await renderSync(() => <SyncMergeScreen sharer={sharer} onDone={() => done.push("done")} />, {
      connected: true,
      synced: false,
      transport: server(),
    });
    await ui.env.repo.create("account", ACCOUNT);
    await fireEvent.press(screen.getByRole("button", { name: "Cancel and disconnect" }));
    await waitFor(() => expect(done).toEqual(["done"]));
    expect(ui.tokens.value).toBeNull();
    expect(await ui.env.repo.list("account")).toHaveLength(1);
  });
});
