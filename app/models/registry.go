package models

import (
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// TenantModel is a domain model whose rows belong to one workspace.
type TenantModel struct {
	// Table is the table of the model.
	Table string
	// New returns a zero value of the model.
	New func() tenancy.Scoped
}

func tenantModel[T any, P interface {
	*T
	tenancy.Scoped
}]() TenantModel {
	return TenantModel{Table: orm.TableOf[T](), New: func() tenancy.Scoped { return P(new(T)) }}
}

// TenantModels lists every model scoped to a workspace. A model that holds
// a WorkspaceID joins this list in the same change that adds it: the
// isolation guard test (tests/isolation_guard_test.go) fails otherwise.
func TenantModels() []TenantModel {
	return []TenantModel{
		tenantModel[Account](),
		tenantModel[Category](),
		tenantModel[Transaction](),
		tenantModel[Budget](),
		tenantModel[Import](),
		tenantModel[Invite](),
		tenantModel[MonthlySummarySent](),
		tenantModel[AuditEvent](),
		tenantModel[Change](),
		tenantModel[SyncState](),
		tenantModel[SyncConflict](),
	}
}
