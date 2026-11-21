package changes

import (
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem/ulid"
)

// MaxFuture is how far ahead of the server clock a client uid may be stamped.
const MaxFuture = 24 * time.Hour

// ErrInvalidUID is the answer to a uid a client may not use.
var ErrInvalidUID = errors.New("changes: invalid uid")

// ValidUID checks a client-supplied uid: 26 upper-case Crockford characters
// and a timestamp at most MaxFuture after now. Uniqueness in the workspace is
// the database's check (unique on workspace_id, uid).
func ValidUID(uid string, now time.Time) error {
	at, ok := ulid.TimeOf(uid)
	if !ok || at.After(now.Add(MaxFuture)) {
		return ErrInvalidUID
	}

	return nil
}
