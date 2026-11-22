package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// contributionsShown is how many contributions each goal lists on the page.
const contributionsShown = 5

// Goals lists the goals with their progress.
func (h *Handlers) Goals(c *web.Ctx) error {
	ctx := c.Context()

	goals, err := h.Fin.Goals(ctx)
	if err != nil {
		return err
	}

	recent, err := h.Fin.RecentContributions(ctx, contributionsShown)
	if err != nil {
		return err
	}

	props := pages.GoalsProps{Goals: goals, Contributions: []models.GoalContribution{}, Today: h.Fin.Today(ctx)}
	for _, g := range goals {
		props.Contributions = append(props.Contributions, recent[g.ID]...)
	}

	return pages.Goals.Render(c, props)
}

// StoreGoal creates a goal.
func (h *Handlers) StoreGoal(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.GoalSchema)
	if err != nil {
		return err
	}

	_, err = h.Fin.CreateGoal(c.Context(), finance.GoalInput{Name: in.Name, Target: in.Target, TargetDate: in.TargetDate})
	if err != nil {
		return back(c, "/goals", err)
	}

	return done(c, "/goals", "Goal saved.")
}

// UpdateGoal changes a goal.
func (h *Handlers) UpdateGoal(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	gid, err := c.ParamInt("goal")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.GoalSchema)
	if err != nil {
		return err
	}

	_, err = h.Fin.UpdateGoal(c.Context(), gid, finance.GoalInput{Name: in.Name, Target: in.Target, TargetDate: in.TargetDate})
	if err != nil {
		return back(c, "/goals", err)
	}

	h.Fin.RequestAlerts(c.Context())

	return done(c, "/goals", "Goal saved.")
}

// DestroyGoal deletes a goal with its contributions.
func (h *Handlers) DestroyGoal(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	gid, err := c.ParamInt("goal")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteGoal(c.Context(), gid); err != nil {
		return err
	}

	return done(c, "/goals", "Goal removed.")
}

// ArchiveGoal moves a goal out of the open list.
func (h *Handlers) ArchiveGoal(c *web.Ctx) error { return h.archiveGoal(c, true) }

// UnarchiveGoal brings a goal back to the open list.
func (h *Handlers) UnarchiveGoal(c *web.Ctx) error { return h.archiveGoal(c, false) }

func (h *Handlers) archiveGoal(c *web.Ctx, archive bool) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	gid, err := c.ParamInt("goal")
	if err != nil {
		return err
	}

	if archive {
		_, err = h.Fin.ArchiveGoal(c.Context(), gid)
	} else {
		_, err = h.Fin.UnarchiveGoal(c.Context(), gid)
	}

	if err != nil {
		return err
	}

	return done(c, "/goals", "Goal saved.")
}

// StoreContribution puts an amount towards a goal.
func (h *Handlers) StoreContribution(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	gid, err := c.ParamInt("goal")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.ContributionSchema)
	if err != nil {
		return err
	}

	_, err = h.Fin.AddContribution(c.Context(), gid, finance.ContributionInput{Amount: in.Amount, Date: in.Date, Note: in.Note})
	if err != nil {
		return back(c, "/goals", err)
	}

	h.Fin.RequestAlerts(c.Context())

	return done(c, "/goals", "Contribution saved.")
}

// DestroyContribution removes a contribution of a goal.
func (h *Handlers) DestroyContribution(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	gid, err := c.ParamInt("goal")
	if err != nil {
		return err
	}

	cid, err := c.ParamInt("contribution")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteContribution(c.Context(), gid, cid); err != nil {
		return err
	}

	return done(c, "/goals", "Contribution removed.")
}
