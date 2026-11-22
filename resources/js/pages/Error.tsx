import { Head, Link } from "@inertiajs/react";
import { Button } from "../components/ui/button";
import { useFinanceI18n } from "../lib/i18n";
import GuideSummary from "../tutorial/GuideSummary";

// Rendered by the server for errors on Inertia visits (404, 403, 500).
// message is safe to show: 5xx only ever carry the status text.
export default function ErrorPage({ status, message }: { status: number; message: string }) {
  const { t } = useFinanceI18n();
  return (
    <>
      <Head title={`${status} ${message}`} />
      <section className="flex flex-col items-start gap-4 py-8">
        <p className="text-label text-brand-text tabular-nums">{status}</p>
        <h1 className="text-h1-auth text-balance">{message}</h1>
        <Button asChild variant="outline" size="lg" data-tour="error-back">
          <Link href="/">{t("authErrorBack")}</Link>
        </Button>
      </section>
      <GuideSummary screenId="Error" />
    </>
  );
}
