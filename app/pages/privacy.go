package pages

import "github.com/AlexGalhardo/vukem/inertia"

// AccountDeletion is what a person sees after signing in while their account
// waits to be deleted: when it will be erased, and the way to cancel.
var AccountDeletion = inertia.Define[AccountDeletionProps]("Auth/AccountDeletion")

// AccountDeletionProps are the props of Auth/AccountDeletion.
type AccountDeletionProps struct {
	Email     string `json:"email"`
	PurgeDate string `json:"purgeDate"`
}

// Privacy is the page of the data lifecycle: the personal export, the deletion
// of the workspace and the deletion of the account.
var Privacy = inertia.Define[PrivacyProps]("Privacy")

// PrivacyProps are the props of Privacy.
type PrivacyProps struct {
	WorkspaceName string `json:"workspaceName"`
	// IsOwner is true when the person owns the workspace: only the owner can
	// delete it.
	IsOwner bool `json:"isOwner"`
	// Personal is true in the personal edition, which has one workspace that is
	// not deleted by itself: the person erases the local data instead.
	Personal bool `json:"personal"`
	// Pending is true while the workspace waits to be purged; PurgeDate is the
	// day (YYYY-MM-DD) it will be.
	Pending   bool   `json:"pending"`
	PurgeDate string `json:"purgeDate"`
	// ExportReady is true right after the password was confirmed: the page then
	// starts the download of /account/export.json.
	ExportReady bool `json:"exportReady"`
	// CanErase is true when the person may erase all local data: the owner, in
	// the personal edition. Erased is true right after they did.
	CanErase bool `json:"canErase"`
	Erased   bool `json:"erased"`
	// GraceDays is the length of the grace period.
	GraceDays int `json:"graceDays"`
}
