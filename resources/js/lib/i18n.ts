import { usePage } from "@inertiajs/react";
import { buildDictionaries, type Dictionary, type Locale } from "../i18n/load";
import type { Key } from "../i18n/keys";
import type { SharedProps } from "../page-props";
import { formatMoney, monthLabel } from "./money";

export type { Key };

const dictionaries = buildDictionaries(import.meta.glob<{ default: Dictionary }>("../i18n/*/*.ts", { eager: true }));

export function useFinanceI18n() {
  const shared = usePage<SharedProps>().props as SharedProps & { locale?: string };
  const locale: Locale = shared.locale === "pt-BR" || shared.locale === "es" ? shared.locale : "en";
  const dictionary = dictionaries[locale];
  return {
    locale,
    t: (key: Key): string => dictionary[key] ?? key,
    money: (cents: number, currency: string) => formatMoney(cents, currency, locale),
    month: (value: string) => monthLabel(value, locale),
    date: (value: string) =>
      new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "UTC" }).format(
        new Date(`${value.slice(0, 10)}T12:00:00Z`),
      ),
  };
}
