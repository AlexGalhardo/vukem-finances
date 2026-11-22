package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

const recoveryCSV = "Date,Description,Amount\n2026-03-01,Coffee,-4.50\n2026-03-02,Salary,1500.00\n2026-03-03,Books,-30.25\n"

// queuedImport leaves an import the way a crash leaves it: confirmed and
// queued, with the job lost (the sync queue keeps nothing).
func queuedImport(t *testing.T, app *vukemtest.App, w who, acc int64) int64 {
	t.Helper()

	fin := softService(t, app)
	ctx := wsCtx(t, w)

	if _, err := fin.StartImport(ctx, acc, "lost.csv", []byte(recoveryCSV)); err != nil {
		t.Fatal(err)
	}

	imp := latestImport(t, app, w)

	err := fin.ConfirmImport(ctx, imp.ID, csvMapping, func(context.Context, int64, int64) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	if got := latestImport(t, app, w); got.Status != models.ImportQueued {
		t.Fatalf("status = %q, want queued", got.Status)
	}

	return imp.ID
}

// TestImportRecoveryRunsAQueuedImportOnce covers QA-501: a queued
// import whose job died with the process is run again at boot, and running a
// job twice, in sequence or at once, writes the rows once.
func TestImportRecoveryRunsAQueuedImportOnce(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	id := queuedImport(t, app, ana, acc)
	fin := softService(t, app)

	var mu sync.Mutex

	var dispatched []int64

	run := func(ctx context.Context, _, importID int64) error {
		mu.Lock()
		dispatched = append(dispatched, importID)
		mu.Unlock()

		return fin.RunImport(ctx, importID)
	}

	n, err := fin.RecoverQueuedImports(t.Context(), run)
	if err != nil || n != 1 {
		t.Fatalf("recovered %d (%v), want 1", n, err)
	}

	if got := latestImport(t, app, ana); got.ID != id || got.Status != models.ImportDone || got.Imported != 3 {
		t.Fatalf("import = %+v, want done with 3 rows", got)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)

	// A second boot finds nothing to do, and a stray second run does nothing.
	if n, err := fin.RecoverQueuedImports(t.Context(), run); err != nil || n != 0 {
		t.Fatalf("second recovery = %d (%v), want 0", n, err)
	}

	if err := fin.RunImport(wsCtx(t, ana), id); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)
}

// TestImportJobRunConcurrentlyWritesRowsOnce: a re-dispatch next to a job that
// is still running (another process) must not write the rows twice, because
// the rows, the counts and the status are one transaction under the
// workspace lock.
func TestImportJobRunConcurrentlyWritesRowsOnce(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	id := queuedImport(t, app, ana, acc)
	fin := softService(t, app)

	var wg sync.WaitGroup

	for range 6 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = fin.RunImport(wsCtx(t, ana), id)
		}()
	}

	wg.Wait()

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)

	if got := latestImport(t, app, ana); got.Status != models.ImportDone || got.Imported != 3 {
		t.Fatalf("import = %+v, want done with 3 rows", got)
	}
}
