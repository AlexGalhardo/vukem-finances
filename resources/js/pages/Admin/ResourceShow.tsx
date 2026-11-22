import { Link } from "@inertiajs/react";
import { ArrowLeft } from "lucide-react";
import { useEffect, useState } from "react";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { buttonVariants } from "../../components/ui/button";
import AdminLayout from "../../layouts/AdminLayout";
import { getJSON, type RecordData } from "../../lib/adminApi";
import { useFinanceI18n } from "../../lib/i18n";

export default function ResourceShow({ slug, id, dataUrl }: { slug: string; id: string; dataUrl: string }) {
  const { t } = useFinanceI18n();
  const [data, setData] = useState<RecordData | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const ctl = new AbortController();
    getJSON<RecordData>(`${dataUrl}/${encodeURIComponent(slug)}/${encodeURIComponent(id)}`, ctl.signal)
      .then(setData)
      .catch((e: Error) => e.name !== "AbortError" && setError(e.message));
    return () => ctl.abort();
  }, [dataUrl, slug, id]);

  return (
    <AdminLayout title={data ? `${data.resource.singular} ${data.id}` : id}>
      <Link
        href={`/admin/${encodeURIComponent(slug)}`}
        className={`${buttonVariants({ variant: "ghost", size: "sm" })} self-start`}
      >
        <ArrowLeft aria-hidden="true" />
        {t("admBack")}
      </Link>
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
      {data && (
        <dl
          data-tour="admin-show-fields"
          className="grid max-w-3xl gap-x-6 gap-y-3 text-body sm:grid-cols-[12rem_minmax(0,1fr)]"
        >
          {(data.fields ?? []).map((f) => (
            <div key={f.label} className="contents">
              <dt className="text-muted-foreground">{f.label}</dt>
              <dd className="break-words">{f.value}</dd>
            </div>
          ))}
        </dl>
      )}
    </AdminLayout>
  );
}
