/** Largest money amount in cents any synced field may hold (2^53 - 1). */
export const MAX_CENTS = Number.MAX_SAFE_INTEGER;

const AMOUNT_SHAPE = /^[^\d.,]*([\d.,]+)[^\d.,]*$/u;
const WHITESPACE = /\p{White_Space}+/u;

function ungroup(s: string): string | null {
	let sep = ".";
	if (s.includes(",")) {
		if (s.includes(".")) {
			return null;
		}
		sep = ",";
	}

	const parts = s.split(sep);
	const head = parts[0] ?? "";
	if (head.length < 1 || head.length > 3) {
		return null;
	}
	for (const p of parts.slice(1)) {
		if (p.length !== 3) {
			return null;
		}
	}

	return parts.join("");
}

function splitAmount(num: string): { whole: string; frac: string } | null {
	const dec = Math.max(num.lastIndexOf("."), num.lastIndexOf(","));
	if (dec < 0) {
		return { whole: num, frac: "" };
	}

	const sepChar = num.charAt(dec);
	// "1.234.567" has no decimal part: the separator repeats.
	if (num.split(sepChar).length - 1 > 1) {
		const whole = ungroup(num);
		return whole === null ? null : { whole, frac: "" };
	}

	const whole = num.slice(0, dec);
	const frac = num.slice(dec + 1);
	if (!/[.,]/u.test(whole)) {
		return { whole, frac };
	}

	const grouped = ungroup(whole);
	return grouped === null ? null : { whole: grouped, frac };
}

/**
 * Port of Go's finance.ParseAmount: reads "12.34", "12,34", "1,234.56",
 * "1.234,56", "-$5", "(5.00)" into integer cents, or null when the text is
 * not an amount. Both parsers are held to testdata/amount-vectors.json.
 */
export function parseAmount(input: string): number | null {
	const s = input.trim();
	const neg = s.startsWith("-") || (s.startsWith("(") && s.endsWith(")"));

	const compact = s.split(WHITESPACE).join("");
	const m = AMOUNT_SHAPE.exec(compact);
	const num = m?.[1];
	if (num === undefined) {
		return null;
	}

	const split = splitAmount(num);
	if (split === null || split.frac.length > 2 || (split.whole === "" && split.frac === "")) {
		return null;
	}

	const frac = split.frac + "0".repeat(2 - split.frac.length);
	const whole = split.whole === "" ? "0" : split.whole;

	// BigInt keeps the cap check exact for text beyond 2^53.
	const total = BigInt(whole) * 100n + BigInt(frac);
	if (total > BigInt(MAX_CENTS)) {
		return null;
	}

	const cents = Number(total);
	return neg && cents !== 0 ? -cents : cents;
}

/** Integer cents as a machine decimal: "-12.34", "0.05". No float is involved. */
export function formatDecimalCents(cents: number): string {
	assertCents(cents);
	const mag = Math.abs(cents);
	const frac = String(mag % 100).padStart(2, "0");
	const sign = cents < 0 ? "-" : "";
	return `${sign}${Math.floor(mag / 100)}.${frac}`;
}

function assertCents(cents: number): void {
	if (!Number.isSafeInteger(cents)) {
		throw new RangeError("money must be a safe integer of cents");
	}
}

/**
 * Locale display of integer cents. The whole units (an exact integer) go
 * through Intl.NumberFormat; the two decimals are placed with the locale's
 * own decimal separator, so no float arithmetic touches the amount.
 */
export function formatMoney(cents: number, currency: string, locale: string): string {
	assertCents(cents);
	const mag = Math.abs(cents);
	const units = Math.floor(mag / 100);
	const frac = String(mag % 100).padStart(2, "0");

	const nf = new Intl.NumberFormat(locale, {
		style: "currency",
		currency,
		minimumFractionDigits: 0,
		maximumFractionDigits: 0,
	});
	const probe = new Intl.NumberFormat(locale, {
		style: "currency",
		currency,
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	});
	const decimal = probe.formatToParts(1.5).find((p) => p.type === "decimal")?.value ?? ".";
	const parts = nf.formatToParts(units);
	let lastInteger = -1;
	parts.forEach((p, i) => {
		if (p.type === "integer" || p.type === "group") {
			lastInteger = i;
		}
	});

	const out = parts.map((p, i) => (i === lastInteger ? `${p.value}${decimal}${frac}` : p.value)).join("");
	return cents < 0 ? `-${out}` : out;
}
