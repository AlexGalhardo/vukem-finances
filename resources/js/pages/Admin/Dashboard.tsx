import { Link } from "@inertiajs/react";
import { useEffect, useState } from "react";
import Stat from "../../components/Stat";
import { Alert, AlertDescription } from "../../components/ui/alert";
import AdminLayout from "../../layouts/AdminLayout";
import { type DashboardData, getJSON } from "../../lib/adminApi";
import { useFinanceI18n } from "../../lib/i18n";

export default function Dashboard({ dataUrl }: { dataUrl: string }) {
  const { t } = useFinanceI18n();
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const ctl = new AbortController();
    getJSON<DashboardData>(dataUrl, ctl.signal)
      .then(setData)
      .catch((e: Error) => e.name !== "AbortError" && setError(e.message));
    return () => ctl.abort();
  }, [dataUrl]);

  return (
    <AdminLayout title={t("admDashboard")}>
      {error && (
        <Alert variant="error">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {!data && !error && (
        <p role="status" className="text-muted-foreground">
          {t("admLoading")}
        </p>
      )}
      <div data-tour="admin-overview" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {(data?.cards ?? []).map((card) => (
          <Link
            key={card.url}
            href={`/admin/${card.url.replace(`${dataUrl}/`, "")}`}
            className="rounded-lg border border-border p-5 transition-colors duration-(--vf-duration-fast) hover:bg-muted"
          >
            <Stat label={card.label} value={card.count} size="hero" />
          </Link>
        ))}
      </div>
    </AdminLayout>
  );
}
