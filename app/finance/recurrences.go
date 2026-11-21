package finance

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	// generateCap is how many occurrences one run creates for one rule: an
	// installation that was off for a year catches up over a few runs.
	generateCap = 500
	// backfillCap is how many past occurrences a new rule may create at once.
	backfillCap = 36
	maxInterval = 999
	maxCount    = 9999
	// A rule or a purchase starts and ends between these years: a date far
	// outside them is a mistake or an attack, never a bill.
	minRuleYear = 1900
	maxRuleYear = 2999
)

// saneYear reports whether an ISO date is in the years rules and installments accept.
func saneYear(date string) bool {
	d, err := time.Parse(isoDate, date)

	return err == nil && d.Year() >= minRuleYear && d.Year() <= maxRuleYear
}

var (
	errFrequency = errs.New(http.StatusUnprocessableEntity, "The frequency must be daily, weekly, monthly or yearly.")
	errInterval  = errs.New(http.StatusUnprocessableEntity, "Repeat every 1 to 999 days, weeks, months or years.")
	errEndDate   = errs.New(http.StatusUnprocessableEntity, "The end date must look like 2026-11-30 and not be before the start date.")
	errMaxCount  = errs.New(http.StatusUnprocessableEntity, "The number of occurrences must be between 1 and 9999.")
	errRuleYear  = errs.New(http.StatusUnprocessableEntity, "Dates must be between the years 1900 and 2999.")
	errBackfill  = errs.New(http.StatusUnprocessableEntity,
		"That start date is more than 36 occurrences in the past. Choose a later start date or turn off the backfill.")
)

// RecurrenceInput is a rule as the form sends it. Backfill says whether a
// start date in the past creates the occurrences already due (create only).
type RecurrenceInput struct {
	AccountID   int64
	CategoryID  int64
	Kind        string
	Amount      string
	Description string
	Frequency   string
	Interval    int
	StartDate   string
	EndDate     string
	MaxCount    int
	Active      bool
	Backfill    bool
}

// ruleOf is the pure rule of a stored recurrence.
func ruleOf(r models.Recurrence) Rule {
	out := Rule{Frequency: r.Frequency, Interval: r.Interval, StartDate: r.StartDate, EndDate: r.EndDate.V}
	if r.MaxCount.Valid {
		out.MaxCount = int(r.MaxCount.V)
	}

	return out
}

// recurrenceFields validates in. keepAccount and keepCategory are what the
// stored rule already uses (0 for a new one): they stay editable after an
// archive, like a transaction's.
func recurrenceFields(ctx context.Context, ex database.Executor, in RecurrenceInput, keepAccount, keepCategory int64) (models.Recurrence, error) {
	tx, err := transactionInput(ctx, ex, TxInput{
		AccountID: in.AccountID, CategoryID: in.CategoryID, Kind: in.Kind, Amount: in.Amount,
		Description: in.Description, Date: in.StartDate,
	}, keepAccount, keepCategory)
	if err != nil {
		return models.Recurrence{}, err
	}

	if !validFrequency(in.Frequency) {
		return models.Recurrence{}, errFrequency
	}

	if !saneYear(in.StartDate) || (in.EndDate != "" && validDate(in.EndDate) && !saneYear(in.EndDate)) {
		return models.Recurrence{}, errRuleYear
	}

	if in.Interval < 1 || in.Interval > maxInterval {
		return models.Recurrence{}, errInterval
	}

	out := models.Recurrence{
		AccountID: in.AccountID, CategoryID: tx.CategoryID, Kind: in.Kind, AmountCents: tx.AmountCents,
		Description: tx.Description, Frequency: in.Frequency, Interval: in.Interval, StartDate: in.StartDate,
		Active: in.Active,
	}

	if in.EndDate != "" {
		if !validDate(in.EndDate) || in.EndDate < in.StartDate {
			return models.Recurrence{}, errEndDate
		}

		out.EndDate = sql.Null[string]{V: in.EndDate, Valid: true}
	}

	if in.MaxCount != 0 {
		if in.MaxCount < 1 || in.MaxCount > maxCount {
			return models.Recurrence{}, errMaxCount
		}

		out.MaxCount = sql.Null[int64]{V: int64(in.MaxCount), Valid: true}
	}

	return out, nil
}

func nullable[T any](v sql.Null[T]) any {
	if !v.Valid {
		return nil
	}

	return v.V
}

func shiftDay(date string, n int) string {
	d, err := time.Parse(isoDate, date)
	if err != nil {
		return date
	}

	return d.AddDate(0, 0, n).Format(isoDate)
}

func dayBefore(date string) string { return shiftDay(date, -1) }
func nextDay(date string) string   { return shiftDay(date, 1) }

// Recurrences lists the rules of the workspace.
func (s *Service) Recurrences(ctx context.Context) ([]models.Recurrence, error) {
	return workspaces.Scoped[models.Recurrence](ctx, s.DB).OrderBy("start_date", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
}

// CreateRecurrence stores a rule and, at once, generates what is due: a rule
// that starts today posts today. A start date in the past creates the past
// occurrences when in.Backfill is set (at most 36, more is refused) and
// skips them otherwise.
func (s *Service) CreateRecurrence(ctx context.Context, in RecurrenceInput) (models.Recurrence, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Recurrence{}, err
	}

	today := s.nowIn(ws).Format(isoDate)
	in.Active = true

	var rule models.Recurrence

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		rule, err = recurrenceFields(ctx, tx, in, 0, 0)
		if err != nil {
			return err
		}

		if in.StartDate < today {
			if in.Backfill {
				due := ruleOf(rule).Occurrences(in.StartDate, today, backfillCap+1)
				if len(due) > backfillCap {
					return errBackfill
				}
			} else {
				rule.GeneratedThrough = sql.Null[string]{V: dayBefore(today), Valid: true}
			}
		}

		return orm.Create(ctx, tx, &rule)
	})
	if err != nil {
		return models.Recurrence{}, err
	}

	if _, err := s.generateRule(ctx, ws, rule.ID, today); err != nil {
		return models.Recurrence{}, err
	}

	return reload[models.Recurrence](ctx, s.DB, rule.ID)
}

// UpdateRecurrence changes a rule. Only the occurrences that are not
// generated yet follow the change; generated transactions keep their values.
// Resuming a paused rule does not post what was due while it was paused.
func (s *Service) UpdateRecurrence(ctx context.Context, id int64, in RecurrenceInput) (models.Recurrence, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Recurrence{}, err
	}

	today := s.nowIn(ws).Format(isoDate)

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		cur, err := workspaces.Scoped[models.Recurrence](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		row, err := recurrenceFields(ctx, tx, in, cur.AccountID, cur.CategoryID.V)
		if err != nil {
			return err
		}

		set := orm.Set{
			"account_id": row.AccountID, "category_id": nullable(row.CategoryID), "kind": row.Kind,
			"amount_cents": row.AmountCents, "description": row.Description, "frequency": row.Frequency,
			"interval": row.Interval, "start_date": row.StartDate, "end_date": nullable(row.EndDate),
			"max_count": nullable(row.MaxCount), "active": row.Active, "last_error": nil,
		}

		if row.Active && !cur.Active && (!cur.GeneratedThrough.Valid || cur.GeneratedThrough.V < dayBefore(today)) {
			set["generated_through"] = dayBefore(today)
		}

		_, err = workspaces.Scoped[models.Recurrence](ctx, tx).Where("id", id).Update(ctx, set)

		return err
	})
	if err != nil {
		return models.Recurrence{}, err
	}

	if _, err := s.generateRule(ctx, ws, id, today); err != nil {
		return models.Recurrence{}, err
	}

	return reload[models.Recurrence](ctx, s.DB, id)
}

// DeleteRecurrence soft-deletes a rule and keeps what it generated. With
// deleteFuture the generated transactions dated after today go too.
func (s *Service) DeleteRecurrence(ctx context.Context, id int64, deleteFuture bool) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()
	today := s.nowIn(ws).Format(isoDate)

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		rule, err := workspaces.Scoped[models.Recurrence](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		if err := softDelete[models.Recurrence](ctx, tx, id, at); err != nil {
			return err
		}

		if !deleteFuture {
			return nil
		}

		_, err = workspaces.Scoped[models.Transaction](ctx, tx).Where("recurrence_uid", rule.UID).
			Where("occurred_on", ">", today).Update(ctx, orm.Set{"deleted_at": at})

		return err
	})
}

// RestoreRecurrence brings back a rule deleted less than RestoreWindow ago
// whose account is alive. What fell due while it was deleted is not posted.
func (s *Service) RestoreRecurrence(ctx context.Context, id int64) (models.Recurrence, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Recurrence{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		r, err := trashed[models.Recurrence](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if err := requireLive[models.Account](ctx, tx, r.AccountID); err != nil {
			return err
		}

		set := orm.Set{"deleted_at": nil}

		if r.CategoryID.Valid {
			live, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", r.CategoryID.V).Exists(ctx)
			if err != nil {
				return err
			}

			if !live {
				set["category_id"] = nil
			}
		}

		if yesterday := dayBefore(s.nowIn(ws).Format(isoDate)); !r.GeneratedThrough.Valid || r.GeneratedThrough.V < yesterday {
			set["generated_through"] = yesterday
		}

		_, err = workspaces.Scoped[models.Recurrence](ctx, tx).OnlyTrashed().Where("id", id).Update(ctx, set)

		return err
	})
	if err != nil {
		return models.Recurrence{}, err
	}

	return reload[models.Recurrence](ctx, s.DB, id)
}

// GenerateDue creates the transactions that fell due for every active rule
// of the workspace of ctx and returns how many it made. A rule that fails
// (the Free quota, an archived account) keeps the message in last_error and
// is retried on the next run without blocking the others.
func (s *Service) GenerateDue(ctx context.Context) (int, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return 0, err
	}

	today := s.nowIn(ws).Format(isoDate)

	ids, err := workspaces.Scoped[models.Recurrence](ctx, s.DB).Where("active", true).OrderBy("id", orm.Asc).Pluck[int64](ctx, "id")
	if err != nil {
		return 0, err
	}

	var (
		total  int
		failed error
	)

	for _, id := range ids {
		n, err := s.generateRule(ctx, ws, id, today)
		total += n
		failed = errors.Join(failed, err)
	}

	return total, failed
}

// GenerateAllDue runs GenerateDue for every workspace; one that fails does
// not starve the others.
func (s *Service) GenerateAllDue(ctx context.Context) error {
	all, err := orm.QueryOf[models.Workspace](s.DB).Get(ctx)
	if err != nil {
		return err
	}

	var failed error

	for _, ws := range all {
		if _, err := s.GenerateDue(workspaces.WithTenant(ctx, ws.ID)); err != nil {
			failed = errors.Join(failed, err)
		}
	}

	return failed
}

// generateRule makes the occurrences of one rule after its watermark up to
// today, in order, through persistTransaction (the workspace lock and the
// atomic quota check). Everything, the watermark included, is one database
// transaction. The rule is read again under the lock, so two concurrent runs
// never make the same occurrence; the natural key is the second guard, and
// it also holds a deleted occurrence in place.
func (s *Service) generateRule(ctx context.Context, ws models.Workspace, id int64, today string) (int, error) {
	var created []models.Transaction

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		created = created[:0]

		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		rule, err := workspaces.Scoped[models.Recurrence](ctx, tx).Where("id", id).First(ctx)
		if errors.Is(err, orm.ErrNotFound) || (err == nil && !rule.Active) {
			return nil
		}

		if err != nil {
			return err
		}

		from := rule.StartDate
		if rule.GeneratedThrough.Valid {
			from = nextDay(rule.GeneratedThrough.V)
		}

		var (
			done    string
			failure error
		)

		for _, date := range ruleOf(rule).Occurrences(from, today, generateCap) {
			made, err := s.makeOccurrence(ctx, tx, ws.ID, rule, date)
			if err != nil {
				if _, ok := errors.AsType[*errs.HTTPError](err); !ok {
					return err
				}

				failure = err

				break
			}

			if made.ID != 0 {
				created = append(created, made)
			}

			done = date
		}

		set := orm.Set{}
		if done != "" {
			set["generated_through"] = done
		}

		switch {
		case failure != nil && (!rule.LastError.Valid || rule.LastError.V != lastErrorText(failure)):
			set["last_error"] = lastErrorText(failure)
		case failure == nil && rule.LastError.Valid:
			set["last_error"] = nil
		}

		if len(set) == 0 {
			return nil
		}

		_, err = workspaces.Scoped[models.Recurrence](ctx, tx).Where("id", id).Update(ctx, set)

		return err
	})
	if err != nil {
		return 0, err
	}

	for _, t := range created {
		// The transactions are committed: a failed alert mail must not undo
		// them or stop the next rule, and the next crossing alerts again.
		_ = s.notifyBudget(ctx, t, models.Transaction{})
	}

	return len(created), nil
}

func lastErrorText(err error) string {
	if e, ok := errors.AsType[*errs.HTTPError](err); ok {
		return strings.TrimSpace(e.Message)
	}

	return err.Error()
}

// makeOccurrence creates the transaction of one occurrence unless a row
// (deleted ones included) already holds its natural key, in which case it
// returns the zero transaction.
func (s *Service) makeOccurrence(ctx context.Context, tx *database.Tx, workspaceID int64, rule models.Recurrence, date string) (models.Transaction, error) {
	held, err := workspaces.Scoped[models.Transaction](ctx, tx).WithTrashed().
		Where("recurrence_uid", rule.UID).Where("occurrence_on", date).Exists(ctx)
	if err != nil || held {
		return models.Transaction{}, err
	}

	var out, previous models.Transaction

	in := TxInput{
		AccountID: rule.AccountID, CategoryID: rule.CategoryID.V, Kind: rule.Kind, Amount: plainAmount(rule.AmountCents),
		Description: rule.Description, Date: date, recurrenceUID: rule.UID, occurrenceOn: date,
	}

	if err := s.persistTransaction(ctx, tx, workspaceID, 0, in, &out, &previous); err != nil {
		return models.Transaction{}, err
	}

	return out, nil
}
