import { describe, expect, test } from "bun:test";
import { fill, isFinal, safeQr } from "./pix";

describe("pix helpers", () => {
  test("fill replaces known placeholders and keeps unknown ones", () => {
    expect(fill("{months} month(s), {total}", { months: 3, total: "R$ 59,70" })).toBe("3 month(s), R$ 59,70");
    expect(fill("until {date}", {})).toBe("until {date}");
  });

  test("only a pending charge keeps the dialog polling", () => {
    expect(isFinal("pending")).toBe(false);
    for (const status of ["paid", "expired", "refunded", "disputed"] as const) expect(isFinal(status)).toBe(true);
  });

  test("safeQr passes a PNG data URL and nothing else", () => {
    expect(safeQr("data:image/png;base64,AAAA")).toBe("data:image/png;base64,AAAA");
    expect(safeQr("javascript:alert(1)")).toBeUndefined();
    expect(safeQr("https://example.test/q.png")).toBeUndefined();
    expect(safeQr(undefined)).toBeUndefined();
  });
});
