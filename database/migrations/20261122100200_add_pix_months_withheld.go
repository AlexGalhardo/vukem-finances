package migrations

import "github.com/AlexGalhardo/vukem/schema"

// months_withheld is what an open dispute actually took from the entitlement.
// Winning the dispute gives back exactly that, so disputing a payment whose
// months were already used cannot mint free access.
func m20261122100200AddPixMonthsWithheld() schema.Migration {
	return schema.Migration{
		ID: "20261122100200_add_pix_months_withheld",
		Up: func(s *schema.Schema) error {
			return s.Table("pix_payments", func(t *schema.Table) {
				t.Integer("months_withheld").Default(0)
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("pix_payments", func(t *schema.Table) { t.DropColumn("months_withheld") })
		},
	}
}
