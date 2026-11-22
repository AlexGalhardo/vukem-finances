import { Head, router, useForm, usePage } from "@inertiajs/react";
import { CircleCheck, CircleSlash } from "lucide-react";
import { type FormEvent, useId, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import DataTable from "../components/DataTable";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import SettingsSection from "../components/settings/SettingsSection";
import { Select } from "../components/ui";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Checkbox } from "../components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Label } from "../components/ui/label";
import { HelpHint } from "../help/HelpHint";
import { type Key, useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";

const providerLabels: Record<string, Key> = {
  anthropic: "aiProviderAnthropic",
  openai: "aiProviderOpenai",
  openai_compatible: "aiProviderCompatible",
};

function CheckRow({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  const id = useId();
  return (
    <div className="flex items-center gap-2">
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onChange(value === true)} />
      <Label htmlFor={id} className="font-normal">
        {label}
      </Label>
    </div>
  );
}

export default function AI({ settings, credential, usage, providers, canManage, privateHosts }: Page<"AI">) {
  const { t } = useFinanceI18n();
  const { edition } = usePage().props as { edition?: string };
  const consentId = useId();
  const [consentOpen, setConsentOpen] = useState(false);
  const [consent, setConsent] = useState(false);

  // The key lives in this form's state only until it is sent, and the field is cleared after every attempt.
  const keyForm = useForm({ provider: providers[0] ?? "anthropic", apiSecret: "", label: "", baseUrl: "", model: "" });
  const switches = useForm({
    enabled: settings.enabled,
    categorize: settings.features.categorize === true,
    assistant: settings.features.assistant === true,
    autoApply: settings.autoApplySuggestions,
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

      <div className="flex flex-col">
        <SettingsSection title={t("aiStatus")} description={t("aiStatusDescription")} tour="ai-status">
          <p className="flex items-center gap-2">
            <Badge variant={settings.enabled ? "success" : "neutral"}>
              {settings.enabled ? (
                <CircleCheck aria-hidden="true" data-icon="inline-start" />
              ) : (
                <CircleSlash aria-hidden="true" data-icon="inline-start" />
              )}
              {settings.enabled ? t("aiStatusOn") : t("aiStatusOff")}
            </Badge>
          </p>
          {!canManage && <p className="text-caption text-muted-foreground">{t("aiOwnerOnly")}</p>}
          {edition === "personal" && <p className="text-caption text-muted-foreground">{t("aiNeedsInternet")}</p>}
        </SettingsSection>

        {canManage && (
          <>
            <SettingsSection
              title={t("aiKeyTitle")}
              description={t("aiKeyDescription")}
              tour="ai-key"
              help={<HelpHint id="hintAiProviderKey" labelKey="aiHelpKey" tutorialStep="AI:1" />}
            >
              {credential ? (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="min-w-0">
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
                        <Button type="button" variant="destructive-outline">
                          {t("delete")}
                        </Button>
                      }
                    />
                  </div>
                </div>
              ) : (
                <p>{t("aiNoKey")}</p>
              )}

              <form onSubmit={saveKey} className="flex flex-col gap-4" autoComplete="off">
                <div className="grid gap-4 sm:grid-cols-2">
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
                  <div className="flex flex-col gap-1.5">
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
                    <p className="text-caption text-muted-foreground">{t("aiKeyHelp")}</p>
                  </div>
                  {keyForm.data.provider === "openai_compatible" && (
                    <div className="flex flex-col gap-1.5 sm:col-span-2">
                      <div className="flex items-end gap-1">
                        <div className="min-w-0 flex-1">
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
                        </div>
                        <HelpHint id="hintAiBaseUrl" labelKey="aiHelpBaseUrl" side="left" />
                      </div>
                      <p className="text-caption text-muted-foreground">{t("aiBaseUrlHelp")}</p>
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
                </div>
                <div>
                  <Button type="submit" loading={keyForm.processing}>
                    {t("aiSaveKey")}
                  </Button>
                </div>
              </form>
            </SettingsSection>

            <SettingsSection
              title={t("aiSettingsTitle")}
              description={t("aiSettingsDescription")}
              tour="ai-limits"
              help={<HelpHint id="hintAiCallLimit" labelKey="aiHelpLimit" tutorialStep="AI:2" />}
            >
              <form onSubmit={saveSettings} className="flex flex-col gap-4">
                <div className="flex flex-col gap-3">
                  <CheckRow
                    label={t("aiEnable")}
                    checked={switches.data.enabled}
                    onChange={(value) => switches.setData("enabled", value)}
                  />
                  <CheckRow
                    label={t("aiFeatureCategorize")}
                    checked={switches.data.categorize}
                    onChange={(value) => switches.setData("categorize", value)}
                  />
                  <CheckRow
                    label={t("aiFeatureAssistant")}
                    checked={switches.data.assistant}
                    onChange={(value) => switches.setData("assistant", value)}
                  />
                  <CheckRow
                    label={t("aiAutoApply")}
                    checked={switches.data.autoApply}
                    onChange={(value) => switches.setData("autoApply", value)}
                  />
                </div>
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
                  <Button type="submit" variant="outline">
                    {t("aiSaveSettings")}
                  </Button>
                </div>
              </form>
            </SettingsSection>
          </>
        )}

        <SettingsSection title={t("aiUsageTitle")} description={t("aiUsageDescription")}>
          <DataTable
            caption={t("aiUsageTitle")}
            rows={usage.byFeature}
            rowKey={(row) => `${row.feature}/${row.provider}`}
            empty={t("aiUsageNone")}
            columns={[
              { id: "feature", header: t("aiUsageFeature"), cell: (row) => row.feature },
              { id: "provider", header: t("aiProvider"), cell: (row) => row.provider },
              { id: "calls", header: t("aiUsageCalls"), cell: (row) => row.calls, align: "end" },
              { id: "input", header: t("aiUsageInput"), cell: (row) => row.inputTokens, align: "end" },
              { id: "output", header: t("aiUsageOutput"), cell: (row) => row.outputTokens, align: "end" },
            ]}
          />
          <p className="text-caption text-muted-foreground">
            {usage.month}: {usage.calls} / {settings.monthlyCallLimit} {t("aiUsageCalls").toLowerCase()}.{" "}
            {t("aiBillingNote")}
          </p>
        </SettingsSection>
      </div>

      <Dialog open={consentOpen} onOpenChange={setConsentOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiConsentTitle")}</DialogTitle>
            <DialogDescription>{t("aiConsentBody")}</DialogDescription>
          </DialogHeader>
          <div className="flex items-start gap-2">
            <Checkbox
              id={consentId}
              className="mt-1"
              checked={consent}
              onCheckedChange={(value) => setConsent(value === true)}
            />
            <Label htmlFor={consentId} className="font-normal">
              {t("aiConsentCheckbox")}
            </Label>
          </div>
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
