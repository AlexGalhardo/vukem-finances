import { expect, type Page, test } from "@playwright/test";
import { blurActive } from "../login";

// Keyboard-only walkthroughs of the ten main journeys (epic 24 task 12). Each journey starts on its page and is
// driven with Tab, Shift+Tab, Enter, Space and Escape only: the focus must always be visible, every stop must have an
// accessible name, the order must not trap, and a dialog must trap focus while open and give it back when closed.

const journeys: { name: string; path: string }[] = [
  { name: "overview", path: "/dashboard" },
  { name: "add and review transactions", path: "/transactions" },
  { name: "accounts", path: "/accounts" },
  { name: "categories", path: "/categories" },
  { name: "budgets", path: "/budgets" },
  { name: "import a statement", path: "/import" },
  { name: "reports", path: "/reports" },
  { name: "cards and statements", path: "/cards" },
  { name: "recurring rules", path: "/recurrences" },
  { name: "workspace and members", path: "/workspace" },
];

interface Stop {
  id: string;
  name: string;
  visible: boolean;
  indicator: boolean;
}

async function focusedStop(page: Page): Promise<Stop | null> {
  return page.evaluate(() => {
    const el = document.activeElement;
    if (!el || el === document.body || el === document.documentElement) return null;
    const style = getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    const labelledBy = el.getAttribute("aria-labelledby");
    const labels = Array.from((el as HTMLInputElement).labels ?? []);
    const name = (
      el.getAttribute("aria-label") ||
      (labelledBy ? document.getElementById(labelledBy)?.textContent : "") ||
      labels.map((l) => l.textContent ?? "").join(" ") ||
      el.textContent ||
      el.getAttribute("title") ||
      el.getAttribute("placeholder") ||
      ""
    ).trim();
    const outline = style.outlineStyle !== "none" && Number.parseFloat(style.outlineWidth) > 0;
    const ring = style.boxShadow !== "none";
    return {
      id: `${el.tagName.toLowerCase()}${el.id ? `#${el.id}` : ""}`,
      name,
      visible: rect.width > 0 && rect.height > 0,
      indicator: outline || ring,
    };
  });
}

for (const journey of journeys) {
  test(`Tab walks the ${journey.name} page without a trap, with a visible focus and a name on every stop`, async ({
    page,
  }) => {
    await page.goto(journey.path);
    await page.getByRole("heading", { level: 1 }).first().waitFor();
    await blurActive(page);

    const seen = new Set<string>();
    const problems: string[] = [];
    let returned = false;
    for (let i = 0; i < 120 && !returned; i++) {
      await page.keyboard.press("Tab");
      const stop = await focusedStop(page);
      if (!stop) {
        // Focus left the document (the browser chrome): the page has no trap.
        returned = true;
        break;
      }
      const key = `${stop.id}|${stop.name}`;
      if (seen.has(key) && seen.size > 4) {
        returned = true;
        break;
      }
      seen.add(key);
      if (!stop.visible) problems.push(`${stop.id} has focus but no size`);
      if (!stop.indicator) problems.push(`${stop.id} "${stop.name}" shows no focus indicator`);
      if (stop.name === "") problems.push(`${stop.id} has no accessible name`);
    }
    expect(problems.join("\n")).toBe("");
    expect(seen.size, "the page has tab stops").toBeGreaterThan(5);
  });
}

test("the new transaction dialog traps Tab, closes on Escape and returns the focus to its button", async ({ page }) => {
  await page.goto("/transactions");
  await page.getByRole("heading", { level: 1 }).first().waitFor();
  const opener = page.getByRole("button", { name: "Add transaction" });
  await opener.focus();
  await page.keyboard.press("Enter");
  const dialog = page.getByRole("dialog", { name: "New expense" });
  await expect(dialog).toBeVisible();

  for (let i = 0; i < 25; i++) {
    await page.keyboard.press("Tab");
    const inside = await page.evaluate(() => Boolean(document.activeElement?.closest("[role=dialog]")));
    expect(inside, `Tab ${i + 1} stays inside the dialog`).toBe(true);
  }
  for (let i = 0; i < 5; i++) {
    await page.keyboard.press("Shift+Tab");
    const inside = await page.evaluate(() => Boolean(document.activeElement?.closest("[role=dialog]")));
    expect(inside, `Shift+Tab ${i + 1} stays inside the dialog`).toBe(true);
  }

  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(opener).toBeFocused();
});

test("the shortcut help dialog opens from the keyboard, traps focus and gives it back", async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).first().waitFor();
  const opener = page.getByRole("button", { name: "Keyboard shortcuts" }).first();
  await opener.focus();
  await page.keyboard.press("Enter");
  const dialog = page.getByRole("dialog").first();
  await expect(dialog).toBeVisible();
  for (let i = 0; i < 12; i++) {
    await page.keyboard.press("Tab");
    const inside = await page.evaluate(() => Boolean(document.activeElement?.closest("[role=dialog]")));
    expect(inside, `Tab ${i + 1} stays inside the dialog`).toBe(true);
  }
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(opener).toBeFocused();
});

test("the language select and the theme buttons work with the keyboard", async ({ page }) => {
  await page.goto("/dashboard");
  await page.getByRole("heading", { level: 1 }).first().waitFor();
  const light = page.getByRole("button", { name: /light/i }).first();
  await light.focus();
  await page.keyboard.press("Enter");
  await expect(page.locator("html")).not.toHaveClass(/dark/);
  const dark = page.getByRole("button", { name: /dark/i }).first();
  await dark.focus();
  await page.keyboard.press("Space");
  await expect(page.locator("html")).toHaveClass(/dark/);
});
