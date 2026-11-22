package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 17: the Telegram bot. Every table is server state (no uid, never
// synced) and none carries a workspace_id: a link belongs to a person and the
// workspace is resolved from the live membership at every message; the codes,
// drafts and counters are keyed by chat or code. The tables are on the
// allow-list of the isolation guard for that reason.
//
//   - telegram_links: one row per chat (chat_id is unique, a revoked row is
//     reused when the chat links again).
//   - telegram_link_codes: single-use codes, stored as a SHA-256 hash.
//   - telegram_pending: drafts waiting for a tap, and the 60 second undo of
//     the entry a tap created.
//   - telegram_updates: processed update ids, the idempotency key.
//   - telegram_state: the polling offset and the lease of the polling worker.
//   - telegram_counters: the durable counters of the bot rate limits.
//
// It also adds the telegram switch to the alert preferences and the delivery
// claim to notifications, next to the mail ones.
func m20261123100000CreateTelegramTables() schema.Migration {
	return schema.Migration{
		ID: "20261123100000_create_telegram_tables",
		Up: func(s *schema.Schema) error {
			if err := s.Create("telegram_links", func(t *schema.Table) {
				t.ID()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.BigInteger("chat_id").Unique()
				t.ForeignID("current_workspace_id").Nullable().Constrained("workspaces").NullOnDelete()
				t.String("default_account_uid", 26).Nullable()
				t.String("language", 8).Default("en")
				t.Timestamp("revoked_at").Nullable()
				t.Timestamp("last_seen_at").Nullable()
				t.Timestamps()
				t.Index("user_id")
			}); err != nil {
				return err
			}

			if err := s.Create("telegram_link_codes", func(t *schema.Table) {
				t.ID()
				t.String("code_hash", 64).Unique()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.ForeignID("for_workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.Timestamp("expires_at")
				t.Timestamp("used_at").Nullable()
				t.Timestamps()
				t.Index("user_id")
				t.Index("expires_at")
			}); err != nil {
				return err
			}

			if err := s.Create("telegram_pending", func(t *schema.Table) {
				t.ID()
				t.String("ref", 26).Unique()
				t.BigInteger("chat_id")
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.ForeignID("for_workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.Text("payload")
				t.Timestamp("expires_at")
				t.Timestamp("consumed_at").Nullable()
				t.String("transaction_uid", 26).Nullable()
				t.Timestamp("undone_at").Nullable()
				t.Timestamps()
				t.Index("chat_id")
				t.Index("expires_at")
			}); err != nil {
				return err
			}

			if err := s.Create("telegram_updates", func(t *schema.Table) {
				t.ID()
				t.BigInteger("update_id").Unique()
				t.Timestamps()
				t.Index("created_at")
			}); err != nil {
				return err
			}

			if err := s.Create("telegram_state", func(t *schema.Table) {
				t.ID()
				t.String("name", 40).Unique()
				t.BigInteger("next_offset").Default(0)
				t.String("lease_owner", 64).Default("")
				t.Timestamp("lease_until").Nullable()
				t.Timestamps()
			}); err != nil {
				return err
			}

			if err := s.Create("telegram_counters", func(t *schema.Table) {
				t.ID()
				t.String("bucket", 120).Unique()
				t.BigInteger("count").Default(0)
				t.Timestamp("expires_at")
				t.Timestamps()
				t.Index("expires_at")
			}); err != nil {
				return err
			}

			if err := s.Table("notification_preferences", func(t *schema.Table) {
				t.Boolean("telegram").Default(false)
			}); err != nil {
				return err
			}

			return s.Table("notifications", func(t *schema.Table) {
				t.Boolean("telegram").Default(false)
				t.Timestamp("telegrammed_at").Nullable()
			})
		},
		Down: func(s *schema.Schema) error {
			if err := s.Table("notifications", func(t *schema.Table) { t.DropColumn("telegram", "telegrammed_at") }); err != nil {
				return err
			}

			if err := s.Table("notification_preferences", func(t *schema.Table) { t.DropColumn("telegram") }); err != nil {
				return err
			}

			for _, table := range []string{
				"telegram_counters", "telegram_state", "telegram_updates", "telegram_pending",
				"telegram_link_codes", "telegram_links",
			} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
