import type { ReactNode } from "react";
import { Text, View } from "react-native";
import { Button } from "../components/Button";
import { EmptyState } from "../components/EmptyState";
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import { useApp, useLoad } from "../core/state";
import { useSync } from "../sync/context";
import { listMissingRefs } from "../sync/integrity";
import { listProblems } from "../sync/results";

/** Mutations the server refused for good, and references that still name nothing after a sync (the data check list). */
export function SyncProblemsScreen(props: { onBack?: () => void }): ReactNode {
  const ui = useUi();
  const app = useApp();
  const sync = useSync();
  const data = useLoad(async (a) => ({ problems: await listProblems(a.db), missing: await listMissingRefs(a.db) }));

  return (
    <Page title={ui.t("titleSyncProblems")} onBack={props.onBack}>
      {data === undefined ? null : data.problems.length === 0 && data.missing.length === 0 ? (
        <EmptyState message={ui.t("syncProblemsEmpty")} />
      ) : (
        <>
          {data.problems.map((p) => (
            <View key={p.id} style={{ gap: ui.spacing[2] }}>
              <Text style={{ color: ui.colors.text }}>
                {ui.t("syncProblemLine", { entity: p.entity, message: p.message === "" ? p.code : p.message })}
              </Text>
              <Button
                label={ui.t("syncTryAgain")}
                onPress={() => {
                  void sync.retryProblem(p.id).then(() => app.changed());
                }}
              />
              <Button
                label={ui.t("syncDiscard")}
                variant="destructive"
                onPress={() => {
                  void sync.discardProblem(p.id).then(() => app.changed());
                }}
              />
            </View>
          ))}
          {data.missing.length === 0 ? null : (
            <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
              {ui.t("syncDataCheck")}
            </Text>
          )}
          {data.missing.map((m) => (
            <Text key={`${m.entity}/${m.uid}/${m.field}`} style={{ color: ui.colors.text }}>
              {ui.t("syncDataCheckLine", { entity: m.entity, field: m.field })}
            </Text>
          ))}
        </>
      )}
    </Page>
  );
}
