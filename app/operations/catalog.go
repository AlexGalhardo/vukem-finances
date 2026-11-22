package operations

import (
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Services are the finance services the operations call. A Catalog built with
// none only describes the operations (the inventory, the REST table); Run needs
// them.
type Services struct {
	WS  *workspaces.Service
	Fin *finance.Service
	Rep *reports.Service
}

// Catalog is the registry of the application: every operation, declared once,
// with the handle an adapter invokes.
type Catalog struct {
	*Registry
	Services

	Accounts   accountOps
	Categories categoryOps
	Tags       tagOps

	Transactions transactionOps
	Budgets      budgetOps

	Cards       cardOps
	Recurrences recurrenceOps
	Goals       goalOps
	Reports     reportOps
	Exports     exportOps
}

// NewCatalog declares every operation of the application.
func NewCatalog(cfg Config, svc Services) *Catalog {
	c := &Catalog{Registry: NewRegistry(cfg), Services: svc}

	c.defineAccounts()
	c.defineCategories()
	c.defineTags()
	c.defineTransactions()
	c.defineBudgets()
	c.defineCards()
	c.defineRecurrences()
	c.defineGoals()
	c.defineReports()
	c.defineExports()

	return c
}
