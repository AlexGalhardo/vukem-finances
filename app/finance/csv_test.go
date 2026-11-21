package finance

import "testing"

var _ StatementParser = CSVParser{}

func TestImportCSVParserReadsRowsThroughTheMapping(t *testing.T) {
	t.Parallel()

	p := CSVParser{Mapping: Mapping{
		Date: "Data", Description: "Historico", Amount: "Valor", Category: "Cat", DateFormat: "02/01/2006", Invert: true,
	}}

	rows, err := p.Parse([]byte("Data;Historico;Valor;Cat\n05/03/2026;Aluguel;1.234,56;Housing\nbad;x;1,00;\n06/03/2026;Zero;0,00;\n07/03/2026;Short\n"))
	if err != nil {
		t.Fatal(err)
	}

	want := []StatementRow{
		{Line: 2, Date: "2026-03-05", Description: "Aluguel", Cents: -123456, Category: "Housing"},
		{Line: 3, Problem: "bad date"},
		{Line: 4, Problem: "bad amount"},
		{Line: 5, Problem: "too few columns"},
	}

	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d: %+v", len(rows), len(want), rows)
	}

	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}

	if _, err := (CSVParser{Mapping: Mapping{Date: "Nope", Description: "Historico", Amount: "Valor"}}).Parse([]byte("Data;Historico;Valor\n1;2;3\n")); err == nil {
		t.Error("a mapping to a missing column must fail")
	}
}
