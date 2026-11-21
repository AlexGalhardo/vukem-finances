package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Categories gain a parent (one level), a color token, an icon name and
// archived_at; tags and their join table are new; transactions gain tag_uids
// (the synced image of the join) and import_hash (server only, the duplicate
// check of the importer). The change-log triggers of categories and
// transactions are reinstalled because their writable columns grew; the
// entities are written out so this migration's checksum never moves with the
// registry.
func m20261121220000AddCategoriesTagsAndImportHash() schema.Migration {
	categories := changes.Entity{
		Name: "category", Table: "categories",
		Fields: []string{"name", "kind", "parent_id", "color", "icon", "archived_at"},
	}
	transactions := changes.Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid", "tag_uids",
		},
	}
	tags := changes.Entity{Name: "tag", Table: "tags", Fields: []string{"name", "color"}}

	return schema.Migration{
		ID: "20261121220000_add_categories_tags_and_import_hash",
		Up: func(s *schema.Schema) error {
			if has, err := s.HasColumn("categories", "parent_id"); err != nil {
				return err
			} else if !has {
				if err := s.Table("categories", func(t *schema.Table) {
					t.BigInteger("parent_id").Nullable()
					t.String("color", 20).Default("")
					t.String("icon", 20).Default("")
					t.Timestamp("archived_at").Nullable()
					t.Index("workspace_id", "parent_id")
				}); err != nil {
					return err
				}
			}

			if has, err := s.HasColumn("transactions", "import_hash"); err != nil {
				return err
			} else if !has {
				if err := s.Table("transactions", func(t *schema.Table) {
					t.Text("tag_uids").Default("[]")
					t.String("import_hash", 80).Nullable()
					t.Index("workspace_id", "import_hash")
				}); err != nil {
					return err
				}
			}

			if err := s.Create("tags", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("name", 40)
				t.String("color", 20).Default("")
				t.Timestamps()
				syncColumns(t)
			}); err != nil {
				return err
			}

			// Case-insensitive, and only among live tags, so a deleted name can be reused.
			if err := s.Raw("CREATE UNIQUE INDEX tags_workspace_id_name_lower_unique ON tags (workspace_id, lower(name)) WHERE deleted_at IS NULL"); err != nil {
				return err
			}

			if err := s.Create("transaction_tags", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("transaction_id").Constrained("transactions").CascadeOnDelete()
				t.ForeignID("tag_id").Constrained("tags").CascadeOnDelete()
				t.Timestamps()
				t.UniqueIndex("transaction_id", "tag_id")
				t.Index("workspace_id", "tag_id")
			}); err != nil {
				return err
			}

			for _, e := range []changes.Entity{categories, transactions, tags} {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			prev := []changes.Entity{
				{Name: "category", Table: "categories", Fields: []string{"name", "kind"}},
				{
					Name: "transaction", Table: "transactions",
					Fields: []string{
						"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid",
					},
				},
			}

			for _, table := range []string{"transaction_tags", "tags"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			for _, stmt := range []string{
				"DROP INDEX IF EXISTS transactions_workspace_id_import_hash_index",
				"DROP INDEX IF EXISTS categories_workspace_id_parent_id_index",
			} {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			if err := s.Table("transactions", func(t *schema.Table) {
				t.DropColumn("tag_uids")
				t.DropColumn("import_hash")
			}); err != nil {
				return err
			}

			if err := s.Table("categories", func(t *schema.Table) {
				t.DropColumn("parent_id")
				t.DropColumn("color")
				t.DropColumn("icon")
				t.DropColumn("archived_at")
			}); err != nil {
				return err
			}

			for _, e := range prev {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
