package finance

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// AICategorize is the job that proposes categories for uncategorized
// transactions (app/ai/categorize): the memory of descriptions first, then the
// model when the owner turned the feature on. It is defined here because the
// write path enqueues it. ImportID or TransactionUID narrow it to the rows of
// an import or to one transaction; both zero means every uncategorized
// transaction without a suggestion, up to the cap of a run.
type AICategorize struct {
	WorkspaceID    int64  `json:"workspaceId"`
	ImportID       int64  `json:"importId"`
	TransactionUID string `json:"transactionUid"`
}

// Kind names the job's handler; keep it stable across deploys.
func (AICategorize) Kind() string { return "ai_categorize" }

// RequestCategorize enqueues a categorization run for the workspace of ctx. It
// never fails the write that asked: the "Categorize uncategorized" button
// catches whatever a failed enqueue misses.
func (s *Service) RequestCategorize(ctx context.Context, importID int64, transactionUID string) {
	id, ok := workspaces.TenantID(ctx)
	if !ok {
		return
	}

	job := AICategorize{WorkspaceID: id, ImportID: importID, TransactionUID: transactionUID}
	if err := s.Queue.Dispatch(ctx, job); err != nil && s.Log != nil {
		s.Log.Warn("ai: could not enqueue a categorization", "workspace", id, "error", err)
	}
}
