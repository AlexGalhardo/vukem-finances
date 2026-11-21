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

// Import is an uploaded bank statement (CSV) and what became of it.
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
}
