package models

import "github.com/AlexGalhardo/vukem/ulid"

var uids = ulid.NewGenerator() //nolint:gochecknoglobals // the generator is safe for concurrent use and keeps ids increasing

// NewUID returns a fresh ULID for a synced row made on the server.
func NewUID() string { return uids.New() }

// stampUID keeps a uid a client invented and fills it in otherwise; the
// sync layer validates a client uid with changes.ValidUID before it gets here.
func stampUID(uid string) string {
	if uid != "" {
		return uid
	}

	return NewUID()
}
