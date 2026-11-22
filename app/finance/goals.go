package finance

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	maxGoals         = 100
	maxContributions = 2000
	maxGoalNameLen   = 120
	maxNoteLen       = 200
	// maxProjectionMonths is how far ahead a projected date is shown; beyond
	// it the pace is too slow to mean anything.
	maxProjectionMonths = 1200
)

var (
	errGoalName     = errs.New(http.StatusUnprocessableEntity, "The name must have 1 to 120 characters.")
	errGoalTarget   = errs.New(http.StatusUnprocessableEntity, "The target must be a positive amount like 1200.00.")
	errGoalDate     = errs.New(http.StatusUnprocessableEntity, "The date must look like 2026-11-30, between the years 1900 and 2999.")
	errGoalNote     = errs.New(http.StatusUnprocessableEntity, "The note must have at most 200 characters.")
	errGoalAmount   = errs.New(http.StatusUnprocessableEntity, "The amount must be a non-zero number like 50.00 (negative to withdraw).")
	errTooManyGoals = errs.New(http.StatusUnprocessableEntity, "A workspace holds at most 100 goals. Delete one first.")
	errTooManyParts = errs.New(http.StatusUnprocessableEntity, "A goal holds at most 2000 contributions.")
)

// GoalInput is a goal as the form sends it. TargetDate is optional.
type GoalInput struct {
	Name       string
	Target     string
	TargetDate string
}

// ContributionInput is an amount put towards a goal; a negative amount is a
// withdrawal. An empty Date means today.
type ContributionInput struct {
	Amount string
	Date   string
	Note   string
}

// GoalStatus is a goal with its progress.
type GoalStatus struct {
	ID          int64  `json:"id"`
	UID         string `json:"uid"`
	Name        string `json:"name"`
	TargetCents int64  `json:"targetCents"`
	TargetDate  string `json:"targetDate"`
	Archived    bool   `json:"archived"`

	SavedCents     int64 `json:"savedCents"`
	RemainingCents int64 `json:"remainingCents"`
	Percent        int   `json:"percent"`
	Reached        bool  `json:"reached"`
	Contributions  int   `json:"contributions"`

	// MonthsLeft and RequiredMonthlyCents are set for a goal with a future
	// target date that is not reached; Overdue marks a target date that has
	// passed with the goal not reached.
	MonthsLeft           int   `json:"monthsLeft"`
	RequiredMonthlyCents int64 `json:"requiredMonthlyCents"`
	Overdue              bool  `json:"overdue"`
	// ProjectedDate is when the average pace of the last three full months
	// would reach the target; empty when there are fewer than two
	// contributions or the pace is not positive.
	ProjectedDate string `json:"projectedDate"`
}

// satAddSigned adds two amounts, clamping at +-spentCap.
func satAddSigned(a, b int64) int64 {
	return max(-spentCap, min(spentCap, a+b)) // both are within +-spentCap, so the sum fits an int64
}

// monthsUntil is how many whole months, counted with AddMonths (a day that
// does not exist in a month clamps to its last day), fit between from and
// target: the largest n with AddMonths(from, n) not after target. It is
// negative or zero when target is before from.
func monthsUntil(from, target time.Time) int {
	n := (target.Year()-from.Year())*12 + int(target.Month()) - int(from.Month())

	for n > 0 && timeutil.AddMonths(from, n).After(target) {
		n--
	}

	return n
}

// requiredMonthly is what must be put aside each month to reach the target
// by the target date, rounded up to the cent, and the months it is spread
// over: at least one while the date has not passed. ok is false for a passed
// date.
func requiredMonthly(remaining int64, today, target time.Time) (cents int64, months int, ok bool) {
	if target.Before(today) {
		return remaining, 0, false
	}

	months = max(1, monthsUntil(today, target))
	n := int64(months)

	return (remaining + n - 1) / n, months, true
}

// projectedDate is the date the average monthly pace would reach remaining;
// ok is false without a positive pace.
func projectedDate(remaining, monthlyPace int64, today time.Time) (time.Time, bool) {
	if remaining <= 0 || monthlyPace <= 0 {
		return time.Time{}, false
	}

	months := (remaining + monthlyPace - 1) / monthlyPace
	if months > maxProjectionMonths {
		return time.Time{}, false
	}

	return timeutil.AddMonths(today, int(months)), true
}

// goalStatus derives the progress of one goal from its contributions, as of
// today.
func goalStatus(g models.Goal, parts []models.GoalContribution, today time.Time) GoalStatus {
	st := GoalStatus{
		ID: g.ID, UID: g.UID, Name: g.Name, TargetCents: g.TargetCents, TargetDate: g.TargetDate.V,
		Archived: g.ArchivedAt.Valid, Contributions: len(parts),
	}

	var saved, recent int64

	firstMonth := timeutil.AddMonths(time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC), -3).Format(isoDate)
	thisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).Format(isoDate)

	for _, p := range parts {
		saved = satAddSigned(saved, p.AmountCents)

		// Dates are ISO and were checked on write; they compare as text.
		if p.OnDate >= firstMonth && p.OnDate < thisMonth {
			recent = satAddSigned(recent, p.AmountCents)
		}
	}

	st.SavedCents = saved
	st.Reached = saved >= g.TargetCents
	st.RemainingCents = max(0, g.TargetCents-saved)
	st.Percent = int(max(0, saved) * 100 / g.TargetCents)

	if st.Reached {
		return st
	}

	if g.TargetDate.Valid {
		if target, err := time.Parse(isoDate, g.TargetDate.V); err == nil {
			if cents, months, ok := requiredMonthly(st.RemainingCents, today, target); ok {
				st.RequiredMonthlyCents, st.MonthsLeft = cents, months
			} else {
				st.Overdue = true
			}
		}
	}

	if len(parts) >= 2 {
		if d, ok := projectedDate(st.RemainingCents, recent/3, today); ok {
			st.ProjectedDate = d.Format(isoDate)
		}
	}

	return st
}

func goalFields(in GoalInput) (name string, target int64, date sql.Null[string], err error) {
	name = strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxGoalNameLen {
		return "", 0, date, errGoalName
	}

	target, perr := ParseAmount(in.Target)
	if perr != nil || target <= 0 {
		return "", 0, date, errGoalTarget
	}

	if d := strings.TrimSpace(in.TargetDate); d != "" {
		if !saneYear(d) {
			return "", 0, date, errGoalDate
		}

		date = sql.Null[string]{V: d, Valid: true}
	}

	return name, target, date, nil
}

func (s *Service) todayTime(ctx context.Context) time.Time {
	t, _ := time.Parse(isoDate, s.Today(ctx))

	return t
}

// Goals lists the goals with their progress, open ones first, then the
// archived ones.
func (s *Service) Goals(ctx context.Context) ([]GoalStatus, error) {
	goals, err := workspaces.Scoped[models.Goal](ctx, s.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil || len(goals) == 0 {
		return []GoalStatus{}, err
	}

	parts, err := workspaces.Scoped[models.GoalContribution](ctx, s.DB).OrderBy("on_date", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	byGoal := map[int64][]models.GoalContribution{}
	for _, p := range parts {
		byGoal[p.GoalID] = append(byGoal[p.GoalID], p)
	}

	today := s.todayTime(ctx)
	open, archived := make([]GoalStatus, 0, len(goals)), []GoalStatus{}

	for _, g := range goals {
		st := goalStatus(g, byGoal[g.ID], today)
		if st.Archived {
			archived = append(archived, st)
		} else {
			open = append(open, st)
		}
	}

	return append(open, archived...), nil
}

// GoalContributions lists the contributions of a goal, newest first.
func (s *Service) GoalContributions(ctx context.Context, goalID int64) ([]models.GoalContribution, error) {
	if ok, err := workspaces.Scoped[models.Goal](ctx, s.DB).Where("id", goalID).Exists(ctx); err != nil || !ok {
		return nil, firstErr(err, errs.ErrNotFound)
	}

	return workspaces.Scoped[models.GoalContribution](ctx, s.DB).Where("goal_id", goalID).
		OrderBy("on_date", orm.Desc).OrderBy("id", orm.Desc).Get(ctx)
}

// CreateGoal adds a goal. A workspace holds at most 100.
func (s *Service) CreateGoal(ctx context.Context, in GoalInput) (models.Goal, error) {
	name, target, date, err := goalFields(in)
	if err != nil {
		return models.Goal{}, err
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Goal{}, err
	}

	g := models.Goal{Name: name, TargetCents: target, TargetDate: date}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		n, err := workspaces.Scoped[models.Goal](ctx, tx).Count(ctx)
		if err != nil {
			return err
		}

		if n >= maxGoals {
			return errTooManyGoals
		}

		return orm.Create(ctx, tx, &g)
	})
	if err != nil {
		return models.Goal{}, err
	}

	return reload[models.Goal](ctx, s.DB, g.ID)
}

// UpdateGoal changes the name, the target and the target date of a goal.
func (s *Service) UpdateGoal(ctx context.Context, id int64, in GoalInput) (models.Goal, error) {
	name, target, date, err := goalFields(in)
	if err != nil {
		return models.Goal{}, err
	}

	var td any
	if date.Valid {
		td = date.V
	}

	n, err := workspaces.Scoped[models.Goal](ctx, s.DB).Where("id", id).
		Update(ctx, orm.Set{"name": name, "target_cents": target, "target_date": td})
	if err != nil {
		return models.Goal{}, err
	}

	if n == 0 {
		return models.Goal{}, errs.ErrNotFound
	}

	return reload[models.Goal](ctx, s.DB, id)
}

// ArchiveGoal hides a goal from the open list; its contributions stay.
func (s *Service) ArchiveGoal(ctx context.Context, id int64) (models.Goal, error) {
	return s.setGoalArchived(ctx, id, s.now().UTC())
}

// UnarchiveGoal brings an archived goal back to the open list.
func (s *Service) UnarchiveGoal(ctx context.Context, id int64) (models.Goal, error) {
	return s.setGoalArchived(ctx, id, nil)
}

func (s *Service) setGoalArchived(ctx context.Context, id int64, at any) (models.Goal, error) {
	n, err := workspaces.Scoped[models.Goal](ctx, s.DB).Where("id", id).Update(ctx, orm.Set{"archived_at": at})
	if err != nil {
		return models.Goal{}, err
	}

	if n == 0 {
		return models.Goal{}, errs.ErrNotFound
	}

	return reload[models.Goal](ctx, s.DB, id)
}

// DeleteGoal deletes a goal and, with the same timestamp, its contributions.
func (s *Service) DeleteGoal(ctx context.Context, id int64) error {
	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := softDelete[models.Goal](ctx, tx, id, at); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.GoalContribution](ctx, tx).Where("goal_id", id).
			Update(ctx, orm.Set{"deleted_at": at})

		return err
	})
}

// RestoreGoal brings back a goal deleted less than RestoreWindow ago with the
// contributions that were deleted with it (not the ones deleted earlier).
func (s *Service) RestoreGoal(ctx context.Context, id int64) (models.Goal, error) {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		g, err := trashed[models.Goal](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Goal](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil}); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.GoalContribution](ctx, tx).OnlyTrashed().Where("goal_id", id).
			Where("deleted_at", g.DeletedAt.V).Update(ctx, orm.Set{"deleted_at": nil})

		return err
	})
	if err != nil {
		return models.Goal{}, err
	}

	return reload[models.Goal](ctx, s.DB, id)
}

// AddContribution puts an amount towards a goal. The goal must be open
// (archived goals take none).
func (s *Service) AddContribution(ctx context.Context, goalID int64, in ContributionInput) (models.GoalContribution, error) {
	cents, err := ParseAmount(in.Amount)
	if err != nil || cents == 0 {
		return models.GoalContribution{}, errGoalAmount
	}

	date := strings.TrimSpace(in.Date)
	if date == "" {
		date = s.Today(ctx)
	}

	if !saneYear(date) {
		return models.GoalContribution{}, errGoalDate
	}

	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > maxNoteLen {
		return models.GoalContribution{}, errGoalNote
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.GoalContribution{}, err
	}

	c := models.GoalContribution{GoalID: goalID, AmountCents: cents, OnDate: date, Note: note}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		g, err := workspaces.Scoped[models.Goal](ctx, tx).Where("id", goalID).First(ctx)
		if err != nil {
			return err
		}

		if g.ArchivedAt.Valid {
			return errGoalArchived
		}

		n, err := workspaces.Scoped[models.GoalContribution](ctx, tx).Where("goal_id", goalID).Count(ctx)
		if err != nil {
			return err
		}

		if n >= maxContributions {
			return errTooManyParts
		}

		return orm.Create(ctx, tx, &c)
	})
	if err != nil {
		return models.GoalContribution{}, err
	}

	return reload[models.GoalContribution](ctx, s.DB, c.ID)
}

var errGoalArchived = errs.New(http.StatusConflict, "Unarchive the goal before adding to it.")

// DeleteContribution deletes one contribution of a goal of the workspace.
func (s *Service) DeleteContribution(ctx context.Context, id int64) error {
	return softDelete[models.GoalContribution](ctx, s.DB, id, s.now().UTC())
}
