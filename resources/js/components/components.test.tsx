import { describe, expect, test } from "bun:test";
import { renderToString } from "react-dom/server";
import DataTable, { ariaSortFor, type DataTableColumn } from "./DataTable";
import { formatMoneyParts, MoneyText } from "./MoneyText";

describe("MoneyText", () => {
  test("turns cents into text and a spoken label", () => {
    const expense = formatMoneyParts(-1250, "USD", "en");
    expect(expense.text).toBe("-$12.50");
    expect(expense.label).toBe("minus 12.50 US dollars");
    expect(expense.negative).toBe(true);
  });

  test("always signs income when asked, and never otherwise", () => {
    expect(formatMoneyParts(1250, "USD", "en", "always").text).toBe("+$12.50");
    expect(formatMoneyParts(1250, "USD", "en", "always").label).toBe("plus 12.50 US dollars");
    expect(formatMoneyParts(1250, "USD", "en").text).toBe("$12.50");
    expect(formatMoneyParts(-1250, "USD", "en", "never").text).toBe("$12.50");
    expect(formatMoneyParts(0, "USD", "en", "always").text).toBe("$0.00");
  });

  test("uses the locale and the translated sign words", () => {
    const pt = formatMoneyParts(-1250, "BRL", "pt-BR", "auto", { minus: "menos", plus: "mais" });
    expect(pt.label.startsWith("menos 12,50")).toBe(true);
  });

  test("renders the visible text hidden from assistive tech and the label for it", () => {
    const html = renderToString(<MoneyText cents={-1250} currency="USD" tone />);
    expect(html).toContain('aria-hidden="true">-$12.50<');
    expect(html).toContain("minus 12.50 US dollars");
    expect(html).toContain("tabular-nums");
  });
});

interface Row {
  id: number;
  name: string;
}

const columns: DataTableColumn<Row>[] = [
  { id: "name", header: "Name", cell: (row) => row.name, sortable: true },
  { id: "id", header: "Id", cell: (row) => row.id, align: "end" },
];

describe("DataTable", () => {
  test("marks the sorted column with aria-sort and other sortable columns as none", () => {
    expect(ariaSortFor(columns[0], { column: "name", direction: "descending" })).toBe("descending");
    expect(ariaSortFor(columns[0], { column: "other", direction: "ascending" })).toBe("none");
    expect(ariaSortFor(columns[1], { column: "id", direction: "ascending" })).toBeUndefined();

    const html = renderToString(
      <DataTable
        columns={columns}
        rows={[{ id: 1, name: "Rent" }]}
        rowKey={(row) => row.id}
        caption="Things"
        sort={{ column: "name", direction: "ascending" }}
      />,
    );
    expect(html).toContain('aria-sort="ascending"');
    expect(html).toContain("<caption");
    expect(html).toContain("Rent");
    expect(html).not.toContain('<button type="button"><span>Id');
  });

  test("shows the empty state and the loading state", () => {
    const empty = renderToString(
      <DataTable columns={columns} rows={[]} rowKey={(row) => row.id} caption="Things" empty={<p>Nothing here</p>} />,
    );
    expect(empty).toContain("Nothing here");

    const loading = renderToString(
      <DataTable
        columns={columns}
        rows={[]}
        rowKey={(row) => row.id}
        caption="Things"
        loading
        loadingLabel="Loading"
      />,
    );
    expect(loading).toContain('aria-busy="true"');
    expect(loading).toContain('role="status"');
    expect(loading).not.toContain("Nothing here");
  });
});
