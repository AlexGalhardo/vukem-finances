export type ThemeChoice = "light" | "dark";

const KEY = "theme";
const QUERY = "(prefers-color-scheme: dark)";
const listeners = new Set<() => void>();

function stored(): ThemeChoice | null {
  try {
    const value = localStorage.getItem(KEY);
    return value === "light" || value === "dark" ? value : null;
  } catch {
    return null;
  }
}

function prefersDark(): boolean {
  return typeof matchMedia === "function" && matchMedia(QUERY).matches;
}

function paint(): void {
  const choice = stored();
  document.documentElement.classList.toggle("dark", choice ? choice === "dark" : prefersDark());
  for (const listener of listeners) listener();
}

/** Saves an explicit choice and repaints. Without a stored choice the theme follows the system. */
export function setTheme(choice: ThemeChoice): void {
  try {
    localStorage.setItem(KEY, choice);
  } catch {
    /* Storage may be disabled: the choice still applies to this page. */
  }
  document.documentElement.classList.toggle("dark", choice === "dark");
  for (const listener of listeners) listener();
}

/** Removes the stored choice so the theme follows the operating system again. */
export function followSystem(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* Nothing to remove. */
  }
  paint();
}

export function isDark(): boolean {
  return document.documentElement.classList.contains("dark");
}

export function subscribeTheme(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** Keeps the page in step with the system while no choice is stored, and with other tabs. */
export function watchTheme(): void {
  if (typeof matchMedia === "function") {
    matchMedia(QUERY).addEventListener("change", () => {
      if (stored() === null) paint();
    });
  }
  addEventListener("storage", (event) => {
    if (event.key === KEY || event.key === null) paint();
  });
}
