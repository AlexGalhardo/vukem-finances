import { router, usePage } from "@inertiajs/react";
import { type FormEvent, useCallback, useEffect, useId, useRef, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import type { SharedProps } from "../../page-props";
import { useAction } from "../../shortcuts/useAction";
import { readProblem } from "../transactions/entryForm";
import { Button } from "../ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "../ui/dialog";
import { Input } from "../ui/input";

interface Figure {
  label: string;
  cents: number;
}

interface ResultTable {
  title: string;
  rows: Figure[];
  truncated: boolean;
}

interface Reply {
  conversationId: number;
  answered: boolean;
  text: string;
  tables: ResultTable[];
  queries: string[];
  figuresDiffer: boolean;
}

interface Turn {
  role: "user" | "assistant";
  text: string;
  reply?: Reply;
}

interface Status {
  enabled: boolean;
  conversations: { id: number; title: string; updatedAt: string }[];
  questionMax: number;
}

async function getJSON<T>(url: string): Promise<T | null> {
  try {
    const response = await fetch(url, { headers: { Accept: "application/json" }, credentials: "same-origin" });
    return response.ok ? ((await response.json()) as T) : null;
  } catch {
    return null;
  }
}

async function send(url: string, method: "POST" | "DELETE", body?: unknown): Promise<Response | null> {
  try {
    return await fetch(url, {
      method,
      credentials: "same-origin",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    return null;
  }
}

/**
 * Owns the assistant dialog for every signed-in page and registers its actions: `ask-assistant` (`a s`) opens it and
 * `review-suggestions` (`g s`) goes to the suggestions. The model's text is rendered as plain text (React escapes it:
 * never HTML, Markdown or links); the tables beside it are the server's own figures and the authoritative ones.
 */
export default function AssistantHost() {
  const { t, money } = useFinanceI18n();
  const workspace = usePage<SharedProps>().props.workspace;
  const currency = workspace?.current.currency ?? "USD";
  const canAsk = workspace?.role !== undefined && workspace.role !== "viewer";
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<Status | null>(null);
  const [turns, setTurns] = useState<Turn[]>([]);
  const [conversation, setConversation] = useState(0);
  const [question, setQuestion] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const opener = useRef<HTMLElement | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const inputId = useId();

  const show = useCallback((): void => {
    opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setError("");
    setOpen(true);
  }, []);

  useAction({
    id: "ask-assistant",
    labelKey: "askAssistant",
    group: "actions",
    keys: "a s",
    enabled: () => canAsk,
    run: show,
  });
  useAction({
    id: "review-suggestions",
    labelKey: "reviewSuggestions",
    group: "actions",
    keys: "g s",
    run: () => router.visit("/suggestions"),
  });

  useEffect(() => {
    if (!open) return;
    let alive = true;
    void getJSON<Status>("/assistant/conversations").then((s) => alive && setStatus(s));
    return () => {
      alive = false;
    };
  }, [open]);

  const load = async (id: number): Promise<void> => {
    const body = await getJSON<{ messages: { role: "user" | "assistant"; content: string; reply: Reply | null }[] }>(
      `/assistant/conversations/${id}`,
    );
    if (!body) return;
    setConversation(id);
    setTurns(body.messages.map((m) => ({ role: m.role, text: m.content, reply: m.reply ?? undefined })));
  };

  const ask = async (event: FormEvent): Promise<void> => {
    event.preventDefault();
    const text = question.trim();
    if (busy || text === "") return;
    setBusy(true);
    setError("");
    const response = await send("/assistant/ask", "POST", {
      question: text,
      conversationId: conversation === 0 ? "" : String(conversation),
    });
    if (response?.ok) {
      const reply = (await response.json()) as Reply;
      setConversation(reply.conversationId);
      setTurns((current) => [
        ...current,
        { role: "user", text },
        { role: "assistant", text: reply.answered ? reply.text : t("assistantUnanswered"), reply },
      ]);
      setQuestion("");
      void getJSON<Status>("/assistant/conversations").then(setStatus);
    } else {
      const body: unknown = await response?.json().catch(() => null);
      setError(readProblem(body, t("assistantError")).message);
    }
    setBusy(false);
    inputRef.current?.focus();
  };

  const deleteAll = async (): Promise<void> => {
    await send("/assistant/conversations", "DELETE");
    setTurns([]);
    setConversation(0);
    setStatus((s) => (s ? { ...s, conversations: [] } : s));
  };

  const examples = [t("assistantExample1"), t("assistantExample2"), t("assistantExample3")];

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
        className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl"
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          inputRef.current?.focus();
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          if (opener.current?.isConnected) opener.current.focus();
        }}
      >
        <DialogHeader>
          <DialogTitle>{t("assistantTitle")}</DialogTitle>
          <DialogDescription>{t("assistantDisclosure")}</DialogDescription>
        </DialogHeader>
        {status !== null && !status.enabled ? (
          <p role="alert" className="text-sm">
            {t("assistantOff")}
          </p>
        ) : (
          <div className="flex flex-col gap-4">
            <div role="log" aria-live="polite" aria-label={t("assistantLog")} className="flex flex-col gap-4">
              {turns.length === 0 && <p className="text-sm text-muted-foreground">{t("assistantEmpty")}</p>}
              {turns.map((turn, index) => (
                // The turns only ever grow, so the position is a stable key.
                // biome-ignore lint/suspicious/noArrayIndexKey: append-only list
                <div key={index} className="flex flex-col gap-2">
                  <p className="text-xs font-medium text-muted-foreground">
                    {turn.role === "user" ? t("assistantYou") : t("assistantName")}
                  </p>
                  <p className="whitespace-pre-wrap text-sm">{turn.text}</p>
                  {turn.reply?.figuresDiffer && (
                    <p role="note" className="text-xs text-destructive">
                      {t("assistantDiffers")}
                    </p>
                  )}
                  {turn.reply?.tables.map((table) => (
                    <div key={table.title} className="overflow-x-auto rounded-lg border">
                      <table className="w-full text-sm">
                        <caption className="p-2 text-left text-xs text-muted-foreground">{table.title}</caption>
                        <tbody>
                          {table.rows.map((row, i) => (
                            // biome-ignore lint/suspicious/noArrayIndexKey: labels can repeat
                            <tr key={i} className="border-t">
                              <th scope="row" className="p-2 text-left font-normal">
                                {row.label}
                              </th>
                              <td className="p-2 text-right tabular-nums">{money(row.cents, currency)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                      {table.truncated && (
                        <p className="p-2 text-xs text-muted-foreground">{t("assistantTruncated")}</p>
                      )}
                    </div>
                  ))}
                  {turn.reply && turn.reply.queries.length > 0 && (
                    <p className="text-xs text-muted-foreground">
                      {t("assistantBasedOn")}: {turn.reply.queries.join("; ")}. {t("assistantTablesNote")}
                    </p>
                  )}
                </div>
              ))}
            </div>
            {error !== "" && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            <form onSubmit={ask} className="flex items-end gap-2">
              <div className="flex flex-1 flex-col gap-1">
                <label htmlFor={inputId} className="text-xs text-muted-foreground">
                  {t("assistantQuestion")}
                </label>
                <Input
                  id={inputId}
                  ref={inputRef}
                  autoComplete="off"
                  maxLength={status?.questionMax ?? 500}
                  placeholder={t("assistantPlaceholder")}
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                />
              </div>
              <Button type="submit" disabled={busy || question.trim() === ""}>
                {busy ? t("assistantThinking") : t("assistantAsk")}
              </Button>
            </form>
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-muted-foreground">{t("assistantExamples")}:</span>
              {examples.map((example) => (
                <Button
                  key={example}
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setQuestion(example);
                    inputRef.current?.focus();
                  }}
                >
                  {example}
                </Button>
              ))}
            </div>
            {status !== null && status.conversations.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                {status.conversations.slice(0, 5).map((c) => (
                  <Button key={c.id} type="button" variant="ghost" size="sm" onClick={() => void load(c.id)}>
                    {c.title}
                  </Button>
                ))}
                <Button type="button" variant="outline" size="sm" onClick={() => void deleteAll()}>
                  {t("assistantDeleteAll")}
                </Button>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
