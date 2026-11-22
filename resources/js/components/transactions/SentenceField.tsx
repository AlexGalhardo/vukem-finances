import { useEffect, useId, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import { Button } from "../ui/button";
import { Field, FieldDescription, FieldError, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";
import { readProblem } from "./entryForm";

/** What the server read from the sentence. Nothing is saved: the person confirms in the form. */
export interface SentenceDraft {
  kind: "income" | "expense";
  amount: string;
  date: string;
  description: string;
  categoryId: number;
  accountId: number;
}

/**
 * "Describe it in a sentence": offered only when AI categorization is on for the workspace and the person may record.
 * The server returns a draft (amount read by its own parser, date from a short list); this fills the form and the
 * person still presses Save.
 */
export default function SentenceField({ onDraft }: { onDraft: (draft: SentenceDraft) => void }) {
  const { t } = useFinanceI18n();
  const id = useId();
  const [enabled, setEnabled] = useState(false);
  const [limit, setLimit] = useState(300);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    fetch("/transactions/draft/status", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
      signal: controller.signal,
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((body: { enabled?: boolean; textLimit?: number } | null) => {
        setEnabled(body?.enabled === true);
        if (typeof body?.textLimit === "number") setLimit(body.textLimit);
      })
      .catch(() => undefined);
    return () => controller.abort();
  }, []);

  if (!enabled) return null;

  const fill = async (): Promise<void> => {
    if (busy || text.trim() === "") return;
    setBusy(true);
    setError("");
    try {
      const response = await fetch("/transactions/draft", {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
      });
      if (response.ok) {
        onDraft((await response.json()) as SentenceDraft);
        setText("");
      } else {
        const body: unknown = await response.json().catch(() => null);
        setError(readProblem(body, t("draftError"), t("aiOffline")).message);
      }
    } catch {
      setError(t("draftError"));
    }
    setBusy(false);
  };

  return (
    <Field data-invalid={error !== ""}>
      <FieldLabel htmlFor={id}>{t("draftSentence")}</FieldLabel>
      <div className="flex gap-2">
        <Input
          id={id}
          type="text"
          autoComplete="off"
          maxLength={limit}
          placeholder={t("draftHint")}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            // Enter fills the form here; it must not submit (save) the transaction.
            if (e.key === "Enter") {
              e.preventDefault();
              void fill();
            }
          }}
        />
        <Button type="button" variant="outline" disabled={busy || text.trim() === ""} onClick={() => void fill()}>
          {t("draftFill")}
        </Button>
      </div>
      <FieldDescription>{t("draftHint")}</FieldDescription>
      <FieldError>{error}</FieldError>
    </Field>
  );
}
