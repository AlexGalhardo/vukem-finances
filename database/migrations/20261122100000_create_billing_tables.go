package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 13: revocation of Stripe access, PIX payments and entitlements, and the
// billing audit trail. None of these tables has a uid column on purpose: they
// are server-only data, never synced to a device. public_ref is the id used in
// URLs and logs. Payment records follow decision D-022 (5 years, no legal
// review).
func m20261122100000CreateBillingTables() schema.Migration {
	return schema.Migration{
		ID: "20261122100000_create_billing_tables",
		Up: func(s *schema.Schema) error {
			steps := []func() error{
				func() error {
					return s.Table("workspace_subscriptions", func(t *schema.Table) {
						t.Timestamp("access_revoked_at").Nullable()
						t.String("revoked_reason", 20).Default("")
						t.String("revoked_ref", 100).Default("")
					})
				},
				func() error {
					return s.Create("pix_payments", func(t *schema.Table) {
						t.ID()
						t.String("public_ref", 26).Unique()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.String("provider_ref", 100).Unique()
						t.String("status", 20)
						t.BigInteger("amount_cents")
						t.String("currency", 3).Default("BRL")
						t.Integer("months")
						t.Text("br_code")
						t.Text("qr_png_base64")
						t.Timestamp("expires_at")
						t.Timestamp("paid_at").Nullable()
						t.BigInteger("created_by")
						t.Timestamps()
						t.Index("workspace_id", "status")
					})
				},
				func() error {
					return s.Create("pix_entitlements", func(t *schema.Table) {
						t.ID()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.Timestamp("anchor_at")
						t.Integer("paid_months").Default(0)
						t.Timestamp("valid_until")
						t.Timestamp("revoked_at").Nullable()
						t.String("revoked_reason", 20).Default("")
						t.Timestamps()
						t.UniqueIndex("workspace_id")
					})
				},
				func() error {
					return s.Create("billing_events", func(t *schema.Table) {
						t.ID()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.String("source", 10)
						t.String("kind", 40)
						t.String("ref", 100).Default("")
						t.String("summary", 200).Default("")
						t.Timestamps()
						t.Index("workspace_id", "id")
					})
				},
			}

			for _, step := range steps {
				if err := step(); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"billing_events", "pix_entitlements", "pix_payments"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return s.Table("workspace_subscriptions", func(t *schema.Table) {
				t.DropColumn("access_revoked_at")
				t.DropColumn("revoked_reason")
				t.DropColumn("revoked_ref")
			})
		},
	}
}
