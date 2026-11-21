import { type FormEvent, type RefObject, useRef, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import AmountInput from "../AmountInput";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { Field, FieldError, FieldGroup, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";
import {
  emptyTransferDraft,
  type PickerAccount,
  type QuickAddOptions,
  sendJSON,
  type TransferDraft,
} from "./entryForm";
import PickerField from "./PickerField";

/** A transfer being edited: the id of one of its legs, its current values and the accounts it already uses. */
export interface TransferEdit {
  legId: number;
  draft: TransferDraft;
  /** Accounts the transfer uses even if archived, so the pickers can still show them. */
  accounts: PickerAccount[];
}

export interface TransferDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Gives focus back to whatever opened the dialog (the dialog opens after a fetch, so Radix has no trigger). */
  returnFocus: () => void;
  options: QuickAddOptions | null;
  edit: TransferEdit | null;
  onSaved: (updated: boolean) => void;
}

/** New transfer or edit transfer. Both legs are written by the server in one step. */
export default function TransferDialog({
  open,
  onOpenChange,
  returnFocus,
  options,
  edit,
  onSaved,
}: TransferDialogProps) {
  const { t } = useFinanceI18n();
  const amountRef = useRef<HTMLInputElement>(null);
  const accounts = options
    ? [...options.accounts, ...(edit?.accounts ?? []).filter((a) => !options.accounts.some((o) => o.id === a.id))]
    : [];
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
          <DialogTitle>{t(edit ? "editTransfer" : "newTransfer")}</DialogTitle>
          <DialogDescription>{t("transferDescription")}</DialogDescription>
        </DialogHeader>
        {options === null ? (
          <p role="alert" className="text-sm text-destructive">
            {t("quickAddLoadError")}
          </p>
        ) : accounts.length < 2 ? (
          <p role="alert" className="text-sm">
            {t("needTwoAccounts")}
          </p>
        ) : (
          <TransferForm
            options={options}
            accounts={accounts}
            edit={edit}
            amountRef={amountRef}
            onCancel={() => onOpenChange(false)}
            onSaved={onSaved}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function TransferForm({
  options,
  accounts,
  edit,
  amountRef,
  onCancel,
  onSaved,
}: {
  options: QuickAddOptions;
  accounts: PickerAccount[];
  edit: TransferEdit | null;
  amountRef: RefObject<HTMLInputElement | null>;
  onCancel: () => void;
  onSaved: (updated: boolean) => void;
}) {
  const { t } = useFinanceI18n();
  const [draft, setDraft] = useState<TransferDraft>(() => edit?.draft ?? emptyTransferDraft(options));
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const alertRef = useRef<HTMLParagraphElement>(null);
  const set = <K extends keyof TransferDraft>(key: K, value: TransferDraft[K]): void =>
    setDraft((current) => ({ ...current, [key]: value }));

  async function submit(event: FormEvent): Promise<void> {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setMessage("");
    setFields({});
    const result = edit
      ? await sendJSON(`/transfers/${edit.legId}`, "PATCH", draft, t("quickAddSaveError"))
      : await sendJSON("/transfers", "POST", draft, t("quickAddSaveError"));
    setBusy(false);
    if (result.ok) {
      onSaved(edit !== null);
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
          <FieldLabel htmlFor="transfer-amount">{t("amount")}</FieldLabel>
          <AmountInput
            id="transfer-amount"
            name="amount"
            ref={amountRef}
            required
            placeholder="100.00"
            value={draft.amount}
            aria-invalid={!!fields.amount}
            onChange={(event) => set("amount", event.target.value)}
          />
          <FieldError>{fields.amount}</FieldError>
        </Field>
        <PickerField
          label={t("fromAccount")}
          items={accounts}
          value={draft.fromAccountId}
          onChange={(id) => set("fromAccountId", id)}
          error={fields.fromAccountId}
        />
        <PickerField
          label={t("toAccount")}
          items={accounts}
          value={draft.toAccountId}
          onChange={(id) => set("toAccountId", id)}
          error={fields.toAccountId}
        />
        <Field data-invalid={!!fields.date}>
          <FieldLabel htmlFor="transfer-date">{t("date")}</FieldLabel>
          <Input
            id="transfer-date"
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
          <FieldLabel htmlFor="transfer-description">{t("description")}</FieldLabel>
          <Input
            id="transfer-description"
            name="description"
            type="text"
            autoComplete="off"
            value={draft.description}
            aria-invalid={!!fields.description}
            onChange={(event) => set("description", event.target.value)}
          />
          <FieldError>{fields.description}</FieldError>
        </Field>
      </FieldGroup>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={busy}>
          {t(busy ? "working" : edit ? "saveTransfer" : "addTransfer")}
        </Button>
      </DialogFooter>
    </form>
  );
}
