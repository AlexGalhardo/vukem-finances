package finance

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"

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
	Log    *slog.Logger

	now func() time.Time
}

// New builds the service from the app and the workspace service.
func New(app *vukem.App, ws *workspaces.Service) (*Service, error) {
	q, err := app.Queue()
	if err != nil {
		return nil, err
	}

	return &Service{DB: ws.DB, WS: ws, Mailer: ws.Mailer, Queue: q, URL: ws.URL, Log: app.Log, now: app.Now}, nil
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
	// Tags are tag names; nil leaves the tags of the transaction as they are
	// (none on a new one), a list, even an empty one, replaces them. Names
	// that do not exist yet are created.
	Tags []string
	// UID is the id a client chose for a new transaction (a ULID, checked by
	// the caller); empty makes the service generate one. Ignored on an edit.
	UID string

	// Set only by the service (a generated occurrence, an installment part):
	// unexported, so no request can claim a recurrence or a group.
	recurrenceUID string
	occurrenceOn  string
	installment   installmentMark
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
	TransferAccountID   *int64  `json:"transferAccountId"`

	Tags []TagRef `json:"tags"`
}

func validDate(s string) bool {
	_, err := time.Parse(isoDate, s)
	return err == nil
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
