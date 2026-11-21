// Helpers of the billing page and the PIX dialog. The server is the only source of prices and of the outcome of a
// payment: this file formats and polls, it decides nothing.

export type PixCharge = {
  ref: string;
  status: "pending" | "paid" | "expired" | "refunded" | "disputed";
  months: number;
  amountCents: number;
  brCode?: string;
  qr?: string;
  expiresAt: string;
  plan?: string;
};

/** Seconds between two polls of an open charge (the server limits the rate as well). */
export const POLL_SECONDS = 3;

/** A charge that will not change by waiting: the dialog stops polling. */
export function isFinal(status: PixCharge["status"]): boolean {
  return status !== "pending";
}

/** Replaces `{name}` placeholders: translations carry them because word order differs between languages. */
export function fill(text: string, values: Record<string, string | number>): string {
  return text.replace(/\{(\w+)\}/g, (whole, name: string) => (name in values ? String(values[name]) : whole));
}

/** The QR image is shown only when the server sent a PNG data URL; anything else is dropped. */
export function safeQr(value: string | undefined): string | undefined {
  return value?.startsWith("data:image/png;base64,") ? value : undefined;
}
