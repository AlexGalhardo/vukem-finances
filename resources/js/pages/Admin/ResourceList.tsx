import { Link } from "@inertiajs/react";
import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import DataTable, { type DataTableColumn } from "../../components/DataTable";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import AdminLayout from "../../layouts/AdminLayout";
import { getJSON, type ListData } from "../../lib/adminApi";
import { useFinanceI18n } from "../../lib/i18n";
import { fill } from "../../lib/pix";

type Row = NonNullable<ListData["rows"]>[number];

export default function ResourceList({ slug, dataUrl }: { slug: string; dataUrl: string }) {
  const { t } = useFinanceI18n();
  const [data, setData] = useState<ListData | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState(() => (typeof window === "undefined" ? "" : window.location.search));

  useEffect(() => {
    const ctl = new AbortController();
    getJSON<ListData>(`${dataUrl}/${encodeURIComponent(slug)}${query}`, ctl.signal)
      .then((d) => {
        setData(d);
        setError("");
      })
      .catch((e: Error) => e.name !== "AbortError" && setError(e.message));
    return () => ctl.abort();
  }, [dataUrl, slug, query]);

  const go = (qs: URLSearchParams) => {
    const next = qs.toString() ? `?${qs.toString()}` : "";
    window.history.replaceState(null, "", `/admin/${encodeURIComponent(slug)}${next}`);
    setQuery(next);
  };
  const params = () => new URLSearchParams(query);

  const search = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const qs = params();
    const q = new FormData(e.currentTarget).get("q");
    if (q) qs.set("q", String(q));
    else qs.delete("q");
    qs.delete("page");
    go(qs);
  };
  const sortBy = (sortUrl?: string) => {
    if (sortUrl) go(new URL(sortUrl, window.location.origin).searchParams);
  };
  const page = (n: number) => {
    const qs = params();
    qs.set("page", String(n));
    go(qs);
  };

  const panelColumns = data?.columns ?? [];
  const columns: DataTableColumn<Row>[] = panelColumns.map((col, index) => ({
    id: `c${index}`,
    header: col.label,
    sortable: Boolean(col.sortUrl),
    cell: (row) =>
      index === 0 ? (
        <Link
          href={`/admin/${encodeURIComponent(slug)}/${encodeURIComponent(row.id)}`}
          className="text-brand-text underline underline-offset-4"
        >
          {row.cells[index]}
        </Link>
      ) : (
        row.cells[index]
      ),
  }));
  const sortedIndex = panelColumns.findIndex((col) => col.sorted);

  return (
    <AdminLayout title={data?.resource.plural ?? slug}>
      {data?.searchable && (
        <search className="max-w-md">
          <form onSubmit={search} className="flex gap-2">
            <Input name="q" defaultValue={data.search} aria-label={t("admSearch")} placeholder={t("admSearch")} />
            <Button type="submit" variant="outline">
              <Search aria-hidden="true" data-icon="inline-start" />
              {t("admSearch")}
            </Button>
          </form>
        </search>
      )}
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
        <div data-tour="admin-list-table" className="overflow-x-auto">
          <DataTable
            caption={data.resource.plural}
            columns={columns}
            rows={data.rows ?? []}
            rowKey={(row) => row.id}
            sort={
              sortedIndex >= 0
                ? { column: `c${sortedIndex}`, direction: data.dir === "desc" ? "descending" : "ascending" }
                : undefined
            }
            onSort={(columnId) => sortBy(panelColumns[Number(columnId.slice(1))]?.sortUrl)}
          />
        </div>
      )}
      {data && data.pager.last > 1 && (
        <nav className="flex flex-wrap items-center gap-3 text-body" aria-label={data.resource.plural}>
          <Button
            type="button"
            variant="outline"
            disabled={data.pager.page <= 1}
            onClick={() => page(data.pager.page - 1)}
          >
            <ChevronLeft aria-hidden="true" data-icon="inline-start" />
            {t("admPrevious")}
          </Button>
          <span className="text-muted-foreground tabular-nums">
            {fill(t("admPager"), { page: data.pager.page, last: data.pager.last, total: data.total })}
          </span>
          <Button
            type="button"
            variant="outline"
            disabled={data.pager.page >= data.pager.last}
            onClick={() => page(data.pager.page + 1)}
          >
            {t("admNext")}
            <ChevronRight aria-hidden="true" data-icon="inline-end" />
          </Button>
        </nav>
      )}
    </AdminLayout>
  );
}
