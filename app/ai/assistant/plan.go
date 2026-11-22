// Package assistant answers questions about the person's own money in plain
// language, in two steps that work with any provider (no tool calling):
//
//  1. Plan: the model sees the question, today's date, the currency and the
//     names of the reports it may ask for, and returns up to four queries. The
//     server validates each one against an allowlist, caps the range (5 years)
//     and the rows (200), and runs them with the report functions of the
//     workspace. The model never writes SQL and never names a table.
//  2. Answer: the model sees only the aggregated results and returns text.
//
// The figures the person reads come from the server's results, never from the
// model: the answer is a sentence next to the authoritative tables.
package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Limits of a plan.
const (
	MaxQueries = 4
	// MaxRangeDays is five years, leap days included.
	MaxRangeDays = 5*365 + 2
	MaxRows      = 200
	defaultRows  = 50
	isoDate      = "2006-01-02"
)

// The reports a plan may name.
const (
	ReportByCategory   = "by_category"
	ReportCashFlow     = "cash_flow"
	ReportByAccount    = "by_account"
	ReportByTag        = "by_tag"
	ReportNetWorth     = "net_worth"
	ReportCompare      = "compare"
	ReportTransactions = "transactions"
)

// Reports is the allowlist.
var Reports = []string{ //nolint:gochecknoglobals // read-only
	ReportByCategory, ReportCashFlow, ReportByAccount, ReportByTag, ReportNetWorth, ReportCompare, ReportTransactions,
}

// ErrInvalidPlan means the model's plan cannot be run as given. The assistant
// then says it could not answer rather than guess.
var ErrInvalidPlan = errors.New("assistant: the plan is not valid")

// Query is one query of a plan, as the model wrote it.
type Query struct {
	Report      string `json:"report"`
	From        string `json:"from"`
	To          string `json:"to"`
	Kind        string `json:"kind"`
	CategoryUID string `json:"category_uid"`
	Limit       int    `json:"limit"`
}

// Plan is what the planning step returns.
type Plan struct {
	Queries []Query `json:"queries"`
}

// Resolved is a query that passed validation: its fields are normalized and
// the category is a row of the workspace.
type Resolved struct {
	Report     string `json:"report"`
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	CategoryID int64  `json:"-"`
	Limit      int    `json:"limit"`
}

// Describe is the "Based on" line of a query.
func (r Resolved) Describe() string {
	name := map[string]string{
		ReportByCategory: "spending by category", ReportCashFlow: "cash flow", ReportByAccount: "totals by account",
		ReportByTag: "totals by tag", ReportNetWorth: "net worth", ReportCompare: "comparison with the previous period",
		ReportTransactions: "transactions",
	}[r.Report]

	if r.Report == ReportByCategory && r.Kind == models.KindIncome {
		name = "income by category"
	}

	return name + ", " + r.From + " to " + r.To
}

// ParsePlan reads the planning reply. Anything but the JSON of the schema is
// ErrInvalidPlan; fields it does not know are ignored.
func ParsePlan(text string) (Plan, error) {
	var p Plan
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &p); err != nil {
		return Plan{}, ErrInvalidPlan
	}

	return p, nil
}

// Validate checks every query of the plan and returns the normalized ones. One
// bad query refuses the whole plan: a half-answered question misleads. The
// category, when named, must be one of the workspace of ctx.
func (s *Service) Validate(ctx context.Context, p Plan) ([]Resolved, error) {
	if len(p.Queries) == 0 || len(p.Queries) > MaxQueries {
		return nil, ErrInvalidPlan
	}

	out := make([]Resolved, 0, len(p.Queries))

	for _, q := range p.Queries {
		r, err := s.resolve(ctx, q)
		if err != nil {
			return nil, err
		}

		out = append(out, r)
	}

	return out, nil
}

func (s *Service) resolve(ctx context.Context, q Query) (Resolved, error) {
	if !slices.Contains(Reports, q.Report) {
		return Resolved{}, ErrInvalidPlan
	}

	from, err := time.Parse(isoDate, q.From)
	if err != nil || from.Format(isoDate) != q.From {
		return Resolved{}, ErrInvalidPlan
	}

	to, err := time.Parse(isoDate, q.To)
	if err != nil || to.Format(isoDate) != q.To || to.Before(from) {
		return Resolved{}, ErrInvalidPlan
	}

	if int(to.Sub(from).Hours()/24)+1 > MaxRangeDays {
		return Resolved{}, ErrInvalidPlan
	}

	r := Resolved{Report: q.Report, From: q.From, To: q.To, Kind: q.Kind}

	switch q.Kind {
	case "", models.KindIncome, models.KindExpense:
	default:
		return Resolved{}, ErrInvalidPlan
	}

	if q.Report == ReportByCategory && r.Kind == "" {
		r.Kind = models.KindExpense
	}

	if q.Report == ReportTransactions {
		r.Limit = q.Limit
		if r.Limit <= 0 {
			r.Limit = defaultRows
		}

		r.Limit = min(r.Limit, MaxRows)

		if q.CategoryUID != "" {
			c, err := workspaces.Scoped[models.Category](ctx, s.DB).Where("uid", q.CategoryUID).First(ctx)
			if errors.Is(err, orm.ErrNotFound) {
				return Resolved{}, ErrInvalidPlan
			}

			if err != nil {
				return Resolved{}, err
			}

			r.CategoryID = c.ID
		}
	}

	return r, nil
}
