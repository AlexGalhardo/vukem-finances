import { type FormEvent, useRef, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import AmountInput from "../AmountInput";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { Field, FieldError, FieldGroup, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";
import {
  categoriesFor,
  type EntryKind,
  emptyTransactionDraft,
  type QuickAddOptions,
  sendJSON,
  type TransactionDraft,
} from "./entryForm";
import PickerField from "./PickerField";
import TagsField from "./TagsField";

export interface TransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Gives focus back to whatever opened the dialog (the dialog opens after a fetch, so Radix has no trigger). */
  returnFocus: () => void;
  /** Null when the pickers could not be loaded: the dialog says so instead of showing a form that cannot work. */
  options: QuickAddOptions | null;
  kind: EntryKind;
  onSaved: () => void;
}

/** New expense or new income. The amount is the first field and holds focus when the dialog opens. */
export default function TransactionDialog({
  open,
  onOpenChange,
  returnFocus,
  options,
  kind,
  onSaved,
}: TransactionDialogProps) {
  const { t } = useFinanceI18n();
  const amountRef = useRef<HTMLInputElement>(null);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          amountRef.current?.focus();
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus();
        }}
      >
        <DialogHeader>
          <DialogTitle>{t(kind === "expense" ? "newExpense" : "newIncome")}</DialogTitle>
          <DialogDescription>{t("quickAddDescription")}</DialogDescription>
        </DialogHeader>
        {options === null ? (
          <p role="alert" className="text-sm text-destructive">
            {t("quickAddLoadError")}
          </p>
        ) : options.accounts.length === 0 ? (
          <p role="alert" className="text-sm">
            {t("noAccountsToPick")}
          </p>
        ) : (
          <TransactionForm
            options={options}
            kind={kind}
            amountRef={amountRef}
            onCancel={() => onOpenChange(false)}
            onSaved={onSaved}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function TransactionForm({
  options,
  kind,
  amountRef,
  onCancel,
  onSaved,
}: {
  options: QuickAddOptions;
  kind: EntryKind;
  amountRef: React.RefObject<HTMLInputElement | null>;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const { t } = useFinanceI18n();
  const [draft, setDraft] = useState<TransactionDraft>(() => emptyTransactionDraft(kind, options));
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const alertRef = useRef<HTMLParagraphElement>(null);
  const set = <K extends keyof TransactionDraft>(key: K, value: TransactionDraft[K]): void =>
    setDraft((current) => ({ ...current, [key]: value }));

  async function submit(event: FormEvent): Promise<void> {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setMessage("");
    setFields({});
    const result = await sendJSON("/transactions", "POST", draft, t("quickAddSaveError"));
    setBusy(false);
    if (result.ok) {
      onSaved();
      return;
    }
    setMessage(result.message);
    setFields(result.fields);
    requestAnimationFrame(() => alertRef.current?.focus());
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-6" noValidate>
      {message !== "" && (
        <p ref={alertRef} tabIndex={-1} role="alert" className="text-sm text-destructive outline-none">
          {message}
        </p>
      )}
      <FieldGroup>
        <Field data-invalid={!!fields.amount}>
          <FieldLabel htmlFor="quick-amount">{t("amount")}</FieldLabel>
          <AmountInput
            id="quick-amount"
            name="amount"
            ref={amountRef}
            required
            placeholder="12.50"
            value={draft.amount}
            aria-invalid={!!fields.amount}
            onChange={(event) => set("amount", event.target.value)}
          />
          <FieldError>{fields.amount}</FieldError>
        </Field>
        <PickerField
          label={t("account")}
          items={options.accounts}
          value={draft.accountId}
          onChange={(id) => set("accountId", id)}
          error={fields.accountId}
        />
        <PickerField
          label={t("category")}
          items={categoriesFor(options, draft.kind)}
          value={draft.categoryId}
          onChange={(id) => set("categoryId", id)}
          error={fields.categoryId}
          nullable
        />
        <Field data-invalid={!!fields.date}>
          <FieldLabel htmlFor="quick-date">{t("date")}</FieldLabel>
          <Input
            id="quick-date"
            name="date"
            type="date"
            required
            value={draft.date}
            aria-invalid={!!fields.date}
            onChange={(event) => set("date", event.target.value)}
          />
          <FieldError>{fields.date}</FieldError>
        </Field>
        <Field data-invalid={!!fields.description}>
          <FieldLabel htmlFor="quick-description">{t("description")}</FieldLabel>
          <Input
            id="quick-description"
            name="description"
            type="text"
            autoComplete="off"
            value={draft.description}
            aria-invalid={!!fields.description}
            onChange={(event) => set("description", event.target.value)}
          />
          <FieldError>{fields.description}</FieldError>
        </Field>
        <TagsField
          tags={draft.tags}
          onChange={(tags) => set("tags", tags)}
          suggestions={options.tags}
          error={fields.tags}
        />
      </FieldGroup>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={busy}>
          {t(busy ? "working" : "add")}
        </Button>
      </DialogFooter>
    </form>
  );
}
