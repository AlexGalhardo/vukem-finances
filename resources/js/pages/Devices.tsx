import { Head, router } from "@inertiajs/react";
import { Plus } from "lucide-react";
import { useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import DataTable from "../components/DataTable";
import PageHeader from "../components/PageHeader";
import SettingsSection from "../components/settings/SettingsSection";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { HelpHint } from "../help/HelpHint";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

export default function Devices({ devices, pairing }: Page<"Devices">) {
  const { t, date } = useFinanceI18n();
  const [dismissed, setDismissed] = useState(false);
  const [copied, setCopied] = useState(false);

  const statusLabel = { live: t("devicesStatusLive"), revoked: t("devicesStatusRevoked") };
  const statusVariant = { live: "success", revoked: "neutral" } as const;

  return (
    <>
      <Head title={t("devices")} />
      <PageHeader
        title={t("devices")}
        description={t("devicesIntro")}
        actions={
          <Button
            type="button"
            variant="brand"
            onClick={() => {
              setDismissed(false);
              setCopied(false);
              router.post("/devices/pairing", {}, { preserveScroll: true });
            }}
          >
            <Plus aria-hidden="true" data-icon="inline-start" />
            {t("devicesAdd")}
          </Button>
        }
      />

      <div className="flex flex-col">
        {pairing && !dismissed && (
          <SettingsSection title={t("devicesPairTitle")} description={t("devicesPairDescription")}>
            <p>{t("devicesPairBody")}</p>
            <p className="break-all">
              {t("devicesPairServer")}: <code data-testid="pairing-server">{pairing.server}</code>
            </p>
            <p>
              {t("devicesPairCode")}: <code data-testid="pairing-code">{pairing.code}</code>
            </p>
            <p className="text-caption text-muted-foreground">
              {t("devicesPairExpires")} {date(pairing.expiresAt)}
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={async () => {
                  await navigator.clipboard?.writeText(pairing.code);
                  setCopied(true);
                }}
              >
                {t("devicesPairCopy")}
              </Button>
              <span role="status" className="text-caption text-muted-foreground">
                {copied ? t("devicesCopied") : ""}
              </span>
              <Button type="button" variant="brand" onClick={() => setDismissed(true)}>
                {t("devicesPairDismiss")}
              </Button>
            </div>
          </SettingsSection>
        )}

        <SettingsSection
          title={t("devicesListTitle")}
          description={t("devicesListDescription")}
          tour="devices-table"
          help={<HelpHint id="hintDevLastPull" labelKey="devicesHelpLastPull" tutorialStep="Devices:0" />}
        >
          <DataTable
            caption={t("devicesListTitle")}
            rows={devices}
            rowKey={(device) => device.id}
            empty={t("devicesNone")}
            columns={[
              { id: "name", header: t("devicesName"), cell: (device) => device.name },
              { id: "platform", header: t("devicesPlatform"), cell: (device) => device.platform },
              {
                id: "lastSeen",
                header: t("devicesLastSeen"),
                cell: (device) => (device.lastSeenAt ? date(device.lastSeenAt) : t("devicesNeverSeen")),
              },
              { id: "lastPull", header: t("devicesLastPull"), cell: (device) => device.lastPullSeq, align: "end" },
              {
                id: "status",
                header: t("devicesStatus"),
                cell: (device) => (
                  <Badge variant={statusVariant[device.status as keyof typeof statusVariant] ?? "neutral"}>
                    {statusLabel[device.status as keyof typeof statusLabel] ?? device.status}
                  </Badge>
                ),
              },
              {
                id: "actions",
                header: <span className="sr-only">{t("devicesRevoke")}</span>,
                align: "end",
                cell: (device) =>
                  device.status === "live" && (
                    <span data-tour="devices-revoke">
                      <ConfirmDelete
                        href={`/devices/${device.id}`}
                        name={device.name}
                        trigger={
                          <Button type="button" variant="destructive-outline" size="sm">
                            {t("devicesRevoke")}
                          </Button>
                        }
                      />
                    </span>
                  ),
              },
            ]}
          />
          <p className="text-caption text-muted-foreground">{t("devicesRevokeNote")}</p>
        </SettingsSection>
      </div>
    </>
  );
}
