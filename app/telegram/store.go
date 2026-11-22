package telegram

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Store is the bot's own tables. None of them has a workspace_id (see the
// migration), so nothing here is tenant data; every query is keyed by chat,
// person, code or reference.
type Store struct {
	DB  *database.DB
	Now func() time.Time
}

// The ways a link can fail.
var (
	// ErrBadCode answers a link code that is unknown, expired or already used.
	ErrBadCode = errors.New("telegram: link code is invalid, expired or used")
	// ErrChatTaken answers a chat that is already linked to another person.
	ErrChatTaken = errors.New("telegram: chat is linked to another account")
)

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))

	return hex.EncodeToString(sum[:])
}

// ---- idempotency ----

// ClaimUpdate records an update id and reports whether this call is the first
// to see it. The unique index is the arbiter, so a webhook retry, a poll
// overlap or two instances never handle one update twice.
func (s *Store) ClaimUpdate(ctx context.Context, id int64) (bool, error) {
	err := orm.Create(ctx, s.DB, &models.TelegramUpdate{UpdateID: id})
	if err == nil {
		return true, nil
	}

	if s.DB.UniqueViolation(err) {
		return false, nil
	}

	return false, err
}

// ReleaseUpdate forgets a claim whose handling failed for an internal reason,
// so the redelivery of the update is handled.
func (s *Store) ReleaseUpdate(ctx context.Context, id int64) error {
	_, err := orm.QueryOf[models.TelegramUpdate](s.DB).Where("update_id", id).Delete(ctx)

	return err
}

// PruneUpdates drops the update ids older than the retention.
func (s *Store) PruneUpdates(ctx context.Context) error {
	_, err := orm.QueryOf[models.TelegramUpdate](s.DB).
		Where("created_at", "<", s.Now().UTC().Add(-config.TelegramUpdateRetention)).Delete(ctx)

	return err
}

// ---- links ----

// LinkByChat returns the live link of a chat.
func (s *Store) LinkByChat(ctx context.Context, chat int64) (models.TelegramLink, bool, error) {
	l, err := orm.QueryOf[models.TelegramLink](s.DB).Where("chat_id", chat).WhereNull("revoked_at").First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.TelegramLink{}, false, nil
	}

	return l, err == nil, err
}

// LinksOf lists the live links of a person, newest first.
func (s *Store) LinksOf(ctx context.Context, userID int64) ([]models.TelegramLink, error) {
	return orm.QueryOf[models.TelegramLink](s.DB).Where("user_id", userID).WhereNull("revoked_at").
		OrderBy("id", orm.Desc).Get(ctx)
}

// Revoke ends a link. The chat can link again later.
func (s *Store) Revoke(ctx context.Context, linkID int64) error {
	_, err := orm.QueryOf[models.TelegramLink](s.DB).Where("id", linkID).WhereNull("revoked_at").
		Update(ctx, orm.Set{"revoked_at": s.Now().UTC()})

	return err
}

// RevokeOf ends a link of userID, and only of userID: the web page names the
// link by id, and another person's id is a no-op.
func (s *Store) RevokeOf(ctx context.Context, userID, linkID int64) (bool, error) {
	n, err := orm.QueryOf[models.TelegramLink](s.DB).Where("id", linkID).Where("user_id", userID).
		WhereNull("revoked_at").Update(ctx, orm.Set{"revoked_at": s.Now().UTC()})

	return n > 0, err
}

// Touch records that the chat was just seen.
func (s *Store) Touch(ctx context.Context, linkID int64) {
	_, _ = orm.QueryOf[models.TelegramLink](s.DB).Where("id", linkID).Update(ctx, orm.Set{"last_seen_at": s.Now().UTC()})
}

// SetLinkFields changes the preferences of a link.
func (s *Store) SetLinkFields(ctx context.Context, linkID int64, set orm.Set) error {
	_, err := orm.QueryOf[models.TelegramLink](s.DB).Where("id", linkID).WhereNull("revoked_at").Update(ctx, set)

	return err
}

// ---- link codes ----

// NewCode makes the single-use code the signed-in person shows to Telegram:
// 128 bits of randomness in base64url (22 characters, inside the 64 that
// /start accepts). Only its hash is stored, and a person holds one live code
// at a time: making another spends the earlier ones.
func (s *Store) NewCode(ctx context.Context, userID, workspaceID int64) (string, time.Time, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", time.Time{}, err
	}

	code := base64.RawURLEncoding.EncodeToString(raw[:])
	now := s.Now().UTC()
	expires := now.Add(config.TelegramLinkCodeTTL)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := orm.QueryOf[models.TelegramLinkCode](tx).Where("user_id", userID).WhereNull("used_at").
			Update(ctx, orm.Set{"used_at": now}); err != nil {
			return err
		}

		row := models.TelegramLinkCode{CodeHash: hashCode(code), UserID: userID, ForWorkspaceID: workspaceID, ExpiresAt: expires}

		return orm.Create(ctx, tx, &row)
	})
	if err != nil {
		return "", time.Time{}, err
	}

	return code, expires, nil
}

// Redeem links chat to the person who made code. The code is spent by one
// conditional UPDATE and the link is written in the same database
// transaction, so of any number of concurrent attempts with one code exactly
// one links. A chat already linked to another person is refused and the code
// stays unspent.
func (s *Store) Redeem(ctx context.Context, code string, chat int64, language string) (models.TelegramLink, error) {
	var link models.TelegramLink

	hash := hashCode(code)
	now := s.Now().UTC()

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.TelegramLinkCode](tx).Where("code_hash", hash).WhereNull("used_at").
			Where("expires_at", ">", now).Update(ctx, orm.Set{"used_at": now})
		if err != nil {
			return err
		}

		if n != 1 {
			return ErrBadCode
		}

		row, err := orm.QueryOf[models.TelegramLinkCode](tx).Where("code_hash", hash).First(ctx)
		if err != nil {
			return err
		}

		existing, err := orm.QueryOf[models.TelegramLink](tx).Where("chat_id", chat).First(ctx)

		switch {
		case errors.Is(err, orm.ErrNotFound):
			link = models.TelegramLink{
				UserID: row.UserID, ChatID: chat, Language: language,
				CurrentWorkspaceID: sql.Null[int64]{V: row.ForWorkspaceID, Valid: true},
			}

			return orm.Create(ctx, tx, &link)
		case err != nil:
			return err
		case !existing.RevokedAt.Valid && existing.UserID != row.UserID:
			return ErrChatTaken
		}

		// A revoked row, or a repeat by the same person: reuse the row.
		if _, err := orm.QueryOf[models.TelegramLink](tx).Where("id", existing.ID).Update(ctx, orm.Set{
			"user_id": row.UserID, "current_workspace_id": row.ForWorkspaceID, "default_account_uid": nil,
			"revoked_at": nil, "language": language,
		}); err != nil {
			return err
		}

		link, err = orm.QueryOf[models.TelegramLink](tx).Find(ctx, existing.ID)

		return err
	})

	return link, err
}

// PruneCodes drops the codes that expired more than a day ago.
func (s *Store) PruneCodes(ctx context.Context) error {
	_, err := orm.QueryOf[models.TelegramLinkCode](s.DB).
		Where("expires_at", "<", s.Now().UTC().Add(-24*time.Hour)).Delete(ctx)

	return err
}

// ---- drafts ----

// NewPending stores a draft for chat; ref is its id and the uid of the
// transaction a tap will create.
func (s *Store) NewPending(ctx context.Context, chat, userID, workspaceID int64, ref, payload string) error {
	row := models.TelegramPending{
		Ref: ref, ChatID: chat, UserID: userID, ForWorkspaceID: workspaceID, Payload: payload,
		ExpiresAt: s.Now().UTC().Add(config.TelegramPendingTTL),
	}

	return orm.Create(ctx, s.DB, &row)
}

// Consume claims a draft of chat with one conditional UPDATE, so a double tap
// or two instances create one transaction. ok is false when the draft is
// unknown, belongs to another chat, expired or was already taken.
func (s *Store) Consume(ctx context.Context, chat int64, ref string) (models.TelegramPending, bool, error) {
	now := s.Now().UTC()

	n, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).Where("chat_id", chat).
		WhereNull("consumed_at").Where("expires_at", ">", now).Update(ctx, orm.Set{"consumed_at": now})
	if err != nil || n != 1 {
		return models.TelegramPending{}, false, err
	}

	p, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).First(ctx)

	return p, err == nil, err
}

// Unconsume gives a claimed draft back after the transaction failed to save.
func (s *Store) Unconsume(ctx context.Context, ref string) error {
	_, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).WhereNull("transaction_uid").
		Update(ctx, orm.Set{"consumed_at": nil})

	return err
}

// MarkCreated remembers the transaction a draft made, for the undo.
func (s *Store) MarkCreated(ctx context.Context, ref, uid string) error {
	_, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).Update(ctx, orm.Set{"transaction_uid": uid})

	return err
}

// ClaimUndo spends the undo of a saved entry: one conditional UPDATE inside
// the window, for the chat that made the entry. window reports a claim that
// failed only because the 60 seconds are over.
func (s *Store) ClaimUndo(ctx context.Context, chat int64, ref string) (p models.TelegramPending, ok, late bool, err error) {
	now := s.Now().UTC()

	n, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).Where("chat_id", chat).
		WhereNotNull("transaction_uid").WhereNull("undone_at").
		Where("consumed_at", ">=", now.Add(-config.TelegramUndoWindow)).Update(ctx, orm.Set{"undone_at": now})
	if err != nil {
		return p, false, false, err
	}

	row, ferr := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).Where("chat_id", chat).First(ctx)
	if ferr != nil && !errors.Is(ferr, orm.ErrNotFound) {
		return p, false, false, ferr
	}

	if n == 1 {
		return row, true, false, nil
	}

	return row, false, ferr == nil && row.TransactionUID.Valid && !row.UndoneAt.Valid, nil
}

// ReleaseUndo gives the undo back after the delete failed.
func (s *Store) ReleaseUndo(ctx context.Context, ref string) error {
	_, err := orm.QueryOf[models.TelegramPending](s.DB).Where("ref", ref).Update(ctx, orm.Set{"undone_at": nil})

	return err
}

// PrunePending drops the drafts that expired, and the spent ones past their
// undo window, a day later.
func (s *Store) PrunePending(ctx context.Context) error {
	_, err := orm.QueryOf[models.TelegramPending](s.DB).
		Where("expires_at", "<", s.Now().UTC().Add(-24*time.Hour)).Delete(ctx)

	return err
}

// ---- the polling lease and offset ----

const pollerName = "poller"

// AcquireLease takes or renews the lease of the polling worker for owner and
// returns the offset to poll from. ok is false while another owner holds a
// lease that has not expired, so two instances never both consume updates.
func (s *Store) AcquireLease(ctx context.Context, owner string, ttl time.Duration) (offset int64, ok bool, err error) {
	if err := orm.Create(ctx, s.DB, &models.TelegramState{Name: pollerName}); err != nil && !s.DB.UniqueViolation(err) {
		return 0, false, err
	}

	now := s.Now().UTC()

	n, err := orm.QueryOf[models.TelegramState](s.DB).Where("name", pollerName).
		WhereGroup(func(g orm.Query[models.TelegramState]) orm.Query[models.TelegramState] {
			return g.Where("lease_owner", owner).OrWhereNull("lease_until").OrWhere("lease_until", "<", now)
		}).Update(ctx, orm.Set{"lease_owner": owner, "lease_until": now.Add(ttl)})
	if err != nil {
		return 0, false, err
	}

	if n != 1 {
		return 0, false, nil
	}

	st, err := orm.QueryOf[models.TelegramState](s.DB).Where("name", pollerName).First(ctx)
	if err != nil {
		return 0, false, err
	}

	return st.NextOffset, true, nil
}

// SaveOffset stores the next offset, only while owner holds the lease.
func (s *Store) SaveOffset(ctx context.Context, owner string, next int64) (bool, error) {
	n, err := orm.QueryOf[models.TelegramState](s.DB).Where("name", pollerName).Where("lease_owner", owner).
		Update(ctx, orm.Set{"next_offset": next})

	return n == 1, err
}

// ReleaseLease frees the lease at a clean shutdown.
func (s *Store) ReleaseLease(ctx context.Context, owner string) error {
	_, err := orm.QueryOf[models.TelegramState](s.DB).Where("name", pollerName).Where("lease_owner", owner).
		Update(ctx, orm.Set{"lease_owner": "", "lease_until": nil})

	return err
}

// Prune drops what the bot no longer needs: processed update ids past the
// retention, expired codes and drafts, and finished rate windows.
func (s *Store) Prune(ctx context.Context) error {
	return errors.Join(s.PruneUpdates(ctx), s.PruneCodes(ctx), s.PrunePending(ctx), PruneCounters(ctx, s.DB, time.Now()))
}
