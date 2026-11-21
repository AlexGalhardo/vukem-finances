package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/pages"
)

// Dashboard renders the charts of the month in ?month=YYYY-MM (default: this month).
func (h *Handlers) Dashboard(c *web.Ctx) error {
	month := c.Query("month")
	if len(month) != 7 {
		month = h.Fin.Month(c.Context())
	}

	d, err := h.Fin.Dashboard(c.Context(), month)
	if err != nil {
		return err
	}

	upcoming, err := h.Fin.Upcoming(c.Context(), upcomingDays)
	if err != nil {
		return err
	}

	return pages.Dashboard.Render(c, pages.DashboardProps{Dashboard: d, Upcoming: upcoming})
}
