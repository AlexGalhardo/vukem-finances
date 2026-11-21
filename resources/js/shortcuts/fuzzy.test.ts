import { describe, expect, test } from "bun:test";
import { rank, scoreSubsequence } from "./fuzzy";

describe("scoreSubsequence", () => {
  test("requires every character, in order", () => {
    expect(scoreSubsequence("trn", "Transactions")).not.toBeNull();
    expect(scoreSubsequence("xtr", "Transactions")).toBeNull();
    expect(scoreSubsequence("zz", "Transactions")).toBeNull();
  });

  test("an empty query matches everything", () => {
    expect(scoreSubsequence("", "anything")).toBe(0);
  });

  test("ignores case and spaces in the query", () => {
    expect(scoreSubsequence("GO  DASH", "go dashboard")).not.toBeNull();
  });

  test("prefix and adjacent matches beat scattered ones", () => {
    const prefix = scoreSubsequence("acc", "Accounts") as number;
    const scattered = scoreSubsequence("acc", "A big cross cube") as number;
    expect(prefix).toBeGreaterThan(scattered);
  });
});

describe("rank", () => {
  const items = ["Budgets", "Categories", "Import statement", "Transactions"];

  test("drops non-matches and puts the best first", () => {
    expect(rank("tra", items, (i) => i)[0]).toBe("Transactions");
    expect(rank("zzz", items, (i) => i)).toEqual([]);
  });

  test("keeps the original order for an empty query and for ties", () => {
    expect(rank("", items, (i) => i)).toEqual(items);
    expect(rank("  ", items, (i) => i)).toEqual(items);
  });
});
