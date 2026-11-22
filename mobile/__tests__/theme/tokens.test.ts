import { readFileSync } from "node:fs";
import { buildTokens } from "../../../scripts/tokens-lib";
import { colors, duration, fontSize, radius, spacing, targetMin } from "../../src/theme/tokens";

const lf = (s: string): string => s.replace(/\r\n/gu, "\n");

describe("theme tokens", () => {
	it("is what scripts/export-tokens.ts makes from resources/css/tokens.css", () => {
		const css = readFileSync(`${__dirname}/../../../resources/css/tokens.css`, "utf8");
		const have = readFileSync(`${__dirname}/../../src/theme/tokens.ts`, "utf8");
		expect(lf(have)).toBe(lf(buildTokens(css)));
	});

	it("carries the palette of both themes with the same names", () => {
		expect(Object.keys(colors.dark)).toEqual(Object.keys(colors.light));
		expect(colors.light.brand).toBe("#1c7448");
		expect(colors.dark.text).toBe("#edf3ef");
	});

	it("converts rem to density-independent pixels", () => {
		expect(spacing[4]).toBe(16);
		expect(fontSize.base).toBe(16);
		expect(radius).toBe(10);
		expect(duration.base).toBe(200);
	});

	it("keeps touch targets at least 44 dp (the spec asks 48 on the controls that use it)", () => {
		expect(targetMin).toBeGreaterThanOrEqual(44);
	});
});
