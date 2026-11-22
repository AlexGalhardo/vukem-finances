package operations

import (
	"context"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// GoalOut is a savings goal with its progress. A goal is not tied to an
// account: contributions are what the person puts towards it.
type GoalOut struct {
	UID                  string  `json:"uid"                    example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version              int64   `json:"version"                example:"1"`
	Name                 string  `json:"name"                   example:"Trip"`
	Currency             string  `json:"currency"               example:"USD"`
	TargetCents          int64   `json:"target_cents"           example:"500000"`
	TargetDate           *string `json:"target_date"`
	Archived             bool    `json:"archived"`
	SavedCents           int64   `json:"saved_cents"            example:"120000"`
	RemainingCents       int64   `json:"remaining_cents"        example:"380000"`
	Percent              int     `json:"percent"                example:"24"`
	Reached              bool    `json:"reached"`
	Contributions        int     `json:"contributions"          example:"3"`
	MonthsLeft           int     `json:"months_left"            doc:"Months to the target date, for a goal with a future date that is not reached" example:"8"`
	RequiredMonthlyCents int64   `json:"required_monthly_cents" doc:"What to put aside each month to reach the target by its date" example:"47500"`
	Overdue              bool    `json:"overdue"                doc:"The target date has passed and the goal is not reached"`
	ProjectedDate        *string `json:"projected_date"         doc:"When the pace of the last months would reach the target"`
}

// ContributionOut is an amount put towards a goal; a negative one is a withdrawal.
type ContributionOut struct {
	UID         string `json:"uid"          example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version     int64  `json:"version"      example:"1"`
	GoalUID     string `json:"goal_uid"     example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	AmountCents int64  `json:"amount_cents" example:"50000"`
	Date        string `json:"date"         example:"2026-11-21"`
	Note        string `json:"note"         example:"Bonus"`
	CreatedAt   string `json:"created_at"   example:"2026-11-21T12:00:00Z"`
}

// GoalCreateIn is the body of goals.create.
type GoalCreateIn struct {
	Name        string `json:"name"         validate:"required,max=120" example:"Trip"`
	TargetCents int64  `json:"target_cents" doc:"The amount to reach, a positive whole number of cents" example:"500000"`
	TargetDate  string `json:"target_date"  validate:"max=10" doc:"When to reach it, YYYY-MM-DD; leave out for no date"`
}

// GoalUpdateIn is the body of goals.update; a field left out stays as it is
// and target_date "" removes the date.
type GoalUpdateIn struct {
	UID         string  `json:"uid"          validate:"required,max=26"`
	Name        *string `json:"name"         validate:"max=120"`
	TargetCents *int64  `json:"target_cents"`
	TargetDate  *string `json:"target_date"  validate:"max=10"`
}

// ContributionAddIn is the body of goals.contributions.add.
type ContributionAddIn struct {
	UID         string `json:"uid"          validate:"required,max=26" doc:"The uid of the goal"`
	AmountCents int64  `json:"amount_cents" doc:"A non-zero whole number of cents; negative withdraws" example:"50000"`
	Date        string `json:"date"         validate:"max=10" doc:"The date of the contribution; today when left out"`
	Note        string `json:"note"         validate:"max=200"`
}

// ContributionDeleteIn is the input of goals.contributions.delete.
type ContributionDeleteIn struct {
	UID             string `json:"uid"              validate:"required,max=26" doc:"The uid of the goal"`
	ContributionUID string `json:"contribution_uid" validate:"required,max=26" doc:"The uid of the contribution"`
}

type goalOps struct {
	List      *Op[NoInput, List[GoalOut]]
	Create    *Op[GoalCreateIn, GoalOut]
	Update    *Op[GoalUpdateIn, GoalOut]
	Archive   *Op[UIDIn, GoalOut]
	Unarchive *Op[UIDIn, GoalOut]
	Delete    *Op[UIDIn, Empty]
	Add       *Op[ContributionAddIn, ContributionOut]
	Remove    *Op[ContributionDeleteIn, Empty]
}

func (c *Catalog) defineGoals() {
	c.Goals = goalOps{
		List: Define(c.Registry, Operation[NoInput, List[GoalOut]]{
			Name: "goals.list", Summary: "List the savings goals with their progress", Scope: ScopeGoalsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[GoalOut], error) { return c.goalsList(ctx) },
		}),
		Create: Define(c.Registry, Operation[GoalCreateIn, GoalOut]{
			Name: "goals.create", Summary: "Create a savings goal", Scope: ScopeGoalsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in GoalCreateIn) (GoalOut, error) { return c.goalsCreate(ctx, in) },
		}),
		Update: Define(c.Registry, Operation[GoalUpdateIn, GoalOut]{
			Name: "goals.update", Summary: "Change the name, target and date of a goal", Scope: ScopeGoalsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in GoalUpdateIn) (GoalOut, error) { return c.goalsUpdate(ctx, in) },
		}),
		Archive: Define(c.Registry, Operation[UIDIn, GoalOut]{
			Name: "goals.archive", Summary: "Archive a goal; its contributions stay", Scope: ScopeGoalsWrite, Kind: KindDestructive, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (GoalOut, error) { return c.goalsArchive(ctx, in, true) },
		}),
		Unarchive: Define(c.Registry, Operation[UIDIn, GoalOut]{
			Name: "goals.unarchive", Summary: "Bring an archived goal back", Scope: ScopeGoalsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (GoalOut, error) { return c.goalsArchive(ctx, in, false) },
		}),
		Delete: Define(c.Registry, Operation[UIDIn, Empty]{
			Name: "goals.delete", Summary: "Delete a goal and its contributions", Scope: ScopeGoalsWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (Empty, error) { return c.goalsDelete(ctx, in) },
		}),
		// A contribution is recorded money, like a transaction: it asks the
		// recording ability of the role, as it does on the goals page.
		Add: Define(c.Registry, Operation[ContributionAddIn, ContributionOut]{
			Name: "goals.contributions.add", Summary: "Put an amount towards a goal", Scope: ScopeGoalsWrite,
			Ability: workspaces.AbilityRecord, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in ContributionAddIn) (ContributionOut, error) {
				return c.contributionsAdd(ctx, in)
			},
		}),
		Remove: Define(c.Registry, Operation[ContributionDeleteIn, Empty]{
			Name: "goals.contributions.delete", Summary: "Delete a contribution of a goal", Scope: ScopeGoalsWrite,
			Ability: workspaces.AbilityRecord, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in ContributionDeleteIn) (Empty, error) {
				return c.contributionsDelete(ctx, in)
			},
		}),
	}
}

func (c *Catalog) goalByUID(ctx context.Context, uid string) (models.Goal, error) {
	x, err := workspaces.Scoped[models.Goal](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

// goalViews reads the goals with their progress, keyed by uid, and the
// currency; the versions come from the rows, which the status does not carry.
func (c *Catalog) goalViews(ctx context.Context) ([]GoalOut, error) {
	all, err := c.Fin.Goals(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := workspaces.Scoped[models.Goal](ctx, c.WS.DB).Get(ctx)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]int64, len(rows))
	for _, g := range rows {
		versions[g.UID] = g.Version
	}

	ws, err := c.WS.Current(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]GoalOut, 0, len(all))

	for _, g := range all {
		v := GoalOut{
			UID: g.UID, Version: versions[g.UID], Name: g.Name, Currency: ws.Currency, TargetCents: g.TargetCents,
			Archived: g.Archived, SavedCents: g.SavedCents, RemainingCents: g.RemainingCents, Percent: g.Percent,
			Reached: g.Reached, Contributions: g.Contributions, MonthsLeft: g.MonthsLeft,
			RequiredMonthlyCents: g.RequiredMonthlyCents, Overdue: g.Overdue,
		}

		if g.TargetDate != "" {
			v.TargetDate = &g.TargetDate
		}

		if g.ProjectedDate != "" {
			v.ProjectedDate = &g.ProjectedDate
		}

		out = append(out, v)
	}

	return out, nil
}

func (c *Catalog) goalView(ctx context.Context, uid string) (GoalOut, error) {
	views, err := c.goalViews(ctx)
	if err != nil {
		return GoalOut{}, err
	}

	for _, v := range views {
		if v.UID == uid {
			return v, nil
		}
	}

	return GoalOut{}, errs.ErrNotFound
}

func (c *Catalog) goalsList(ctx context.Context) (List[GoalOut], error) {
	data, err := c.goalViews(ctx)

	return List[GoalOut]{Data: data}, err
}

func (c *Catalog) goalsCreate(ctx context.Context, in GoalCreateIn) (GoalOut, error) {
	g, err := c.Fin.CreateGoal(ctx, finance.GoalInput{Name: in.Name, Target: centsText(in.TargetCents), TargetDate: in.TargetDate})
	if err != nil {
		return GoalOut{}, err
	}

	return c.goalView(ctx, g.UID)
}

func (c *Catalog) goalsUpdate(ctx context.Context, in GoalUpdateIn) (GoalOut, error) {
	cur, err := c.goalByUID(ctx, in.UID)
	if err != nil {
		return GoalOut{}, err
	}

	next := finance.GoalInput{Name: cur.Name, Target: centsText(cur.TargetCents), TargetDate: cur.TargetDate.V}

	if in.Name != nil {
		next.Name = *in.Name
	}

	if in.TargetCents != nil {
		next.Target = centsText(*in.TargetCents)
	}

	if in.TargetDate != nil {
		next.TargetDate = *in.TargetDate
	}

	if _, err := c.Fin.UpdateGoal(ctx, cur.ID, next); err != nil {
		return GoalOut{}, err
	}

	return c.goalView(ctx, cur.UID)
}

func (c *Catalog) goalsArchive(ctx context.Context, in UIDIn, archive bool) (GoalOut, error) {
	g, err := c.goalByUID(ctx, in.UID)
	if err != nil {
		return GoalOut{}, err
	}

	if archive {
		_, err = c.Fin.ArchiveGoal(ctx, g.ID)
	} else {
		_, err = c.Fin.UnarchiveGoal(ctx, g.ID)
	}

	if err != nil {
		return GoalOut{}, err
	}

	return c.goalView(ctx, g.UID)
}

func (c *Catalog) goalsDelete(ctx context.Context, in UIDIn) (Empty, error) {
	g, err := c.goalByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	return Empty{}, c.Fin.DeleteGoal(ctx, g.ID)
}

func (c *Catalog) contributionsAdd(ctx context.Context, in ContributionAddIn) (ContributionOut, error) {
	g, err := c.goalByUID(ctx, in.UID)
	if err != nil {
		return ContributionOut{}, err
	}

	p, err := c.Fin.AddContribution(ctx, g.ID, finance.ContributionInput{
		Amount: centsText(in.AmountCents), Date: in.Date, Note: in.Note,
	})
	if err != nil {
		return ContributionOut{}, err
	}

	return ContributionOut{
		UID: p.UID, Version: p.Version, GoalUID: g.UID, AmountCents: p.AmountCents, Date: p.OnDate, Note: p.Note,
		CreatedAt: Stamp(p.CreatedAt),
	}, nil
}

func (c *Catalog) contributionsDelete(ctx context.Context, in ContributionDeleteIn) (Empty, error) {
	g, err := c.goalByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	p, err := workspaces.Scoped[models.GoalContribution](ctx, c.WS.DB).Where("uid", in.ContributionUID).
		Where("goal_id", g.ID).First(ctx)
	if err != nil {
		return Empty{}, lookupErr(err)
	}

	return Empty{}, c.Fin.DeleteContribution(ctx, g.ID, p.ID)
}
