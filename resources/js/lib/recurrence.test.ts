import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { addMonths, installmentDates, occurrences, previewCents, type Rule, splitInstallments } from "./recurrence";

interface Vector {
  name: string;
  rule: Rule;
  from: string;
  to: string;
  expect: string[];
}

const vectors: Vector[] = JSON.parse(
  readFileSync(join(import.meta.dir, "../../../testdata/recurrence-vectors.json"), "utf8"),
);

describe("recurrence vectors", () => {
  test("there are at least 60", () => {
    expect(vectors.length).toBeGreaterThanOrEqual(60);
  });

  for (const v of vectors) {
    test(v.name, () => {
      expect(occurrences(v.rule, v.from, v.to)).toEqual(v.expect);
    });
  }
});

describe("month arithmetic", () => {
  test("clamps to the end of the month", () => {
    expect(addMonths("2027-01-31", 1)).toBe("2027-02-28");
    expect(addMonths("2028-01-31", 1)).toBe("2028-02-29");
    expect(addMonths("2028-02-29", 12)).toBe("2029-02-28");
    expect(addMonths("2027-03-31", -1)).toBe("2027-02-28");
  });
});

describe("installments", () => {
  test("100.00 in 3 is 33.34, 33.33, 33.33", () => {
    expect(splitInstallments(10000, 3)).toEqual([3334, 3333, 3333]);
  });

  test("the parts always add up and are at least one cent", () => {
    for (let parts = 2; parts <= 12; parts++) {
      for (let total = parts; total <= 500; total++) {
        const split = splitInstallments(total, parts);
        expect(split.reduce((a, b) => a + b, 0)).toBe(total);
        expect(Math.min(...split)).toBeGreaterThanOrEqual(1);
      }
    }
  });

  test("a purchase on the 31st keeps month ends", () => {
    expect(installmentDates("2027-01-31", 4)).toEqual(["2027-01-31", "2027-02-28", "2027-03-31", "2027-04-30"]);
  });

  test("the preview reads only plain amounts", () => {
    expect(previewCents("12.50")).toBe(1250);
    expect(previewCents("12,5")).toBe(1250);
    expect(previewCents("7")).toBe(700);
    expect(previewCents("1.234")).toBeNull();
    expect(previewCents("-1")).toBeNull();
    expect(previewCents("abc")).toBeNull();
  });
});
