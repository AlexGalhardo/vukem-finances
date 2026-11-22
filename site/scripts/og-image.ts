// Generates public/og.png (1200 by 630), the picture shared by Open Graph and Twitter cards. The picture is
// drawn here from the tokens of docs/design.md and carries no screenshot and no asset of another product.
//
//   cd site && bun run og
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";

const out = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "public", "og.png");

const html = `<!doctype html><html><body style="margin:0;width:1200px;height:630px;background:#0e1310;color:#edf3ef;
font-family:system-ui,'Segoe UI',Roboto,Arial,sans-serif;display:flex;flex-direction:column;justify-content:center;
padding:0 96px;box-sizing:border-box;border-left:16px solid #62d596">
<div style="font-size:34px;color:#62d596;font-weight:700;letter-spacing:.02em">vukem-finances</div>
<div style="font-size:78px;font-weight:800;line-height:1.08;margin-top:24px;letter-spacing:-.02em">Your money,<br>in an app you own</div>
<div style="font-size:34px;color:#a6b6ac;margin-top:32px">Open source. Self-hosted. Your data stays yours.</div>
</body></html>`;

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1200, height: 630 } });
  await page.setContent(html);
  fs.writeFileSync(out, await page.screenshot({ type: "png" }));
} finally {
  await browser.close();
}
console.log(`og.png: ${fs.statSync(out).size} bytes`);
