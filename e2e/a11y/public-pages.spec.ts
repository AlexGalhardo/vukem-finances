import { expect, test } from "@playwright/test";
import { locales, themes, useLocale, useTheme, violations } from "./axe";

// The pages a person sees before signing in, audited with axe in two themes and three languages.
test.use({ storageState: { cookies: [], origins: [] } });

const pages = ["/login", "/forgot-password"];

for (const locale of locales) {
  for (const theme of themes) {
    test(`axe finds nothing on the sign-in pages in ${theme} and ${locale.code}`, async ({ page }) => {
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
