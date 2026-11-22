import { Head, useForm } from "@inertiajs/react";
import { type FormEvent, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import DataTable from "../components/DataTable";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import SettingsSection from "../components/settings/SettingsSection";
import { Select } from "../components/ui";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Checkbox } from "../components/ui/checkbox";
import { Label } from "../components/ui/label";
import { HelpHint } from "../help/HelpHint";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Tokens({ tokens, scopes, expiryDays, max, created }: Page<"Tokens">) {
  const { t, date } = useFinanceI18n();
  const form = useForm({ name: "", scopes: [] as string[], days: String(expiryDays[0] ?? 30) });
  const [dismissed, setDismissed] = useState(false);
  const [copied, setCopied] = useState(false);

  const toggle = (scope: string, on: boolean) =>
    form.setData("scopes", on ? [...form.data.scopes, scope] : form.data.scopes.filter((s) => s !== scope));

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setDismissed(false);
    setCopied(false);
    form.post("/tokens", { preserveScroll: true, onSuccess: () => form.reset("name", "scopes") });
  };

  const copy = async (secret: string) => {
    await navigator.clipboard?.writeText(secret);
    setCopied(true);
  };

  const statusLabel = {
    live: t("tokensStatusLive"),
    expired: t("tokensStatusExpired"),
    revoked: t("tokensStatusRevoked"),
  };
  const statusVariant = { live: "success", expired: "warning", revoked: "neutral" } as const;

  return (
    <>
      <Head title={t("tokens")} />
      <PageHeader title={t("tokens")} description={t("tokensIntro")} />

      <div className="flex flex-col">
        {created && !dismissed && (
          <SettingsSection title={t("tokensCreatedTitle")} description={t("tokensCreatedBody")} tour="tokens-copy">
            <p className="break-all">
              <strong>{created.name}</strong>: <code data-testid="token-secret">{created.secret}</code>
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <Button type="button" variant="outline" onClick={() => void copy(created.secret)}>
                {t("tokensCopy")}
              </Button>
              <span role="status" className="text-caption text-muted-foreground">
                {copied ? t("tokensCopied") : ""}
              </span>
              <Button type="button" variant="brand" onClick={() => setDismissed(true)}>
                {t("tokensCreatedDismiss")}
              </Button>
            </div>
          </SettingsSection>
        )}

        <SettingsSection title={t("tokensCreateTitle")} description={t("tokensCreateDescription")} tour="tokens-new">
          <form onSubmit={submit} className="flex flex-col gap-4" autoComplete="off">
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
            <fieldset className="flex flex-col gap-2">
              <legend className="mb-2 flex items-center gap-1 text-label">
                {t("tokensScopes")}
                <HelpHint id="hintTokScopes" labelKey="tokensHelpScopes" />
              </legend>
              <div className="grid gap-2 sm:grid-cols-2">
                {scopes.map((scope) => (
                  <div key={scope} className="flex items-center gap-2">
                    <Checkbox
                      id={`scope-${scope}`}
                      checked={form.data.scopes.includes(scope)}
                      onCheckedChange={(checked) => toggle(scope, checked === true)}
                    />
                    <Label htmlFor={`scope-${scope}`} className="min-w-0 font-normal break-all">
                      <code>{scope}</code>
                    </Label>
                  </div>
                ))}
              </div>
              {form.errors.scopes && <p className="text-caption text-expense">{form.errors.scopes}</p>}
            </fieldset>
            <div className="flex max-w-xs items-end gap-1">
              <div className="min-w-0 flex-1">
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
              <HelpHint id="hintTokExpiry" labelKey="tokensHelpExpiry" side="right" />
            </div>
            <p className="text-caption text-muted-foreground">{t("tokensLimit").replace("{max}", String(max))}</p>
            <div>
              <Button type="submit" variant="brand" loading={form.processing}>
                {t("tokensCreate")}
              </Button>
            </div>
          </form>
        </SettingsSection>

        <SettingsSection title={t("tokensListTitle")} description={t("tokensListDescription")} tour="tokens-table">
          <DataTable
            caption={t("tokensListTitle")}
            rows={tokens}
            rowKey={(tok) => tok.id}
            empty={t("tokensNone")}
            columns={[
              { id: "name", header: t("tokensName"), cell: (tok) => tok.name },
              { id: "scopes", header: t("tokensScopes"), cell: (tok) => tok.scopes.join(", ") },
              {
                id: "lastUsed",
                header: t("tokensLastUsed"),
                cell: (tok) => (tok.lastUsedAt ? date(tok.lastUsedAt) : t("tokensNeverUsed")),
              },
              { id: "expires", header: t("tokensExpires"), cell: (tok) => (tok.expiresAt ? date(tok.expiresAt) : "") },
              {
                id: "status",
                header: t("tokensStatus"),
                cell: (tok) => (
                  <Badge variant={statusVariant[tok.status as keyof typeof statusVariant] ?? "neutral"}>
                    {statusLabel[tok.status as keyof typeof statusLabel] ?? tok.status}
                  </Badge>
                ),
              },
              {
                id: "actions",
                header: <span className="sr-only">{t("tokensRevoke")}</span>,
                align: "end",
                cell: (tok) =>
                  tok.status === "live" && (
                    <ConfirmDelete
                      href={`/tokens/${tok.id}`}
                      name={tok.name}
                      trigger={
                        <Button type="button" variant="destructive-outline" size="sm">
                          {t("tokensRevoke")}
                        </Button>
                      }
                    />
                  ),
              },
            ]}
          />
        </SettingsSection>
      </div>
    </>
  );
}
