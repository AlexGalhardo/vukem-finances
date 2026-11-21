import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

type Theme = "light" | "dark";

function readTokens(theme: Theme): Record<string, string> {
  const css = readFileSync(join(import.meta.dir, "../../css/tokens.css"), "utf8");
  const selector = theme === "light" ? ":root {" : ".dark {";
  const start = css.indexOf(selector);
  const end = css.indexOf("}", start);
  const tokens: Record<string, string> = {};
  for (const match of css.slice(start, end).matchAll(/--(vf-[a-z-]+):\s*(#[0-9a-f]{6});/gi)) {
    tokens[match[1]] = match[2];
  }
  return tokens;
}

function channel(value: number): number {
  const c = value / 255;
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
}

export function luminance(hex: string): number {
  const n = Number.parseInt(hex.slice(1), 16);
  return 0.2126 * channel(n >> 16) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255);
}

export function contrast(a: string, b: string): number {
  const x = luminance(a);
  const y = luminance(b);
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
}

const surfaces = ["vf-bg", "vf-surface", "vf-surface-muted"] as const;
const textOnSurface = ["vf-text", "vf-text-muted", "vf-brand", "vf-income", "vf-expense"] as const;
const nonText = ["vf-control-border", "vf-focus"] as const;

describe("contrast", () => {
  test("the helper matches known values", () => {
    expect(contrast("#000000", "#ffffff")).toBeCloseTo(21, 5);
    expect(contrast("#777777", "#ffffff")).toBeCloseTo(4.48, 1);
  });

  for (const theme of ["light", "dark"] as const) {
    const tokens = readTokens(theme);

    test(`${theme}: every token is present`, () => {
      for (const name of [...surfaces, ...textOnSurface, ...nonText, "vf-on-brand", "vf-border"]) {
        expect(tokens[name], name).toBeDefined();
      }
    });

    for (const surface of surfaces) {
      for (const name of textOnSurface) {
        test(`${theme}: ${name} text on ${surface} is at least 4.5:1`, () => {
          expect(contrast(tokens[name], tokens[surface])).toBeGreaterThanOrEqual(4.5);
        });
      }
      for (const name of nonText) {
        test(`${theme}: ${name} on ${surface} is at least 3:1`, () => {
          expect(contrast(tokens[name], tokens[surface])).toBeGreaterThanOrEqual(3);
        });
      }
    }

    test(`${theme}: text on the brand color is at least 4.5:1`, () => {
      expect(contrast(tokens["vf-on-brand"], tokens["vf-brand"])).toBeGreaterThanOrEqual(4.5);
    });
  }
});
