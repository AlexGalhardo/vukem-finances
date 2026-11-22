/** Basis points (1/100 of a percent, an integer from the server) as the text of a percentage, for reading only. */
export function percentText(bps: number, locale = "en", signed = false): string {
  return new Intl.NumberFormat(locale, {
    style: "percent",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    signDisplay: signed ? "exceptZero" : "auto",
  }).format(bps / 10000);
}

export const presets = ["this_month", "last_month", "last_3_months", "last_12_months", "this_year", "custom"] as const;
export type Preset = (typeof presets)[number];

/** The query string of a period: a preset alone, or custom with its two dates. */
export function periodQuery(preset: string, from: string, to: string): Record<string, string> {
  return preset === "custom" ? { period: "custom", from, to } : { period: preset };
}
