import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { describeError } from "../core/errors";
import { type App, useApp, useLoad } from "../core/state";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { DeleteButton } from "../components/DeleteButton";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { Page } from "../components/Page";
import { TextField } from "../components/TextField";
import { useUi } from "../components/ui";
import type { Row } from "../db/entities";

/** Categories, income and expense, with their subcategories under them; create, edit, move, archive and delete. */
export function CategoriesScreen(props: { onBack?: () => void }): ReactNode {
  const app = useApp();
  const ui = useUi();
  const [editing, setEditing] = useState<string | "new" | null>(null);
  const [name, setName] = useState("");
  const [kind, setKind] = useState("expense");
  const [parent, setParent] = useState<string | null>(null);
  const [reassign, setReassign] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const categories = useLoad((a: App) => a.repo.list("category"));

  function open(target: string | "new"): void {
    const c = categories?.find((x) => x.uid === target);
    setName(c?.name ?? "");
    setKind(c?.kind ?? "expense");
    setParent(c?.parent_uid ?? null);
    setReassign(null);
    setError(null);
    setEditing(target);
  }

  async function save(): Promise<void> {
    if (name.trim() === "") {
      setError(ui.t("required"));
      return;
    }
    try {
      if (editing === "new") {
        await app.ledger.createCategory({ name: name.trim(), kind, parent_uid: parent, color: "", icon: "" });
      } else if (editing !== null) {
        await app.repo.update("category", editing, { name: name.trim() });
        const current = categories?.find((c) => c.uid === editing);
        if (current !== undefined && current.parent_uid !== parent) {
          await app.ledger.moveCategory(editing, parent);
        }
      }
    } catch (e) {
      setError(describeError(ui, e));
      return;
    }
    setEditing(null);
    app.changed();
  }

  async function archive(c: Row<"category">): Promise<void> {
    await app.repo.update("category", c.uid, { archived_at: c.archived_at === null ? new Date().toISOString() : null });
    setEditing(null);
    app.changed();
  }

  async function remove(uid: string): Promise<void> {
    try {
      await app.ledger.deleteCategory(uid, reassign);
    } catch (e) {
      setError(describeError(ui, e));
      return;
    }
    setEditing(null);
    app.changed();
  }

  if (editing !== null) {
    const current = categories?.find((c) => c.uid === editing);
    const tops = (categories ?? []).filter(
      (c) => c.parent_uid === null && c.kind === kind && c.uid !== editing && c.archived_at === null,
    );
    const sameKind = (categories ?? []).filter((c) => c.kind === kind && c.uid !== editing && c.archived_at === null);
    return (
      <Page title={editing === "new" ? ui.t("categoryNew") : ui.t("categoryEdit")} onBack={() => setEditing(null)}>
        <TextField label={ui.t("name")} value={name} onChangeText={setName} />
        {editing === "new" ? (
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
            <Chip
              role="radio"
              label={ui.t("expense")}
              selected={kind === "expense"}
              onPress={() => {
                setKind("expense");
                setParent(null);
              }}
            />
            <Chip
              role="radio"
              label={ui.t("income")}
              selected={kind === "income"}
              onPress={() => {
                setKind("income");
                setParent(null);
              }}
            />
          </View>
        ) : null}
        <Text style={{ color: ui.colors.textMuted }}>{ui.t("categoryParent")}</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
          <Chip role="radio" label={ui.t("none")} selected={parent === null} onPress={() => setParent(null)} />
          {tops.map((t) => (
            <Chip
              key={t.uid}
              role="radio"
              label={t.name}
              selected={parent === t.uid}
              onPress={() => setParent(t.uid)}
            />
          ))}
        </View>
        {error === null ? null : (
          <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
            {error}
          </Text>
        )}
        <Button label={ui.t("save")} onPress={() => void save()} />
        {current === undefined ? null : (
          <>
            <Button
              variant="secondary"
              label={current.archived_at === null ? ui.t("archive") : ui.t("unarchive")}
              onPress={() => void archive(current)}
            />
            <Text style={{ color: ui.colors.textMuted }}>{ui.t("categoryReassign")}</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] }}>
              <Chip
                role="radio"
                label={ui.t("uncategorized")}
                selected={reassign === null}
                onPress={() => setReassign(null)}
              />
              {sameKind.map((c) => (
                <Chip
                  key={c.uid}
                  role="radio"
                  label={c.name}
                  selected={reassign === c.uid}
                  onPress={() => setReassign(c.uid)}
                />
              ))}
            </View>
            <DeleteButton onConfirm={() => void remove(current.uid)} />
          </>
        )}
      </Page>
    );
  }

  const list = categories ?? [];
  const ordered = (k: string): Row<"category">[] =>
    list
      .filter((c) => c.kind === k && c.parent_uid === null)
      .flatMap((top) => [top, ...list.filter((c) => c.parent_uid === top.uid)]);

  return (
    <Page
      title={ui.t("titleCategories")}
      onBack={props.onBack}
      action={{ label: ui.t("add"), onPress: () => open("new") }}
    >
      {categories === undefined ? null : list.length === 0 ? (
        <EmptyState
          message={ui.t("categoriesEmpty")}
          action={{ label: ui.t("categoryNew"), onPress: () => open("new") }}
        />
      ) : (
        (["expense", "income"] as const).map((k) => (
          <View key={k}>
            <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
              {ui.t(k)}
            </Text>
            {ordered(k).map((c) => (
              <ListRow
                key={c.uid}
                title={`${c.parent_uid === null ? "" : "↳ "}${c.name}${c.archived_at === null ? "" : ` (${ui.t("accountArchived")})`}`}
                onPress={() => open(c.uid)}
              />
            ))}
          </View>
        ))
      )}
    </Page>
  );
}
