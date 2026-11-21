package finance

// StatementRow is one line of a bank statement as a parser read it, before it
// becomes a transaction. Money is signed int64 cents: a negative amount is an
// expense. A row the parser could not read keeps its line number and the
// reason in Problem and carries no data.
type StatementRow struct {
	// Line is the 1-based line of the file the row starts on (for CSV, the
	// header is line 1).
	Line int
	// Date is an ISO date, "2026-11-12".
	Date        string
	Description string
	Cents       int64
	// Category is the category name the file holds for the row, if any; the
	// importer matches it to a category of the workspace.
	Category string
	// FITID is the bank's own id of the transaction (OFX); it makes the
	// duplicate check exact.
	FITID string
	// Problem is why the row cannot be imported; empty for a good row.
	Problem string
}

// StatementParser reads one statement format. Parse never trusts the content:
// it bounds what it reads, and a row it cannot use is a StatementRow with a
// Problem, not an error. An error means the file as a whole is unusable.
type StatementParser interface {
	Parse(content []byte) ([]StatementRow, error)
}
