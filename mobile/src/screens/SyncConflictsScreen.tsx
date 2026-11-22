import type { ReactNode } from "react";
import { Text, View } from "react-native";
import { Button } from "../components/Button";
import { EmptyState } from "../components/EmptyState";
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import { useApp, useLoad } from "../core/state";
import { useSync } from "../sync/context";

interface ConflictRow {
  id: number;
  entity: string;
  uid: string;
  field: string;
  kind: string;
  lost_value: string;
  applied_value: string;
}

function show(json: string): string {
  const v = JSON.parse(json) as unknown;
  return v === null ? "-" : typeof v === "string" ? v : JSON.stringify(v);
}

/** The values that lost a merge, each with both sides. "Apply again" writes the replaced value as a normal edit. */
export function SyncConflictsScreen(props: { onBack?: () => void }): ReactNode {
  const ui = useUi();
  const app = useApp();
  const sync = useSync();
  const rows = useLoad((a) =>
    a.db.all<ConflictRow>(
      "SELECT id, entity, uid, field, kind, lost_value, applied_value FROM sync_conflicts WHERE dismissed_at IS NULL ORDER BY id",
    ),
  );

  return (
    <Page title={ui.t("titleSyncConflicts")} onBack={props.onBack}>
      {rows === undefined ? null : rows.length === 0 ? (
        <EmptyState message={ui.t("syncConflictsEmpty")} />
      ) : (
        rows.map((c) => (
          <View key={c.id} style={{ gap: ui.spacing[2] }}>
            <Text style={{ color: ui.colors.text }}>
              {c.kind === "edited_deleted_row"
                ? ui.t("syncConflictDeleted", { entity: c.entity, field: c.field, lost: show(c.lost_value) })
                : ui.t("syncConflictField", {
                    entity: c.entity,
                    field: c.field,
                    applied: show(c.applied_value),
                    lost: show(c.lost_value),
                  })}
            </Text>
            <Button
              label={c.kind === "edited_deleted_row" ? ui.t("syncRestoreAndApply") : ui.t("syncApplyAgain")}
              onPress={() => {
                void sync.applyConflictAgain(c.id).then(() => app.changed());
              }}
            />
            <Button
              label={ui.t("syncDismiss")}
              variant="secondary"
              onPress={() => {
                void sync.dismissConflict(c.id).then(() => app.changed());
              }}
            />
          </View>
        ))
      )}
    </Page>
  );
}
