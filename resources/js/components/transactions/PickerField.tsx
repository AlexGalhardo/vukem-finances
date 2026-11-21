import { useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import Combobox from "../Combobox";
import { Field, FieldError, FieldLabel } from "../ui/field";
import { filterPicker } from "./entryForm";

interface PickerItem {
  id: number;
  name: string;
}

export interface PickerFieldProps {
  label: string;
  items: readonly PickerItem[];
  /** The chosen id as text, or "" for none. */
  value: string;
  onChange: (id: string) => void;
  error?: string;
  /** Offer an empty choice (the category picker). */
  nullable?: boolean;
}

/** A searchable picker built on the combobox: type to narrow the list, arrows and Enter to choose. */
export default function PickerField({ label, items, value, onChange, error, nullable = false }: PickerFieldProps) {
  const { t } = useFinanceI18n();
  const selected = items.find((item) => String(item.id) === value);
  const [query, setQuery] = useState(selected?.name ?? "");

  const shown = filterPicker(items, query, selected).map((item) => ({ id: String(item.id), label: item.name }));
  const choices = nullable ? [{ id: "", label: t("none") }, ...shown] : shown;

  return (
    <Field data-invalid={!!error}>
      <FieldLabel>{label}</FieldLabel>
      <Combobox
        items={choices}
        query={query}
        onQueryChange={setQuery}
        onSelect={(item) => {
          onChange(item.id);
          setQuery(item.id === "" ? "" : item.label);
        }}
        label={label}
        placeholder={nullable ? t("none") : undefined}
        resultsLabel={(count) => `${count} ${t(count === 1 ? "resultOne" : "resultMany")}`}
        inputClassName={error ? "border-destructive" : undefined}
        // Leaving the field shows the chosen name again instead of half-typed text.
        onInputBlur={() => setQuery(selected?.name ?? "")}
        listClassName="mt-1 rounded-lg border bg-popover shadow-sm"
      />
      <FieldError>{error}</FieldError>
    </Field>
  );
}
