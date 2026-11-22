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
import { Checkbox } from "../../components/ui/checkbox";
import { FieldSeparator } from "../../components/ui/field";
import { Label } from "../../components/ui/label";
import { useFocusFirstError } from "../../hooks/useFocusFirstError";
import { useFinanceI18n } from "../../lib/i18n";
import GuideSummary from "../../tutorial/GuideSummary";

export default function Login() {
  const { t } = useFinanceI18n();
  const { googleEnabled: google, edition } = usePage().props as { googleEnabled?: boolean; edition?: string };
  const googleEnabled = Boolean(google);
  const form = useForm({ email: "", password: "", remember: false });
  const formRef = useFocusFirstError(form.errors);

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post("/login", { onFinish: () => form.reset("password") });
  }

  return (
    <>
      <Head title={t("login")} />
      <AuthHeading title={t("welcome")} description={t("authDescription")} />
      {googleEnabled && (
        <>
          <GoogleButton label={t("google")} />
          {edition === "personal" && <p className="text-caption text-muted-foreground">{t("googleNeedsInternet")}</p>}
          <FieldSeparator>{t("orEmail")}</FieldSeparator>
        </>
      )}
      <AuthSection>
        <AuthErrorSummary errors={form.errors} />
        <form ref={formRef} onSubmit={submit} data-tour="login-form" className="flex flex-col gap-6">
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
              label={t("password")}
              name="password"
              autoComplete="current-password"
              inputSize="lg"
              value={form.data.password}
              onChange={(value) => form.setData("password", value)}
              error={form.errors.password}
            />
            <div className="flex items-center gap-2">
              <Checkbox
                id="auth-remember"
                checked={form.data.remember}
                onCheckedChange={(checked) => form.setData("remember", checked === true)}
              />
              <Label htmlFor="auth-remember" className="font-normal">
                {t("remember")}
              </Label>
            </div>
          </div>
          <Button type="submit" variant="brand" size="lg" loading={form.processing}>
            {t("login")}
            <ArrowUpRight data-icon="inline-end" />
          </Button>
        </form>
      </AuthSection>
      <div className="flex flex-wrap justify-between gap-x-4 gap-y-2 text-body">
        {edition !== "personal" && (
          <Link href="/register" className={authLinkClass}>
            {t("register")}
          </Link>
        )}
        <Link href="/forgot-password" data-tour="login-forgot" className={authLinkClass}>
          {t("forgot")}
        </Link>
      </div>
      <GuideSummary screenId="Login" />
    </>
  );
}
