import { type ReactNode, useState } from "react";
import { Text } from "react-native";
import { backupFileName, buildBackup } from "../backup/export";
import type { Sharer } from "../backup/share";
import { Button } from "../components/Button";
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import { useApp } from "../core/state";
import { useSync } from "../sync/context";

type View = { name: "choose" } | { name: "confirm"; rows: number; pending: number } | { name: "merged"; count: number };

/**
 * The first connection of a phone that already has data: merge, replace (after a
 * confirmation that says local data will be deleted, with an export offered first)
 * or cancel.
 */
export function SyncMergeScreen(props: { sharer: Sharer; onDone: () => void; onBack?: () => void }): ReactNode {
  const ui = useUi();
  const app = useApp();
  const sync = useSync();
  const [view, setView] = useState<View>({ name: "choose" });
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function run(work: () => Promise<void>): Promise<void> {
    setBusy(true);
    setMessage(null);
    try {
      await work();
    } catch {
      setMessage(ui.t("errorGeneric"));
    } finally {
      setBusy(false);
    }
  }

  async function exportFirst(): Promise<void> {
    await run(async () => {
      const backup = await buildBackup(
        app.db,
        { name: ui.t("appName"), currency: app.settings.currency, timezone: app.settings.timezone },
        new Date(),
      );
      await props.sharer.shareText(backupFileName(app.today()), JSON.stringify(backup, null, 2));
    });
  }

  if (view.name === "merged") {
    return (
      <Page title={ui.t("titleSyncMerge")}>
        <Text accessibilityLiveRegion="polite" style={{ color: ui.colors.text }}>
          {ui.t("syncMergeDone", { count: view.count })}
        </Text>
        <Button
          label={ui.t("done")}
          onPress={() => {
            app.changed();
            props.onDone();
          }}
        />
      </Page>
    );
  }

  if (view.name === "confirm") {
    return (
      <Page title={ui.t("syncReplaceTitle")} onBack={() => setView({ name: "choose" })}>
        <Text style={{ color: ui.colors.text }}>
          {ui.t("syncReplaceBody", { count: view.rows, pending: view.pending })}
        </Text>
        <Button
          label={ui.t("syncReplaceExport")}
          variant="secondary"
          disabled={busy}
          onPress={() => void exportFirst()}
        />
        <Button
          label={ui.t("syncReplaceConfirm")}
          variant="destructive"
          disabled={busy}
          onPress={() =>
            void run(async () => {
              await sync.replace();
              app.changed();
              props.onDone();
            })
          }
        />
        <Button label={ui.t("cancel")} variant="secondary" onPress={() => setView({ name: "choose" })} />
        {message === null ? null : (
          <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
            {message}
          </Text>
        )}
      </Page>
    );
  }

  return (
    <Page title={ui.t("titleSyncMerge")} onBack={props.onBack}>
      <Text style={{ color: ui.colors.text }}>{ui.t("syncMergeIntro")}</Text>
      <Button
        label={ui.t("syncMergeButton")}
        disabled={busy}
        onPress={() =>
          void run(async () => {
            const result = await sync.merge();
            setView({ name: "merged", count: result.merged });
          })
        }
      />
      <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("syncMergeHelp")}</Text>
      <Button
        label={ui.t("syncReplaceButton")}
        variant="destructive"
        disabled={busy}
        onPress={() =>
          void run(async () => {
            const d = await sync.describeReplace();
            setView({ name: "confirm", rows: d.rows, pending: d.pending });
          })
        }
      />
      <Button
        label={ui.t("syncMergeCancel")}
        variant="secondary"
        disabled={busy}
        onPress={() =>
          void run(async () => {
            await sync.disconnect(false);
            props.onDone();
          })
        }
      />
      {message === null ? null : (
        <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
          {message}
        </Text>
      )}
    </Page>
  );
}
