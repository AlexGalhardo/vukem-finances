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
