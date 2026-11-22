import { Head, Link, usePage } from "@inertiajs/react";
import { AuthHeading } from "../../components/auth/AuthParts";
import { Button } from "../../components/ui/button";
import { useFinanceI18n } from "../../lib/i18n";
import GuideSummary from "../../tutorial/GuideSummary";
import type { SharedProps } from "../../types";

export default function VerifyEmail() {
  const { auth } = usePage<SharedProps>().props;
  const { t } = useFinanceI18n();

  return (
    <>
      <Head title={t("authVerifyTitle")} />
      <AuthHeading title={t("authVerifyTitle")} />
      <p className="text-body text-pretty text-muted-foreground">
        {t("authVerifyBody").replace("{email}", auth?.user?.email ?? "")}
      </p>
      <Button asChild variant="brand" size="lg" data-tour="verify-send-again">
        <Link href="/email/verification-notification" method="post" as="button">
          {t("authVerifyResend")}
        </Link>
      </Button>
      <GuideSummary screenId="VerifyEmail" />
    </>
  );
}
