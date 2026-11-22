import vectors from "../../../testdata/amount-vectors.json";
import { formatDecimalCents, formatMoney, MAX_CENTS, parseAmount } from "../../src/domain/money";

describe("parseAmount", () => {
	it("holds at least 60 shared vectors", () => {
		expect(vectors.length).toBeGreaterThanOrEqual(60);
	});

	it.each(vectors.map((v) => [JSON.stringify(v.in), v] as const))("vector %s", (_name, v) => {
		expect(parseAmount(v.in)).toBe(v.cents);
	});

	it("reads the headline cases", () => {
		expect(parseAmount("1.234,56")).toBe(123456);
		expect(parseAmount("1,234,567")).toBe(123456700);
		expect(parseAmount("(5.00)")).toBe(-500);
		expect(parseAmount("-$5.25")).toBe(-525);
		expect(parseAmount("1.005")).toBeNull();
		expect(parseAmount("")).toBeNull();
	});

	it("never returns negative zero", () => {
		expect(Object.is(parseAmount("-0"), 0)).toBe(true);
	});
});

describe("formatDecimalCents", () => {
	it("round-trips through parseAmount", () => {
		for (const c of [0, 5, -5, 1234, -123456, MAX_CENTS, -MAX_CENTS]) {
			expect(parseAmount(formatDecimalCents(c))).toBe(c);
		}
	});

	it("pads and signs without a float", () => {
		expect(formatDecimalCents(5)).toBe("0.05");
		expect(formatDecimalCents(-1234)).toBe("-12.34");
	});

	it("refuses a non-integer", () => {
		expect(() => formatDecimalCents(1.5)).toThrow(RangeError);
	});
});

describe("formatMoney", () => {
	it("groups units and places the decimals by locale", () => {
		expect(formatMoney(123456, "USD", "en-US")).toBe("$1,234.56");
		expect(formatMoney(5, "USD", "en-US")).toBe("$0.05");
		expect(formatMoney(-1250, "USD", "en-US")).toBe("-$12.50");
	});

	it("uses the comma of pt-BR", () => {
		expect(formatMoney(123456, "BRL", "pt-BR").replace(/\s/gu, " ")).toBe("R$ 1.234,56");
	});

	it("is exact at the cap", () => {
		expect(formatMoney(MAX_CENTS, "USD", "en-US")).toBe("$90,071,992,547,409.91");
	});
});
