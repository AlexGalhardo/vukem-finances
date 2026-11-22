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

// QA-003: a bank file in Windows-1252 (or with a BOM) is read as text, never
// kept as invalid UTF-8.
func TestImportCSVParserDecodesWindows1252AndDropsTheBOM(t *testing.T) {
	t.Parallel()

	m := Mapping{Date: "Date", Description: "Description", Amount: "Amount", DateFormat: "2006-01-02"}

	latin := []byte("Date,Description,Amount\n2026-03-01,Caf")
	latin = append(latin, 0xE9, ' ', 0x93)
	latin = append(latin, "bom"...)
	latin = append(latin, 0x94)
	latin = append(latin, ",-3.50\n"...)

	rows, err := CSVParser{Mapping: m}.Parse(latin)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows = %+v, err = %v", rows, err)
	}

	if want := "Caf\u00e9 \u201cbom\u201d"; rows[0].Description != want {
		t.Errorf("description = %q, want %q", rows[0].Description, want)
	}

	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Date,Description,Amount\n2026-03-01,Coffee,-3.50\n")...)

	rows, err = CSVParser{Mapping: m}.Parse(withBOM)
	if err != nil || len(rows) != 1 || rows[0].Problem != "" || rows[0].Description != "Coffee" {
		t.Errorf("a file with a BOM: rows = %+v, err = %v", rows, err)
	}
}
