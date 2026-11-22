import { Head, Link, useForm } from "@inertiajs/react";
import { type FormEvent, useState } from "react";
import Field from "../../components/Field";
import PageHeader from "../../components/PageHeader";
import PasswordField from "../../components/PasswordField";
import SettingsSection from "../../components/settings/SettingsSection";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { HelpHint } from "../../help/HelpHint";
import { useFinanceI18n } from "../../lib/i18n";
import { fill } from "../../lib/pix";

interface Props {
  enabled: boolean;
  pending: boolean;
  secret: string | null;
  qr: string | null;
  recoveryCodes: string[] | null;
  recoveryCodesLeft: number;
}

export default function TwoFactor({ enabled, pending, secret, qr, recoveryCodes, recoveryCodesLeft }: Props) {
  const { t } = useFinanceI18n();
  const confirm = useForm({ code: "" });
  const regenerate = useForm({ password: "" });
  const disable = useForm({ password: "" });
  const [copied, setCopied] = useState(false);

  const submitConfirm = (e: FormEvent) => {
    e.preventDefault();
    confirm.post("/user/two-factor/confirm", { onFinish: () => confirm.reset("code") });
  };

  const submitRegenerate = (e: FormEvent) => {
    e.preventDefault();
    setCopied(false);
    regenerate.post("/user/two-factor/recovery-codes", { onFinish: () => regenerate.reset("password") });
  };

  const submitDisable = (e: FormEvent) => {
    e.preventDefault();
    disable.post("/user/two-factor/disable", { onFinish: () => disable.reset("password") });
  };

  const copyCodes = async (codes: string[]) => {
    await navigator.clipboard?.writeText(codes.join("\n"));
    setCopied(true);
  };

  return (
    <>
      <Head title={t("tfaTitle")} />
      <PageHeader title={t("tfaTitle")} />
      {enabled ? (
        <div className="flex flex-col">
          <SettingsSection title={t("tfaStatusTitle")}>
            <p>
              <Badge variant="success">{t("tfaOn")}</Badge>
            </p>
          </SettingsSection>

          <SettingsSection
            title={t("tfaRecoveryTitle")}
            description={t("tfaRecoveryDescription")}
            tour="twofactor-recovery"
            help={<HelpHint id="hintTfaRecoveryCodes" labelKey="tfaHelpRecovery" tutorialStep="TwoFactor:1" />}
          >
            {recoveryCodes ? (
              <>
                <p>
                  <strong>{t("tfaRecoveryStore")}</strong>
                </p>
                <ul className="grid grid-cols-1 gap-2 font-mono sm:grid-cols-2">
                  {recoveryCodes.map((code) => (
                    <li key={code} className="break-all rounded-md bg-muted px-3 py-2">
                      {code}
                    </li>
                  ))}
                </ul>
                <div className="flex items-center gap-2">
                  <Button type="button" variant="outline" onClick={() => void copyCodes(recoveryCodes)}>
                    {t("tfaCopyCodes")}
                  </Button>
                  <span role="status" className="text-caption text-muted-foreground">
                    {copied ? t("tfaCopied") : ""}
                  </span>
                </div>
              </>
            ) : (
              <p>{fill(t("tfaRecoveryLeft"), { count: recoveryCodesLeft })}</p>
            )}
            <form onSubmit={submitRegenerate} className="flex max-w-md flex-col gap-4">
              <PasswordField
                label={t("tfaCurrentPassword")}
                name="password"
                autoComplete="current-password"
                value={regenerate.data.password}
                onChange={(value) => regenerate.setData("password", value)}
                error={regenerate.errors.password}
              />
              <div>
                <Button type="submit" variant="outline" loading={regenerate.processing}>
                  {t("tfaRegenerate")}
                </Button>
              </div>
            </form>
          </SettingsSection>

          <SettingsSection danger title={t("tfaTurnOff")} description={t("tfaTurnOffDescription")}>
            <form onSubmit={submitDisable} className="flex max-w-md flex-col gap-4">
              <PasswordField
                label={t("tfaCurrentPassword")}
                name="password"
                autoComplete="current-password"
                value={disable.data.password}
                onChange={(value) => disable.setData("password", value)}
                error={disable.errors.password}
              />
              <div>
                <Button type="submit" variant="destructive" loading={disable.processing}>
                  {t("tfaDisable")}
                </Button>
              </div>
            </form>
          </SettingsSection>
        </div>
      ) : pending && qr ? (
        <div className="flex flex-col">
          <SettingsSection title={t("tfaSetupTitle")} description={t("tfaScan")}>
            <img
              className="w-fit rounded-md border border-border p-2"
              style={{ backgroundColor: "white" }}
              src={qr}
              width={200}
              height={200}
              alt={t("tfaQrAlt")}
            />
            <p className="break-all">
              {t("tfaSetupKey")}: <code className="font-mono">{secret}</code>
            </p>
            <form onSubmit={submitConfirm} className="flex max-w-xs flex-col gap-4">
              <Field
                label={t("tfaCode")}
                name="code"
                inputMode="numeric"
                autoComplete="one-time-code"
                required
                value={confirm.data.code}
                onChange={(e) => confirm.setData("code", e.target.value)}
                error={confirm.errors.code}
              />
              <div>
                <Button type="submit" variant="brand" loading={confirm.processing}>
                  {t("tfaConfirm")}
                </Button>
              </div>
            </form>
          </SettingsSection>
        </div>
      ) : (
        <div className="flex flex-col">
          <SettingsSection title={t("tfaStatusTitle")} description={t("tfaOff")} tour="twofactor-enable">
            <div>
              <Button asChild variant="brand">
                <Link href="/user/two-factor" method="post" as="button">
                  {t("tfaEnable")}
                </Link>
              </Button>
            </div>
          </SettingsSection>
        </div>
      )}
      <p className="text-body">
        <Link href="/dashboard" className="text-brand-text underline underline-offset-4">
          {t("tfaBack")}
        </Link>
      </p>
    </>
  );
}
