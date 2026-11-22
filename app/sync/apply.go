package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// abilityFor is the role ability a mutation of an entity needs, the same one
// the web pages and the API ask for.
func abilityFor(entity string) string {
	switch entity {
	case "transaction", "recurrence", "goal_contribution":
		return workspaces.AbilityRecord
	}

	return workspaces.AbilityManageData
}

func findRow(ctx context.Context, tx *database.Tx, ws int64, entity, uid string) (StoredRow, bool, error) {
	ent, ok := changes.Lookup(entity)
	if !ok {
		return StoredRow{}, false, fmt.Errorf("sync: unknown entity %s", entity)
	}

	rows, err := selectRows(ctx, tx, ent, ws, "t.uid = ?", []any{uid}, "", 1)
	if err != nil || len(rows) == 0 {
		return StoredRow{}, false, err
	}

	return rows[0], true, nil
}

func maxSeq(ctx context.Context, tx *database.Tx, ws int64) (int64, error) {
	var seq int64

	err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(seq), 0) FROM changes WHERE workspace_id = ?", ws).Scan(&seq)

	return seq, err
}

func rowSeq(ctx context.Context, tx *database.Tx, ws int64, entity, uid string) (int64, error) {
	var seq int64

	err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(seq), 0) FROM changes WHERE workspace_id = ? AND entity = ? AND uid = ?", ws, entity, uid).Scan(&seq)

	return seq, err
}

// sinceOf reads the logged changes of one row after base.
func sinceOf(ctx context.Context, tx *database.Tx, ws int64, entity, uid string, base int64) ([]changes.Change, error) {
	rs, err := tx.QueryContext(ctx,
		"SELECT seq, op, fields FROM changes WHERE workspace_id = ? AND entity = ? AND uid = ? AND seq > ? ORDER BY seq",
		ws, entity, uid, base)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rs.Close() }()

	var out []changes.Change

	for rs.Next() {
		var (
			c   changes.Change
			raw string
		)

		if err := rs.Scan(&c.Seq, &c.Op, &raw); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(raw), &c.Fields); err != nil {
			return nil, fmt.Errorf("sync: change %d has bad fields: %w", c.Seq, err)
		}

		out = append(out, c)
	}

	return out, rs.Err()
}

func specsOf(e changes.Entity) []FieldSpec {
	fields, _ := Fields(e)

	return fields
}

// apply validates one mutation and, when it passes, merges it into the row.
// It returns a rejected Result for what is wrong with the data the client sent
// (the caller rolls the savepoint back) and an error only for a failure of the
// server.
func (s *Service) apply(ctx context.Context, tx *database.Tx, env pushEnv, m MutationIn, res Result) (Result, error) {
	ws := env.a.WorkspaceID

	if changes.ValidUID(m.OpID, env.now) != nil {
		return rejected(res, reject(CodeInvalidUID, "op_id", "op_id must be a ULID.")), nil
	}

	ent, ok := changes.Lookup(m.Entity)
	if !ok {
		return rejected(res, reject(CodeUnknownEntity, "entity", "%q is not a synced entity.", m.Entity)), nil
	}

	if changes.ValidUID(m.UID, env.now) != nil {
		return rejected(res, reject(CodeInvalidUID, "uid", "uid must be a ULID.")), nil
	}

	if !slices.Contains([]string{"upsert", "delete", "restore"}, m.Op) {
		return rejected(res, reject(CodeInvalidOp, "op", "op must be upsert, delete or restore.")), nil
	}

	if m.BaseSeq < 0 {
		return rejected(res, reject(CodeInvalidField, "base_seq", "base_seq must be zero or a seq you saw.")), nil
	}

	if !workspaces.RoleCan(env.a.Role, abilityFor(m.Entity)) {
		return rejected(res, reject(CodeForbidden, "", "Your role in this workspace cannot change %s rows.", m.Entity)), nil
	}

	if m.Op == "delete" && len(m.Fields) > 0 {
		return rejected(res, reject(CodeInvalidField, "fields", "A delete carries no fields.")), nil
	}

	cur, exists, err := findRow(ctx, tx, ws, m.Entity, m.UID)
	if err != nil {
		return Result{}, err
	}

	fields, merr := s.validateFields(ent, m.Fields, m.Op == "upsert" && !exists)
	if merr != nil {
		return rejected(res, merr), nil
	}

	if out, done, err := s.naturalKey(ctx, tx, ws, m, fields, exists, res); err != nil || done {
		return out, err
	}

	refs, err := s.resolve(ctx, tx, ws, ent, fields)
	if err != nil {
		return Result{}, err
	}

	if refs.tagErr != nil {
		return rejected(res, refs.tagErr), nil
	}

	if refs.ok() {
		if merr := s.relations(ctx, tx, ws, ent, cur, exists, m.UID, fields, refs, m.Op); merr != nil {
			return rejected(res, merr), nil
		}

		if err := s.settleStatement(ctx, tx, ws, ent, cur, exists, m.Op, fields, refs); err != nil {
			return Result{}, err
		}
	}

	var since []changes.Change

	if exists {
		if since, err = sinceOf(ctx, tx, ws, m.Entity, m.UID, m.BaseSeq); err != nil {
			return Result{}, err
		}
	}

	verdict := changes.Merge(
		changes.Row{Exists: exists, Deleted: cur.Deleted, Version: cur.Version, Fields: cur.Fields},
		changes.Mutation{
			Create: m.Op == "upsert", Delete: m.Op == "delete", Restore: m.Op == "restore", BaseSeq: m.BaseSeq,
			Device: env.dev.DeviceID, Fields: fields, Today: env.today, Parents: refs.states,
		}, since)

	res.Conflicts = conflictsOf(verdict)

	switch verdict.Outcome {
	case changes.OutcomeRejected:
		return rejected(res, reject(verdict.Code, verdict.Field, "The mutation was refused: %s.", verdict.Code)), nil
	case changes.OutcomeDropped:
		return rejected(res, reject(CodeEditedDeleted, "", "The row was deleted meanwhile: the edit was dropped.")), nil
	case changes.OutcomeNoop:
		res.Status = StatusDuplicate

		return s.fill(ctx, tx, ws, ent, res)
	}

	return s.write(ctx, tx, env, ent, m, cur, exists, verdict, refs, len(since) > 0, res)
}

// settleStatement makes the statement of an entry the one its card and date
// give, as every other write path does: what a device sends is not trusted, or
// a purchase could land on a statement already paid. A transfer leg keeps its
// value, because a card payment names the statement it pays.
func (s *Service) settleStatement(
	ctx context.Context, tx *database.Tx, ws int64, ent changes.Entity, cur StoredRow, exists bool, op string,
	fields map[string]any, refs refSet,
) error {
	if ent.Name != "transaction" || op != "upsert" {
		return nil
	}

	_, account := fields["account_uid"]
	_, date := fields["occurred_on"]
	_, sent := fields["statement_ref"]

	if !account && !date && !sent {
		return nil
	}

	eff := map[string]any{}
	if exists {
		maps.Copy(eff, cur.Fields)
	}

	maps.Copy(eff, fields)

	if t, _ := eff["transfer_uid"].(string); t != "" {
		return nil
	}

	accountID, ok := refs.ids["account_uid"]
	if !ok {
		uid, _ := eff["account_uid"].(string)

		row, found, err := findRow(ctx, tx, ws, "account", uid)
		if err != nil || !found {
			return err
		}

		accountID = row.ID
	}

	on, _ := eff["occurred_on"].(string)

	ref, err := s.Fin.StatementRefIn(ctx, tx, accountID, on)
	if err != nil {
		return err
	}

	var want any
	if ref.Valid {
		want = ref.V
	}

	if (exists && cur.Fields["statement_ref"] == want) || (!exists && want == nil) {
		delete(fields, "statement_ref")

		return nil
	}

	fields["statement_ref"] = want

	return nil
}

func conflictsOf(v changes.Result) []ConflictOut {
	out := []ConflictOut{}

	for _, c := range v.Conflicts {
		var applied any
		if c.Kind == models.ConflictField {
			applied = v.Apply[c.Field]
		}

		out = append(out, ConflictOut{Field: c.Field, Kind: c.Kind, LostValue: c.LostValue, AppliedValue: applied, LostBy: c.LostByDevice})
	}

	return out
}

// fill completes a result with the current row, its version and its newest seq.
func (s *Service) fill(ctx context.Context, tx *database.Tx, ws int64, ent changes.Entity, res Result) (Result, error) {
	row, ok, err := findRow(ctx, tx, ws, ent.Name, res.UID)
	if err != nil {
		return Result{}, err
	}

	if ok {
		res.Version = row.Version
		if !row.Deleted {
			res.Row = row.Wire()
		}
	}

	res.Seq, err = rowSeq(ctx, tx, ws, ent.Name, res.UID)

	return res, err
}

// naturalKey answers a create of a recurrence occurrence whose natural key
// (recurrence_uid, occurrence_on) already belongs to another row of the
// workspace, deleted ones included: the client adopts that row.
func (s *Service) naturalKey(
	ctx context.Context, tx *database.Tx, ws int64, m MutationIn, fields map[string]any, exists bool, res Result,
) (Result, bool, error) {
	if m.Entity != "transaction" || m.Op != "upsert" || exists {
		return res, false, nil
	}

	rule, _ := fields["recurrence_uid"].(string)
	on, _ := fields["occurrence_on"].(string)

	if rule == "" || on == "" {
		return res, false, nil
	}

	var other string

	err := tx.QueryRowContext(ctx,
		"SELECT uid FROM transactions WHERE workspace_id = ? AND recurrence_uid = ? AND occurrence_on = ? AND uid <> ?",
		ws, rule, on, m.UID).Scan(&other)
	if errors.Is(err, sql.ErrNoRows) {
		return res, false, nil
	}

	if err != nil {
		return Result{}, true, err
	}

	res.Status, res.ReplacedByUID, res.UID = StatusDuplicate, other, m.UID

	ent, _ := changes.Lookup("transaction")
	row, ok, err := findRow(ctx, tx, ws, "transaction", other)

	if err != nil {
		return Result{}, true, err
	}

	if ok {
		res.Version = row.Version
		if !row.Deleted {
			res.Row = row.Wire()
		}
	}

	if res.Seq, err = rowSeq(ctx, tx, ws, ent.Name, other); err != nil {
		return Result{}, true, err
	}

	return res, true, nil
}

// refSet is what resolving the uids of a mutation found: the state of each
// parent for Merge, the row ids to write, and the parent rows for the checks.
type refSet struct {
	states map[string]changes.ParentState
	ids    map[string]int64
	rows   map[string]StoredRow
	tags   []int64
	// tagErr is a tag of tag_uids that is missing or deleted.
	tagErr *mutationError
}

func (r refSet) ok() bool {
	for _, st := range r.states {
		if st != changes.ParentOK {
			return false
		}
	}

	return r.tagErr == nil
}

// resolve looks up every uid the fields point at, in the workspace of the
// token only: a uid of another workspace is simply not found, the same answer
// as a uid that does not exist.
func (s *Service) resolve(ctx context.Context, tx *database.Tx, ws int64, ent changes.Entity, fields map[string]any) (refSet, error) {
	out := refSet{states: map[string]changes.ParentState{}, ids: map[string]int64{}, rows: map[string]StoredRow{}}

	for _, f := range specsOf(ent) {
		uid, _ := fields[f.Name].(string)

		switch {
		case f.Type == TypeRef && uid != "":
			row, found, err := findRow(ctx, tx, ws, f.Ref, uid)
			if err != nil {
				return out, err
			}

			switch {
			case !found:
				out.states[f.Name] = changes.ParentMissing
			case row.Deleted:
				out.states[f.Name] = changes.ParentDeleted
			default:
				out.states[f.Name], out.ids[f.Name], out.rows[f.Name] = changes.ParentOK, row.ID, row
			}
		case f.Type == TypeUIDList:
			list, _ := fields[f.Name].([]any)

			for _, item := range list {
				tagUID, _ := item.(string)

				row, found, err := findRow(ctx, tx, ws, "tag", tagUID)
				if err != nil {
					return out, err
				}

				switch {
				case !found:
					out.tagErr = reject(changes.CodeUnknownReference, f.Name, "A tag in %s does not exist.", f.Name)
				case row.Deleted:
					out.tagErr = reject(changes.CodeParentDeleted, f.Name, "A tag in %s is deleted.", f.Name)
				default:
					out.tags = append(out.tags, row.ID)
				}
			}
		}
	}

	return out, nil
}

// relations are the rules between fields that the services enforce and the
// registry cannot express: kinds that must agree, archived parents, nesting.
func (s *Service) relations(
	ctx context.Context, tx *database.Tx, ws int64, ent changes.Entity, cur StoredRow, exists bool,
	uid string, fields map[string]any, refs refSet, op string,
) *mutationError {
	eff := maps.Clone(cur.Fields)
	if eff == nil || !exists {
		eff = map[string]any{}
	}

	maps.Copy(eff, fields)

	_, accountSet := fields["account_uid"]
	_, categorySet := fields["category_uid"]
	_, kindSet := fields["kind"]

	switch ent.Name {
	case "transaction", "recurrence":
		if r, ok := refs.rows["account_uid"]; ok && accountSet && r.Fields["archived_at"] != nil && op != "delete" {
			return reject(CodeInvalidState, "account_uid", "The account is archived and takes no new entries.")
		}

		if (categorySet || kindSet) && eff["category_uid"] != nil {
			cat, found, err := s.category(ctx, tx, ws, refs, eff)
			if err != nil {
				return reject(CodeInvalidState, "category_uid", "The category cannot be read.")
			}

			if found && cat.Fields["kind"] != eff["kind"] {
				return reject(CodeInvalidState, "category_uid", "The category must be of the same kind as the entry.")
			}

			if found && categorySet && cat.Fields["archived_at"] != nil {
				return reject(CodeInvalidState, "category_uid", "The category is archived and takes no new entries.")
			}
		}

		if ent.Name == "recurrence" {
			end, _ := eff["end_date"].(string)
			start, _ := eff["start_date"].(string)

			if end != "" && start != "" && end < start {
				return reject(CodeInvalidField, "end_date", "end_date cannot be before start_date.")
			}
		}
	case "category":
		if p, ok := refs.rows["parent_uid"]; ok {
			if p.UID == uid || p.Fields["parent_uid"] != nil || p.Fields["kind"] != eff["kind"] {
				return reject(CodeInvalidState, "parent_uid", "A parent is a top-level category of the same kind.")
			}
		}
	case "budget":
		if c, ok := refs.rows["category_uid"]; ok && c.Fields["kind"] != models.KindExpense {
			return reject(CodeInvalidState, "category_uid", "A budget needs an expense category.")
		}
	case "credit_card":
		for _, name := range []string{"account_uid", "payment_account_uid"} {
			if a, ok := refs.rows[name]; ok && name == "account_uid" && a.Fields["kind"] != models.AccountCreditCard {
				return reject(CodeInvalidState, name, "A card belongs to an account of kind credit_card.")
			}
		}
	case "goal_contribution":
		if g, ok := refs.rows["goal_uid"]; ok && !exists && g.Fields["archived_at"] != nil {
			return reject(CodeInvalidState, "goal_uid", "Unarchive the goal before adding to it.")
		}
	}

	return nil
}

// category finds the category the effective fields name: the one just resolved
// or, when only the kind changed, the stored one.
func (s *Service) category(ctx context.Context, tx *database.Tx, ws int64, refs refSet, eff map[string]any) (StoredRow, bool, error) {
	if r, ok := refs.rows["category_uid"]; ok {
		return r, true, nil
	}

	uid, _ := eff["category_uid"].(string)
	if uid == "" {
		return StoredRow{}, false, nil
	}

	return findRow(ctx, tx, ws, "category", uid)
}

// write stores the merged mutation: restore, columns, tags and delete, then
// reads the row back for the result.
func (s *Service) write(
	ctx context.Context, tx *database.Tx, env pushEnv, ent changes.Entity, m MutationIn, cur StoredRow, exists bool,
	verdict changes.Result, refs refSet, concurrent bool, res Result,
) (Result, error) {
	ws := env.a.WorkspaceID

	before, err := maxSeq(ctx, tx, ws)
	if err != nil {
		return Result{}, err
	}

	columns := maps.Clone(verdict.Apply)
	del, hasDel := columns["deleted_at"].(bool)

	delete(columns, "deleted_at")

	restoring := hasDel && !del

	if restoring && cur.DeletedAt.Before(env.now.Add(-finance.RestoreWindow)) {
		return rejected(res, reject(CodeRestoreWindow, "", "The row was deleted more than 30 days ago and cannot be restored.")), nil
	}

	eff := map[string]any{}
	if exists {
		maps.Copy(eff, cur.Fields)
	}

	maps.Copy(eff, columns)

	if err := s.quota(ctx, tx, env, ent.Name, exists && !cur.Deleted, cur, eff); err != nil {
		return Result{}, err
	}

	id := cur.ID

	if restoring {
		if err := s.Fin.RestoreIn(ctx, tx, ent.Name, id); err != nil {
			return Result{}, err
		}
	}

	switch {
	case !exists:
		if id, err = s.insert(ctx, tx, env, ent, m.UID, columns, refs); err != nil {
			return Result{}, err
		}
	case len(columns) > 0:
		if err := s.update(ctx, tx, env, ent, id, columns, refs); err != nil {
			return Result{}, err
		}
	}

	if _, ok := columns["tag_uids"]; ok && ent.Name == "transaction" {
		if err := replaceTags(ctx, tx, ws, id, refs.tags, env.now); err != nil {
			return Result{}, err
		}
	}

	if hasDel && del {
		if err := s.Fin.DeleteIn(ctx, tx, ent.Name, id); err != nil {
			return Result{}, err
		}
	}

	res.Status = StatusApplied
	if (concurrent && m.Op != "delete") || len(res.Conflicts) > 0 {
		res.Status = StatusMerged
	}

	if res, err = s.fill(ctx, tx, ws, ent, res); err != nil {
		return Result{}, err
	}

	res.Effects, err = effects(ctx, tx, ws, before, ent.Name, m.UID)

	return res, err
}

func effects(ctx context.Context, tx *database.Tx, ws, before int64, entity, uid string) ([]EffectOut, error) {
	rs, err := tx.QueryContext(ctx,
		`SELECT seq, entity, uid, op FROM changes WHERE workspace_id = ? AND seq > ? AND NOT (entity = ? AND uid = ?)
		 ORDER BY seq LIMIT ?`, ws, before, entity, uid, maxEffects)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rs.Close() }()

	out := []EffectOut{}

	for rs.Next() {
		var e EffectOut
		if err := rs.Scan(&e.Seq, &e.Entity, &e.UID, &e.Op); err != nil {
			return nil, err
		}

		out = append(out, e)
	}

	return out, rs.Err()
}

// dbValue turns a canonical wire value into what the column holds.
func dbValue(f FieldSpec, v any, refs refSet) (any, error) {
	if v == nil {
		return nil, nil //nolint:nilnil // a null column
	}

	switch f.Type {
	case TypeRef:
		return refs.ids[f.Name], nil
	case TypeTimestamp:
		str, _ := v.(string)

		return time.Parse(time.RFC3339, str)
	case TypeUIDList:
		raw, err := json.Marshal(v)

		return string(raw), err
	}

	return v, nil
}

func (s *Service) insert(ctx context.Context, tx *database.Tx, env pushEnv, ent changes.Entity, uid string, columns map[string]any, refs refSet) (int64, error) {
	cols := []string{"workspace_id", "uid", "created_at", "updated_at"}
	args := []any{env.a.WorkspaceID, uid, env.now, env.now}

	for _, f := range specsOf(ent) {
		v, ok := columns[f.Name]
		if !ok {
			continue
		}

		dv, err := dbValue(f, v, refs)
		if err != nil {
			return 0, err
		}

		cols, args = append(cols, f.Column), append(args, dv)
	}

	res, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		ent.Table, strings.Join(cols, ", "), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ")), args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (s *Service) update(ctx context.Context, tx *database.Tx, env pushEnv, ent changes.Entity, id int64, columns map[string]any, refs refSet) error {
	sets := []string{"updated_at = ?"}
	args := []any{env.now}

	for _, f := range specsOf(ent) {
		v, ok := columns[f.Name]
		if !ok {
			continue
		}

		dv, err := dbValue(f, v, refs)
		if err != nil {
			return err
		}

		sets, args = append(sets, f.Column+" = ?"), append(args, dv)
	}

	args = append(args, id, env.a.WorkspaceID)
	_, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET %s WHERE id = ? AND workspace_id = ?", ent.Table, strings.Join(sets, ", ")), args...)

	return err
}

// replaceTags makes the join rows of a transaction equal to its tag_uids.
func replaceTags(ctx context.Context, tx *database.Tx, ws, txID int64, tagIDs []int64, now time.Time) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM transaction_tags WHERE workspace_id = ? AND transaction_id = ?", ws, txID); err != nil {
		return err
	}

	for _, tagID := range tagIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO transaction_tags (workspace_id, transaction_id, tag_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			ws, txID, tagID, now, now); err != nil {
			return err
		}
	}

	return nil
}

// quota enforces the plan the way the services do, inside the batch
// transaction and after the workspace lock, so two pushes cannot both pass.
func (s *Service) quota(
	ctx context.Context, tx *database.Tx, env pushEnv, entity string, wasLive bool, cur StoredRow, eff map[string]any,
) error {
	switch entity {
	case "account":
		if eff["archived_at"] != nil {
			return nil
		}

		if wasLive && cur.Fields["archived_at"] == nil {
			return nil
		}

		lim, err := s.WS.LimitsIn(ctx, tx)
		if err != nil {
			return err
		}

		var n int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM accounts WHERE workspace_id = ? AND deleted_at IS NULL AND archived_at IS NULL", env.a.WorkspaceID).Scan(&n); err != nil {
			return err
		}

		return lim.CheckAccounts(n + 1)
	case "budget":
		if wasLive {
			return nil
		}

		lim, err := s.WS.LimitsIn(ctx, tx)
		if err != nil {
			return err
		}

		return lim.CheckBudgets()
	case "transaction":
		if eff["transfer_uid"] != nil {
			return nil
		}

		on, _ := eff["occurred_on"].(string)
		if len(on) < 7 {
			return nil
		}

		was, _ := cur.Fields["occurred_on"].(string)
		if wasLive && cur.Fields["transfer_uid"] == nil && len(was) >= 7 && was[:7] == on[:7] {
			return nil
		}

		lim, err := s.WS.LimitsIn(ctx, tx)
		if err != nil {
			return err
		}

		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM transactions WHERE workspace_id = ? AND deleted_at IS NULL AND transfer_uid IS NULL
			 AND substr(occurred_on, 1, 7) = ?`, env.a.WorkspaceID, on[:7]).Scan(&n); err != nil {
			return err
		}

		return lim.CheckTransactions(n + 1)
	}

	return nil
}
