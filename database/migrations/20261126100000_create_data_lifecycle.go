package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 34: deletion with a grace period. A workspace or a person moves to
// pending_deletion with a purge date; the purge job removes the rows when the
// date passes. The three new tables carry no workspace_id on purpose: they
// hold only what may outlive the data they describe.
//
//   - privacy_audit: the event, the time and a non-reversible subject reference.
//   - payment_records: what the law asks to keep after a deletion (amount, date,
//     provider reference, subject reference), until retain_until (D-022).
//   - purged_tokens: the public id of an API token of a purged workspace, so a
//     phone that syncs late is told 410 workspace_deleted and not 401.
func m20261126100000CreateDataLifecycle() schema.Migration {
	return schema.Migration{
		ID: "20261126100000_create_data_lifecycle",
		Up: func(s *schema.Schema) error {
			if err := s.Table("workspaces", func(t *schema.Table) {
				t.String("state", 20).Default("active")
				t.Timestamp("purge_at").Nullable()
				t.Timestamp("deletion_requested_at").Nullable()
				t.BigInteger("deletion_requested_by").Default(0)
				t.String("purge_error", 120).Default("")
			}); err != nil {
				return err
			}

			if err := s.Table("users", func(t *schema.Table) {
				t.Timestamp("deletion_requested_at").Nullable()
				t.Timestamp("purge_at").Nullable()
			}); err != nil {
				return err
			}

			if err := s.Create("privacy_audit", func(t *schema.Table) {
				t.ID()
				t.String("event", 60)
				t.String("subject_ref", 64)
				t.Timestamps()
				t.Index("subject_ref")
			}); err != nil {
				return err
			}

			if err := s.Create("payment_records", func(t *schema.Table) {
				t.ID()
				t.String("subject_ref", 64)
				t.String("provider", 20)
				t.String("provider_ref", 255)
				t.BigInteger("amount_cents")
				t.String("currency", 3)
				t.Timestamp("paid_at")
				t.Timestamp("retain_until")
				t.Timestamps()
				t.UniqueIndex("provider", "provider_ref")
				t.Index("retain_until")
			}); err != nil {
				return err
			}

			return s.Create("purged_tokens", func(t *schema.Table) {
				t.ID()
				t.String("token_id", 64).Unique()
				t.Timestamps()
			})
		},
		Down: func(s *schema.Schema) error {
			for _, name := range []string{"purged_tokens", "payment_records", "privacy_audit"} {
				if err := s.DropIfExists(name); err != nil {
					return err
				}
			}

			if err := s.Table("users", func(t *schema.Table) {
				t.DropColumn("purge_at")
				t.DropColumn("deletion_requested_at")
			}); err != nil {
				return err
			}

			return s.Table("workspaces", func(t *schema.Table) {
				t.DropColumn("purge_error")
				t.DropColumn("deletion_requested_by")
				t.DropColumn("deletion_requested_at")
				t.DropColumn("purge_at")
				t.DropColumn("state")
			})
		},
	}
}
