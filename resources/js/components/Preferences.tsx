import { router } from "@inertiajs/react";
import { useEffect, useState } from "react";
import { Moon, Sun, Zap } from "lucide-react";
import { useFinanceI18n } from "../lib/i18n";
import { setSingleKeyEnabled, useSingleKeyShortcuts } from "../shortcuts/singleKey";
import { Button } from "./ui/button";
import { NativeSelect, NativeSelectOption } from "./ui/native-select";

export default function Preferences({ withShortcuts = false }: { withShortcuts?: boolean }) {
  const { t, locale } = useFinanceI18n();
  const [dark, setDark] = useState(false);
  const [singleKeys] = useSingleKeyShortcuts();
  useEffect(() => {
    setDark(document.documentElement.classList.contains("dark"));
  }, []);
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  function toggle() {
    const next = !dark;
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
    try {
      localStorage.setItem("theme", next ? "dark" : "light");
    } catch {
      /* Storage may be disabled. */
    }
  }
  return (
    <div className="flex items-center gap-2">
      <NativeSelect
        aria-label={t("language")}
        value={locale}
        onChange={(e) => router.post("/locale", { language: e.target.value }, { preserveScroll: true })}
      >
        <NativeSelectOption value="en">English</NativeSelectOption>
        <NativeSelectOption value="pt-BR">Português (Brasil)</NativeSelectOption>
        <NativeSelectOption value="es">Español</NativeSelectOption>
      </NativeSelect>
      {withShortcuts && (
        <Button
          variant={singleKeys ? "secondary" : "ghost"}
          size="icon"
          aria-pressed={singleKeys}
          aria-label={t("singleKey")}
          title={t("singleKey")}
          onClick={() => setSingleKeyEnabled(!singleKeys)}
        >
          <Zap aria-hidden="true" />
        </Button>
      )}
      <Button
        variant="ghost"
        size="icon"
        onClick={toggle}
        aria-label={t(dark ? "light" : "dark")}
        title={t(dark ? "light" : "dark")}
      >
        {dark ? <Sun /> : <Moon />}
      </Button>
    </div>
  );
}
