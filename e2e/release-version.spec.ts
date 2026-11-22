import { expect, test } from "@playwright/test";

for (const path of ["/dashboard", "/transactions", "/admin"]) {
  test(`${path} shows the release in the bottom right corner`, async ({ page }) => {
    await page.goto(path);
    const badge = page.getByTestId("release-version");
    await expect(badge).toHaveText(/^v\d+\.\d+\.\d+/);

    const box = await badge.boundingBox();
    const viewport = page.viewportSize();
    if (!box || !viewport) throw new Error("no layout to measure");
    expect(box.x + box.width).toBeGreaterThan(viewport.width - 40);
    expect(box.y + box.height).toBeGreaterThan(viewport.height - 40);
    await expect(badge).toHaveCSS("pointer-events", "none");
  });
}
