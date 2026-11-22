import { Link } from "@inertiajs/react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useFinanceI18n } from "../lib/i18n";
import { buttonVariants } from "./ui/button";

export interface PaginationProps {
  page: number;
  pageCount: number;
  /** Builds the URL of a page, so paging stays a real link that works without scripts. */
  hrefFor: (page: number) => string;
}

export default function Pagination({ page, pageCount, hrefFor }: PaginationProps) {
  const { t } = useFinanceI18n();
  if (pageCount <= 1) return null;
  const linkClass = buttonVariants({ variant: "outline" });
  const disabledClass = linkClass;
  return (
    <nav aria-label={t("pagination")} className="flex items-center justify-between gap-3">
      {page > 1 ? (
        <Link href={hrefFor(page - 1)} className={linkClass} preserveScroll>
          <ChevronLeft aria-hidden="true" />
          {t("previousPage")}
        </Link>
      ) : (
        <span aria-disabled="true" className={disabledClass}>
          <ChevronLeft aria-hidden="true" />
          {t("previousPage")}
        </span>
      )}
      <span className="text-body tabular-nums text-muted-foreground">
        {t("page")} {page} {t("of")} {pageCount}
      </span>
      {page < pageCount ? (
        <Link href={hrefFor(page + 1)} className={linkClass} preserveScroll>
          {t("nextPage")}
          <ChevronRight aria-hidden="true" />
        </Link>
      ) : (
        <span aria-disabled="true" className={disabledClass}>
          {t("nextPage")}
          <ChevronRight aria-hidden="true" />
        </span>
      )}
    </nav>
  );
}
