import { describe, expect, test } from "bun:test";
import { canExportJson, transactionsCsvUrl } from "./exports";

describe("transactionsCsvUrl", () => {
  test("carries only the filters that are set", () => {
    expect(transactionsCsvUrl({ q: "", from: "2026-01-01", to: "", kind: "expense" })).toBe(
      "/transactions/export.csv?from=2026-01-01&kind=expense",
    );
    expect(transactionsCsvUrl({})).toBe("/transactions/export.csv");
  });

  test("encodes what a person typed", () => {
    expect(transactionsCsvUrl({ q: "a&b=c d" })).toBe("/transactions/export.csv?q=a%26b%3Dc+d");
  });

  test("the Excel flavor asks for a BOM and semicolons", () => {
    expect(transactionsCsvUrl({ from: "2026-01-01" }, { excel: true })).toBe(
      "/transactions/export.csv?from=2026-01-01&bom=1&delimiter=semicolon",
    );
  });
});

describe("canExportJson", () => {
  test("is for owners and admins only", () => {
    expect(["owner", "admin", "member", "viewer", undefined].map(canExportJson)).toEqual([
      true,
      true,
      false,
      false,
      false,
    ]);
  });
});
