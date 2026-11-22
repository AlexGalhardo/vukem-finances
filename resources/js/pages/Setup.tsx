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
import type { SetupProps } from "../page-props";
import GuideSummary from "../tutorial/GuideSummary";

export default function Setup({ currency, codeRequired }: SetupProps) {
  const { t } = useFinanceI18n();
  const form = useForm({ name: "", email: "", password: "", password_confirmation: "", currency, code: "" });
  const formRef = useFocusFirstError(form.errors);

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post("/setup", { onFinish: () => form.reset("password", "password_confirmation") });
  }

  return (
    <>
      <Head title={t("setupTitle")} />
      <AuthHeading title={t("setupTitle")} description={t("setupDescription")} />
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} className="flex flex-col gap-6">
          <div className="flex flex-col gap-4">
            {codeRequired && (
              <div data-tour="setup-code" className="flex flex-col gap-2">
                <Field
                  label={t("setupCode")}
                  type="password"
                  name="code"
                  autoComplete="off"
                  inputSize="lg"
                  required
                  value={form.data.code}
                  onChange={(e) => form.setData("code", e.target.value)}
                  error={form.errors.code}
                />
                <p className="text-caption text-muted-foreground">{t("setupCodeHelp")}</p>
              </div>
            )}
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
            <div data-tour="setup-currency">
              <Field
                label={t("setupCurrency")}
                type="text"
                name="currency"
                maxLength={3}
                inputSize="lg"
                required
                value={form.data.currency}
                onChange={(e) => form.setData("currency", e.target.value.toUpperCase())}
                error={form.errors.currency}
              />
            </div>
          </div>
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("setupSubmit")}
            <ArrowUpRight data-icon="inline-end" />
          </Button>
        </form>
      </AuthSection>
      <GuideSummary screenId="Setup" />
    </>
  );
}
