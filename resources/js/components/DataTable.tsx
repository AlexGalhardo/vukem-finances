import { ArrowDown, ArrowUp, ChevronsUpDown } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../lib/utils";
import { Skeleton } from "./ui/skeleton";
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

export type SortDirection = "ascending" | "descending";

export interface DataTableColumn<Row> {
  id: string;
  header: ReactNode;
  cell: (row: Row) => ReactNode;
  sortable?: boolean;
  align?: "start" | "end";
}

export interface DataTableProps<Row> {
  columns: readonly DataTableColumn<Row>[];
  rows: readonly Row[];
  rowKey: (row: Row) => string | number;
  /** Names the table for screen readers; visually hidden. */
  caption: string;
  sort?: { column: string; direction: SortDirection };
  onSort?: (column: string) => void;
  loading?: boolean;
  loadingLabel?: string;
  /** Shown in place of the rows when there are none. */
  empty?: ReactNode;
  /** 36 px rows instead of 44. */
  compact?: boolean;
  /** Rows for which this returns true get the selected style and aria-selected. */
  isSelected?: (row: Row) => boolean;
}

export function ariaSortFor(
  column: { id: string; sortable?: boolean },
  sort: DataTableProps<unknown>["sort"],
): SortDirection | "none" | undefined {
  if (!column.sortable) return undefined;
  return sort?.column === column.id ? sort.direction : "none";
}

/** A real table: semantic headers, `aria-sort` on sortable columns, and explicit loading and empty states. */
export default function DataTable<Row>({
  columns,
  rows,
  rowKey,
  caption,
  sort,
  onSort,
  loading = false,
  loadingLabel = "Loading",
  empty,
  compact = false,
  isSelected,
}: DataTableProps<Row>) {
  // Below md a table of more than three columns becomes a list: the first column and the last amount share the
  // first line, the others follow. The DOM stays one table, so there is no duplicated text.
  const listed = columns.length > 3;
  const lastEnd = columns.map((column) => column.align).lastIndexOf("end");
  const cellLayout = (index: number): string => {
    if (!listed) return "";
    if (index === 0) return "max-md:col-start-1 max-md:row-start-1 max-md:font-medium";
    if (index === lastEnd) return "max-md:col-start-2 max-md:row-start-1";
    return "max-md:col-span-2 max-md:py-0 max-md:text-caption max-md:text-muted-foreground";
  };
  const rowClass = cn(
    compact && "h-9",
    listed &&
      "max-md:grid max-md:h-auto max-md:grid-cols-[1fr_auto] max-md:items-baseline max-md:gap-x-3 max-md:px-0 max-md:py-2",
  );
  const alignment = (align: DataTableColumn<Row>["align"]): string => (align === "end" ? "text-right" : "text-left");
  return (
    <Table aria-busy={loading || undefined} className={listed ? "max-md:block" : undefined}>
      <TableCaption className="sr-only">{caption}</TableCaption>
      <TableHeader className={listed ? "max-md:sr-only" : undefined}>
        <TableRow>
          {columns.map((column) => {
            const ariaSort = ariaSortFor(column, sort);
            const Icon = ariaSort === "ascending" ? ArrowUp : ariaSort === "descending" ? ArrowDown : ChevronsUpDown;
            return (
              <TableHead
                key={column.id}
                scope="col"
                aria-sort={ariaSort}
                className={cn("sticky top-0 z-(--vf-z-header) bg-background", alignment(column.align))}
              >
                {column.sortable ? (
                  <button
                    type="button"
                    onClick={() => onSort?.(column.id)}
                    className={cn(
                      "inline-flex items-center gap-1 rounded-md py-1 text-overline hover:text-foreground pointer-coarse:min-h-11",
                      column.align === "end" && "flex-row-reverse",
                    )}
                  >
                    {column.header}
                    <Icon aria-hidden="true" className="size-3.5" />
                  </button>
                ) : (
                  column.header
                )}
              </TableHead>
            );
          })}
        </TableRow>
      </TableHeader>
      <TableBody className={listed ? "max-md:block" : undefined}>
        {loading ? (
          [0, 1, 2].map((n) => (
            <TableRow key={n} className={rowClass}>
              {columns.map((column, index) => (
                <TableCell key={column.id} className={cn(cellLayout(index), listed && "max-md:whitespace-normal")}>
                  <Skeleton className="h-4 w-full" />
                </TableCell>
              ))}
            </TableRow>
          ))
        ) : rows.length === 0 ? (
          <TableRow>
            <TableCell colSpan={columns.length}>{empty}</TableCell>
          </TableRow>
        ) : (
          rows.map((row) => {
            const selected = isSelected?.(row) ?? false;
            return (
              <TableRow
                key={rowKey(row)}
                className={rowClass}
                aria-selected={isSelected ? selected : undefined}
                data-state={selected ? "selected" : undefined}
              >
                {columns.map((column, index) => (
                  <TableCell
                    key={column.id}
                    className={cn(alignment(column.align), listed && "max-md:whitespace-normal", cellLayout(index))}
                  >
                    {column.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            );
          })
        )}
      </TableBody>
      {loading && (
        <tfoot className="sr-only">
          <tr>
            <td role="status">{loadingLabel}</td>
          </tr>
        </tfoot>
      )}
    </Table>
  );
}
