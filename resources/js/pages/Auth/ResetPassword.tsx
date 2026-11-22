import { Head, useForm } from "@inertiajs/react";
import type { FormEvent } from "react";
import { AuthErrorSummary, AuthHeading, AuthSection } from "../../components/auth/AuthParts";
import Field from "../../components/Field";
import PasswordField from "../../components/PasswordField";
import { Button } from "../../components/ui/button";
import { useFocusFirstError } from "../../hooks/useFocusFirstError";
import { useFinanceI18n } from "../../lib/i18n";
import { checkConfirmation, checkPassword, passwordAdvice } from "../../lib/password-policy";
import GuideSummary from "../../tutorial/GuideSummary";

export default function ResetPassword({ token, email }: { token: string; email: string }) {
  const { t } = useFinanceI18n();
  const form = useForm({ token, email, password: "", password_confirmation: "" });
  const formRef = useFocusFirstError(form.errors);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post("/reset-password", { onFinish: () => form.reset("password", "password_confirmation") });
  };

  return (
    <>
      <Head title={t("authResetTitle")} />
      <AuthHeading title={t("authResetTitle")} description={t("authResetDescription")} />
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} data-tour="reset-form" className="flex flex-col gap-6">
          <div className="flex flex-col gap-4">
            <Field
              label={t("email")}
              type="email"
              name="email"
              autoComplete="email"
              inputSize="lg"
              required
              value={form.data.email}
              onChange={(e) => form.setData("email", e.target.value)}
              error={form.errors.email}
            />
            <PasswordField
              label={t("authNewPassword")}
              name="password"
              autoComplete="new-password"
              inputSize="lg"
              value={form.data.password}
              onChange={(value) => form.setData("password", value)}
              error={form.errors.password}
              checks={checkPassword(form.data.password)}
              advice={passwordAdvice(form.data.password)}
            />
            <PasswordField
              label={t("confirmPassword")}
              name="password_confirmation"
              autoComplete="new-password"
              inputSize="lg"
              value={form.data.password_confirmation}
              onChange={(value) => form.setData("password_confirmation", value)}
              error={form.errors.password_confirmation}
              checks={checkConfirmation(form.data.password, form.data.password_confirmation)}
            />
          </div>
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("authResetSubmit")}
          </Button>
        </form>
      </AuthSection>
      <GuideSummary screenId="ResetPassword" />
    </>
  );
}
