// Package privacy is the data lifecycle of epic 34: the personal data export,
// the deletion of a workspace or an account with a grace period, the purge job
// and the minimal audit record. Deletion is two steps on purpose: a request
// moves the subject to pending_deletion with a purge date 30 days ahead, and the
// purge job removes the rows only when that date has passed.
package privacy
