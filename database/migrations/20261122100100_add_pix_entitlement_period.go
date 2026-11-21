package migrations

import "github.com/AlexGalhardo/vukem/schema"

// A PIX entitlement starts a new period when access begins again from a new
// anchor. Each payment records the period its months count in, so refunding a
// payment of an ended period takes nothing from the one the customer paid for
// afterwards.
func m20261122100100AddPixEntitlementPeriod() schema.Migration {
	return schema.Migration{
		ID: "20261122100100_add_pix_entitlement_period",
		Up: func(s *schema.Schema) error {
			if err := s.Table("pix_entitlements", func(t *schema.Table) {
				t.Integer("period").Default(1)
			}); err != nil {
				return err
			}

			return s.Table("pix_payments", func(t *schema.Table) {
				t.Integer("entitlement_period").Default(1)
			})
		},
		Down: func(s *schema.Schema) error {
			if err := s.Table("pix_payments", func(t *schema.Table) { t.DropColumn("entitlement_period") }); err != nil {
				return err
			}

			return s.Table("pix_entitlements", func(t *schema.Table) { t.DropColumn("period") })
		},
	}
}
