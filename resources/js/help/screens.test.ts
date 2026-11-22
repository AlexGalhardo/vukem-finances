import { describe, expect, test } from "bun:test";
import { screenIdForPath, screenRoutes } from "./screens";

describe("screenIdForPath", () => {
  test("maps static and parameterised routes, ignoring query and hash", () => {
    expect(screenIdForPath("/dashboard")).toBe("Dashboard");
    expect(screenIdForPath("/dashboard?month=2026-11")).toBe("Dashboard");
    expect(screenIdForPath("/cards/12")).toBe("CardStatement");
    expect(screenIdForPath("/cards")).toBe("Cards");
    expect(screenIdForPath("/invites/abc")).toBe("InviteAccept");
    expect(screenIdForPath("/invites/abc/register")).toBe("InviteRegister");
    expect(screenIdForPath("/transactions#search")).toBe("Transactions");
  });

  test("a static admin route wins over the resource routes", () => {
    expect(screenIdForPath("/admin")).toBe("AdminDashboard");
    expect(screenIdForPath("/admin/revenue")).toBe("AdminRevenue");
    expect(screenIdForPath("/admin/users")).toBe("AdminResourceList");
    expect(screenIdForPath("/admin/users/7")).toBe("AdminResourceShow");
  });

  test("an unknown path has no screen, and every screen id is unique", () => {
    expect(screenIdForPath("/nope/at/all")).toBeUndefined();
    expect(new Set(screenRoutes.map((screen) => screen.id)).size).toBe(screenRoutes.length);
    expect(screenRoutes).toHaveLength(36);
  });
});
