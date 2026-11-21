import { test as setup } from "@playwright/test";
import { STORAGE_STATE } from "./env";
import { login } from "./login";

// One sign-in for the whole run: the app throttles repeated logins.
setup("sign in as the seeded administrator", async ({ page }) => {
  await login(page);
  await page.context().storageState({ path: STORAGE_STATE });
});
