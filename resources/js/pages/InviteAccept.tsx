import { Head, useForm } from "@inertiajs/react";
import type { FormEvent } from "react";
import { AuthHeading } from "../components/auth/AuthParts";
import { Button } from "../components/ui/button";
import { useFinanceI18n } from "../lib/i18n";
import type { Page } from "../page-props";
import GuideSummary from "../tutorial/GuideSummary";

export default function InviteAccept({ token }: Page<"InviteAccept">) {
  const { t } = useFinanceI18n();
  const form = useForm({});

  const submit = (e: FormEvent) => {
    e.preventDefault();
    form.post(`/invites/${token}`);
  };

  return (
    <>
      <Head title={t("authInviteTitle")} />
      <AuthHeading title={t("authInviteTitle")} description={t("authInviteBody")} />
      <form onSubmit={submit}>
        <Button
          type="submit"
          variant="brand"
          size="lg"
          loading={form.processing}
          data-tour="invite-accept"
          className="w-full"
        >
          {t("authInviteAccept")}
        </Button>
      </form>
      <GuideSummary screenId="InviteAccept" />
    </>
  );
}
