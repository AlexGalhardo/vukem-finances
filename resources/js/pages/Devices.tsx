import { Head, router } from "@inertiajs/react";
import { useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import PageHeader from "../components/PageHeader";
import { Card, danger } from "../components/ui";
import { Button } from "../components/ui/button";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Devices({ devices, pairing }: Page<"Devices">) {
  const { t, date } = useFinanceI18n();
  const [dismissed, setDismissed] = useState(false);

  const statusLabel = { live: t("devicesStatusLive"), revoked: t("devicesStatusRevoked") };

  return (
    <>
      <Head title={t("devices")} />
      <PageHeader title={t("devices")} description={t("devicesIntro")} />

      {pairing && !dismissed && (
        <Card title={t("devicesPairTitle")}>
          <p className="mb-2">{t("devicesPairBody")}</p>
          <p>
            {t("devicesPairServer")}: <code data-testid="pairing-server">{pairing.server}</code>
          </p>
          <p className="mb-2">
            {t("devicesPairCode")}: <code data-testid="pairing-code">{pairing.code}</code>
          </p>
          <p className="mb-3 text-sm">
            {t("devicesPairExpires")} {date(pairing.expiresAt)}
          </p>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => navigator.clipboard?.writeText(pairing.code)}>
              {t("devicesPairCopy")}
            </Button>
            <Button type="button" onClick={() => setDismissed(true)}>
              {t("devicesPairDismiss")}
            </Button>
          </div>
        </Card>
      )}

      <div className="mb-4">
        <Button
          type="button"
          onClick={() => {
            setDismissed(false);
            router.post("/devices/pairing", {}, { preserveScroll: true });
          }}
        >
          {t("devicesAdd")}
        </Button>
      </div>

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
