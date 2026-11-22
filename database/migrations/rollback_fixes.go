package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Full rollback fixes. Committed migrations are frozen, so the Downs that
// cannot run are replaced or wrapped here, in the registry. Only Down changes:
// the checksum of an applied migration covers Up, so nothing recorded moves.

// withTriggersDroppedBeforeDown wraps the Down of a committed migration so the
// change-log triggers of the tables go away before it runs. SQLite refuses
// ALTER TABLE DROP COLUMN while a trigger of the table still names the column
// (the update trigger reads OLD.<col>), so the Downs of the credit cards, the
// transfers and the tags migrations failed on the column they drop. Each of
// those Downs reinstalls the triggers of the earlier column set itself, so
// every table ends with the same triggers as before.
func withTriggersDroppedBeforeDown(mig schema.Migration, tables ...string) schema.Migration {
	down := mig.Down

	mig.Down = func(s *schema.Schema) error {
		for _, table := range tables {
			for _, name := range changes.TriggerNames(table) {
				if err := s.Raw("DROP TRIGGER IF EXISTS " + name); err != nil {
					return err
				}
			}
		}

		return down(s)
	}

	return mig
}

// withSyncColumnsDown replaces the Down of the sync columns migration. The
// original rebuilt each table to its legacy definition without saying that
// uid, deleted_at, version and the unique index on uid are meant to go, which
// Rebuild refuses.
func withSyncColumnsDown(mig schema.Migration) schema.Migration {
	mig.Down = func(s *schema.Schema) error {
		for _, table := range []string{"sync_conflicts", "sync_state", "changes"} {
			if err := s.DropIfExists(table); err != nil {
				return err
			}
		}

		for _, table := range syncTables {
			err := s.Rebuild(table, legacyDefinition(table), schema.DropTriggers(),
				schema.DropColumns("uid", "deleted_at", "version"), schema.DropIndexes(table+"_workspace_id_uid_unique"))
			if err != nil {
				return err
			}
		}

		return nil
	}

	return mig
}
