import { Head, router, useForm } from "@inertiajs/react";
import { type FormEvent, useId, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import { Card, danger, primary, Select, secondary } from "../components/ui";
import { Button } from "../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { type Key, useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

const providerLabels: Record<string, Key> = {
  anthropic: "aiProviderAnthropic",
  openai: "aiProviderOpenai",
  openai_compatible: "aiProviderCompatible",
};

function Check({ label, ...input }: { label: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="flex items-center gap-2">
      <input type="checkbox" className="size-4" {...input} />
      <span>{label}</span>
    </label>
  );
}

export default function AI({ settings, credential, usage, providers, canManage, privateHosts }: Page<"AI">) {
  const { t } = useFinanceI18n();
  const consentId = useId();
  const [consentOpen, setConsentOpen] = useState(false);
  const [consent, setConsent] = useState(false);

  // The key lives in this form's state only until it is sent, and the field is cleared after every attempt.
  const keyForm = useForm({ provider: providers[0] ?? "anthropic", apiSecret: "", label: "", baseUrl: "", model: "" });
  const switches = useForm({
    enabled: settings.enabled,
    categorize: settings.features.categorize === true,
    assistant: settings.features.assistant === true,
    limit: String(settings.monthlyCallLimit),
  });

  const saveKey = (e: FormEvent) => {
    e.preventDefault();
    keyForm.post("/ai/key", { preserveScroll: true, onFinish: () => keyForm.reset("apiSecret") });
  };

  const sendSettings = (withConsent: boolean) =>
    router.put(
      "/ai/settings",
      { ...switches.data, consent: withConsent },
      {
        preserveScroll: true,
        onFinish: () => {
          setConsentOpen(false);
          setConsent(false);
        },
      },
    );

  const saveSettings = (e: FormEvent) => {
    e.preventDefault();
    // Turning AI on asks first what is sent; turning it off or changing a limit needs no question.
    if (switches.data.enabled && !settings.enabled) setConsentOpen(true);
    else sendSettings(false);
  };

  return (
    <>
      <Head title={t("ai")} />
      <PageHeader title={t("ai")} description={t("aiIntro")} />

      <Card title={t("aiStatus")}>
        <p>{settings.enabled ? t("aiStatusOn") : t("aiStatusOff")}</p>
        {!canManage && <p className="mt-2 text-sm text-muted-foreground">{t("aiOwnerOnly")}</p>}
      </Card>

      {canManage && (
        <>
          <Card title={t("aiKeyTitle")}>
            {credential ? (
              <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                <p>
                  {t("aiKeySaved")}: {t(providerLabels[credential.provider] ?? "aiProvider")}
                  {credential.keyHint && (
                    <>
                      {" "}
                      {t("aiKeyEnding")} <code>{credential.keyHint}</code>
                    </>
                  )}
                  . {t("aiLastUsed")}: {credential.lastUsedAt || t("aiNeverUsed")}.
                </p>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => router.post("/ai/check", {}, { preserveScroll: true })}
                  >
                    {t("aiTestKey")}
                  </Button>
                  <ConfirmDelete
                    href={`/ai/key/${credential.id}`}
                    name={credential.provider}
                    trigger={
                      <button type="button" className={danger}>
                        {t("delete")}
                      </button>
                    }
                  />
                </div>
              </div>
            ) : (
              <p className="mb-4">{t("aiNoKey")}</p>
            )}

            <form onSubmit={saveKey} className="grid gap-3 sm:grid-cols-2" autoComplete="off">
              <Select
                label={t("aiProvider")}
                value={keyForm.data.provider}
                onChange={(e) => keyForm.setData("provider", e.target.value)}
                error={keyForm.errors.provider}
              >
                {providers.map((p) => (
                  <option key={p} value={p}>
                    {t(providerLabels[p] ?? "aiProvider")}
                  </option>
                ))}
              </Select>
              <div>
                <Field
                  label={t("aiKey")}
                  type="password"
                  name="apiSecret"
                  autoComplete="off"
                  required
                  value={keyForm.data.apiSecret}
                  onChange={(e) => keyForm.setData("apiSecret", e.target.value)}
                  error={keyForm.errors.apiSecret}
                />
                <p className="mt-1 text-sm text-muted-foreground">{t("aiKeyHelp")}</p>
              </div>
              {keyForm.data.provider === "openai_compatible" && (
                <div>
                  <Field
                    label={t("aiBaseUrl")}
                    type="url"
                    name="baseUrl"
                    required
                    placeholder={privateHosts ? "http://localhost:11434/v1" : "https://api.example.com/v1"}
                    value={keyForm.data.baseUrl}
                    onChange={(e) => keyForm.setData("baseUrl", e.target.value)}
                    error={keyForm.errors.baseUrl}
                  />
                  <p className="mt-1 text-sm text-muted-foreground">{t("aiBaseUrlHelp")}</p>
                </div>
              )}
              <Field
                label={t("aiModel")}
                name="model"
                value={keyForm.data.model}
                onChange={(e) => keyForm.setData("model", e.target.value)}
                error={keyForm.errors.model}
              />
              <Field
                label={t("aiLabel")}
                name="label"
                value={keyForm.data.label}
                onChange={(e) => keyForm.setData("label", e.target.value)}
                error={keyForm.errors.label}
              />
              <div className="self-end">
                <button type="submit" disabled={keyForm.processing} className={primary}>
                  {t("aiSaveKey")}
                </button>
              </div>
            </form>
          </Card>

          <Card title={t("aiSettingsTitle")}>
            <form onSubmit={saveSettings} className="flex flex-col gap-3">
              <Check
                label={t("aiEnable")}
                checked={switches.data.enabled}
                onChange={(e) => switches.setData("enabled", e.target.checked)}
              />
              <Check
                label={t("aiFeatureCategorize")}
                checked={switches.data.categorize}
                onChange={(e) => switches.setData("categorize", e.target.checked)}
              />
              <Check
                label={t("aiFeatureAssistant")}
                checked={switches.data.assistant}
                onChange={(e) => switches.setData("assistant", e.target.checked)}
              />
              <div className="max-w-xs">
                <Field
                  label={t("aiMonthlyLimit")}
                  type="number"
                  min={1}
                  max={1000000}
                  name="limit"
                  value={switches.data.limit}
                  onChange={(e) => switches.setData("limit", e.target.value)}
                  error={switches.errors.limit}
                />
              </div>
              <div>
                <button type="submit" className={secondary}>
                  {t("aiSaveSettings")}
                </button>
              </div>
            </form>
          </Card>
        </>
      )}

      <Card title={t("aiUsageTitle")}>
        {usage.byFeature.length === 0 ? (
          <p>{t("aiUsageNone")}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left">
                <th scope="col">{t("aiUsageFeature")}</th>
                <th scope="col">{t("aiProvider")}</th>
                <th scope="col">{t("aiUsageCalls")}</th>
                <th scope="col">{t("aiUsageInput")}</th>
                <th scope="col">{t("aiUsageOutput")}</th>
              </tr>
            </thead>
            <tbody>
              {usage.byFeature.map((row) => (
                <tr key={`${row.feature}/${row.provider}`}>
                  <td>{row.feature}</td>
                  <td>{row.provider}</td>
                  <td>{row.calls}</td>
                  <td>{row.inputTokens}</td>
                  <td>{row.outputTokens}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <p className="mt-3 text-sm text-muted-foreground">
          {usage.month}: {usage.calls} / {settings.monthlyCallLimit} {t("aiUsageCalls").toLowerCase()}.{" "}
          {t("aiBillingNote")}
        </p>
      </Card>

      <Dialog open={consentOpen} onOpenChange={setConsentOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiConsentTitle")}</DialogTitle>
            <DialogDescription>{t("aiConsentBody")}</DialogDescription>
          </DialogHeader>
          <label htmlFor={consentId} className="flex items-start gap-2">
            <input
              id={consentId}
              type="checkbox"
              className="mt-1 size-4"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
            />
            <span>{t("aiConsentCheckbox")}</span>
          </label>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setConsentOpen(false)}>
              {t("cancel")}
            </Button>
            <Button type="button" disabled={!consent} onClick={() => sendSettings(true)}>
              {t("aiConsentConfirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
