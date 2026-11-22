import { Moon, Sun } from "lucide-react";
import { type ReactElement, useEffect, useState } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";

/** Shows both themes side by side, the current one pressed. */
export default function ThemeToggle(): ReactElement {
  const { t } = useFinanceI18n();
  const [dark, setDark] = useState(false);

  useEffect(() => {
    setDark(document.documentElement.classList.contains("dark"));
  }, []);

  function choose(next: boolean): void {
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
    try {
      localStorage.setItem("theme", next ? "dark" : "light");
    } catch {
      /* Storage may be disabled. */
    }
  }

  return (
    <fieldset aria-label={t("theme")} className="flex min-w-0 items-center gap-0.5 rounded-md border p-0.5">
      <Button
        type="button"
        variant={dark ? "ghost" : "secondary"}
        size="icon"
        className="size-7"
        aria-pressed={!dark}
        aria-label={t("light")}
        title={t("light")}
        onClick={() => choose(false)}
      >
        <Sun aria-hidden="true" />
      </Button>
      <Button
        type="button"
        variant={dark ? "secondary" : "ghost"}
        size="icon"
        className="size-7"
        aria-pressed={dark}
        aria-label={t("dark")}
        title={t("dark")}
        onClick={() => choose(true)}
      >
        <Moon aria-hidden="true" />
      </Button>
    </fieldset>
  );
}
