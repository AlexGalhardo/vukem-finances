package models

import (
	"strconv"

	"github.com/AlexGalhardo/vukem/orm"
)

// User is a registered account (vukem install:auth).
type User struct {
	orm.Model

	Name            string       `json:"name"`
	Email           string       `json:"email"`
	Password        string       `json:"-"`
	EmailVerifiedAt orm.NullTime `json:"email_verified_at"`
	SessionVersion  int64        `json:"-"`

	// DeletionRequestedAt is set while the account is in its grace period;
	// PurgeAt is when the purge job removes it.
	DeletionRequestedAt orm.NullTime `json:"-"`
	PurgeAt             orm.NullTime `json:"-"`
}

// PendingDeletion reports whether the account asked to be deleted and the grace
// period has not ended.
func (u User) PendingDeletion() bool { return u.DeletionRequestedAt.Valid }

// AuthID implements auth.Authenticatable.
func (u User) AuthID() string { return strconv.FormatInt(u.ID, 10) }

// AuthPassword implements auth.Authenticatable.
func (u User) AuthPassword() string { return u.Password }

// AuthSessionVersion implements auth.SessionVersioner: sessions logged in
// with another version are signed out.
func (u User) AuthSessionVersion() int64 { return u.SessionVersion }

// EmailVerified implements auth.Verifiable.
func (u User) EmailVerified() bool { return u.EmailVerifiedAt.Valid }
