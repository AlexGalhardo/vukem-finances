import { Head } from "@inertiajs/react";
import PageHeader from "../components/PageHeader";
import { Card, danger, primary, secondary } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

/** The consent screen of the remote MCP authorization server. The form is a plain POST to the authorization endpoint: the answer is a redirect to the client. */
export default function OAuthConsent(props: Page<"OAuthConsent">) {
  const { t } = useFinanceI18n();
  const { clientName, clientId, redirectHost, workspace, role, workspaces, scopes, action, fields, returnTo } = props;

  return (
    <>
      <Head title={t("consentTitle")} />
      <PageHeader title={t("consentTitle")} description={t("consentIntro").replace("{client}", clientName)} />

      <Card title={t("consentClient")}>
        <dl className="grid gap-2 text-sm sm:grid-cols-[12rem_1fr]">
          <dt className="text-muted-foreground">{t("consentClient")}</dt>
          <dd>{clientName}</dd>
          <dt className="text-muted-foreground">{t("consentClientId")}</dt>
          <dd className="break-all">{clientId}</dd>
          <dt className="text-muted-foreground">{t("consentRedirect")}</dt>
          <dd>{redirectHost}</dd>
        </dl>
      </Card>

      <Card title={t("consentWorkspace")}>
        <p className="font-medium">{workspace}</p>
        <p className="text-sm text-muted-foreground">{t("consentRole").replace("{role}", role)}</p>
        {workspaces.length > 1 && (
          <form method="post" action="/oauth/workspace" className="mt-3 flex items-center gap-2">
            <input type="hidden" name="return" value={returnTo} />
            <label className="sr-only" htmlFor="consent-workspace">
              {t("consentWorkspacePick")}
            </label>
            <select id="consent-workspace" name="workspace" defaultValue={workspaces.find((w) => w.current)?.id}>
              {workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
            <button type="submit" className={secondary}>
              {t("consentWorkspacePick")}
            </button>
          </form>
        )}
      </Card>

      <Card title={t("consentScopes")}>
        <ul className="grid gap-1 text-sm">
          {scopes.map((s) => (
            <li key={s.name} className={s.allowed ? "" : "text-muted-foreground line-through"}>
              <code>{s.name}</code>
              {s.writes && <span> ({t("consentScopeWrites")})</span>}
              {!s.allowed && <span className="no-underline"> {t("consentScopeNotAllowed")}</span>}
            </li>
          ))}
        </ul>
      </Card>

      <form method="post" action={action} className="flex gap-3">
        {fields.map((f) => (
          <input key={`${f.name}=${f.value}`} type="hidden" name={f.name} value={f.value} />
        ))}
        <button type="submit" name="decision" value="approve" className={primary}>
          {t("consentApprove")}
        </button>
        <button type="submit" name="decision" value="deny" className={danger}>
          {t("consentDeny")}
        </button>
      </form>

      <p className="text-sm text-muted-foreground">{t("consentNote")}</p>
    </>
  );
}
