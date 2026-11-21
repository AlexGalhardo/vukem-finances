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

describe("the year bound", () => {
  const rule = (over: Partial<Rule>): Rule => ({
    frequency: "monthly",
    interval: 1,
    start_date: "9999-11-30",
    end_date: null,
    max_count: null,
    ...over,
  });

  test("a rule at the bound lists its last dates and stops", () => {
    expect(occurrences(rule({}), "9999-01-01", "9999-12-31")).toEqual(["9999-11-30", "9999-12-30"]);
    expect(occurrences(rule({ frequency: "yearly", start_date: "9999-02-28" }), "9999-01-01", "9999-12-31")).toEqual([
      "9999-02-28",
    ]);
    expect(occurrences(rule({ frequency: "daily", start_date: "9999-12-30" }), "9999-12-01", "9999-12-31")).toEqual([
      "9999-12-30",
      "9999-12-31",
    ]);
  });

  test("a date past the bound is never produced, even for the widest window", () => {
    const end = occurrences(rule({ frequency: "daily", start_date: "9999-12-31" }), "0001-01-01", "9999-12-31");
    expect(end).toEqual(["9999-12-31"]);
    expect(occurrences(rule({ frequency: "weekly", start_date: "9999-12-25" }), "9999-12-25", "9999-12-31")).toEqual([
      "9999-12-25",
    ]);
  });

  test("a five-digit year is not a date", () => {
    expect(occurrences(rule({ start_date: "10000-01-01" }), "2027-01-01", "9999-12-31")).toEqual([]);
    expect(occurrences(rule({}), "2027-01-01", "10000-01-01")).toEqual([]);
    expect(() => addMonths("9999-12-31", 1)).toThrow();
    expect(() => installmentDates("9999-12-31", 3)).toThrow();
  });

  test("one call walks a bounded number of occurrences", () => {
    const daily = rule({ frequency: "daily", start_date: "0001-01-01" });
    expect(occurrences(daily, "0001-01-01", "9999-12-31").length).toBe(10000);
    expect(occurrences(daily, "9990-01-01", "9990-01-03")).toEqual(["9990-01-01", "9990-01-02", "9990-01-03"]);
  });

  test("a year below 100 is a year, not 19xx", () => {
    expect(occurrences(rule({ frequency: "daily", start_date: "0050-02-28" }), "0050-02-27", "0050-03-02")).toEqual([
      "0050-02-28",
      "0050-03-01",
      "0050-03-02",
    ]);
  });
});
