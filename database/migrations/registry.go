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
		withSyncColumnsDown(m20261121200000AddSyncColumns()),
		m20261121200100InstallChangeTriggers(),
		m20261121200000CreateAITables(),
		withTriggersDroppedBeforeDown(m20261121210000AddAccountArchiveAndTransfers(), "accounts", "transactions"),
		withTriggersDroppedBeforeDown(m20261121220000AddCategoriesTagsAndImportHash(), "categories", "transactions"),
		m20261121230000AddImportReport(),
		m20261122100000CreateBillingTables(),
		m20261122100100AddPixEntitlementPeriod(),
		m20261122100200AddPixMonthsWithheld(),
		withTriggersDroppedBeforeDown(m20261122110000CreateCreditCards(), "transactions"),
		m20261122120000AddTokenResource(),
		m20261122130000CreateRecurrences(),
		m20261122140000AddBudgetRolloverAndOverall(),
		m20261122140100CreateGoals(),
		m20261122140200CreateNotifications(),
		m20261123100000AddReportIndexes(),
		m20261123100000CreateTelegramTables(),
		m20261123180000CreateSyncDevicesAndOps(),
		m20261124163000CreateAIAssistantTables(),
		m20261124170000CreatePairingCodes(),
		m20261124170100IndexPairingCodesByWorkspace(),
		m20261125100000CreateMCPOAuthTables(),
		m20261125110000AddMCPGrantScopes(),
		m20261126100000CreateDataLifecycle(),
		m20261126120000AddTransactionCoverIndexes(),
		m20261127100000CreateTransactionMonthTotals(),
	}
}
