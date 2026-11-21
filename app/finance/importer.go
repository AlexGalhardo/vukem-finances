package finance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// MaxImportBytes bounds an uploaded statement (CSV or OFX); the content is
// kept in the imports table until the job has run, then dropped.
const MaxImportBytes = 2 << 20

const (
	// maxImportRows bounds what one import reads: a file with more is refused, never cut short.
	maxImportRows = 20000
	// maxReportFailures is how many failed rows the stored report lists.
	maxReportFailures = 100
	// previewRows is how many rows the preview shows.
	previewRows = 20
	// maxDescriptionLength is the longest description an import stores, in characters.
	maxDescriptionLength = 200
	// lookupChunk is how many hashes one duplicate lookup asks for.
	lookupChunk = 400
)

// Import formats.
const (
	FormatCSV = "csv"
	FormatOFX = "ofx"
)

var (
	errImportTooBig = errs.New(http.StatusRequestEntityTooLarge, "The file is too big: the limit is 2 MiB.")
	errImportCSV    = errs.New(http.StatusUnprocessableEntity, "The file is not a CSV with a header row and a line.")
	errImportState  = errs.New(http.StatusConflict, "This import was already started.")
	errMapping      = errs.New(http.StatusUnprocessableEntity, "Choose the date, description and amount columns.")
	errImportFailed = errs.New(http.StatusInternalServerError, "The import failed and nothing was imported.")
)

// ImportFailure is a row that could not be imported.
type ImportFailure struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// PreviewRow is a row as it will be stored.
type PreviewRow struct {
	Line        int    `json:"line"`
	Date        string `json:"date"`
	Description string `json:"description"`
	// Cents is signed: negative is an expense.
	Cents     int64  `json:"cents"`
	Category  string `json:"category"`
	Duplicate bool   `json:"duplicate"`
	Problem   string `json:"problem"`
}

// MappedPreview is what an import would do: the first rows as they will be
// stored and the totals of the whole file.
type MappedPreview struct {
	Rows []PreviewRow `json:"rows"`
	// Read is every row of the file; New would be imported, Duplicates are
	// already in the account (imported anyway if AllowDuplicates), Failed
	// cannot be read.
	Read       int  `json:"read"`
	New        int  `json:"new"`
	Duplicates int  `json:"duplicates"`
	Failed     int  `json:"failed"`
	Allowed    bool `json:"allowDuplicates"`
	// From and To are the first and last date of the readable rows.
	From     string          `json:"from"`
	To       string          `json:"to"`
	Failures []ImportFailure `json:"failures"`
}

// Preview is what the upload and mapping steps show. A CSV answers with its
// header and a few raw rows to map columns from; an OFX file, and a CSV once
// it is mapped, answer with Mapped.
type Preview struct {
	ImportID int64          `json:"importId"`
	Filename string         `json:"filename"`
	Format   string         `json:"format"`
	Headers  []string       `json:"headers"`
	Rows     [][]string     `json:"rows"`
	Mapped   *MappedPreview `json:"mapped"`
}

// ImportView is an import with its report decoded for the page.
type ImportView struct {
	models.Import

	Failures []ImportFailure `json:"failures"`
}

// StartImport stores an uploaded statement for account (Pro) and returns the
// preview: the columns to map for a CSV, the rows as they will be stored for
// an OFX file. A file that cannot be read stores nothing.
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

	format, err := DetectFormat(filename, content)
	if err != nil {
		return Preview{}, err
	}

	acc, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("id", accountID).First(ctx)
	if err != nil {
		return Preview{}, orElse(err, errAccount)
	}

	if err := refuseArchived(ctx, s.DB, accountID); err != nil {
		return Preview{}, err
	}

	out := Preview{Filename: strings.TrimSpace(filename), Format: format}

	var rows []StatementRow

	if format == FormatOFX {
		if rows, err = (OFXParser{}).Parse(content); err != nil {
			return Preview{}, err
		}
	} else {
		var raw [][]string
		if out.Headers, raw, err = readCSV(string(content)); err != nil {
			return Preview{}, err
		}

		out.Rows = raw[:min(5, len(raw))]
	}

	imp := models.Import{
		AccountID: accountID, Filename: out.Filename, Content: string(content), Format: format,
		Status: models.ImportUploaded,
	}
	if err := orm.Create(ctx, s.DB, &imp); err != nil {
		return Preview{}, err
	}

	out.ImportID = imp.ID

	if format == FormatOFX {
		if out.Mapped, err = s.mappedPreview(ctx, s.DB, acc, rows, false); err != nil {
			return Preview{}, err
		}
	}

	return out, nil
}

// PreviewImport shows what the import would do with a column mapping (a CSV)
// and the duplicate switch, without creating anything.
func (s *Service) PreviewImport(ctx context.Context, id int64, m Mapping) (Preview, error) {
	imp, err := workspaces.Scoped[models.Import](ctx, s.DB).Find(ctx, id)
	if err != nil {
		return Preview{}, err
	}

	if imp.Status != models.ImportUploaded {
		return Preview{}, errImportState
	}

	m, err = checkMapping(imp, m)
	if err != nil {
		return Preview{}, err
	}

	acc, err := reload[models.Account](ctx, s.DB, imp.AccountID)
	if err != nil {
		return Preview{}, err
	}

	rows, err := parserFor(imp, m).Parse([]byte(imp.Content))
	if err != nil {
		return Preview{}, err
	}

	out := Preview{ImportID: imp.ID, Filename: imp.Filename, Format: imp.Format}

	if imp.Format == FormatCSV {
		var raw [][]string
		if out.Headers, raw, err = readCSV(imp.Content); err != nil {
			return Preview{}, err
		}

		out.Rows = raw[:min(5, len(raw))]
	}

	out.Mapped, err = s.mappedPreview(ctx, s.DB, acc, rows, m.AllowDuplicates)

	return out, err
}

// checkMapping validates a mapping against the file. An OFX import has no
// columns: only the duplicate switch of the mapping counts.
func checkMapping(imp models.Import, m Mapping) (Mapping, error) {
	if imp.Format == FormatOFX {
		return Mapping{AllowDuplicates: m.AllowDuplicates}, nil
	}

	headers, _, err := readCSV(imp.Content)
	if err != nil {
		return m, err
	}

	for _, col := range []string{m.Date, m.Description, m.Amount} {
		if indexOf(headers, col) < 0 {
			return m, errMapping
		}
	}

	if m.Category != "" && indexOf(headers, m.Category) < 0 {
		return m, errMapping
	}

	if m.DateFormat == "" {
		m.DateFormat = DateFormats[0]
	}

	if !contains(DateFormats, m.DateFormat) {
		return m, errMapping
	}

	return m, nil
}

func parserFor(imp models.Import, m Mapping) StatementParser {
	if imp.Format == FormatOFX {
		return OFXParser{}
	}

	return CSVParser{Mapping: m}
}

// ConfirmImport saves the column mapping and the duplicate switch, and queues
// the job that creates the transactions.
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

	m, err = checkMapping(imp, m)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// Only an uploaded import moves to queued, so two confirmations queue one job.
	n, err := workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).Where("status", models.ImportUploaded).
		Update(ctx, orm.Set{"mapping": string(raw), "status": models.ImportQueued})
	if err != nil {
		return err
	}

	if n == 0 {
		return errImportState
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

// Imports lists the latest imports of the workspace with their reports.
func (s *Service) Imports(ctx context.Context) ([]ImportView, error) {
	imps, err := workspaces.Scoped[models.Import](ctx, s.DB).OrderBy("id", orm.Desc).Limit(20).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ImportView, len(imps))

	for i, imp := range imps {
		out[i] = ImportView{Import: imp, Failures: []ImportFailure{}}

		if imp.Report != "" {
			// A report this code wrote; an unreadable one shows no rows rather than failing the page.
			_ = json.Unmarshal([]byte(imp.Report), &out[i].Failures)
		}
	}

	return out, nil
}

// ---- duplicates ----

// normalizeDescription makes two spellings of one description equal: case and
// runs of white space do not matter.
func normalizeDescription(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func sum(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))

	return hex.EncodeToString(h[:])
}

// importHash is the duplicate key of a row: the SHA-256 of account, date,
// signed amount and normalized description, or, when the bank gave an id, the
// SHA-256 of account and FITID with the prefix "fitid:". The account is part
// of it, so the same row in two accounts is not a duplicate.
func importHash(accountUID string, r StatementRow) string {
	if r.FITID != "" {
		return "fitid:" + sum(accountUID, r.FITID)
	}

	return sum(accountUID, r.Date, strconv.FormatInt(r.Cents, 10), normalizeDescription(r.Description))
}

// candidate is a readable row with its hash and whether the account has it.
type candidate struct {
	row  StatementRow
	hash string
	dup  bool
}

// classify hashes the readable rows and marks the duplicates. Two identical
// rows in one file with no FITID are two purchases: the n-th copy gets the
// ordinal n in its hash, so a second import of the file finds both. The same
// FITID twice in a file is one transaction, so the second copy is a duplicate.
func (s *Service) classify(
	ctx context.Context, ex database.Executor, acc models.Account, rows []StatementRow,
) ([]candidate, error) {
	var out []candidate

	seen := map[string]int{}

	for _, r := range rows {
		if r.Problem != "" {
			continue
		}

		h := importHash(acc.UID, r)
		n := seen[h]
		seen[h]++

		c := candidate{row: r, hash: h}

		switch {
		case n > 0 && r.FITID != "":
			c.dup = true
		case n > 0:
			c.hash = sum(h, strconv.Itoa(n))
		}

		out = append(out, c)
	}

	hashes := make([]any, len(out))
	for i, c := range out {
		hashes[i] = c.hash
	}

	have := map[string]bool{}

	for start := 0; start < len(hashes); start += lookupChunk {
		got, err := workspaces.Scoped[models.Transaction](ctx, ex).Where("account_id", acc.ID).
			WhereIn("import_hash", hashes[start:min(len(hashes), start+lookupChunk)]...).Pluck[string](ctx, "import_hash")
		if err != nil {
			return nil, err
		}

		for _, h := range got {
			have[h] = true
		}
	}

	for i := range out {
		out[i].dup = out[i].dup || have[out[i].hash]
	}

	return out, nil
}

func (s *Service) mappedPreview(
	ctx context.Context, ex database.Executor, acc models.Account, rows []StatementRow, allow bool,
) (*MappedPreview, error) {
	cands, err := s.classify(ctx, ex, acc, rows)
	if err != nil {
		return nil, err
	}

	p := &MappedPreview{Read: len(rows), Allowed: allow, Failures: []ImportFailure{}, Rows: []PreviewRow{}}

	for _, r := range rows {
		if r.Problem != "" {
			p.Failed++

			if len(p.Failures) < maxReportFailures {
				p.Failures = append(p.Failures, ImportFailure{Line: r.Line, Reason: r.Problem})
			}
		}
	}

	for _, c := range cands {
		if c.dup {
			p.Duplicates++
		} else {
			p.New++
		}

		if p.From == "" || c.row.Date < p.From {
			p.From = c.row.Date
		}

		if c.row.Date > p.To {
			p.To = c.row.Date
		}
	}

	// The first rows of the file in file order, bad ones included, so the
	// person sees the file the way it will be read.
	byLine := map[int]candidate{}
	for _, c := range cands {
		byLine[c.row.Line] = c
	}

	for _, r := range rows {
		if len(p.Rows) >= previewRows {
			break
		}

		if r.Problem != "" {
			p.Rows = append(p.Rows, PreviewRow{Line: r.Line, Problem: r.Problem})
			continue
		}

		c := byLine[r.Line]
		p.Rows = append(p.Rows, PreviewRow{
			Line: r.Line, Date: r.Date, Description: shortDescription(r.Description), Cents: r.Cents,
			Category: r.Category, Duplicate: c.dup,
		})
	}

	return p, nil
}

func shortDescription(s string) string { return strings.TrimSpace(clip(s, maxDescriptionLength)) }

// ---- the job ----

// RunImport turns the rows of an import into transactions. It runs in the
// queued job, under the tenant context of the workspace that started it, and
// is all or nothing: the rows, the counts and the status are written in one
// database transaction under the workspace lock, so a failed import leaves no
// row behind and a second run of the same job does nothing. Rows that do not
// parse are counted and reported, not fatal; rows already imported are
// skipped unless the import allows duplicates; a statement that cannot be
// read at all, or that would break the plan's monthly limit, marks the
// import failed. Budget alerts are checked once per category and month at
// the end.
func (s *Service) RunImport(ctx context.Context, id int64) error {
	imp, err := workspaces.Scoped[models.Import](ctx, s.DB).Find(ctx, id)
	if err != nil {
		return err
	}

	if imp.Status != models.ImportQueued {
		return nil // a retry of a job that already finished
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return err
	}

	var made []models.Transaction

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		fresh, err := reload[models.Import](ctx, tx, id)
		if err != nil {
			return err
		}

		if fresh.Status != models.ImportQueued {
			return nil
		}

		made, err = s.importRows(ctx, tx, fresh, lim)

		return err
	})
	if err != nil {
		return s.failImport(ctx, id, err)
	}

	return s.alertImported(ctx, made)
}

// failImport records why an import failed. The reason is shown on the page,
// so only a message this package wrote reaches it; anything else (a database
// error) is logged by the job runner and shown as a plain failure.
func (s *Service) failImport(ctx context.Context, id int64, cause error) error {
	msg := errImportFailed.Message

	if e, ok := errors.AsType[*errs.HTTPError](cause); ok && e.Status < http.StatusInternalServerError {
		msg = e.Message
	}

	_, err := workspaces.Scoped[models.Import](ctx, s.DB).Where("id", id).
		Update(ctx, orm.Set{"status": models.ImportFailed, "error": msg, "content": ""})
	if err != nil {
		return err
	}

	// The failure is recorded on the import; retrying the job would not help,
	// except after a database error, which the runner should see.
	if _, ok := errors.AsType[*errs.HTTPError](cause); ok {
		return nil
	}

	return cause
}

func (s *Service) alertImported(ctx context.Context, made []models.Transaction) error {
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

	return nil
}

func (s *Service) importRows(
	ctx context.Context, tx *database.Tx, imp models.Import, lim workspaces.Limits,
) ([]models.Transaction, error) {
	if err := refuseArchived(ctx, tx, imp.AccountID); err != nil {
		return nil, err
	}

	var m Mapping
	if err := json.Unmarshal([]byte(imp.Mapping), &m); err != nil {
		return nil, errMapping
	}

	acc, err := reload[models.Account](ctx, tx, imp.AccountID)
	if err != nil {
		return nil, err
	}

	rows, err := parserFor(imp, m).Parse([]byte(imp.Content))
	if err != nil {
		return nil, err
	}

	cands, err := s.classify(ctx, tx, acc, rows)
	if err != nil {
		return nil, err
	}

	cats, err := workspaces.Scoped[models.Category](ctx, tx).Get(ctx)
	if err != nil {
		return nil, err
	}

	byName := map[string]models.Category{}

	for _, c := range cats {
		if !c.ArchivedAt.Valid {
			byName[strings.ToLower(c.Name)+"/"+c.Kind] = c
		}
	}

	var made []models.Transaction

	dups := 0

	for _, c := range cands {
		if c.dup && !m.AllowDuplicates {
			dups++
			continue
		}

		made = append(made, rowToTransaction(c, imp, byName))
	}

	if err := checkImportQuota(ctx, tx, lim, made); err != nil {
		return nil, err
	}

	for i := range made {
		if err := orm.Create(ctx, tx, &made[i]); err != nil {
			return nil, err
		}
	}

	report, notes, bad := failureReport(rows)

	_, err = workspaces.Scoped[models.Import](ctx, tx).Where("id", imp.ID).Update(ctx, orm.Set{
		"status": models.ImportDone, "imported": len(made), "failed": bad, "duplicates": dups,
		"error": strings.Join(notes, "\n"), "report": report, "content": "",
	})

	return made, err
}

// failureReport lists the unreadable rows: the JSON of up to 100 of them, the
// first five as text, and how many there are.
func failureReport(rows []StatementRow) (report string, notes []string, bad int) {
	list := []ImportFailure{}

	for _, r := range rows {
		if r.Problem == "" {
			continue
		}

		bad++

		if len(list) < maxReportFailures {
			list = append(list, ImportFailure{Line: r.Line, Reason: r.Problem})
		}

		if len(notes) < 5 {
			notes = append(notes, fmt.Sprintf("line %d: %s", r.Line, r.Problem))
		}
	}

	raw, _ := json.Marshal(list)

	return string(raw), notes, bad
}

// checkImportQuota refuses an import that would take a month over the
// transactions the plan allows (none of the plans that can import has such a
// limit today; the check keeps an import from being a way around one).
func checkImportQuota(ctx context.Context, tx *database.Tx, lim workspaces.Limits, made []models.Transaction) error {
	if lim.Transactions == workspaces.Unlimited {
		return nil
	}

	added := map[string]int{}
	for i := range made {
		added[MonthOf(made[i].OccurredOn)]++
	}

	existing := map[string]int{}

	for month := range added {
		from, to := monthBounds(month)

		n, err := workspaces.Scoped[models.Transaction](ctx, tx).WhereNull("transfer_uid").
			WhereBetween("occurred_on", from, to).Count(ctx)
		if err != nil {
			return err
		}

		existing[month] = int(n)
	}

	return overQuota(lim, existing, added)
}

// overQuota fails when any month, with the rows an import adds to what it
// already holds, goes over the transactions per month of the plan.
func overQuota(lim workspaces.Limits, existing, added map[string]int) error {
	for month, n := range added {
		if err := lim.CheckTransactions(existing[month] + n); err != nil {
			return err
		}
	}

	return nil
}

// rowToTransaction builds the transaction of a readable row. It is always an
// ordinary income or expense: an import never creates, joins or matches the
// legs of a transfer.
func rowToTransaction(c candidate, imp models.Import, cats map[string]models.Category) models.Transaction {
	r := c.row

	t := models.Transaction{
		AccountID: imp.AccountID, Description: shortDescription(r.Description), OccurredOn: r.Date,
		ImportID: sql.Null[int64]{V: imp.ID, Valid: true}, Kind: models.KindIncome, AmountCents: r.Cents,
		ImportHash: sql.Null[string]{V: c.hash, Valid: true},
	}

	if r.Cents < 0 {
		t.Kind, t.AmountCents = models.KindExpense, -r.Cents
	}

	if c, ok := cats[strings.ToLower(r.Category)+"/"+t.Kind]; ok && r.Category != "" {
		t.CategoryID = sql.Null[int64]{V: c.ID, Valid: true}
	}

	return t
}
