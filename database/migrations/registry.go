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
		m20261121220000AddCategoriesTagsAndImportHash(),
		m20261121230000AddImportReport(),
		m20261122100000CreateBillingTables(),
		m20261122100100AddPixEntitlementPeriod(),
		m20261122100200AddPixMonthsWithheld(),
		m20261122110000CreateCreditCards(),
		m20261122120000AddTokenResource(),
		m20261122130000CreateRecurrences(),
		m20261122140000AddBudgetRolloverAndOverall(),
		m20261122140100CreateGoals(),
		m20261122140200CreateNotifications(),
		m20261123100000AddReportIndexes(),
		m20261123100000CreateTelegramTables(),
		m20261123180000CreateSyncDevicesAndOps(),
		m20261124163000CreateAIAssistantTables(),
		m20261124090000CreatePairingCodes(),
	}
}
