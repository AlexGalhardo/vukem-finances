import { ArrowDownLeft, ArrowLeftRight, ArrowUpRight } from "lucide-react";
import { cn } from "../lib/utils";

export type MoneySign = "auto" | "always" | "never";

export interface MoneyParts {
  text: string;
  /** Spoken form such as "minus 12.50 US dollars"; the visible sign is a symbol screen readers may skip. */
  label: string;
  negative: boolean;
}

export const defaultSignLabels = { minus: "minus", plus: "plus" } as const;

/** Formats int64 cents for reading. Never does arithmetic on floats beyond the display division. */
export function formatMoneyParts(
  cents: number,
  currency: string,
  locale = "en",
  sign: MoneySign = "auto",
  labels: { minus: string; plus: string } = defaultSignLabels,
): MoneyParts {
  const negative = cents < 0;
  const signDisplay = sign === "never" ? "never" : sign === "always" ? "exceptZero" : "auto";
  const text = new Intl.NumberFormat(locale, { style: "currency", currency, signDisplay }).format(cents / 100);
  const spoken = new Intl.NumberFormat(locale, { style: "currency", currency, currencyDisplay: "name" }).format(
    Math.abs(cents) / 100,
  );
  const word =
    sign === "never" ? "" : negative ? `${labels.minus} ` : sign === "always" && cents > 0 ? `${labels.plus} ` : "";
  return { text, label: `${word}${spoken}`, negative };
}

export type MoneyKind = "income" | "expense" | "transfer" | "neutral";
export type MoneySize = "sm" | "md" | "lg" | "hero";

export interface MoneyTextProps {
  cents: number;
  currency: string;
  locale?: string;
  /** Defaults to "always" for a colored positive amount, so income never relies on color alone. */
  sign?: MoneySign;
  /** Derived from the sign of the amount when omitted. */
  kind?: MoneyKind;
  size?: MoneySize;
  /** A direction icon before the amount: for lists. */
  showIcon?: boolean;
  /** Colors income and expense; the sign still carries the meaning. */
  colored?: boolean;
  /** The former name of `colored`. */
  tone?: boolean;
  labels?: { minus: string; plus: string };
  className?: string;
}

const sizeClass: Record<MoneySize, string> = {
  sm: "text-caption",
  md: "",
  lg: "text-h2",
  hero: "text-hero-number",
};

const kindClass: Record<MoneyKind, string> = {
  income: "text-income",
  expense: "text-expense",
  transfer: "text-muted-foreground",
  neutral: "",
};

const kindIcon = { income: ArrowDownLeft, expense: ArrowUpRight, transfer: ArrowLeftRight } as const;

function kindOf(cents: number): MoneyKind {
  return cents > 0 ? "income" : cents < 0 ? "expense" : "neutral";
}

/** The only way to show money: tabular figures, the sign before the symbol (U+2212 for minus), a spoken label. */
export function MoneyText({
  cents,
  currency,
  locale,
  sign,
  kind,
  size = "md",
  showIcon = false,
  colored,
  tone = false,
  labels,
  className,
}: MoneyTextProps) {
  const paint = colored ?? tone;
  const resolved = kind ?? kindOf(cents);
  const effectiveSign = sign ?? (paint && cents > 0 ? "always" : "auto");
  const parts = formatMoneyParts(cents, currency, locale, effectiveSign, labels);
  const Icon = showIcon && resolved !== "neutral" ? kindIcon[resolved] : null;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 tabular-nums",
        sizeClass[size],
        paint && cents !== 0 && kindClass[resolved],
        paint && cents === 0 && "text-muted-foreground",
        className,
      )}
    >
      {Icon && <Icon aria-hidden="true" className="size-[1em] shrink-0" />}
      <span aria-hidden="true">{parts.text.replace("-", "−")}</span>
      <span className="sr-only">{parts.label}</span>
    </span>
  );
}
