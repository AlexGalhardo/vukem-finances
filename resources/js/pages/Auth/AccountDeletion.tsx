import { Head, router } from "@inertiajs/react";
import PageHeader from "../../components/PageHeader";
import { Card, primary, secondary } from "../../components/ui";
import { useFinanceI18n } from "../../lib/i18n";
import type { Page } from "../../page-props";

export default function AccountDeletion({ email, purgeDate }: Page<"Auth/AccountDeletion">) {
  const { t, date } = useFinanceI18n();

  return (
    <>
      <Head title={t("accountDeletionTitle")} />
      <PageHeader title={t("accountDeletionTitle")} description={email} />
      <Card>
        <p role="status" className="mb-4">
          {t("accountDeletionBody").replace("{date}", date(purgeDate))}
        </p>
        <div className="flex flex-wrap gap-2">
          <button type="button" className={primary} onClick={() => router.post("/account/deletion/cancel")}>
            {t("accountDeletionCancel")}
          </button>
          <button type="button" className={secondary} onClick={() => router.post("/logout")}>
            {t("accountDeletionSignOut")}
          </button>
        </div>
      </Card>
    </>
  );
}
