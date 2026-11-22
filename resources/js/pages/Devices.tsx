import { Head } from "@inertiajs/react";
import ConfirmDelete from "../components/ConfirmDelete";
import PageHeader from "../components/PageHeader";
import { Card, danger } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Devices({ devices }: Page<"Devices">) {
  const { t, date } = useFinanceI18n();

  const statusLabel = { live: t("devicesStatusLive"), revoked: t("devicesStatusRevoked") };

  return (
    <>
      <Head title={t("devices")} />
      <PageHeader title={t("devices")} description={t("devicesIntro")} />

      <Card title={t("devicesListTitle")}>
        {devices.length === 0 ? (
          <p>{t("devicesNone")}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left">
                <th scope="col">{t("devicesName")}</th>
                <th scope="col">{t("devicesPlatform")}</th>
                <th scope="col">{t("devicesLastSeen")}</th>
                <th scope="col">{t("devicesLastPull")}</th>
                <th scope="col">{t("devicesStatus")}</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {devices.map((device) => (
                <tr key={device.id}>
                  <td>{device.name}</td>
                  <td>{device.platform}</td>
                  <td>{device.lastSeenAt ? date(device.lastSeenAt) : t("devicesNeverSeen")}</td>
                  <td>{device.lastPullSeq}</td>
                  <td>{statusLabel[device.status as keyof typeof statusLabel] ?? device.status}</td>
                  <td>
                    {device.status === "live" && (
                      <ConfirmDelete
                        href={`/devices/${device.id}`}
                        name={device.name}
                        trigger={
                          <button type="button" className={danger}>
                            {t("devicesRevoke")}
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

      <p className="text-sm text-muted-foreground">{t("devicesRevokeNote")}</p>
    </>
  );
}
