// Package migrations holds the schema migrations.
package migrations

import "github.com/AlexGalhardo/vukem/schema"

// All lists the migrations in order; vukem make:migration appends here.
func All() []schema.Migration {
	return []schema.Migration{
		m20261121154436CreateSocialAccountsTable(),
		m20261121154439CreateFinanceTables(),
		m20261121154434CreateUsersTable(),
		m20261121154435CreateAuthTables(),
		m20261121154437AddTwoFactorColumns(),
		m20261121154434CreateAdminAuditLogsTable(),
		m20261121190000AddWorkspaceTimezone(),
		m20261121190100CreateAuditEventsTable(),
		m20261121200000AddSyncColumns(),
		m20261121200100InstallChangeTriggers(),
		m20261121200000CreateAITables(),
		m20261121210000AddAccountArchiveAndTransfers(),
		m20261122100000CreateBillingTables(),
		m20261122100100AddPixEntitlementPeriod(),
	}
}
