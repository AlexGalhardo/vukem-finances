package operations

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// RecurrenceOut is a rule that makes a transaction again and again. Only the
// occurrences that are not generated yet follow a change of the rule.
type RecurrenceOut struct {
	UID              string  `json:"uid"               example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version          int64   `json:"version"           example:"1"`
	AccountUID       string  `json:"account_uid"       example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	CategoryUID      *string `json:"category_uid"`
	Kind             string  `json:"kind"              example:"expense"`
	AmountCents      int64   `json:"amount_cents"      example:"4990"`
	Currency         string  `json:"currency"          example:"USD"`
	Description      string  `json:"description"       example:"Streaming"`
	Frequency        string  `json:"frequency"         doc:"daily, weekly, monthly or yearly" example:"monthly"`
	Interval         int     `json:"interval"          doc:"Repeat every this many days, weeks, months or years" example:"1"`
	StartDate        string  `json:"start_date"        example:"2026-12-01"`
	EndDate          *string `json:"end_date"`
	MaxCount         *int64  `json:"max_count"`
	GeneratedThrough *string `json:"generated_through" doc:"The last date the rule has been run up to"`
	Active           bool    `json:"active"`
	CreatedAt        string  `json:"created_at"        example:"2026-11-21T12:00:00Z"`
	UpdatedAt        string  `json:"updated_at"        example:"2026-11-21T12:00:00Z"`
}

func recurrenceOut(r models.Recurrence, l lookups) RecurrenceOut {
	out := RecurrenceOut{
		UID: r.UID, Version: r.Version, AccountUID: l.accounts[r.AccountID], Kind: r.Kind, AmountCents: r.AmountCents,
		Currency: l.currency, Description: r.Description, Frequency: r.Frequency, Interval: r.Interval, StartDate: r.StartDate,
		Active: r.Active, CreatedAt: Stamp(r.CreatedAt), UpdatedAt: Stamp(r.UpdatedAt),
	}

	if r.CategoryID.Valid {
		if uid, ok := l.categories[r.CategoryID.V]; ok {
			out.CategoryUID = &uid
		}
	}

	if r.EndDate.Valid {
		out.EndDate = &r.EndDate.V
	}

	if r.MaxCount.Valid {
		out.MaxCount = &r.MaxCount.V
	}

	if r.GeneratedThrough.Valid {
		out.GeneratedThrough = &r.GeneratedThrough.V
	}

	return out
}

// RecurrenceCreateIn is the body of recurrences.create.
type RecurrenceCreateIn struct {
	AccountUID  string `json:"account_uid"  validate:"required,max=26"`
	CategoryUID string `json:"category_uid" validate:"max=26" doc:"A category of the same kind; leave out for none"`
	Kind        string `json:"kind"         validate:"required,max=20" doc:"income or expense" example:"expense"`
	AmountCents int64  `json:"amount_cents" doc:"A positive whole number of cents; the kind gives the sign" example:"4990"`
	Description string `json:"description"  validate:"max=200" example:"Streaming"`
	Frequency   string `json:"frequency"    validate:"required,max=20" doc:"daily, weekly, monthly or yearly" example:"monthly"`
	Interval    int    `json:"interval"     validate:"min=1,max=999" doc:"Repeat every this many days, weeks, months or years" example:"1"`
	StartDate   string `json:"start_date"   validate:"required,max=10" example:"2026-12-01"`
	EndDate     string `json:"end_date"     validate:"max=10" doc:"The last date the rule may run; leave out for no end"`
	MaxCount    int    `json:"max_count"    validate:"min=0,max=9999" doc:"At most this many occurrences; 0 for no limit"`
	Backfill    bool   `json:"backfill"     doc:"Post the occurrences that fell before today, at most 36"`
}

// RecurrenceUpdateIn is the body of recurrences.update; a field left out stays
// as it is. category_uid, end_date "" remove them and max_count 0 removes the
// limit; active false pauses the rule.
type RecurrenceUpdateIn struct {
	UID         string  `json:"uid"          validate:"required,max=26"`
	AccountUID  *string `json:"account_uid"  validate:"max=26"`
	CategoryUID *string `json:"category_uid" validate:"max=26"`
	Kind        *string `json:"kind"         validate:"max=20"`
	AmountCents *int64  `json:"amount_cents"`
	Description *string `json:"description"  validate:"max=200"`
	Frequency   *string `json:"frequency"    validate:"max=20"`
	Interval    *int    `json:"interval"     validate:"min=0,max=999"`
	StartDate   *string `json:"start_date"   validate:"max=10"`
	EndDate     *string `json:"end_date"     validate:"max=10"`
	MaxCount    *int    `json:"max_count"    validate:"min=0,max=9999"`
	Active      *bool   `json:"active"`
}

// RecurrenceDeleteIn is the input of recurrences.delete.
type RecurrenceDeleteIn struct {
	UID    string `json:"uid"    validate:"required,max=26"`
	Future bool   `json:"future" doc:"Also delete the generated transactions dated after today"`
}

type recurrenceOps struct {
	List   *Op[NoInput, List[RecurrenceOut]]
	Create *Op[RecurrenceCreateIn, RecurrenceOut]
	Update *Op[RecurrenceUpdateIn, RecurrenceOut]
	Delete *Op[RecurrenceDeleteIn, Empty]
}

func (c *Catalog) defineRecurrences() {
	c.Recurrences = recurrenceOps{
		List: Define(c.Registry, Operation[NoInput, List[RecurrenceOut]]{
			Name: "recurrences.list", Summary: "List the recurring rules", Scope: ScopeRecurrencesRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[RecurrenceOut], error) {
				return c.recurrencesList(ctx)
			},
		}),
		Create: Define(c.Registry, Operation[RecurrenceCreateIn, RecurrenceOut]{
			Name: "recurrences.create", Summary: "Create a recurring rule and post what is already due", Scope: ScopeRecurrencesWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in RecurrenceCreateIn) (RecurrenceOut, error) {
				return c.recurrencesCreate(ctx, in)
			},
		}),
		Update: Define(c.Registry, Operation[RecurrenceUpdateIn, RecurrenceOut]{
			Name: "recurrences.update", Summary: "Change a recurring rule", Scope: ScopeRecurrencesWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in RecurrenceUpdateIn) (RecurrenceOut, error) {
				return c.recurrencesUpdate(ctx, in)
			},
		}),
		Delete: Define(c.Registry, Operation[RecurrenceDeleteIn, Empty]{
			Name: "recurrences.delete", Summary: "Delete a recurring rule and keep what it generated", Scope: ScopeRecurrencesWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in RecurrenceDeleteIn) (Empty, error) {
				return c.recurrencesDelete(ctx, in)
			},
		}),
	}
}

func (c *Catalog) recurrenceByUID(ctx context.Context, uid string) (models.Recurrence, error) {
	x, err := workspaces.Scoped[models.Recurrence](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (c *Catalog) recurrencesList(ctx context.Context) (List[RecurrenceOut], error) {
	all, err := c.Fin.Recurrences(ctx)
	if err != nil {
		return List[RecurrenceOut]{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return List[RecurrenceOut]{}, err
	}

	data := make([]RecurrenceOut, 0, len(all))
	for _, r := range all {
		data = append(data, recurrenceOut(r, l))
	}

	return List[RecurrenceOut]{Data: data}, nil
}

func (c *Catalog) recurrencesCreate(ctx context.Context, in RecurrenceCreateIn) (RecurrenceOut, error) {
	account, err := c.accountID(ctx, in.AccountUID)
	if err != nil {
		return RecurrenceOut{}, err
	}

	category, err := c.categoryID(ctx, in.CategoryUID)
	if err != nil {
		return RecurrenceOut{}, err
	}

	r, err := c.Fin.CreateRecurrence(ctx, finance.RecurrenceInput{
		AccountID: account, CategoryID: category, Kind: in.Kind, Amount: centsText(in.AmountCents), Description: in.Description,
		Frequency: in.Frequency, Interval: in.Interval, StartDate: in.StartDate, EndDate: in.EndDate, MaxCount: in.MaxCount,
		Backfill: in.Backfill,
	})
	if err != nil {
		return RecurrenceOut{}, err
	}

	return c.recurrenceView(ctx, r)
}

func (c *Catalog) recurrenceView(ctx context.Context, r models.Recurrence) (RecurrenceOut, error) {
	l, err := c.lookups(ctx)
	if err != nil {
		return RecurrenceOut{}, err
	}

	return recurrenceOut(r, l), nil
}

func (c *Catalog) recurrencesUpdate(ctx context.Context, in RecurrenceUpdateIn) (RecurrenceOut, error) {
	cur, err := c.recurrenceByUID(ctx, in.UID)
	if err != nil {
		return RecurrenceOut{}, err
	}

	next := finance.RecurrenceInput{
		AccountID: cur.AccountID, CategoryID: cur.CategoryID.V, Kind: cur.Kind, Amount: centsText(cur.AmountCents),
		Description: cur.Description, Frequency: cur.Frequency, Interval: cur.Interval, StartDate: cur.StartDate,
		EndDate: cur.EndDate.V, Active: cur.Active,
	}

	if cur.MaxCount.Valid {
		next.MaxCount = int(cur.MaxCount.V)
	}

	if in.AccountUID != nil {
		if next.AccountID, err = c.accountID(ctx, *in.AccountUID); err != nil {
			return RecurrenceOut{}, err
		}
	}

	if in.CategoryUID != nil {
		if next.CategoryID, err = c.categoryID(ctx, *in.CategoryUID); err != nil {
			return RecurrenceOut{}, err
		}
	}

	if in.Kind != nil {
		next.Kind = *in.Kind
	}

	if in.AmountCents != nil {
		next.Amount = centsText(*in.AmountCents)
	}

	if in.Description != nil {
		next.Description = *in.Description
	}

	if in.Frequency != nil {
		next.Frequency = *in.Frequency
	}

	if in.Interval != nil && *in.Interval != 0 {
		next.Interval = *in.Interval
	}

	if in.StartDate != nil {
		next.StartDate = *in.StartDate
	}

	if in.EndDate != nil {
		next.EndDate = *in.EndDate
	}

	if in.MaxCount != nil {
		next.MaxCount = *in.MaxCount
	}

	if in.Active != nil {
		next.Active = *in.Active
	}

	saved, err := c.Fin.UpdateRecurrence(ctx, cur.ID, next)
	if err != nil {
		return RecurrenceOut{}, err
	}

	return c.recurrenceView(ctx, saved)
}

func (c *Catalog) recurrencesDelete(ctx context.Context, in RecurrenceDeleteIn) (Empty, error) {
	r, err := c.recurrenceByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	return Empty{}, c.Fin.DeleteRecurrence(ctx, r.ID, in.Future)
}
