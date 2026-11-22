/** The 35 screens of the product and the route each one answers (section 15.8 of docs/design-system.md). */
export interface ScreenRoute {
  id: string;
  /** A path with `{name}` segments for parameters, or `*` for the error page that has no route of its own. */
  route: string;
}

export const screenRoutes: readonly ScreenRoute[] = [
  { id: "Login", route: "/login" },
  { id: "Register", route: "/register" },
  { id: "CheckEmail", route: "/register/check-email" },
  { id: "VerifyEmail", route: "/email/verify" },
  { id: "ForgotPassword", route: "/forgot-password" },
  { id: "ResetPassword", route: "/reset-password/{token}" },
  { id: "TwoFactorChallenge", route: "/two-factor-challenge" },
  { id: "Setup", route: "/setup" },
  { id: "InviteAccept", route: "/invites/{token}" },
  { id: "InviteRegister", route: "/invites/{token}/register" },
  { id: "Error", route: "*" },
  { id: "Dashboard", route: "/dashboard" },
  { id: "Transactions", route: "/transactions" },
  { id: "Accounts", route: "/accounts" },
  { id: "Cards", route: "/cards" },
  { id: "CardStatement", route: "/cards/{card}" },
  { id: "Categories", route: "/categories" },
  { id: "Budgets", route: "/budgets" },
  { id: "Goals", route: "/goals" },
  { id: "Recurrences", route: "/recurrences" },
  { id: "Reports", route: "/reports" },
  { id: "Import", route: "/import" },
  { id: "Suggestions", route: "/suggestions" },
  { id: "AI", route: "/ai" },
  { id: "Workspace", route: "/workspace" },
  { id: "Notifications", route: "/notifications" },
  { id: "Telegram", route: "/settings/telegram" },
  { id: "Tokens", route: "/tokens" },
  { id: "Devices", route: "/devices" },
  { id: "Billing", route: "/billing" },
  { id: "TwoFactor", route: "/user/two-factor" },
  { id: "AdminDashboard", route: "/admin" },
  { id: "AdminResourceList", route: "/admin/{slug}" },
  { id: "AdminResourceShow", route: "/admin/{slug}/{id}" },
  { id: "AdminRevenue", route: "/admin/revenue" },
];

function matcher(route: string): RegExp {
  const pattern = route
    .split("/")
    .map((part) => (part.startsWith("{") ? "[^/]+" : part.replace(/[.*+?^$()|[\]\\]/g, "\\$&")))
    .join("/");
  return new RegExp(`^${pattern}/?$`);
}

// Static routes win over parameterised ones: /admin/revenue before /admin/{slug}.
const ordered = screenRoutes
  .filter((screen) => screen.route !== "*")
  .map((screen) => ({ id: screen.id, params: (screen.route.match(/\{/g) ?? []).length, test: matcher(screen.route) }))
  .sort((a, b) => a.params - b.params);

/** The screen id for a URL path (query and hash ignored), or undefined when no screen owns it. */
export function screenIdForPath(url: string): string | undefined {
  const path = url.split(/[?#]/)[0] ?? url;
  return ordered.find((screen) => screen.test.test(path))?.id;
}
