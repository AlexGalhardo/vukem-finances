package finance

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const isoDate = "2006-01-02"

var (
	errAccount  = errs.New(http.StatusUnprocessableEntity, "Choose one of your accounts.")
	errCategory = errs.New(http.StatusUnprocessableEntity, "Choose one of your categories.")
	errKind     = errs.New(http.StatusUnprocessableEntity, "The kind must be income or expense.")
	errDate     = errs.New(http.StatusUnprocessableEntity, "The date must look like 2026-11-30.")
	errAmountIn = errs.New(http.StatusUnprocessableEntity, "The amount must be a positive number like 12.50.")
	errExpense  = errs.New(http.StatusUnprocessableEntity, "A budget needs an expense category.")
	errName     = errs.New(http.StatusConflict, "That name is already taken.")
)

// Service holds the finance rules of the workspace in the context.
type Service struct {
	DB     *database.DB
	WS     *workspaces.Service
	Mailer *mail.Mailer
	Queue  *queue.Queue
	URL    string

	now func() time.Time
}

// New builds the service from the app and the workspace service.
func New(app *vukem.App, ws *workspaces.Service) (*Service, error) {
	q, err := app.Queue()
	if err != nil {
		return nil, err
	}

	return &Service{DB: ws.DB, WS: ws, Mailer: ws.Mailer, Queue: q, URL: ws.URL, now: app.Now}, nil
}

// Today is the current ISO date where the people of the workspace of ctx
// live: the clock read in the workspace timezone, UTC when it has none.
func (s *Service) Today(ctx context.Context) string { return s.localNow(ctx).Format(isoDate) }

// Month is the current "YYYY-MM" in the timezone of the workspace of ctx.
func (s *Service) Month(ctx context.Context) string { return s.localNow(ctx).Format("2006-01") }

func (s *Service) localNow(ctx context.Context) time.Time {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return s.now().UTC()
	}

	return s.nowIn(ws)
}

// nowIn is the clock read in the timezone of ws; a name that no longer loads
// (it was validated on save) falls back to UTC.
func (s *Service) nowIn(ws models.Workspace) time.Time {
	loc, err := workspaces.LoadZone(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}

	return s.now().In(loc)
}

// MonthOf returns the "YYYY-MM" of an ISO date.
func MonthOf(date string) string { return date[:7] }

// monthBounds returns the first and last possible ISO date of a "YYYY-MM"
// month; ISO dates compare as text, so 31 bounds every month.
func monthBounds(month string) (from, to string) { return month + "-01", month + "-31" }

// ---- accounts and categories ----

// Categories lists the categories of the workspace of ctx.
func (s *Service) Categories(ctx context.Context) ([]models.Category, error) {
	return workspaces.Scoped[models.Category](ctx, s.DB).OrderBy("kind", orm.Asc).OrderBy("name", orm.Asc).Get(ctx)
}

// CreateCategory adds a category; names are unique per workspace.
func (s *Service) CreateCategory(ctx context.Context, name, kind string) (models.Category, error) {
	if kind != models.KindIncome && kind != models.KindExpense {
		return models.Category{}, errKind
	}

	name = strings.TrimSpace(name)

	taken, err := workspaces.Scoped[models.Category](ctx, s.DB).Where("name", name).Exists(ctx)
	if err != nil {
		return models.Category{}, err
	}

	if taken {
		return models.Category{}, errName
	}

	c := models.Category{Name: name, Kind: kind}
	if err := orm.Create(ctx, s.DB, &c); err != nil {
		// Two devices that create the same name at once: the partial unique
		// index on live names settles it, and the loser gets the same answer.
		if s.DB.UniqueViolation(err) {
			return models.Category{}, errName
		}

		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, c.ID)
}

// ---- transactions ----

// TxInput is a new transaction as the form sends it.
type TxInput struct {
	AccountID   int64
	CategoryID  int64
	Kind        string
	Amount      string
	Description string
	Date        string
}

// TxFilter narrows the transaction list; zero values mean "any".
type TxFilter struct {
	From, To   string
	CategoryID int64
	AccountID  int64
	Kind       string
}

// TxRow is a transaction with the names the list shows.
type TxRow struct {
	models.Transaction

	CategoryIDValue *int64 `json:"categoryId"`
	AccountName     string `json:"accountName"`
	CategoryName    string `json:"categoryName"`

	// TransferID is set on a transfer leg, and TransferAccountName is the
	// account on the other side of it.
	TransferID          *string `json:"transferId"`
	TransferAccountName string  `json:"transferAccountName"`
}

func validDate(s string) bool {
	_, err := time.Parse(isoDate, s)
	return err == nil
}

// Transactions lists the filtered transactions, newest first (at most 500).
func (s *Service) Transactions(ctx context.Context, f TxFilter) ([]TxRow, error) {
	q := workspaces.Scoped[models.Transaction](ctx, s.DB)

	if f.From != "" && validDate(f.From) {
		q = q.Where("occurred_on", ">=", f.From)
	}

	if f.To != "" && validDate(f.To) {
		q = q.Where("occurred_on", "<=", f.To)
	}

	if f.CategoryID > 0 {
		q = q.Where("category_id", f.CategoryID)
	}

	if f.AccountID > 0 {
		q = q.Where("account_id", f.AccountID)
	}

	if f.Kind == models.KindIncome || f.Kind == models.KindExpense {
		q = q.Where("kind", f.Kind)
	}

	txs, err := q.OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).Limit(500).Get(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := s.Accounts(ctx)
	if err != nil {
		return nil, err
	}

	cats, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}

	an := map[int64]string{}
	for _, a := range accounts {
		an[a.ID] = a.Name
	}

	cn := map[int64]string{}
	for _, c := range cats {
		cn[c.ID] = c.Name
	}

	other, err := s.otherLegAccounts(ctx, txs)
	if err != nil {
		return nil, err
	}

	rows := make([]TxRow, len(txs))
	for i := range txs {
		t := &txs[i]
		rows[i] = TxRow{Transaction: *t, AccountName: an[t.AccountID], CategoryName: cn[t.CategoryID.V]}
		if t.CategoryID.Valid {
			rows[i].CategoryIDValue = &t.CategoryID.V
		}

		if t.TransferUID.Valid {
			rows[i].TransferID = &t.TransferUID.V
			rows[i].TransferAccountName = an[other[t.TransferUID.V][t.ID]]
		}
	}

	return rows, nil
}

// CreateTransaction records a transaction. The account and the category
// must belong to the workspace, the amount is positive, and the Free plan
// stops at its transactions per month. An expense may raise a budget alert.
func (s *Service) CreateTransaction(ctx context.Context, in TxInput) (models.Transaction, error) {
	return s.saveTransaction(ctx, 0, in)
}

func firstErr(err, fallback error) error {
	if err != nil {
		return err
	}

	return fallback
}

// ---- budgets ----

// BudgetStatus is a budget with what the month has spent against it.
type BudgetStatus struct {
	ID         int64  `json:"id"`
	CategoryID int64  `json:"categoryId"`
	Category   string `json:"category"`
	LimitCents int64  `json:"limitCents"`
	SpentCents int64  `json:"spentCents"`
	Percent    int    `json:"percent"`
}

// SetBudget sets the monthly limit of an expense category (Pro).
func (s *Service) SetBudget(ctx context.Context, categoryID int64, limit string) error {
	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return err
	}

	if err := lim.CheckBudgets(); err != nil {
		return err
	}

	cents, err := ParseAmount(limit)
	if err != nil || cents <= 0 {
		return errAmountIn
	}

	ok, err := workspaces.Scoped[models.Category](
		ctx,
		s.DB,
	).Where("id", categoryID).
		Where("kind", models.KindExpense).
		Exists(ctx)
	if err != nil || !ok {
		return firstErr(err, errExpense)
	}

	n, err := workspaces.Scoped[models.Budget](ctx, s.DB).Where("category_id", categoryID).
		Update(ctx, orm.Set{"limit_cents": cents})
	if err != nil || n > 0 {
		return err
	}

	err = orm.Create(ctx, s.DB, &models.Budget{CategoryID: categoryID, LimitCents: cents})
	if s.DB.UniqueViolation(err) {
		// A concurrent writer created the budget first: this one is an edit.
		_, err = workspaces.Scoped[models.Budget](ctx, s.DB).Where("category_id", categoryID).
			Update(ctx, orm.Set{"limit_cents": cents})
	}

	return err
}

func (s *Service) spent(ctx context.Context, categoryID int64, month string) (int64, error) {
	from, to := monthBounds(month)

	return workspaces.Scoped[models.Transaction](ctx, s.DB).Where("category_id", categoryID).
		WhereNull("transfer_uid").Where("kind", models.KindExpense).WhereBetween("occurred_on", from, to).Sum[int64](ctx, "amount_cents")
}

// Budgets lists the budgets with what month has spent against each.
func (s *Service) Budgets(ctx context.Context, month string) ([]BudgetStatus, error) {
	budgets, err := workspaces.Scoped[models.Budget](ctx, s.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	cats, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}

	names := map[int64]string{}
	for _, c := range cats {
		names[c.ID] = c.Name
	}

	out := make([]BudgetStatus, 0, len(budgets))

	for _, b := range budgets {
		spent, err := s.spent(ctx, b.CategoryID, month)
		if err != nil {
			return nil, err
		}

		out = append(out, BudgetStatus{
			ID: b.ID, CategoryID: b.CategoryID, Category: names[b.CategoryID], LimitCents: b.LimitCents,
			SpentCents: spent, Percent: int(spent * 100 / b.LimitCents),
		})
	}

	return out, nil
}

// alertBudget emails the owners when delta (cents just added to the month's
// spending of the category) made it cross 80% or 100% of the budget. It
// compares the spending before and after, so no alert is ever sent twice.
func (s *Service) alertBudget(ctx context.Context, categoryID int64, month string, delta int64) error {
	b, err := workspaces.Scoped[models.Budget](ctx, s.DB).Where("category_id", categoryID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	spent, err := s.spent(ctx, categoryID, month)
	if err != nil {
		return err
	}

	before := spent - delta
	percent := 0

	for _, p := range []int64{80, 100} {
		if before*100 < p*b.LimitCents && spent*100 >= p*b.LimitCents {
			percent = int(p)
		}
	}

	if percent == 0 {
		return nil
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	cat, err := workspaces.Scoped[models.Category](ctx, s.DB).Find(ctx, categoryID)
	if err != nil {
		return err
	}

	owners, err := s.WS.OwnerEmails(ctx, ws.ID)
	if err != nil {
		return err
	}

	for _, to := range owners {
		err := s.WS.Notify(ctx, appmail.BudgetAlert{
			To: to, Workspace: ws.Name, Category: cat.Name, Percent: percent,
			Spent: FormatAmount(spent, ws.Currency), Limit: FormatAmount(b.LimitCents, ws.Currency),
			URL: s.URL + "/budgets",
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// otherLegAccounts maps each transfer uid among txs to, per leg id, the
// account of the other leg, so a row can name the account on the other side
// even when the filter hid that leg.
func (s *Service) otherLegAccounts(ctx context.Context, txs []models.Transaction) (map[string]map[int64]int64, error) {
	var uids []any

	for _, t := range txs {
		if t.TransferUID.Valid {
			uids = append(uids, t.TransferUID.V)
		}
	}

	out := map[string]map[int64]int64{}
	if len(uids) == 0 {
		return out, nil
	}

	legs, err := workspaces.Scoped[models.Transaction](ctx, s.DB).WhereIn("transfer_uid", uids...).Get(ctx)
	if err != nil {
		return nil, err
	}

	byUID := map[string][]models.Transaction{}
	for _, l := range legs {
		byUID[l.TransferUID.V] = append(byUID[l.TransferUID.V], l)
	}

	for uid, pair := range byUID {
		out[uid] = map[int64]int64{}

		for _, l := range pair {
			for _, o := range pair {
				if o.ID != l.ID {
					out[uid][l.ID] = o.AccountID
				}
			}
		}
	}

	return out, nil
}
