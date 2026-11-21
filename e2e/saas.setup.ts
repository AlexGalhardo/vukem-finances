import { test as setup } from "@playwright/test";
import { SAAS_STORAGE_STATE } from "./env";
import { login } from "./login";

// A sign-in on the saas server for the billing spec: the seeded administrator owns a workspace of its own there.
setup("sign in to the saas edition", async ({ page }) => {
  await login(page);
  await page.context().storageState({ path: SAAS_STORAGE_STATE });
});
