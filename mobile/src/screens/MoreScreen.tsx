import type { ReactNode } from "react";
import { ListRow } from "../components/ListRow";
import { Page } from "../components/Page";
import { useUi } from "../components/ui";
import type { MessageKey } from "../i18n";

export type MoreTarget = "cards" | "categories" | "tags" | "budgets" | "recurrences" | "reports" | "settings";

const ITEMS: { target: MoreTarget; label: MessageKey }[] = [
  { target: "cards", label: "titleCards" },
  { target: "categories", label: "titleCategories" },
  { target: "tags", label: "titleTags" },
  { target: "budgets", label: "titleBudgets" },
  { target: "recurrences", label: "titleRecurrences" },
  { target: "reports", label: "titleReports" },
  { target: "settings", label: "titleSettings" },
];

/** The rest of the app: one row per screen that does not have a tab. */
export function MoreScreen(props: { onOpen: (target: MoreTarget) => void }): ReactNode {
  const ui = useUi();
  return (
    <Page title={ui.t("tabMore")}>
      {ITEMS.map((i) => (
        <ListRow key={i.target} title={ui.t(i.label)} onPress={() => props.onOpen(i.target)} />
      ))}
    </Page>
  );
}
