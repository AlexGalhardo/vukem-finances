import { Head, Link, useForm, usePage } from "@inertiajs/react";
import { ArrowUpRight } from "lucide-react";
import type { FormEvent } from "react";
import {
  AuthErrorSummary,
  AuthHeading,
  AuthSection,
  authLinkClass,
  GoogleButton,
} from "../../components/auth/AuthParts";
import Field from "../../components/Field";
import PasswordField from "../../components/PasswordField";
import { Button } from "../../components/ui/button";
import { FieldSeparator } from "../../components/ui/field";
import { useFocusFirstError } from "../../hooks/useFocusFirstError";
import { useFinanceI18n } from "../../lib/i18n";
import { checkConfirmation, checkPassword, passwordAdvice } from "../../lib/password-policy";
import GuideSummary from "../../tutorial/GuideSummary";

export default function Register() {
  const { t } = useFinanceI18n();
  const googleEnabled = Boolean((usePage().props as { googleEnabled?: boolean }).googleEnabled);
  const form = useForm({ name: "", email: "", password: "", password_confirmation: "" });
  const formRef = useFocusFirstError(form.errors);

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post("/register", { onFinish: () => form.reset("password", "password_confirmation") });
  }

  return (
    <>
      <Head title={t("register")} />
      <AuthHeading title={t("createTitle")} description={t("authDescription")} />
      {googleEnabled && (
        <>
          <GoogleButton label={t("google")} />
          <FieldSeparator>{t("orEmail")}</FieldSeparator>
        </>
      )}
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} data-tour="register-form" className="flex flex-col gap-6">
          <div className="flex flex-col gap-4">
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
          </div>
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("createAccount")}
            <ArrowUpRight data-icon="inline-end" />
          </Button>
        </form>
      </AuthSection>
      <div className="flex flex-wrap justify-between gap-4 text-body">
        <Link href="/login" className={authLinkClass}>
          {t("alreadyRegistered")}
        </Link>
      </div>
      <GuideSummary screenId="Register" />
    </>
  );
}
