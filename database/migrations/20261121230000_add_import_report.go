package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Imports gain the format of the file, how many rows were skipped as
// duplicates, and the report: a JSON list (at most 100) of the rows that
// failed, each with its line and reason.
func m20261121230000AddImportReport() schema.Migration {
	return schema.Migration{
		ID: "20261121230000_add_import_report",
		Up: func(s *schema.Schema) error {
			if has, err := s.HasColumn("imports", "format"); err != nil || has {
				return err
			}

			return s.Table("imports", func(t *schema.Table) {
				t.String("format", 10).Default("csv")
				t.Integer("duplicates").Default(0)
				t.Text("report").Default("")
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("imports", func(t *schema.Table) {
				t.DropColumn("format")
				t.DropColumn("duplicates")
				t.DropColumn("report")
			})
		},
	}
}
