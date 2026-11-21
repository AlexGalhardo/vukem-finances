import { Head, useForm } from "@inertiajs/react";
import { type FormEvent, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import { Card, danger, primary, Select } from "../components/ui";
import { Button } from "../components/ui/button";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Tokens({ tokens, scopes, expiryDays, max, created }: Page<"Tokens">) {
  const { t, date } = useFinanceI18n();
  const form = useForm({ name: "", scopes: [] as string[], days: String(expiryDays[0] ?? 30) });
  const [dismissed, setDismissed] = useState(false);

  const toggle = (scope: string, on: boolean) =>
    form.setData("scopes", on ? [...form.data.scopes, scope] : form.data.scopes.filter((s) => s !== scope));

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setDismissed(false);
    form.post("/tokens", { preserveScroll: true, onSuccess: () => form.reset("name", "scopes") });
  };

  const statusLabel = {
    live: t("tokensStatusLive"),
    expired: t("tokensStatusExpired"),
    revoked: t("tokensStatusRevoked"),
  };

  return (
    <>
      <Head title={t("tokens")} />
      <PageHeader title={t("tokens")} description={t("tokensIntro")} />

      {created && !dismissed && (
        <Card title={t("tokensCreatedTitle")}>
          <p className="mb-2">{t("tokensCreatedBody")}</p>
          <p className="mb-3 break-all">
            <strong>{created.name}</strong>: <code data-testid="token-secret">{created.secret}</code>
          </p>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => navigator.clipboard?.writeText(created.secret)}>
              {t("tokensCopy")}
            </Button>
            <Button type="button" onClick={() => setDismissed(true)}>
              {t("tokensCreatedDismiss")}
            </Button>
          </div>
        </Card>
      )}

      <Card title={t("tokensCreateTitle")}>
        <form onSubmit={submit} className="flex flex-col gap-3" autoComplete="off">
          <div className="max-w-md">
            <Field
              label={t("tokensName")}
              name="name"
              required
              maxLength={80}
              value={form.data.name}
              onChange={(e) => form.setData("name", e.target.value)}
              error={form.errors.name}
            />
          </div>
          <fieldset>
            <legend className="mb-1 text-sm font-medium">{t("tokensScopes")}</legend>
            <div className="grid gap-1 sm:grid-cols-2">
              {scopes.map((scope) => (
                <label key={scope} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    className="size-4"
                    checked={form.data.scopes.includes(scope)}
                    onChange={(e) => toggle(scope, e.target.checked)}
                  />
                  <code>{scope}</code>
                </label>
              ))}
            </div>
            {form.errors.scopes && <p className="text-sm text-destructive">{form.errors.scopes}</p>}
          </fieldset>
          <div className="max-w-xs">
            <Select
              label={t("tokensExpiry")}
              value={form.data.days}
              onChange={(e) => form.setData("days", e.target.value)}
              error={form.errors.days}
            >
              {expiryDays.map((d) => (
                <option key={d} value={d}>
                  {d} {t("tokensDays")}
                </option>
              ))}
            </Select>
          </div>
          <p className="text-sm text-muted-foreground">{t("tokensLimit").replace("{max}", String(max))}</p>
          <div>
            <button type="submit" disabled={form.processing} className={primary}>
              {t("tokensCreate")}
            </button>
          </div>
        </form>
      </Card>

      <Card title={t("tokensListTitle")}>
        {tokens.length === 0 ? (
          <p>{t("tokensNone")}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left">
                <th scope="col">{t("tokensName")}</th>
                <th scope="col">{t("tokensScopes")}</th>
                <th scope="col">{t("tokensLastUsed")}</th>
                <th scope="col">{t("tokensExpires")}</th>
                <th scope="col">{t("tokensStatus")}</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {tokens.map((tok) => (
                <tr key={tok.id}>
                  <td>{tok.name}</td>
                  <td>{tok.scopes.join(", ")}</td>
                  <td>{tok.lastUsedAt ? date(tok.lastUsedAt) : t("tokensNeverUsed")}</td>
                  <td>{tok.expiresAt ? date(tok.expiresAt) : ""}</td>
                  <td>{statusLabel[tok.status as keyof typeof statusLabel] ?? tok.status}</td>
                  <td>
                    {tok.status === "live" && (
                      <ConfirmDelete
                        href={`/tokens/${tok.id}`}
                        name={tok.name}
                        trigger={
                          <button type="button" className={danger}>
                            {t("tokensRevoke")}
                          </button>
                        }
                      />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </>
  );
}
