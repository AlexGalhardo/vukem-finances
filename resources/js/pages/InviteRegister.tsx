import { Head, useForm } from "@inertiajs/react";
import { ArrowUpRight } from "lucide-react";
import type { FormEvent } from "react";
import { AuthErrorSummary, AuthHeading, AuthSection } from "../components/auth/AuthParts";
import Field from "../components/Field";
import PasswordField from "../components/PasswordField";
import { Button } from "../components/ui/button";
import { useFocusFirstError } from "../hooks/useFocusFirstError";
import { useFinanceI18n } from "../lib/i18n";
import { checkConfirmation, checkPassword, passwordAdvice } from "../lib/password-policy";
import type { InviteRegisterProps } from "../page-props";
import GuideSummary from "../tutorial/GuideSummary";

export default function InviteRegister({ token, email }: InviteRegisterProps) {
  const { t } = useFinanceI18n();
  const form = useForm({ name: "", password: "", password_confirmation: "" });
  const formRef = useFocusFirstError(form.errors);

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post(`/invites/${token}/register`, { onFinish: () => form.reset("password", "password_confirmation") });
  }

  return (
    <>
      <Head title={t("inviteRegisterTitle")} />
      <AuthHeading title={t("inviteRegisterTitle")} description={t("inviteRegisterDescription")} />
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} data-tour="invite-register-form" className="flex flex-col gap-6">
          <div className="flex flex-col gap-4">
            <Field label={t("email")} type="email" name="email" inputSize="lg" readOnly value={email} />
            <Field
              label={t("name")}
              type="text"
              name="name"
              autoComplete="name"
              inputSize="lg"
              required
              value={form.data.name}
              onChange={(e) => form.setData("name", e.target.value)}
              error={form.errors.name}
            />
            <PasswordField
              label={t("password")}
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
            {t("createAccount")}
            <ArrowUpRight data-icon="inline-end" />
          </Button>
        </form>
      </AuthSection>
      <GuideSummary screenId="InviteRegister" />
    </>
  );
}
