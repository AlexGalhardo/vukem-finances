import AxeBuilder from "@axe-core/playwright";
import { expect, type Page } from "@playwright/test";

export const themes = ["light", "dark"] as const;
export const locales = [
  { code: "en", label: "English" },
  { code: "pt-BR", label: "Português (Brasil)" },
  { code: "es", label: "Español" },
] as const;

export type Locale = (typeof locales)[number]["code"];
export type Theme = (typeof themes)[number];

/** Stores the theme before the page script runs, the way the theme buttons do. */
export async function useTheme(page: Page, theme: Theme): Promise<void> {
  await page.addInitScript((value) => localStorage.setItem("theme", value), theme);
}

/** Sets the language of the session through the select every page header carries. */
export async function useLocale(page: Page, locale: Locale): Promise<void> {
  await page.goto("/login");
  await page.waitForLoadState("load");
  await page.getByRole("combobox").first().selectOption(locale);
  await expect(page.locator("html")).toHaveAttribute("lang", locale);
}

/**
 * Puts the language of a signed-in session back to English. The language lives in the session row, which every spec
 * of the run shares, so an audit that leaves Spanish behind breaks the specs that read English labels.
 */
export async function restoreEnglish(page: Page): Promise<void> {
  await page.goto("/dashboard");
  await page.getByRole("combobox").first().selectOption("en");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
}

/** Runs axe with the WCAG 2.2 AA rule tags and best practices; returns one line per violating rule and node. */
export async function violations(page: Page, label: string): Promise<string[]> {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"])
    .analyze();
  return results.violations.flatMap((v) =>
    v.nodes.map((n) => `${label}: ${v.id} (${v.impact}) ${n.target.join(" ")} :: ${n.html.slice(0, 110)}`),
  );
}

export async function expectNoViolations(page: Page, label: string): Promise<void> {
  expect((await violations(page, label)).join("\n"), label).toBe("");
}
