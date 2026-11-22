import { readdirSync, readFileSync } from "node:fs";

const root = `${__dirname}/..`;

describe("routes", () => {
  // expo-router uses src/app as its routes root when that folder exists, which would turn app state files into routes
  // and hide mobile/app. The shared code lives in src/core, src/screens, src/db and so on instead.
  it("has no src/app folder, so expo-router reads mobile/app", () => {
    expect(readdirSync(`${root}/src`)).not.toContain("app");
  });

  it("has a route for every tab and every screen that the More list opens", () => {
    expect(readdirSync(`${root}/app/(tabs)`).sort()).toEqual([
      "_layout.tsx",
      "accounts.tsx",
      "add.tsx",
      "index.tsx",
      "more.tsx",
      "transactions.tsx",
    ]);
    const more = readFileSync(`${root}/src/screens/MoreScreen.tsx`, "utf8");
    const targets = [...more.matchAll(/target: "(\w+)"/gu)].map((m) => m[1]);
    expect(targets.length).toBeGreaterThanOrEqual(7);
    const routes = readdirSync(`${root}/app`);
    for (const t of targets) expect(routes).toContain(`${t}.tsx`);
  });
});
