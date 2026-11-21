import { Head, useForm } from "@inertiajs/react";
import { ArrowUpRight } from "lucide-react";
import type { FormEvent } from "react";
import PasswordField from "../components/PasswordField";
import { Button } from "../components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { useFinanceI18n } from "../lib/i18n";
import { checkConfirmation, checkPassword, passwordAdvice } from "../lib/password-policy";
import type { InviteRegisterProps } from "../page-props";

export default function InviteRegister({ token, email }: InviteRegisterProps) {
  const { t } = useFinanceI18n();
  const form = useForm({ name: "", password: "", password_confirmation: "" });

  function submit(e: FormEvent) {
    e.preventDefault();
    form.post(`/invites/${token}/register`, { onFinish: () => form.reset("password", "password_confirmation") });
  }

  return (
    <>
      <Head title={t("inviteRegisterTitle")} />
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-semibold tracking-tight">{t("inviteRegisterTitle")}</h1>
        <p className="text-muted-foreground">{t("inviteRegisterDescription")}</p>
      </div>
      <form onSubmit={submit} className="flex flex-col gap-6">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="invite-email">{t("email")}</FieldLabel>
            <Input id="invite-email" type="email" readOnly value={email} />
          </Field>
          <Field data-invalid={!!form.errors.name}>
            <FieldLabel htmlFor="invite-name">{t("name")}</FieldLabel>
            <Input
              id="invite-name"
              type="text"
              autoComplete="name"
              required
              value={form.data.name}
              aria-invalid={!!form.errors.name}
              onChange={(e) => form.setData("name", e.target.value)}
            />
            <FieldError>{form.errors.name}</FieldError>
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
        </FieldGroup>
        <Button type="submit" disabled={form.processing}>
          {t("createAccount")}
          <ArrowUpRight data-icon="inline-end" />
        </Button>
      </form>
    </>
  );
}
