// Package changes is the sync side of the data model
// (docs/specs/sync-data-model.md): the registry of synced entities, the
// triggers that write the change log, the reader of the log, compaction and
// the conflict merge. A table with a uid column must be registered here.
package changes
