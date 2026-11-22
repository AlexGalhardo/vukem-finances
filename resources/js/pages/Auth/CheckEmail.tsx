import { Head, Link } from "@inertiajs/react";
import { AuthHeading, authLinkClass } from "../../components/auth/AuthParts";
import { useFinanceI18n } from "../../lib/i18n";
import GuideSummary from "../../tutorial/GuideSummary";

export default function CheckEmail() {
  const { t } = useFinanceI18n();
  return (
    <>
      <Head title={t("authCheckEmailTitle")} />
      <AuthHeading title={t("authCheckEmailTitle")} />
      <p data-tour="check-email-message" className="text-body text-pretty text-muted-foreground">
        {t("authCheckEmailBody")}
      </p>
      <Link href="/login" className={authLinkClass}>
        {t("authGoToLogin")}
      </Link>
      <GuideSummary screenId="CheckEmail" />
    </>
  );
}
