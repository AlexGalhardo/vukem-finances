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
}: DataTableProps<Row>) {
  const alignment = (align: DataTableColumn<Row>["align"]): string => (align === "end" ? "text-right" : "text-left");
  return (
    <Table aria-busy={loading || undefined}>
      <TableCaption className="sr-only">{caption}</TableCaption>
      <TableHeader>
        <TableRow>
          {columns.map((column) => {
            const ariaSort = ariaSortFor(column, sort);
            const Icon = ariaSort === "ascending" ? ArrowUp : ariaSort === "descending" ? ArrowDown : ChevronsUpDown;
            return (
              <TableHead key={column.id} scope="col" aria-sort={ariaSort} className={alignment(column.align)}>
                {column.sortable ? (
                  <button
                    type="button"
                    onClick={() => onSort?.(column.id)}
                    className={cn(
                      "inline-flex items-center gap-1 rounded-md py-1 font-medium hover:text-foreground pointer-coarse:min-h-11",
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
      <TableBody>
        {loading ? (
          [0, 1, 2].map((n) => (
            <TableRow key={n}>
              {columns.map((column) => (
                <TableCell key={column.id}>
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
          rows.map((row) => (
            <TableRow key={rowKey(row)}>
              {columns.map((column) => (
                <TableCell key={column.id} className={alignment(column.align)}>
                  {column.cell(row)}
                </TableCell>
              ))}
            </TableRow>
          ))
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
