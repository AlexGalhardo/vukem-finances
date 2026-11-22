package sync

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// The pairing flow of docs/specs/sync-protocol.md: a person presses "Add
// device" on the web, reads an 8 character code and types it on the phone,
// which trades it for a token bound to that workspace.
const (
	// PairingCodeLength is the characters of a code.
	PairingCodeLength = 8
	// PairingTTL is how long a code works.
	PairingTTL = 10 * time.Minute
	// PairingTokenDays is the life of the token a pairing issues.
	PairingTokenDays = 365
	// PairingMaxLive is the unused, unexpired codes one person may hold.
	PairingMaxLive = 5
	// PairAttemptsPerMinute is the pair requests one address may make.
	PairAttemptsPerMinute = 5
	// PairFailuresPerHour is the failed pairs the whole server tolerates.
	PairFailuresPerHour = 200

	// CodeInvalidPairing is the one answer for a code that is unknown, used or expired.
	CodeInvalidPairing = "invalid_pairing_code"

	// TokenScope is the only scope a pairing token holds: the role of its person
	// stays the ceiling of what it may change.
	TokenScope = "sync"

	// 32 characters without 0, 1, I and O, so a code read from a screen is hard to mistake.
	pairingAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

var (
	errInvalidPairing = fail(http.StatusUnprocessableEntity, CodeInvalidPairing, "This pairing code is not valid or has expired.")
	errTooManyCodes   = fail(http.StatusUnprocessableEntity, "validation_failed",
		"You already hold the most pairing codes allowed: use one or wait 10 minutes.")
)

// IsInvalidPairing reports whether err is the answer for a code that did not pair.
func IsInvalidPairing(err error) bool { return errors.Is(err, errInvalidPairing) }

// PairingCode is a code as its creator sees it, once.
type PairingCode struct {
	Code      string
	ExpiresAt time.Time
}

// PairWorkspace names the workspace a paired phone syncs.
type PairWorkspace struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

// PairResult is the answer of POST /sync/pair.
type PairResult struct {
	Token      string        `json:"token"`
	DeviceID   string        `json:"device_id"`
	Workspace  PairWorkspace `json:"workspace"`
	ServerTime time.Time     `json:"server_time"`
}

// hashPairingCode is what is stored: the code itself never is.
func hashPairingCode(code string) string {
	sum := sha256.Sum256([]byte(code))

	return hex.EncodeToString(sum[:])
}

// NormalizePairingCode upper-cases a typed code and drops the separators; ok is
// false when what is left is not a code.
func NormalizePairingCode(raw string) (string, bool) {
	code := strings.ToUpper(strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}

		return r
	}, raw))

	if len(code) != PairingCodeLength {
		return "", false
	}

	for _, r := range code {
		if !strings.ContainsRune(pairingAlphabet, r) {
			return "", false
		}
	}

	return code, true
}

func newPairingCode() (string, error) {
	var raw [PairingCodeLength]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	out := make([]byte, PairingCodeLength)
	for i, b := range raw {
		out[i] = pairingAlphabet[b&31]
	}

	return string(out), nil
}

// CreatePairingCode makes a code for the person of a in the workspace of a.
func (s *Service) CreatePairingCode(ctx context.Context, a Actor) (PairingCode, error) {
	now := s.now()

	var out PairingCode

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM pairing_codes WHERE expires_at < ?", now.Add(-time.Hour)); err != nil {
			return err
		}

		var live int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM pairing_codes WHERE user_id = ? AND workspace_id = ? AND used_at IS NULL AND expires_at > ?",
			a.UserID, a.WorkspaceID, now).Scan(&live); err != nil {
			return err
		}

		if live >= PairingMaxLive {
			return errTooManyCodes
		}

		code, err := newPairingCode()
		if err != nil {
			return err
		}

		out = PairingCode{Code: code, ExpiresAt: now.Add(PairingTTL)}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO pairing_codes (workspace_id, user_id, code_hash, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			a.WorkspaceID, a.UserID, hashPairingCode(code), out.ExpiresAt, now, now)

		return err
	})

	return out, err
}

// Pair trades a code for a token and a registered device. The code is
// consumed by one conditional UPDATE, so of any number of concurrent attempts
// exactly one wins. A code that is unknown, used or expired gets the same
// answer, so an attacker learns nothing from it.
func (s *Service) Pair(ctx context.Context, rawCode, deviceName, platform string) (PairResult, error) {
	name := strings.TrimSpace(deviceName)
	if n := utf8.RuneCountInString(name); n == 0 || n > 80 || !utf8.ValidString(name) {
		return PairResult{}, errDeviceName
	}

	if platform == "" {
		platform = models.PlatformOther
	}

	if !contains(models.DevicePlatforms(), platform) {
		return PairResult{}, errPlatform
	}

	code, ok := NormalizePairingCode(rawCode)
	if !ok {
		return PairResult{}, errInvalidPairing
	}

	now := s.now()
	hash := hashPairingCode(code)

	var (
		plain string
		dev   models.SyncDevice
		wsID  int64
	)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE pairing_codes SET used_at = ?, updated_at = ? WHERE code_hash = ? AND used_at IS NULL AND expires_at > ?",
			now, now, hash, now)
		if err != nil {
			return err
		}

		n, err := res.RowsAffected()
		if err != nil {
			// A failing database is not a bad code: it must not count against the failure limit.
			return err
		}

		if n != 1 {
			return errInvalidPairing
		}

		var userID int64
		if err := tx.QueryRowContext(ctx, "SELECT workspace_id, user_id FROM pairing_codes WHERE code_hash = ?", hash).
			Scan(&wsID, &userID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		// The person may have left the workspace since: then the code is worth nothing.
		if _, err := orm.QueryOf[models.Membership](tx).Where("workspace_id", wsID).Where("user_id", userID).First(ctx); err != nil {
			if errors.Is(err, orm.ErrNotFound) {
				return errInvalidPairing
			}

			return err
		}

		tokens := auth.NewTokens(tx)
		tokens.Prefix, tokens.Now, tokens.Log = s.Tokens.Prefix, s.Tokens.Now, s.Tokens.Log

		tokenName := "Phone " + name
		if utf8.RuneCountInString(tokenName) > 80 {
			tokenName = string([]rune(tokenName)[:80])
		}

		var tok auth.Token

		plain, tok, err = tokens.CreateWith(ctx, strconv.FormatInt(userID, 10), tokenName, []string{TokenScope},
			PairingTokenDays*24*time.Hour, auth.CreateOptions{Resource: strconv.FormatInt(wsID, 10)})
		if err != nil {
			return err
		}

		dev = models.SyncDevice{
			WorkspaceID: wsID, UserID: userID, TokenID: tok.ID, DeviceID: models.NewUID(), Name: name, Platform: platform,
		}

		return orm.Create(workspaces.WithTenant(ctx, wsID), tx, &dev)
	})
	if err != nil {
		if errors.Is(err, errInvalidPairing) {
			return PairResult{}, errInvalidPairing
		}

		return PairResult{}, err
	}

	ws, err := s.WS.Current(workspaces.WithTenant(ctx, wsID))
	if err != nil {
		return PairResult{}, err
	}

	return PairResult{
		Token: plain, DeviceID: dev.DeviceID, ServerTime: now,
		Workspace: PairWorkspace{UID: strconv.FormatInt(ws.ID, 10), Name: ws.Name, Currency: ws.Currency, Timezone: ws.Timezone},
	}, nil
}
