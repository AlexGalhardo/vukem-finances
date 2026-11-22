package sync

// MonthCountSQL and MonthArgs expose the monthly quota count to the plan test.
const MonthCountSQL = monthCountSQL

func MonthArgs(workspaceID int64, month string) []any { return monthCountArgs(workspaceID, month) }
