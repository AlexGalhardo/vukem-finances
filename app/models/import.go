package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Import statuses.
const (
	ImportUploaded = "uploaded"
	ImportQueued   = "queued"
	ImportDone     = "done"
	ImportFailed   = "failed"
)

// Import is an uploaded bank statement (CSV or OFX) and what became of it.
type Import struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	AccountID   int64  `json:"account_id"`
	Filename    string `json:"filename"`
	Content     string `json:"-"`
	Mapping     string `json:"-"`
	Status      string `json:"status"`
	Imported    int    `json:"imported"`
	Failed      int    `json:"failed"`
	Error       string `json:"error"`

	// Format is "csv" or "ofx". Duplicates counts the rows skipped because
	// the same row was imported before. Report is a JSON list of the rows
	// that failed (at most 100), read through finance.ImportView.
	Format     string `json:"format"`
	Duplicates int    `json:"duplicates"`
	Report     string `json:"-"`
}
