import { expect, test } from "@playwright/test";
import { locales, restoreEnglish, themes, useLocale, useTheme, violations } from "./axe";

// The pages that exist in the saas edition only, plus the sign-up page, audited like the personal ones.
// /connected-apps answers 404 unless the remote MCP server is configured, so it is not audited here.
const pages = ["/billing", "/admin", "/admin/revenue", "/workspace", "/dashboard"];
const publicPages = ["/register", "/login"];

test.afterEach(async ({ page }) => {
  await restoreEnglish(page);
});

for (const locale of locales) {
  for (const theme of themes) {
    test(`axe finds nothing on the saas pages in ${theme} and ${locale.code}`, async ({ page, browser }) => {
      test.setTimeout(300_000);
      await useTheme(page, theme);
      await useLocale(page, locale.code);
      const found: string[] = [];
      for (const path of pages) {
        await page.goto(path);
        await page.getByRole("heading", { level: 1 }).first().waitFor();
        await page.waitForLoadState("load");
        found.push(...(await violations(page, `${path} ${theme} ${locale.code}`)));
      }

      const anonymous = await browser.newContext({
        baseURL: test.info().project.use.baseURL,
        storageState: { cookies: [], origins: [] },
      });
      const open = await anonymous.newPage();
      await useTheme(open, theme);
      await useLocale(open, locale.code);
      for (const path of publicPages) {
        await open.goto(path);
        await open.getByRole("heading", { level: 1 }).first().waitFor();
        await open.waitForLoadState("load");
        expect(await open.evaluate(() => document.documentElement.classList.contains("dark"))).toBe(theme === "dark");
        found.push(...(await violations(open, `${path} ${theme} ${locale.code}`)));
      }
      await anonymous.close();
      expect(found.join("\n")).toBe("");
    });
  }
}
