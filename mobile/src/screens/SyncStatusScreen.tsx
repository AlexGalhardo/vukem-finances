import { type ReactNode, useState } from "react";
import { Text } from "react-native";
import { Button } from "../components/Button";
import { ListRow } from "../components/ListRow";
import { Page } from "../components/Page";
import type { Ui } from "../components/ui";
import { useUi } from "../components/ui";
import type { MessageKey } from "../i18n";
import { useSync, useSyncView } from "../sync/context";
import type { EngineSnapshot } from "../sync/engine";

export function statusLine(ui: Ui, view: EngineSnapshot): string {
  switch (view.status) {
    case "disconnected":
      return ui.t("syncDisconnected");
    case "idle":
      return ui.t("syncIdle");
    case "syncing":
      return ui.t("syncSyncing");
    case "offline":
      return view.pending > 0 ? ui.t("syncOffline", { count: view.pending }) : ui.t("syncOfflineNone");
    case "auth_failed":
      return ui.t("syncAuthFailed");
    case "needs_merge_choice":
      return ui.t("syncNeedsChoice");
    default: {
      const key: MessageKey =
        view.failure === "unsupported"
          ? "syncErrorUnsupported"
          : view.failure === "rate_limited"
            ? "syncRateLimited"
            : view.failure === "client"
              ? "syncErrorClient"
              : "syncErrorServer";
      return ui.t(key);
    }
  }
}

/** The time of the last sync in UTC, in a fixed shape that does not depend on the device locale. */
function stamp(iso: string): string {
  return `${iso.slice(0, 10)} ${iso.slice(11, 16)} UTC`;
}

/** The state of the sync, what waits, and the actions around it. A polite live region announces each change. */
export function SyncStatusScreen(props: {
  onBack?: () => void;
  onConnect: () => void;
  onChoose: () => void;
  onConflicts: () => void;
  onProblems: () => void;
}): ReactNode {
  const ui = useUi();
  const sync = useSync();
  const view = useSyncView();
  const [asking, setAsking] = useState(false);

  if (view === undefined) {
    return (
      <Page title={ui.t("titleSync")} onBack={props.onBack}>
        {null}
      </Page>
    );
  }

  if (asking) {
    return (
      <Page title={ui.t("syncDisconnectTitle")} onBack={() => setAsking(false)}>
        <Text style={{ color: ui.colors.text }}>{ui.t("syncDisconnectBody")}</Text>
        <Button
          label={ui.t("syncKeepQueue")}
          onPress={() => {
            void sync.disconnect(false).then(() => setAsking(false));
          }}
        />
        <Button
          label={ui.t("syncClearQueue")}
          variant="destructive"
          onPress={() => {
            void sync.disconnect(true).then(() => setAsking(false));
          }}
        />
        <Button label={ui.t("cancel")} variant="secondary" onPress={() => setAsking(false)} />
      </Page>
    );
  }

  const connected = view.status !== "disconnected" && view.connection !== null;

  return (
    <Page title={ui.t("titleSync")} onBack={props.onBack}>
      <Text accessibilityLiveRegion="polite" style={{ color: ui.colors.text, fontWeight: "700" }}>
        {statusLine(ui, view)}
      </Text>

      {Object.entries(view.paused).map(([entity, message]) => (
        <Text key={entity} accessibilityRole="alert" style={{ color: ui.colors.expense }}>
          {ui.t("syncQuota", { message })}
        </Text>
      ))}

      {view.status === "auth_failed" ? <Text style={{ color: ui.colors.text }}>{ui.t("syncAuthExplain")}</Text> : null}

      {connected && view.connection !== null ? (
        <>
          <Text style={{ color: ui.colors.text }}>{ui.t("syncServer", { url: view.connection.serverUrl })}</Text>
          <Text style={{ color: ui.colors.text }}>
            {ui.t("syncWorkspace", { name: view.connection.workspaceName })}
          </Text>
          <Text style={{ color: ui.colors.text }}>{ui.t("syncDevice", { name: view.connection.deviceName })}</Text>
          <Text style={{ color: ui.colors.text }}>
            {view.lastSyncAt === null ? ui.t("syncNever") : ui.t("syncLastSync", { time: stamp(view.lastSyncAt) })}
          </Text>
          <Text style={{ color: ui.colors.text }}>{ui.t("syncPending", { count: view.pending })}</Text>
        </>
      ) : null}

      {view.problems > 0 ? (
        <ListRow
          title={ui.t("syncProblemsCount", { count: view.problems })}
          subtitle={ui.t("syncReviewProblems")}
          onPress={props.onProblems}
        />
      ) : null}
      {view.conflicts > 0 ? (
        <ListRow
          title={ui.t("syncConflictsCount", { count: view.conflicts })}
          subtitle={ui.t("syncReviewConflicts")}
          onPress={props.onConflicts}
        />
      ) : null}

      {view.status === "needs_merge_choice" ? <Button label={ui.t("syncChoose")} onPress={props.onChoose} /> : null}
      {view.status === "auth_failed" ? <Button label={ui.t("syncPairAgain")} onPress={props.onConnect} /> : null}
      {connected && view.status !== "auth_failed" && view.status !== "needs_merge_choice" ? (
        <Button
          label={ui.t("syncNow")}
          disabled={view.status === "syncing"}
          onPress={() => {
            void sync.engine.trigger("manual");
          }}
        />
      ) : null}
      {connected ? (
        <Button label={ui.t("syncDisconnect")} variant="secondary" onPress={() => setAsking(true)} />
      ) : (
        <Button label={ui.t("syncConnect")} onPress={props.onConnect} />
      )}

      <Text style={{ color: ui.colors.textMuted }}>{ui.t("syncLimits")}</Text>
    </Page>
  );
}
