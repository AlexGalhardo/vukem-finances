import { Head, router, useForm, usePage } from "@inertiajs/react";
import { type FormEvent, useEffect } from "react";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import PasswordField from "../components/PasswordField";
import { Card, danger, primary } from "../components/ui";
import { useFinanceI18n } from "../lib/i18n";
import type { Page, SharedProps } from "../page-props";

export default function Privacy({
  workspaceName,
  isOwner,
  personal,
  pending,
  purgeDate,
  exportReady,
  canErase,
  erased,
  graceDays,
}: Page<"Privacy">) {
  const { t, date } = useFinanceI18n();
  const user = (usePage<SharedProps>().props as SharedProps & { auth?: { user?: { email: string } } }).auth?.user;

  const exportForm = useForm({ password: "" });
  const workspaceForm = useForm({ name: "", password: "" });
  const accountForm = useForm({ confirmation: "", password: "" });
  const eraseForm = useForm({ confirmation: "", password: "" });

  // The password was confirmed: the file is a plain GET, so the browser saves it and the page stays.
  useEffect(() => {
    if (!exportReady) return;
    window.location.assign("/account/export.json");
    router.replace({ url: "/account/privacy", preserveState: true, preserveScroll: true });
  }, [exportReady]);

  const submitExport = (e: FormEvent) => {
    e.preventDefault();
    exportForm.post("/account/export", { onFinish: () => exportForm.reset("password") });
  };
  const submitWorkspace = (e: FormEvent) => {
    e.preventDefault();
    workspaceForm.post("/workspace/deletion", { onFinish: () => workspaceForm.reset("password") });
  };
  const submitErase = (e: FormEvent) => {
    e.preventDefault();
    eraseForm.post("/account/erase", { onFinish: () => eraseForm.reset("password") });
  };
  const submitAccount = (e: FormEvent) => {
    e.preventDefault();
    accountForm.post("/account/deletion", { onFinish: () => accountForm.reset("password") });
  };

  return (
    <>
      <Head title={t("privacy")} />
      <PageHeader title={t("privacy")} description={t("privacyIntro")} />

      {pending && (
        <Card title={t("privacyPendingTitle")}>
          <p role="status" className="mb-3">
            {t("privacyPendingBody").replace("{date}", date(purgeDate))}
          </p>
          {isOwner ? (
            <button
              type="button"
              className={primary}
              onClick={() => router.post("/workspace/deletion/cancel", {}, { preserveScroll: true })}
            >
              {t("privacyPendingCancel")}
            </button>
          ) : (
            <p className="text-sm text-muted-foreground">{t("privacyPendingOwnerOnly")}</p>
          )}
        </Card>
      )}

      <Card title={t("privacyExportTitle")}>
        <p className="mb-3 text-sm">{t("privacyExportBody")}</p>
        {exportReady && <p role="status">{t("privacyExportStarted")}</p>}
        <form onSubmit={submitExport} className="flex max-w-md flex-col gap-3">
          <PasswordField
            label={t("privacyPassword")}
            name="password"
            value={exportForm.data.password}
            onChange={(v) => exportForm.setData("password", v)}
            autoComplete="current-password"
            error={exportForm.errors.password}
          />
          <p className="text-sm text-muted-foreground">{t("privacyPasswordHint")}</p>
          <div>
            <button type="submit" disabled={exportForm.processing} className={primary}>
              {t("privacyExportButton")}
            </button>
          </div>
        </form>
      </Card>

      {!pending && (
        <Card title={t("privacyWorkspaceTitle")}>
          {personal ? (
            <p>{t("privacyWorkspacePersonal")}</p>
          ) : !isOwner ? (
            <p>{t("privacyWorkspaceOwnerOnly")}</p>
          ) : (
            <>
              <p className="mb-3 text-sm">{t("privacyWorkspaceBody").replace("{days}", String(graceDays))}</p>
              <form onSubmit={submitWorkspace} className="flex max-w-md flex-col gap-3" autoComplete="off">
                <Field
                  label={`${t("privacyWorkspaceName")}: ${workspaceName}`}
                  name="name"
                  value={workspaceForm.data.name}
                  onChange={(e) => workspaceForm.setData("name", e.target.value)}
                  error={workspaceForm.errors.name}
                />
                <PasswordField
                  label={t("privacyPassword")}
                  name="password"
                  value={workspaceForm.data.password}
                  onChange={(v) => workspaceForm.setData("password", v)}
                  autoComplete="current-password"
                  error={workspaceForm.errors.password}
                />
                <div>
                  <button type="submit" disabled={workspaceForm.processing} className={danger}>
                    {t("privacyWorkspaceDelete")}
                  </button>
                </div>
              </form>
            </>
          )}
        </Card>
      )}

      {erased && (
        <Card>
          <p role="status">{t("privacyErased")}</p>
        </Card>
      )}

      {canErase && !erased && (
        <Card title={t("privacyEraseTitle")}>
          <p className="mb-3 text-sm">{t("privacyEraseBody")}</p>
          <form onSubmit={submitErase} className="flex max-w-md flex-col gap-3" autoComplete="off">
            <Field
              label={t("privacyEraseConfirm")}
              name="confirmation"
              value={eraseForm.data.confirmation}
              onChange={(e) => eraseForm.setData("confirmation", e.target.value)}
              error={eraseForm.errors.confirmation}
            />
            <PasswordField
              label={t("privacyPassword")}
              name="password"
              value={eraseForm.data.password}
              onChange={(v) => eraseForm.setData("password", v)}
              autoComplete="current-password"
              error={eraseForm.errors.password}
            />
            <div>
              <button type="submit" disabled={eraseForm.processing} className={danger}>
                {t("privacyEraseButton")}
              </button>
            </div>
          </form>
        </Card>
      )}

      <Card title={t("privacyAccountTitle")}>
        <p className="mb-3 text-sm">{t("privacyAccountBody").replace("{days}", String(graceDays))}</p>
        <form onSubmit={submitAccount} className="flex max-w-md flex-col gap-3" autoComplete="off">
          <Field
            label={`${t("privacyAccountConfirm")}${user ? `: ${user.email}` : ""}`}
            name="confirmation"
            type="email"
            value={accountForm.data.confirmation}
            onChange={(e) => accountForm.setData("confirmation", e.target.value)}
            error={accountForm.errors.confirmation}
          />
          <PasswordField
            label={t("privacyPassword")}
            name="password"
            value={accountForm.data.password}
            onChange={(v) => accountForm.setData("password", v)}
            autoComplete="current-password"
            error={accountForm.errors.password}
          />
          <div>
            <button type="submit" disabled={accountForm.processing} className={danger}>
              {t("privacyAccountDelete")}
            </button>
          </div>
        </form>
      </Card>
    </>
  );
}
