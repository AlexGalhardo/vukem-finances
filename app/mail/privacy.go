package mail

import "github.com/AlexGalhardo/vukem/mail"

// The privacy mailables (epic 34) are React Email components too. PurgeDate is
// a plain date ("2026-12-22") so the text reads the same in every client.

// WorkspaceDeletionScheduled tells a member that the owner asked to delete the
// workspace: it turns read-only now and is erased on PurgeDate unless the owner
// cancels.
type WorkspaceDeletionScheduled struct {
	To, Workspace, PurgeDate, URL string
}

// Envelope implements mail.Mailable.
func (m WorkspaceDeletionScheduled) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.To}, Subject: m.Workspace + " will be deleted", Tags: tag("workspace_deletion_scheduled")}
}

// Content implements mail.Mailable.
func (m WorkspaceDeletionScheduled) Content() mail.Content {
	return mail.React("workspace_deletion_scheduled", m)
}

// WorkspaceDeletionCancelled tells a member that the deletion was cancelled.
type WorkspaceDeletionCancelled struct {
	To, Workspace, URL string
}

// Envelope implements mail.Mailable.
func (m WorkspaceDeletionCancelled) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.To}, Subject: m.Workspace + " will not be deleted", Tags: tag("workspace_deletion_cancelled")}
}

// Content implements mail.Mailable.
func (m WorkspaceDeletionCancelled) Content() mail.Content {
	return mail.React("workspace_deletion_cancelled", m)
}

// AccountDeletionScheduled confirms to the person that their account will be
// erased on PurgeDate, and how to cancel by signing in before then.
type AccountDeletionScheduled struct {
	To, Name, PurgeDate, URL string
}

// Envelope implements mail.Mailable.
func (m AccountDeletionScheduled) Envelope() mail.Envelope {
	return mail.Envelope{To: []string{m.To}, Subject: "Your account will be deleted", Tags: tag("account_deletion_scheduled")}
}

// Content implements mail.Mailable.
func (m AccountDeletionScheduled) Content() mail.Content {
	return mail.React("account_deletion_scheduled", m)
}
