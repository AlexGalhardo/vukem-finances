package sync

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Statuses of a pushed mutation.
const (
	StatusApplied   = "applied"
	StatusMerged    = "merged"
	StatusDuplicate = "duplicate"
	StatusRejected  = "rejected"
)

// MutationIn is one change a client made offline.
type MutationIn struct {
	OpID    string         `json:"op_id"`
	Entity  string         `json:"entity"`
	UID     string         `json:"uid"`
	Op      string         `json:"op"`
	BaseSeq int64          `json:"base_seq"`
	Fields  map[string]any `json:"fields"`
}

// ConflictOut is a value that lost a merge.
type ConflictOut struct {
	Field        string `json:"field"`
	Kind         string `json:"kind"`
	LostValue    any    `json:"lost_value"`
	AppliedValue any    `json:"applied_value"`
	LostBy       string `json:"lost_by_device,omitempty"`
}

// ErrorOut says why a mutation was rejected.
type ErrorOut struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// EffectOut is another row the mutation changed (the other leg of a
// transfer, the transactions of a deleted account).
type EffectOut struct {
	Entity string `json:"entity"`
	UID    string `json:"uid"`
	Op     string `json:"op"`
	Seq    int64  `json:"seq"`
}

// Result is the outcome of one mutation. It is stored under op_id, so a
// retry gets the same answer and changes nothing.
type Result struct {
	OpID          string         `json:"op_id"`
	Status        string         `json:"status"`
	Entity        string         `json:"entity"`
	UID           string         `json:"uid"`
	Row           map[string]any `json:"row"`
	Version       int64          `json:"version"`
	Seq           int64          `json:"seq"`
	Conflicts     []ConflictOut  `json:"conflicts"`
	Effects       []EffectOut    `json:"effects"`
	ReplacedByUID string         `json:"replaced_by_uid,omitempty"`
	Error         *ErrorOut      `json:"error,omitempty"`
}

// maxEffects bounds the side effects listed in one result; pull has the rest.
const maxEffects = 100

// pushEnv is what every mutation of one batch shares.
type pushEnv struct {
	a      Actor
	dev    models.SyncDevice
	today  string
	now    time.Time
	ranked map[string]int
}

// Push applies a batch of mutations in one database transaction, in entity
// dependency order, each inside a savepoint: a rejected mutation changes
// nothing and the others go on. Results come back in request order.
func (s *Service) Push(ctx context.Context, a Actor, dev models.SyncDevice, muts []MutationIn) ([]Result, error) {
	if len(muts) > MaxMutations {
		return nil, fail(http.StatusRequestEntityTooLarge, CodeBatchTooLarge, "A push takes at most 200 mutations.")
	}

	ctx = workspaces.WithTenant(ctx, a.WorkspaceID)
	now := s.now()

	var recent int

	err := s.DB.SQL().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sync_ops WHERE workspace_id = ? AND device_id = ? AND created_at > ?",
		a.WorkspaceID, dev.DeviceID, now.Add(-time.Hour)).Scan(&recent)
	if err != nil {
		return nil, err
	}

	if recent+len(muts) > MutationsPerHour {
		return nil, fail(http.StatusTooManyRequests, CodeRateLimited, "A device may push at most 5000 mutations per hour.")
	}

	// Read before the transaction: an in-memory database has one connection.
	env := pushEnv{a: a, dev: dev, today: s.Fin.Today(ctx), now: now, ranked: map[string]int{}}
	for i, e := range Order() {
		env.ranked[e.Name] = i
	}

	results := make([]Result, len(muts))

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, a.WorkspaceID); err != nil {
			return err
		}

		broken, err := s.checkTransfers(ctx, tx, a.WorkspaceID, muts)
		if err != nil {
			return err
		}

		for _, i := range order(muts, env.ranked) {
			m := muts[i]

			if res, ok, err := storedResult(ctx, tx, a.WorkspaceID, m.OpID); err != nil {
				return err
			} else if ok {
				results[i] = res
				continue
			}

			res, err := s.applyIsolated(ctx, tx, env, m, broken[i])
			if err != nil {
				return err
			}

			if err := storeConflicts(ctx, tx, a.WorkspaceID, res.Entity, res.UID, dev.DeviceID, res.Conflicts, now); err != nil {
				return err
			}

			if err := storeResult(ctx, tx, a.WorkspaceID, dev.DeviceID, res, now); err != nil {
				return err
			}

			results[i] = res
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// order sorts the indexes of a batch parents first (a category with a parent
// after the others of its entity), keeping the request order otherwise, and the
// request order of the mutations of one row always.
func order(muts []MutationIn, rank map[string]int) []int {
	idx := make([]int, len(muts))
	for i := range idx {
		idx[i] = i
	}

	// A row is a child when any of its mutations names a parent. All of its mutations then
	// weigh the same, so a create and a later edit of it keep their request order.
	children := map[string]bool{}

	for _, m := range muts {
		if parent, ok := m.Fields["parent_uid"]; ok && parent != nil && m.Op != "delete" {
			children[m.Entity+"/"+m.UID] = true
		}
	}

	weight := func(m MutationIn) int {
		w := rank[m.Entity] * 2
		if children[m.Entity+"/"+m.UID] && m.Op != "delete" {
			w++
		}

		if m.Op == "delete" {
			w += 1000
		}

		return w
	}

	slices.SortStableFunc(idx, func(a, b int) int { return weight(muts[a]) - weight(muts[b]) })

	return idx
}

var errRollback = errors.New("sync: rolled back")

// applyIsolated runs one mutation in a savepoint. A rejection (or an error of
// the data the client sent) rolls the savepoint back and becomes a result;
// anything else is a server failure and stops the batch.
func (s *Service) applyIsolated(ctx context.Context, tx *database.Tx, env pushEnv, m MutationIn, preRejected *mutationError) (Result, error) {
	res := Result{OpID: m.OpID, Entity: m.Entity, UID: m.UID, Conflicts: []ConflictOut{}, Effects: []EffectOut{}}

	if preRejected != nil {
		return rejected(res, preRejected), nil
	}

	err := tx.Savepoint(ctx, func(sp *database.Tx) error {
		out, err := s.apply(ctx, sp, env, m, res)
		if err != nil {
			return err
		}

		res = out
		if res.Status == StatusRejected {
			return errRollback
		}

		return nil
	})

	var merr *mutationError

	switch {
	case err == nil, errors.Is(err, errRollback):
		return res, nil
	case errors.As(err, &merr):
		return rejected(res, merr), nil
	case workspaces.IsUpgrade(err):
		return rejected(res, reject(CodeQuotaExceeded, "", "%s", errMessage(err))), nil
	case tx.UniqueViolation(err):
		code := CodeDuplicateKey
		if m.Entity == "category" || m.Entity == "tag" {
			code = CodeDuplicateName
		}

		return rejected(res, reject(code, "", "A live row with the same name or key exists.")), nil
	case finance.IsParentDeleted(err):
		return rejected(res, reject(changes.CodeParentDeleted, "", "Restore the account, category or goal it belongs to first.")), nil
	case finance.IsNameTaken(err):
		return rejected(res, reject(CodeDuplicateName, "", "That name is already taken.")), nil
	case finance.IsNotRestorable(err):
		return rejected(res, reject(CodeNotRestorable, "", "There is nothing to restore here.")), nil
	}

	return Result{}, err
}

func errMessage(err error) string {
	if he, ok := errors.AsType[*errs.HTTPError](err); ok {
		return he.Error()
	}

	return err.Error()
}

func rejected(res Result, e *mutationError) Result {
	res.Status = StatusRejected
	res.Error = &ErrorOut{Code: e.Code, Message: e.Message, Field: e.Field}
	res.Effects = []EffectOut{}

	return res
}
