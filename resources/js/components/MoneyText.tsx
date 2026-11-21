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

export interface MoneyTextProps {
  cents: number;
  currency: string;
  locale?: string;
  sign?: MoneySign;
  /** Colors expenses and income; the sign or label still carries the meaning. */
  tone?: boolean;
  labels?: { minus: string; plus: string };
  className?: string;
}

export function MoneyText({ cents, currency, locale, sign, tone = false, labels, className }: MoneyTextProps) {
  const parts = formatMoneyParts(cents, currency, locale, sign, labels);
  return (
    <span
      className={cn(
        "tabular-nums",
        tone && cents < 0 && "text-(--vf-expense)",
        tone && cents > 0 && "text-(--vf-income)",
        className,
      )}
    >
      <span aria-hidden="true">{parts.text}</span>
      <span className="sr-only">{parts.label}</span>
    </span>
  );
}
