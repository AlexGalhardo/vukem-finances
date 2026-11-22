import "@fontsource-variable/geist/wght.css";
import "@fontsource-variable/geist-mono/wght.css";

import { createInertiaApp } from "@inertiajs/react";

import AppLayout from "./layouts/AppLayout";
import { watchTheme } from "./lib/theme";

// Dark mode follows the "dark" class on <html>. app.html sets it before paint
// from localStorage.theme ("dark" | "light") or the system preference; this
// keeps it live while no choice is stored.
watchTheme();

// The @inertiajs/vite plugin resolves pages from ./pages and mounts the app.
// The nonce lets the progress bar styles through the CSP.
// It mounts on #app, the default inertia.Config.RootID; when you change
// RootID, pass the same value here as `id`.
// react-style-singleton, used by the Radix scroll lock, reads the nonce of its injected <style> here.
(globalThis as { __webpack_nonce__?: string }).__webpack_nonce__ =
  document.querySelector<HTMLMetaElement>('meta[property="csp-nonce"]')?.nonce;

createInertiaApp({
  layout: () => AppLayout,
  nonce: document.querySelector<HTMLMetaElement>('meta[property="csp-nonce"]')?.nonce,
  progress: { color: "#4f46e5" },
}).finally(() => {
  // app.html sets data-hydrating; style it (button:disabled, a pending hint)
  // for the gap before the app mounts.
  document.documentElement.removeAttribute("data-hydrating");
});
