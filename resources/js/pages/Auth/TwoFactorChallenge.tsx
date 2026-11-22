import { Head, useForm } from "@inertiajs/react";
import { type FormEvent, useState } from "react";
import { AuthErrorSummary, AuthHeading, AuthSection, authLinkClass } from "../../components/auth/AuthParts";
import Field from "../../components/Field";
import { Button } from "../../components/ui/button";
import { useFocusFirstError } from "../../hooks/useFocusFirstError";
import { useFinanceI18n } from "../../lib/i18n";
import GuideSummary from "../../tutorial/GuideSummary";

export default function TwoFactorChallenge() {
  const { t } = useFinanceI18n();
  const [recovery, setRecovery] = useState(false);
  const form = useForm({ code: "", recovery_code: "" });
  const formRef = useFocusFirstError(form.errors);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post("/two-factor-challenge", { onFinish: () => form.reset() });
  };

  return (
    <>
      <Head title={t("authTwoFactorTitle")} />
      <AuthHeading
        title={t("authTwoFactorTitle")}
        description={recovery ? t("authTwoFactorRecoveryPrompt") : t("authTwoFactorAppPrompt")}
      />
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} className="flex flex-col gap-6">
          {recovery ? (
            <Field
              label={t("authTwoFactorRecoveryCode")}
              name="recovery_code"
              autoComplete="off"
              inputSize="lg"
              autoFocus
              required
              value={form.data.recovery_code}
              onChange={(e) => form.setData("recovery_code", e.target.value)}
              error={form.errors.recovery_code}
            />
          ) : (
            <div data-tour="challenge-code">
              <Field
                label={t("authTwoFactorCode")}
                name="code"
                inputMode="numeric"
                autoComplete="one-time-code"
                inputSize="lg"
                autoFocus
                required
                value={form.data.code}
                onChange={(e) => form.setData("code", e.target.value)}
                error={form.errors.code}
              />
            </div>
          )}
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("login")}
          </Button>
        </form>
      </AuthSection>
      <p>
        <button
          type="button"
          onClick={() => {
            // Only one field is sent filled: the server checks the app code first.
            setRecovery(!recovery);
            form.reset();
            form.clearErrors();
          }}
          className={`${authLinkClass} rounded-sm pointer-coarse:min-h-11`}
        >
          {recovery ? t("authTwoFactorUseApp") : t("authTwoFactorUseRecovery")}
        </button>
      </p>
      <GuideSummary screenId="TwoFactorChallenge" />
    </>
  );
}
