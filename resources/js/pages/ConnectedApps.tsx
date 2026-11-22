import { Head } from "@inertiajs/react";
import ConfirmDelete from "../components/ConfirmDelete";
import PageHeader from "../components/PageHeader";
import { Card, danger } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function ConnectedApps({ apps }: Page<"ConnectedApps">) {
  const { t, date } = useFinanceI18n();

  const statusLabel = { live: t("connectedAppsStatusLive"), revoked: t("connectedAppsStatusRevoked") };

  return (
    <>
      <Head title={t("connectedApps")} />
      <PageHeader title={t("connectedApps")} description={t("connectedAppsIntro")} />

      <Card title={t("connectedAppsListTitle")}>
        {apps.length === 0 ? (
          <p>{t("connectedAppsNone")}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left">
                <th scope="col">{t("connectedAppsClient")}</th>
                <th scope="col">{t("connectedAppsScopes")}</th>
                <th scope="col">{t("connectedAppsLastUsed")}</th>
                <th scope="col">{t("connectedAppsStatus")}</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {apps.map((app) => (
                <tr key={app.id}>
                  <td>{app.client}</td>
                  <td>{app.scopes.join(", ")}</td>
                  <td>{app.lastUsedAt ? date(app.lastUsedAt) : t("connectedAppsNeverUsed")}</td>
                  <td>{statusLabel[app.status as keyof typeof statusLabel] ?? app.status}</td>
                  <td>
                    {app.status === "live" && (
                      <ConfirmDelete
                        href={`/connected-apps/${app.id}`}
                        name={app.client}
                        trigger={
                          <button type="button" className={danger}>
                            {t("connectedAppsRevoke")}
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

      <p className="text-sm text-muted-foreground">{t("connectedAppsRevokeNote")}</p>
    </>
  );
}
