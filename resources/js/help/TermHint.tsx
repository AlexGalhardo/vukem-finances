import type { ReactElement, ReactNode } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "../components/ui/popover";
import { useFinanceI18n } from "../lib/i18n";
import { helpBodyKey, helpTitleKey, type TermId } from "./ids";

/** A financial word in running text: a dotted-underline button that opens its glossary definition. */
export function TermHint({ term, children }: { term: TermId; children: ReactNode }): ReactElement {
  const { t } = useFinanceI18n();
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="cursor-help rounded-sm underline decoration-dotted decoration-1 underline-offset-4 hover:decoration-solid pointer-coarse:min-h-11"
        >
          {children}
        </button>
      </PopoverTrigger>
      <PopoverContent onOpenAutoFocus={(event) => event.preventDefault()}>
        <p className="text-body text-pretty">
          <strong className="font-semibold">{t(helpTitleKey(term))}.</strong> {t(helpBodyKey(term))}
        </p>
      </PopoverContent>
    </Popover>
  );
}
