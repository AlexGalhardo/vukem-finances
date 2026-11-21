package finance

import (
	"context"
	"sort"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Upcoming sources.
const (
	SourceTransaction = "transaction"
	SourceInstallment = "installment"
	SourceRecurrence  = "recurrence"
)

// upcomingCap bounds the list: a daily rule over a long window must not flood a page.
const upcomingCap = 200

// UpcomingItem is something that will happen soon: a transaction already dated
// in the future, an installment part, or an occurrence of a rule that is not
// generated yet (virtual, computed and never stored).
type UpcomingItem struct {
	Date        string `json:"date"`
	Kind        string `json:"kind"`
	AmountCents int64  `json:"amountCents"`
	Description string `json:"description"`
	Source      string `json:"source"`
	AccountID   int64  `json:"accountId"`
	// TransactionID is set for stored rows, RecurrenceID for virtual ones.
	TransactionID    int64 `json:"transactionId"`
	RecurrenceID     int64 `json:"recurrenceId"`
	InstallmentNo    int64 `json:"installmentNo"`
	InstallmentTotal int64 `json:"installmentTotal"`
}

// Upcoming lists, by date, what is due in the next days: stored transactions
// dated after today, and the occurrences of active rules from today on that no
// row holds yet (a row, even a deleted one, or the watermark of the rule hides
// the virtual occurrence, so nothing shows twice).
func (s *Service) Upcoming(ctx context.Context, days int) ([]UpcomingItem, error) {
	today := s.Today(ctx)
	end := shiftDay(today, max(days, 1))

	rows, err := workspaces.Scoped[models.Transaction](ctx, s.DB).WhereNull("transfer_uid").
		Where("occurred_on", ">", today).Where("occurred_on", "<=", end).
		OrderBy("occurred_on", orm.Asc).OrderBy("id", orm.Asc).Limit(upcomingCap).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]UpcomingItem, 0, len(rows))

	for _, r := range rows {
		item := UpcomingItem{
			Date: r.OccurredOn, Kind: r.Kind, AmountCents: r.AmountCents, Description: r.Description,
			Source: SourceTransaction, AccountID: r.AccountID, TransactionID: r.ID,
		}

		switch {
		case r.InstallmentGroupUID.Valid:
			item.Source, item.InstallmentNo, item.InstallmentTotal = SourceInstallment, r.InstallmentNo.V, r.InstallmentTotal.V
		case r.RecurrenceUID.Valid:
			item.Source = SourceRecurrence
		}

		out = append(out, item)
	}

	rules, err := workspaces.Scoped[models.Recurrence](ctx, s.DB).Where("active", true).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		virtual, err := s.virtualOccurrences(ctx, rule, today, end)
		if err != nil {
			return nil, err
		}

		out = append(out, virtual...)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })

	if len(out) > upcomingCap {
		out = out[:upcomingCap]
	}

	return out, nil
}

func (s *Service) virtualOccurrences(ctx context.Context, rule models.Recurrence, today, end string) ([]UpcomingItem, error) {
	from := today
	if rule.GeneratedThrough.Valid && nextDay(rule.GeneratedThrough.V) > from {
		from = nextDay(rule.GeneratedThrough.V)
	}

	dates := ruleOf(rule).Occurrences(from, end, upcomingCap)
	if len(dates) == 0 {
		return nil, nil
	}

	held, err := workspaces.Scoped[models.Transaction](ctx, s.DB).WithTrashed().Where("recurrence_uid", rule.UID).
		WhereIn("occurrence_on", anys(dates)...).Pluck[string](ctx, "occurrence_on")
	if err != nil {
		return nil, err
	}

	taken := make(map[string]bool, len(held))
	for _, d := range held {
		taken[d] = true
	}

	var out []UpcomingItem

	for _, d := range dates {
		if taken[d] {
			continue
		}

		out = append(out, UpcomingItem{
			Date: d, Kind: rule.Kind, AmountCents: rule.AmountCents, Description: rule.Description,
			Source: SourceRecurrence, AccountID: rule.AccountID, RecurrenceID: rule.ID,
		})
	}

	return out, nil
}
