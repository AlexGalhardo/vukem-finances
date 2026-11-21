package finance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// MaxImportBytes bounds an uploaded statement; the content is kept in the
// imports table until the job has run, then dropped.
const MaxImportBytes = 2 << 20

// maxImportRows bounds what one import creates.
const maxImportRows = 20000

var (
	errImportTooBig = errs.New(http.StatusRequestEntityTooLarge, "The file is too big: the limit is 2 MiB.")
	errImportCSV    = errs.New(http.StatusUnprocessableEntity, "The file is not a CSV with a header row and a line.")
	errImportState  = errs.New(http.StatusConflict, "This import was already started.")
	errMapping      = errs.New(http.StatusUnprocessableEntity, "Choose the date, description and amount columns.")
)

// Preview is what the mapping step shows: the header and a few rows.
type Preview struct {
	ImportID int64      `json:"importId"`
	Filename string     `json:"filename"`
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
}

// StartImport stores an uploaded statement for account (Pro) and returns the
// preview the user maps columns from.
func (s *Service) StartImport(ctx context.Context, accountID int64, filename string, content []byte) (Preview, error) {
	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return Preview{}, err
	}

	if err := lim.CheckImport(); err != nil {
		return Preview{}, err
	}

	if len(content) > MaxImportBytes {
		return Preview{}, errImportTooBig
	}

	if ok, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("id", accountID).Exists(ctx); err != nil || !ok {
		return Preview{}, firstErr(err, errAccount)
	}

	if err := refuseArchived(ctx, s.DB, accountID); err != nil {
		return Preview{}, err
	}

	headers, rows, err := readCSV(string(content))
	if err != nil {
		return Preview{}, err
	}

	imp := models.Import{
		AccountID: accountID, Filename: strings.TrimSpace(filename), Content: string(content),
		Status: models.ImportUploaded,
	}
	if err := orm.Create(ctx, s.DB, &imp); err != nil {
		return Preview{}, err
	}

	return Preview{ImportID: imp.ID, Filename: imp.Filename, Headers: headers, Rows: rows[:min(5, len(rows))]}, nil
}

// ConfirmImport saves the column mapping and queues the job that creates
// the transactions.
func (s *Service) ConfirmImport(
	ctx context.Context,
	id int64,
	m Mapping,
	dispatch func(context.Context, int64, int64) error,
) error {
	imp, err := workspaces.Scoped[models.Import](ctx, s.DB).Find(ctx, id)
	if err != nil {
		return err
	}

	if imp.Status != models.ImportUploaded {
		return errImportState
	}

	if err := refuseArchived(ctx, s.DB, imp.AccountID); err != nil {
		return err
	}

	headers, _, err := readCSV(imp.Content)
	if err != nil {
		return err
	}

	for _, col := range []string{m.Date, m.Description, m.Amount} {
		if indexOf(headers, col) < 0 {
			return errMapping
		}
	}

	if m.Category != "" && indexOf(headers, m.Category) < 0 {
		return errMapping
	}

	if m.DateFormat == "" {
		m.DateFormat = DateFormats[0]
	}

	if !contains(DateFormats, m.DateFormat) {
		return errMapping
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).
		Update(ctx, orm.Set{"mapping": string(raw), "status": models.ImportQueued})
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	return dispatch(ctx, ws, id)
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}

	return -1
}

func contains(list []string, s string) bool { return indexOf(list, s) >= 0 }

// Imports lists the latest imports of the workspace.
func (s *Service) Imports(ctx context.Context) ([]models.Import, error) {
	return workspaces.Scoped[models.Import](ctx, s.DB).OrderBy("id", orm.Desc).Limit(20).Get(ctx)
}

// RunImport turns the rows of an import into transactions. It runs in the
// queued job, under the tenant context of the workspace that started it.
// Rows that do not parse are counted and skipped, not fatal; a statement
// that cannot be read at all marks the import failed. Budget alerts are
// checked once per category and month at the end.
func (s *Service) RunImport(ctx context.Context, id int64) error {
	imp, err := workspaces.Scoped[models.Import](ctx, s.DB).Find(ctx, id)
	if err != nil {
		return err
	}

	if imp.Status != models.ImportQueued {
		return nil // a retry of a job that already finished
	}

	// The account may have been archived while the job waited in the queue.
	if err := refuseArchived(ctx, s.DB, imp.AccountID); err != nil {
		if !errors.Is(err, errArchivedEntry) {
			return err
		}

		_, uerr := workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).
			Update(ctx, orm.Set{"status": models.ImportFailed, "error": err.Error(), "content": ""})

		return uerr
	}

	made, bad, notes, err := s.importRows(ctx, imp)
	if err != nil {
		_, uerr := workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).
			Update(ctx, orm.Set{"status": models.ImportFailed, "error": err.Error(), "content": ""})

		return uerr // the failure is recorded on the import; retrying the job would not help
	}

	type key struct {
		cat   int64
		month string
	}

	added := map[key]int64{}

	for i := range made {
		if t := &made[i]; t.Kind == models.KindExpense && t.CategoryID.Valid {
			added[key{t.CategoryID.V, MonthOf(t.OccurredOn)}] += t.AmountCents
		}
	}

	for k, delta := range added {
		if err := s.alertBudget(ctx, k.cat, k.month, delta); err != nil {
			return err
		}
	}

	_, err = workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).Update(ctx, orm.Set{
		"status": models.ImportDone, "imported": len(made), "failed": bad, "error": strings.Join(notes, "\n"),
		"content": "",
	})

	return err
}

func (s *Service) importRows(
	ctx context.Context,
	imp models.Import,
) (made []models.Transaction, bad int, notes []string, err error) {
	var m Mapping
	if err := json.Unmarshal([]byte(imp.Mapping), &m); err != nil {
		return nil, 0, nil, errMapping
	}

	rows, err := CSVParser{Mapping: m}.Parse([]byte(imp.Content))
	if err != nil {
		return nil, 0, nil, err
	}

	cats, err := s.Categories(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	byName := map[string]models.Category{}

	for _, c := range cats {
		if c.ArchivedAt.Valid {
			continue
		}

		byName[strings.ToLower(c.Name)+"/"+c.Kind] = c
	}

	for _, row := range rows {
		if row.Problem != "" {
			bad++

			if len(notes) < 5 {
				notes = append(notes, fmt.Sprintf("line %d: %s", row.Line, row.Problem))
			}

			continue
		}

		made = append(made, rowToTransaction(row, imp, byName))
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		for i := range made {
			if err := orm.Create(ctx, tx, &made[i]); err != nil {
				return err
			}
		}

		return nil
	})

	return made, bad, notes, err
}

func rowToTransaction(row StatementRow, imp models.Import, cats map[string]models.Category) models.Transaction {
	t := models.Transaction{
		AccountID: imp.AccountID, Description: row.Description, OccurredOn: row.Date,
		ImportID: sql.Null[int64]{V: imp.ID, Valid: true}, Kind: models.KindIncome, AmountCents: row.Cents,
	}

	if row.Cents < 0 {
		t.Kind, t.AmountCents = models.KindExpense, -row.Cents
	}

	if c, ok := cats[strings.ToLower(row.Category)+"/"+t.Kind]; ok && row.Category != "" {
		t.CategoryID = sql.Null[int64]{V: c.ID, Valid: true}
	}

	return t
}
