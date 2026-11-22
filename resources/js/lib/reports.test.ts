import { describe, expect, test } from "bun:test";
import { percentText, periodQuery } from "./reports";

describe("percentText", () => {
  test("shows basis points as a percentage with two decimals", () => {
    expect(percentText(1250)).toBe("12.50%");
    expect(percentText(-6667, "en", true)).toBe("-66.67%");
    expect(percentText(100, "en", true)).toBe("+1.00%");
    expect(percentText(0, "en", true)).toBe("0.00%");
  });
});

describe("periodQuery", () => {
  test("a preset travels alone, custom with its dates", () => {
    expect(periodQuery("last_month", "2026-01-01", "2026-01-31")).toEqual({ period: "last_month" });
    expect(periodQuery("custom", "2026-01-01", "2026-01-31")).toEqual({
      period: "custom",
      from: "2026-01-01",
      to: "2026-01-31",
    });
  });
});
