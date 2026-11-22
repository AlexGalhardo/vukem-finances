import { router } from "@inertiajs/react";
import { Zap } from "lucide-react";
import { useEffect } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { setSingleKeyEnabled, useSingleKeyShortcuts } from "../shortcuts/singleKey";
import { Button } from "./ui/button";
import { NativeSelect, NativeSelectOption } from "./ui/native-select";

export default function Preferences({ withShortcuts = false }: { withShortcuts?: boolean }) {
  const { t, locale } = useFinanceI18n();
  const [singleKeys] = useSingleKeyShortcuts();
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
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
    </div>
  );
}
