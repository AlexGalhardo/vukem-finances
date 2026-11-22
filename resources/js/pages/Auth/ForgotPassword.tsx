import { Head, Link, useForm } from "@inertiajs/react";
import type { FormEvent } from "react";
import { AuthErrorSummary, AuthHeading, AuthSection, authLinkClass } from "../../components/auth/AuthParts";
import Field from "../../components/Field";
import { Button } from "../../components/ui/button";
import { useFocusFirstError } from "../../hooks/useFocusFirstError";
import { useFinanceI18n } from "../../lib/i18n";
import GuideSummary from "../../tutorial/GuideSummary";

export default function ForgotPassword() {
  const { t } = useFinanceI18n();
  const form = useForm({ email: "" });
  const formRef = useFocusFirstError(form.errors);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post("/forgot-password");
  };

  return (
    <>
      <Head title={t("authForgotTitle")} />
      <AuthHeading title={t("authForgotTitle")} description={t("authForgotDescription")} />
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} data-tour="forgot-form" className="flex flex-col gap-6">
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
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("authForgotSubmit")}
          </Button>
        </form>
      </AuthSection>
      <Link href="/login" className={authLinkClass}>
        {t("authBackToLogin")}
      </Link>
      <GuideSummary screenId="ForgotPassword" />
    </>
  );
}
