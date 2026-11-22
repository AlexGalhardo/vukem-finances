import type { Key } from "../i18n/keys";

export type NotificationParams = Record<string, unknown>;

export interface Formatters {
  money: (cents: number, currency: string) => string;
  date: (value: string) => string;
  currency: string;
}

const text = (value: unknown): string => (typeof value === "string" ? value : "");
const cents = (value: unknown): number => (typeof value === "number" ? value : 0);

/** Fills the {placeholders} of a translated message; a missing value stays visible as its name. */
export function fill(message: string, values: Record<string, string | number>): string {
  return message.replace(/\{(\w+)\}/g, (whole, name: string) => (name in values ? String(values[name]) : whole));
}

/** The values a notification title needs, formatted for the person: money in the workspace currency, dates in their locale. */
export function titleValues(params: NotificationParams, f: Formatters): Record<string, string | number> {
  const money = (key: string) => f.money(cents(params[key]), f.currency);
  return {
    name: text(params.name),
    percent: typeof params.percent === "number" ? params.percent : 0,
    spent: money("spent_cents"),
    limit: money("limit_cents"),
    used: money("used_cents"),
    remaining: money("remaining_cents"),
    date: text(params.due_on || params.date) ? f.date(text(params.due_on || params.date)) : "",
  };
}

/** The title of a notification in the language of the person, from its stored key and params. */
export function notificationTitle(
  titleKey: string,
  params: NotificationParams,
  t: (key: Key) => string,
  f: Formatters,
): string {
  return fill(t(titleKey as Key), titleValues(params, f));
}

/** The accessible name of the bell: it carries the unread count. */
export function bellLabel(unread: number, t: (key: Key) => string): string {
  return unread > 0 ? fill(t("notificationsBell"), { n: unread }) : t("notificationsBellNone");
}
