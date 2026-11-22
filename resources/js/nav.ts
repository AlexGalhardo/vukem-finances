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
  Smartphone,
  Target,
  Repeat,
  Sparkles,
  Tags,
  Upload,
  WalletCards,
  Users,
  Wallet,
} from "lucide-react";
import type { Key } from "./i18n/keys";

export type NavEdition = "saas" | "personal";

export interface NavItem {
  id: string;
  labelKey: Key;
  href: string;
  icon: LucideIcon;
  /** "g d" style binding for the matching `go-<id>` action; see docs/shortcuts.md before choosing one. */
  keys?: string;
  /** Shown in one edition only; omit for both. Billing is `saas`. */
  edition?: NavEdition;
}

/** The sidebar and the navigation actions of the palette. Feature epics append their own entries here. */
export const navItems: readonly NavItem[] = [
  { id: "dashboard", labelKey: "dashboard", href: "/dashboard", icon: ChartNoAxesCombined, keys: "g d" },
  { id: "transactions", labelKey: "transactions", href: "/transactions", icon: ArrowLeftRight, keys: "g t" },
  { id: "accounts", labelKey: "accounts", href: "/accounts", icon: Wallet, keys: "g a" },
  { id: "cards", labelKey: "cards", href: "/cards", icon: WalletCards, keys: "g k" },
  { id: "categories", labelKey: "categories", href: "/categories", icon: Tags, keys: "g c" },
  { id: "budgets", labelKey: "budgets", href: "/budgets", icon: PiggyBank, keys: "g b" },
  { id: "goals", labelKey: "goals", href: "/goals", icon: Target, keys: "g g" },
  { id: "recurrences", labelKey: "recurrences", href: "/recurrences", icon: Repeat, keys: "g u" },
  { id: "reports", labelKey: "reports", href: "/reports", icon: ChartPie, keys: "g r" },
  { id: "import", labelKey: "import", href: "/import", icon: Upload, keys: "g i" },
  { id: "workspace", labelKey: "workspace", href: "/workspace", icon: Users, keys: "g w" },
  { id: "suggestions", labelKey: "suggestions", href: "/suggestions", icon: ListChecks },
  { id: "ai", labelKey: "ai", href: "/ai", icon: Sparkles },
  { id: "tokens", labelKey: "tokens", href: "/tokens", icon: KeyRound },
  { id: "devices", labelKey: "devices", href: "/devices", icon: Smartphone },
  { id: "connectedApps", labelKey: "connectedApps", href: "/connected-apps", icon: Plug, edition: "saas" },
  { id: "billing", labelKey: "billing", href: "/billing", icon: CreditCard, edition: "saas" },
];

/** `workspace.edition` is "personal" or "saas"; the personal edition has no billing entry. */
export function navItemsFor(edition: string | undefined, items: readonly NavItem[] = navItems): NavItem[] {
  const current: NavEdition = edition === "personal" ? "personal" : "saas";
  return items.filter((item) => item.edition === undefined || item.edition === current);
}

export function navActionId(item: Pick<NavItem, "id">): string {
  return `go-${item.id}`;
}

export function isActivePath(url: string, href: string): boolean {
  const path = url.split("?")[0] ?? url;
  return path === href || path.startsWith(`${href}/`);
}
