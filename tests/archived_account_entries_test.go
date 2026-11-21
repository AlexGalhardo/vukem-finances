package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const archivedCSV = "date,description,amount\n2026-01-02,Market,10.00\n"

func wantArchivedRefusal(t *testing.T, what string, err error) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), "archived") {
		t.Errorf("%s on an archived account = %v, want a refusal", what, err)
	}
}

// An archived account frees its place in the plan, so it must not keep taking
// new entries: a Free workspace would use more accounts than it pays for.
func TestSecurity_ArchivedAccountTakesNoNewEntries(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)
	noDispatch := func(context.Context, int64, int64) error { return nil }
	mapping := finance.Mapping{Date: "date", Description: "description", Amount: "amount"}
	input := finance.TxInput{AccountID: acc, Kind: models.KindExpense, Amount: "10.00", Date: today()}

	tx, err := fin.CreateTransaction(ctx, input)
	if err != nil {
		t.Fatal(err)
	}

	uploaded, err := fin.StartImport(ctx, acc, "uploaded.csv", []byte(archivedCSV))
	if err != nil {
		t.Fatal(err)
	}

	queued, err := fin.StartImport(ctx, acc, "queued.csv", []byte(archivedCSV))
	if err != nil {
		t.Fatal(err)
	}

	if err := fin.ConfirmImport(ctx, queued.ImportID, mapping, noDispatch); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.ArchiveAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}

	_, err = fin.CreateTransaction(ctx, input)
	wantArchivedRefusal(t, "a new transaction", err)

	_, err = fin.StartImport(ctx, acc, "late.csv", []byte(archivedCSV))
	wantArchivedRefusal(t, "a new import", err)

	wantArchivedRefusal(t, "confirming an import", fin.ConfirmImport(ctx, uploaded.ImportID, mapping, noDispatch))

	if err := fin.RunImport(ctx, queued.ImportID); err != nil {
		t.Fatal(err)
	}

	imports, err := fin.Imports(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, imp := range imports {
		if imp.ID == queued.ImportID && imp.Status != models.ImportFailed {
			t.Errorf("the queued import = %q, want failed", imp.Status)
		}
	}

	rows, err := fin.Transactions(ctx, finance.TxFilter{AccountID: acc})
	if err != nil || len(rows) != 1 {
		t.Errorf("transactions = %d, %v; want only the one from before the archive", len(rows), err)
	}

	// The history stays correctable.
	input.Amount = "12.00"
	if _, err := fin.UpdateTransaction(ctx, tx.ID, input); err != nil {
		t.Errorf("editing an entry of the archived account = %v", err)
	}
}
