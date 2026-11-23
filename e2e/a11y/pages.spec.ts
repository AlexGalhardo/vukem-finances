import { expect, test } from "@playwright/test";
import { locales, restoreEnglish, themes, useLocale, useTheme, violations } from "./axe";

// Every page of the personal edition that a signed-in person reaches, audited with axe in two themes and three
// languages (epic 24 task 12). Pages of the saas edition only are in saas-pages.spec.ts; the sign-in pages in
// public-pages.spec.ts.
const pages = [
  "/dashboard",
  "/transactions",
  "/accounts",
  "/cards",
  "/categories",
  "/budgets",
  "/goals",
  "/recurrences",
  "/reports",
  "/import",
  "/suggestions",
  "/ai",
  "/tokens",
  "/devices",
  "/notifications",
  "/workspace",
  "/account/privacy",
  "/account/deletion",
  "/settings/telegram",
  "/user/two-factor",
];

test.afterEach(async ({ page }) => {
  await restoreEnglish(page);
});

for (const locale of locales) {
  for (const theme of themes) {
    test(`axe finds nothing on any page in ${theme} and ${locale.code}`, async ({ page }) => {
      test.setTimeout(300_000);
      await useTheme(page, theme);
      await useLocale(page, locale.code);
      const found: string[] = [];
      for (const path of pages) {
        await page.goto(path);
        await page.getByRole("heading", { level: 1 }).first().waitFor();
        await page.waitForLoadState("load");
        const isDark = await page.evaluate(() => document.documentElement.classList.contains("dark"));
        expect(isDark, `${path} theme`).toBe(theme === "dark");
        found.push(...(await violations(page, `${path} ${theme} ${locale.code}`)));
      }
      expect(found.join("\n")).toBe("");
    });
  }
}
