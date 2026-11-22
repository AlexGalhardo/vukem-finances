// Package qakit holds what the quality campaign scripts share: a seeded
// database and an app server they can start, kill and start again.
package qakit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/auth/hash"
	"github.com/AlexGalhardo/vukem/database"
)

const (
	crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	// seedMonths is how far back the seeded transactions reach.
	seedMonths = 24
	// SeedPass is the password of every seeded user.
	SeedPass = "Load-Test-Passw0rd-xyz"
)

// Workspace is one seeded workspace and what a client needs to use it.
type Workspace struct {
	ID       int64
	UserID   int64
	Email    string
	Tokens   []string // one per client: the API limits count per token
	Accounts []string
	Cats     []string
	Rows     int
	// Members are extra users of the workspace (the first one only), so a load
	// test can sign in once per client: the web rate limits count per user.
	Members []string
}

// ULID is a 26 character Crockford id with a time part, as the clients make.
func ULID(t time.Time) string {
	var b [26]byte

	ms := uint64(t.UnixMilli())
	for i := 9; i >= 0; i-- {
		b[i] = crockford[ms&31]
		ms >>= 5
	}

	var r [16]byte
	_, _ = rand.Read(r[:])

	for i := range 16 {
		b[10+i] = crockford[r[i]&31]
	}

	return string(b[:])
}

func RandInt(n int) int {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

// Seed fills the migrated database: one workspace of bigRows transactions
// and workspaces small ones of perRows each. Rows go through the real change
// log triggers, so sync tables are as large as production ones would be.
func Seed(ctx context.Context, dbPath string, bigRows, workspaces, perRows, clients int) ([]Workspace, error) {
	db, err := database.Open(ctx, "file:"+dbPath)
	if err != nil {
		return nil, err
	}

	defer func() { _ = db.Close() }()

	hashed, err := hash.Default().Make(SeedPass)
	if err != nil {
		return nil, err
	}

	tokens := auth.NewTokens(db)
	tokens.Prefix = "vf_"

	out := make([]Workspace, 0, workspaces+1)

	for i := 0; i <= workspaces; i++ {
		rows := perRows
		if i == 0 {
			rows = bigRows
		}

		ws, err := seedWorkspace(ctx, db, tokens, i, rows, clients, hashed)
		if err != nil {
			return nil, fmt.Errorf("workspace %d: %w", i, err)
		}

		out = append(out, ws)
	}

	return out, nil
}

func seedWorkspace(ctx context.Context, db *database.DB, tokens *auth.Tokens, n, rows, clients int, hashed string) (Workspace, error) {
	ws := Workspace{Email: fmt.Sprintf("load%d@example.test", n), Rows: rows}
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	tx, err := db.SQL().BeginTx(ctx, nil)
	if err != nil {
		return ws, err
	}

	defer func() { _ = tx.Rollback() }()

	if ws.UserID, err = insertID(ctx, tx,
		"INSERT INTO users (name, email, password, email_verified_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		fmt.Sprintf("Load %d", n), ws.Email, hashed, now, now, now); err != nil {
		return ws, err
	}

	if ws.ID, err = insertID(ctx, tx, "INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES (?, 'USD', ?, ?)",
		fmt.Sprintf("Load %d", n), now, now); err != nil {
		return ws, err
	}

	if _, err = tx.ExecContext(ctx, "INSERT INTO memberships (workspace_id, user_id, role, created_at, updated_at) VALUES (?, ?, 'owner', ?, ?)",
		ws.ID, ws.UserID, now, now); err != nil {
		return ws, err
	}

	if n == 0 {
		for c := range clients {
			email := fmt.Sprintf("member%d@example.test", c)

			uid, err := insertID(ctx, tx,
				"INSERT INTO users (name, email, password, email_verified_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
				fmt.Sprintf("Member %d", c), email, hashed, now, now, now)
			if err != nil {
				return ws, err
			}

			if _, err = tx.ExecContext(ctx, "INSERT INTO memberships (workspace_id, user_id, role, created_at, updated_at) VALUES (?, ?, 'member', ?, ?)",
				ws.ID, uid, now, now); err != nil {
				return ws, err
			}

			ws.Members = append(ws.Members, email)
		}
	}

	accountIDs := make([]int64, 0, 3)

	for i, name := range []string{"Checking", "Savings", "Cash"} {
		uid := ULID(time.Now())

		id, err := insertID(ctx, tx,
			"INSERT INTO accounts (workspace_id, name, kind, opening_cents, uid, created_at, updated_at) VALUES (?, ?, ?, 100000, ?, ?, ?)",
			ws.ID, name, []string{"checking", "savings", "cash"}[i], uid, now, now)
		if err != nil {
			return ws, err
		}

		accountIDs = append(accountIDs, id)
		ws.Accounts = append(ws.Accounts, uid)
	}

	categoryIDs := make([]int64, 0, 12)

	for i := range 12 {
		uid := ULID(time.Now())

		kind := "expense"
		if i < 2 {
			kind = "income"
		}

		id, err := insertID(ctx, tx,
			"INSERT INTO categories (workspace_id, name, kind, uid, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			ws.ID, "Category "+strconv.Itoa(i), kind, uid, now, now)
		if err != nil {
			return ws, err
		}

		categoryIDs = append(categoryIDs, id)
		ws.Cats = append(ws.Cats, uid)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO transactions
		(workspace_id, account_id, category_id, kind, amount_cents, description, occurred_on, uid, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return ws, err
	}

	defer func() { _ = stmt.Close() }()

	// Rows spread evenly over seedMonths, so a filter over the last twelve
	// months matches half of them.
	span := seedMonths * 30
	today := time.Now().UTC()

	for i := range rows {
		day := today.AddDate(0, 0, -(i * span / rows))
		c := RandInt(len(categoryIDs))
		kind, amount := "expense", int64(500+RandInt(20000))

		if c < 2 {
			kind, amount = "income", int64(100000+RandInt(300000))
		}

		_, err := stmt.ExecContext(ctx, ws.ID, accountIDs[RandInt(len(accountIDs))], categoryIDs[c], kind, amount,
			fmt.Sprintf("Purchase %d at shop %d", i, RandInt(500)), day.Format("2006-01-02"), ULID(day), now, now)
		if err != nil {
			return ws, err
		}
	}

	if err := tx.Commit(); err != nil {
		return ws, err
	}

	for c := range clients {
		plain, _, err := tokens.CreateWith(ctx, strconv.FormatInt(ws.UserID, 10), fmt.Sprintf("load %d", c),
			[]string{
				"accounts:read", "categories:read", "transactions:read", "transactions:write", "reports:read", "export:read", "sync",
			}, 24*time.Hour, auth.CreateOptions{Resource: strconv.FormatInt(ws.ID, 10)})
		if err != nil {
			return ws, err
		}

		ws.Tokens = append(ws.Tokens, plain)
	}

	return ws, nil
}

func insertID(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}
