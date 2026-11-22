package alerts

import (
	"context"
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// Evaluate returns the notifications that should exist for the workspace of
// ctx as of now: the budgets at 80 or 100 percent of this month, the cards at
// 80 percent of their limit, the statements falling due within three days, the
// goals reached and the rules that failed. It writes nothing. A part that
// cannot be read is left out and reported in the error; the other parts still
// come back.
func (s *Service) Evaluate(ctx context.Context) ([]Candidate, error) {
	var (
		out    []Candidate
		failed error
	)

	for _, part := range []func(context.Context) ([]Candidate, error){
		s.budgets, s.cards, s.goals, s.recurrences,
	} {
		got, err := part(ctx)
		out = append(out, got...)
		failed = errors.Join(failed, err)
	}

	return out, failed
}

func (s *Service) budgets(ctx context.Context) ([]Candidate, error) {
	month := s.Fin.Month(ctx)

	list, err := s.Fin.Budgets(ctx, month)
	if err != nil {
		return nil, err
	}

	var out []Candidate

	for _, b := range list {
		kind, threshold := "", 0

		switch {
		case b.Percent >= 100 && b.Overall:
			kind, threshold = KindOverallBudget100, 100
		case b.Percent >= 100:
			kind, threshold = KindBudget100, 100
		case b.Percent >= 80 && !b.Overall:
			kind, threshold = KindBudget80, 80
		}

		if kind == "" {
			continue
		}

		out = append(out, Candidate{
			Kind: kind, Subject: "budget:" + b.UID + ":" + month,
			Params: map[string]any{
				"name": b.Category, "threshold": threshold, "percent": b.Percent,
				"spent_cents": b.SpentCents, "limit_cents": b.AvailableCents, "month": month,
			},
		})
	}

	return out, nil
}

func (s *Service) cards(ctx context.Context) ([]Candidate, error) {
	cards, err := s.Fin.CardOverviews(ctx)
	if err != nil {
		return nil, err
	}

	month, today := s.Fin.Month(ctx), s.Fin.Today(ctx)

	var (
		out    []Candidate
		failed error
	)

	for _, c := range cards {
		if c.Archived {
			continue
		}

		// The limit is at most 2^53, so 80 percent of it cannot overflow;
		// the debt is compared to that, never multiplied.
		if c.LimitCents > 0 && c.UsedCents >= (c.LimitCents*80+99)/100 {
			out = append(out, Candidate{
				Kind: KindCardLimit80, Subject: "card:" + c.UID + ":" + month,
				Params: map[string]any{"name": c.Name, "used_cents": c.UsedCents, "limit_cents": c.LimitCents, "month": month},
			})
		}

		list, err := s.Fin.Statements(ctx, c.ID)
		if err != nil {
			failed = errors.Join(failed, err)

			continue
		}

		for _, st := range list {
			if st.RemainingCents > 0 && st.Status == finance.StatementClosed && dueSoon(today, st.DueOn) {
				out = append(out, Candidate{
					Kind: KindStatementDue, Subject: "card:" + c.UID + ":" + st.Ref,
					Params: map[string]any{
						"name": c.Name, "due_on": st.DueOn, "remaining_cents": st.RemainingCents, "ref": st.Ref,
					},
				})
			}
		}
	}

	return out, failed
}

// dueSoon reports whether due is today or up to statementDueDays later. Both
// come from the app's own date arithmetic; a value that does not parse is not
// due.
func dueSoon(today, due string) bool {
	t, err1 := time.Parse("2006-01-02", today)
	d, err2 := time.Parse("2006-01-02", due)

	if err1 != nil || err2 != nil {
		return false
	}

	days := int(d.Sub(t).Hours() / 24)

	return days >= 0 && days <= statementDueDays
}

func (s *Service) goals(ctx context.Context) ([]Candidate, error) {
	list, err := s.Fin.Goals(ctx)
	if err != nil {
		return nil, err
	}

	var out []Candidate

	for _, g := range list {
		if g.Reached && !g.Archived {
			out = append(out, Candidate{
				Kind: KindGoalReached, Subject: "goal:" + g.UID,
				Params: map[string]any{"name": g.Name, "target_cents": g.TargetCents, "saved_cents": g.SavedCents},
			})
		}
	}

	return out, nil
}

func (s *Service) recurrences(ctx context.Context) ([]Candidate, error) {
	rules, err := s.Fin.Recurrences(ctx)
	if err != nil {
		return nil, err
	}

	var out []Candidate

	for _, r := range rules {
		if !r.Active || !r.LastError.Valid {
			continue
		}

		// The occurrence that keeps failing is the one after the watermark;
		// until the rule moves past it the date, and so the alert, is the same.
		date := r.StartDate
		if r.GeneratedThrough.Valid {
			date = r.GeneratedThrough.V
		}

		out = append(out, Candidate{
			Kind: KindRecurrenceFailed, Subject: "recurrence:" + r.UID + ":" + date,
			Params: map[string]any{"name": r.Description, "amount_cents": r.AmountCents, "date": date},
		})
	}

	return out, nil
}
