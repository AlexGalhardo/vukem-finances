import { describe, expect, test } from "bun:test";
import { bellLabel, fill, notificationTitle } from "./notifications";

const t = (key: string): string =>
  ({
    notificationsBell: "Notifications, {n} unread",
    notificationsBellNone: "Notifications, none unread",
    "alerts.budget_80": "{name} is at {percent}% of its budget ({spent} of {limit})",
  })[key] ?? key;

describe("notifications", () => {
  test("the bell's accessible name carries the unread count", () => {
    expect(bellLabel(3, t)).toBe("Notifications, 3 unread");
    expect(bellLabel(0, t)).toBe("Notifications, none unread");
  });

  test("fills placeholders and leaves unknown ones visible", () => {
    expect(fill("{a} and {b}", { a: 1 })).toBe("1 and {b}");
  });

  test("renders a title from its key and params in the person's formats", () => {
    const title = notificationTitle(
      "alerts.budget_80",
      { name: "Groceries", percent: 85, spent_cents: 8500, limit_cents: 10000 },
      t,
      { money: (c, cur) => `${cur} ${(c / 100).toFixed(2)}`, date: (d) => d, currency: "USD" },
    );
    expect(title).toBe("Groceries is at 85% of its budget (USD 85.00 of USD 100.00)");
  });
});
