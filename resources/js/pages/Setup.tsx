import { Head, useForm } from "@inertiajs/react";
import { ArrowUpRight } from "lucide-react";
import type { FormEvent } from "react";
import PasswordField from "../components/PasswordField";
import { Button } from "../components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { useFinanceI18n } from "../lib/i18n";
import { checkConfirmation, checkPassword, passwordAdvice } from "../lib/password-policy";
import type { SetupProps } from "../page-props";

export default function Setup({ currency, codeRequired }: SetupProps) {
  const { t } = useFinanceI18n();
  const form = useForm({ name: "", email: "", password: "", password_confirmation: "", currency, code: "" });

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post("/setup", { onFinish: () => form.reset("password", "password_confirmation") });
  }

  return (
    <>
      <Head title={t("setupTitle")} />
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-semibold tracking-tight">{t("setupTitle")}</h1>
        <p className="text-muted-foreground">{t("setupDescription")}</p>
      </div>
      <form onSubmit={submit} className="flex flex-col gap-6">
        <FieldGroup>
          {codeRequired && (
            <Field data-invalid={!!form.errors.code}>
              <FieldLabel htmlFor="setup-code">{t("setupCode")}</FieldLabel>
              <Input
                id="setup-code"
                type="password"
                autoComplete="off"
                required
                value={form.data.code}
                aria-invalid={!!form.errors.code}
                onChange={(e) => form.setData("code", e.target.value)}
              />
              <p className="text-sm text-muted-foreground">{t("setupCodeHelp")}</p>
              <FieldError>{form.errors.code}</FieldError>
            </Field>
          )}
          <Field data-invalid={!!form.errors.name}>
            <FieldLabel htmlFor="setup-name">{t("name")}</FieldLabel>
            <Input
              id="setup-name"
              type="text"
              autoComplete="name"
              required
              value={form.data.name}
              aria-invalid={!!form.errors.name}
              onChange={(e) => form.setData("name", e.target.value)}
            />
            <FieldError>{form.errors.name}</FieldError>
          </Field>
          <Field data-invalid={!!form.errors.email}>
            <FieldLabel htmlFor="setup-email">{t("email")}</FieldLabel>
            <Input
              id="setup-email"
              type="email"
              autoComplete="email"
              required
              value={form.data.email}
              aria-invalid={!!form.errors.email}
              onChange={(e) => form.setData("email", e.target.value)}
            />
            <FieldError>{form.errors.email}</FieldError>
          </Field>
          <PasswordField
            label={t("password")}
            name="password"
            autoComplete="new-password"
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
            value={form.data.password_confirmation}
            onChange={(value) => form.setData("password_confirmation", value)}
            error={form.errors.password_confirmation}
            checks={checkConfirmation(form.data.password, form.data.password_confirmation)}
          />
          <Field data-invalid={!!form.errors.currency}>
            <FieldLabel htmlFor="setup-currency">{t("setupCurrency")}</FieldLabel>
            <Input
              id="setup-currency"
              type="text"
              maxLength={3}
              required
              value={form.data.currency}
              aria-invalid={!!form.errors.currency}
              onChange={(e) => form.setData("currency", e.target.value.toUpperCase())}
            />
            <FieldError>{form.errors.currency}</FieldError>
          </Field>
        </FieldGroup>
        <Button type="submit" disabled={form.processing}>
          {t("setupSubmit")}
          <ArrowUpRight data-icon="inline-end" />
        </Button>
      </form>
    </>
  );
}
