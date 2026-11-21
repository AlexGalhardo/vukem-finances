package seeders_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/config"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/database/seeders"
)

func TestSecurity_FinanceSeedRejectsProduction(t *testing.T) {
	app := vukemtest.New(t, vukem.New)
	app.Source = config.MapSource(map[string]string{"APP_ENV": "production"})
	if err := (seeders.FinanceAdminSeeder{}).SeedApp(t.Context(), app.App); err == nil {
		t.Fatal("production admin seed accepted")
	}
	if err := (seeders.FinanceDemoSeeder{}).SeedApp(t.Context(), app.App); err == nil {
		t.Fatal("production demo seed accepted")
	}
}

func TestSecurity_FinanceSeedRandomPrivateCredential(t *testing.T) {
	t.Chdir(t.TempDir())
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env: map[string]string{"ADMIN_EMAILS": "admin@example.test"},
	})
	if err := (seeders.FinanceAdminSeeder{}).SeedApp(t.Context(), app.App); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("storage/finance-admin-*.txt")
	if err != nil || len(files) != 1 {
		t.Fatalf("credential files = %v, %v", files, err)
	}
	info, err := os.Stat(files[0])
	if err != nil {
		t.Fatal(err)
	}
	// CreateTemp asks for 0600. Windows does not store those bits, so Go reports 0666.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("credential permissions = %v, want 0600", info.Mode().Perm())
	}
	contents, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	_, password, ok := strings.Cut(string(contents), "Password: ")
	if !ok || len(strings.TrimSpace(password)) < 40 {
		t.Fatal("missing secure generated password")
	}
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	before, err := orm.QueryOf[models.User](db).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !before.EmailVerified() || strings.Contains(string(contents), before.Password) {
		t.Fatal("seed must verify local admin and hash its password")
	}
	if err := (seeders.FinanceAdminSeeder{}).SeedApp(t.Context(), app.App); err != nil {
		t.Fatal(err)
	}
	after, err := orm.QueryOf[models.User](db).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if before.Password != after.Password {
		t.Fatal("rerun replaced existing password")
	}
	files, _ = filepath.Glob("storage/finance-admin-*.txt")
	if len(files) != 1 {
		t.Fatal("rerun created another credential file")
	}
}

func TestFinanceDemoSeedHasTwoAccountsAndATransfer(t *testing.T) {
	t.Chdir(t.TempDir())
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env: map[string]string{"ADMIN_EMAILS": "admin@example.test"},
	})

	for _, seeder := range []vukem.AppSeeder{seeders.FinanceAdminSeeder{}, seeders.FinanceDemoSeeder{}} {
		if err := seeder.SeedApp(t.Context(), app.App); err != nil {
			t.Fatal(err)
		}
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), 1)

	accounts, err := workspaces.Scoped[models.Account](ctx, db).Get(ctx)
	if err != nil || len(accounts) != 2 {
		t.Fatalf("accounts = %d, %v; want 2", len(accounts), err)
	}

	legs, err := workspaces.Scoped[models.Transaction](ctx, db).WhereNotNull("transfer_uid").Get(ctx)
	if err != nil || len(legs) != 2 {
		t.Fatalf("transfer legs = %d, %v; want 2", len(legs), err)
	}

	if legs[0].TransferUID != legs[1].TransferUID || legs[0].AccountID == legs[1].AccountID ||
		legs[0].AmountCents != legs[1].AmountCents || legs[0].Kind == legs[1].Kind || legs[0].CategoryID.Valid || legs[1].CategoryID.Valid {
		t.Fatalf("legs = %+v, want an expense and an income of the same amount on two accounts, uncategorized", legs)
	}

	// Rerunning the seed keeps what exists.
	if err := (seeders.FinanceDemoSeeder{}).SeedApp(t.Context(), app.App); err != nil {
		t.Fatal(err)
	}

	if n, _ := workspaces.Scoped[models.Account](ctx, db).Count(ctx); n != 2 {
		t.Fatalf("a rerun left %d accounts, want 2", n)
	}
}
