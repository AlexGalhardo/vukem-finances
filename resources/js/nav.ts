import {
  ArrowLeftRight,
  ChartNoAxesCombined,
  ChartPie,
  CreditCard,
  KeyRound,
  ListChecks,
  type LucideIcon,
  PiggyBank,
  Plug,
  Repeat,
  ShieldCheck,
  Smartphone,
  Sparkles,
  Tags,
  Target,
  Upload,
  Users,
  Wallet,
  WalletCards,
} from "lucide-react";
import type { Key } from "./i18n/keys";

export type NavEdition = "saas" | "personal";

export type NavGroup = "money" | "planning" | "insights" | "data" | "settings";

/** The sidebar groups in order; items without a group come first, unlabeled. */
export const navGroups: readonly { id: NavGroup; labelKey: Key }[] = [
  { id: "money", labelKey: "navMoney" },
  { id: "planning", labelKey: "navPlanning" },
  { id: "insights", labelKey: "navInsights" },
  { id: "data", labelKey: "navData" },
  { id: "settings", labelKey: "navSettings" },
];

export interface NavItem {
  id: string;
  labelKey: Key;
  href: string;
  icon: LucideIcon;
  /** "g d" style binding for the matching `go-<id>` action; see docs/shortcuts.md before choosing one. */
  keys?: string;
  /**
   * The id of a registered action whose shortcut this item shows, when it is not `go-<id>` (Suggestions uses
   * `review-suggestions`). Do not copy its keys into `keys`: the registry refuses a duplicate binding.
   */
  actionId?: string;
  /** The sidebar group; omit for the two top items. */
  group?: NavGroup;
  /** Shown in one edition only; omit for both. Billing is `saas`. */
  edition?: NavEdition;
}

/** The sidebar and the navigation actions of the palette. Feature epics append their own entries here. */
export const navItems: readonly NavItem[] = [
  { id: "dashboard", labelKey: "dashboard", href: "/dashboard", icon: ChartNoAxesCombined, keys: "g d" },
  { id: "transactions", labelKey: "transactions", href: "/transactions", icon: ArrowLeftRight, keys: "g t" },
  { id: "accounts", group: "money", labelKey: "accounts", href: "/accounts", icon: Wallet, keys: "g a" },
  { id: "cards", group: "money", labelKey: "cards", href: "/cards", icon: WalletCards, keys: "g k" },
  { id: "categories", group: "planning", labelKey: "categories", href: "/categories", icon: Tags, keys: "g c" },
  { id: "budgets", group: "planning", labelKey: "budgets", href: "/budgets", icon: PiggyBank, keys: "g b" },
  { id: "goals", group: "planning", labelKey: "goals", href: "/goals", icon: Target, keys: "g g" },
  { id: "recurrences", group: "money", labelKey: "recurrences", href: "/recurrences", icon: Repeat, keys: "g u" },
  { id: "reports", group: "insights", labelKey: "reports", href: "/reports", icon: ChartPie, keys: "g r" },
  { id: "import", group: "data", labelKey: "import", href: "/import", icon: Upload, keys: "g i" },
  { id: "workspace", group: "settings", labelKey: "workspace", href: "/workspace", icon: Users, keys: "g w" },
  {
    id: "suggestions",
    group: "insights",
    labelKey: "suggestions",
    href: "/suggestions",
    icon: ListChecks,
    actionId: "review-suggestions",
  },
  { id: "ai", group: "insights", labelKey: "ai", href: "/ai", icon: Sparkles },
  { id: "tokens", group: "settings", labelKey: "tokens", href: "/tokens", icon: KeyRound },
  { id: "privacy", group: "settings", labelKey: "privacy", href: "/account/privacy", icon: ShieldCheck },
  { id: "devices", group: "settings", labelKey: "devices", href: "/devices", icon: Smartphone },
  {
    id: "connectedApps",
    group: "settings",
    labelKey: "connectedApps",
    href: "/connected-apps",
    icon: Plug,
    edition: "saas",
  },
  { id: "billing", group: "settings", labelKey: "billing", href: "/billing", icon: CreditCard, edition: "saas" },
];

/** `workspace.edition` is "personal" or "saas"; the personal edition has no billing entry. */
export function navItemsFor(edition: string | undefined, items: readonly NavItem[] = navItems): NavItem[] {
  const current: NavEdition = edition === "personal" ? "personal" : "saas";
  return items.filter((item) => item.edition === undefined || item.edition === current);
}

export function navActionId(item: Pick<NavItem, "id">): string {
  return `go-${item.id}`;
}

/** The action whose shortcut the item shows: its own `actionId`, else the `go-<id>` navigation action. */
export function navHintActionId(item: Pick<NavItem, "id" | "actionId">): string {
  return item.actionId ?? navActionId(item);
}

export function isActivePath(url: string, href: string): boolean {
  const path = url.split("?")[0] ?? url;
  return path === href || path.startsWith(`${href}/`);
}
