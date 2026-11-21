package finance

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// PageSize is how many transactions one page of the list shows.
const PageSize = 50

// maxSearchLength bounds the text of a search, in characters.
const maxSearchLength = 200

// TxFilter narrows the transaction list; zero values mean "any".
type TxFilter struct {
	From, To   string
	CategoryID int64
	AccountID  int64
	Kind       string
	// Q is a case-insensitive text search of the description (ASCII letters
	// fold; SQLite does not fold the others); wildcards in it are literal. At
	// most 200 characters are used.
	Q string
	// TagIDs keeps the transactions that carry every one of these tags.
	TagIDs []int64
	// Min and Max bound the amount (an absolute value, whatever the kind) and
	// are read by ParseAmount: "1.234,56" is 123456 cents. Empty means no bound.
	Min, Max string
	// Page is 1-based; it matters to TransactionPage only.
	Page int
}

// TxPage is one page of the filtered list with the size of the whole result.
type TxPage struct {
	Rows     []TxRow `json:"rows"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

// likeEscape makes s literal inside a LIKE pattern that declares ESCAPE '\':
// the backslash, the percent sign and the underscore are escaped.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// clip keeps the first n characters of s.
func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	return string([]rune(s)[:n])
}

func (s *Service) filtered(ctx context.Context, f TxFilter) (orm.Query[models.Transaction], error) {
	q := workspaces.Scoped[models.Transaction](ctx, s.DB)

	if f.From != "" && validDate(f.From) {
		q = q.Where("occurred_on", ">=", f.From)
	}

	if f.To != "" && validDate(f.To) {
		q = q.Where("occurred_on", "<=", f.To)
	}

	if f.CategoryID > 0 {
		// A parent category also shows what was filed under its subcategories.
		ids := []any{f.CategoryID}

		kids, err := workspaces.Scoped[models.Category](ctx, s.DB).Where("parent_id", f.CategoryID).Pluck[int64](ctx, "id")
		if err != nil {
			return q, err
		}

		for _, k := range kids {
			ids = append(ids, k)
		}

		q = q.WhereIn("category_id", ids...)
	}

	if f.AccountID > 0 {
		q = q.Where("account_id", f.AccountID)
	}

	if f.Kind == models.KindIncome || f.Kind == models.KindExpense {
		q = q.Where("kind", f.Kind)
	}

	if text := strings.TrimSpace(clip(f.Q, maxSearchLength)); text != "" {
		q = q.WhereRaw(`lower(description) LIKE lower(?) ESCAPE '\'`, "%"+likeEscape(text)+"%")
	}

	if len(f.TagIDs) > 0 {
		ws, err := s.WS.Current(ctx)
		if err != nil {
			return q, err
		}

		ids := anys(f.TagIDs)
		args := append([]any{ws.ID}, ids...)
		args = append(args, len(f.TagIDs))
		marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		q = q.WhereRaw(`id IN (SELECT transaction_id FROM transaction_tags WHERE workspace_id = ? AND tag_id IN (`+
			marks+`) GROUP BY transaction_id HAVING COUNT(DISTINCT tag_id) = ?)`, args...)
	}

	for _, bound := range []struct {
		text, op string
	}{{f.Min, ">="}, {f.Max, "<="}} {
		if strings.TrimSpace(bound.text) == "" {
			continue
		}

		cents, err := ParseAmount(bound.text)
		if err != nil || cents < 0 {
			return q, errAmountIn
		}

		q = q.Where("amount_cents", bound.op, cents)
	}

	return q, nil
}

// Transactions lists the filtered transactions, newest first (at most 500).
func (s *Service) Transactions(ctx context.Context, f TxFilter) ([]TxRow, error) {
	q, err := s.filtered(ctx, f)
	if err != nil {
		return nil, err
	}

	txs, err := q.OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).Limit(500).Get(ctx)
	if err != nil {
		return nil, err
	}

	return s.rowsOf(ctx, txs)
}

// TransactionPage lists one page of the filtered transactions, newest first
// (the order is (occurred_on, id), so a page never repeats or skips a row
// while nothing changes), with the total count of the filter.
func (s *Service) TransactionPage(ctx context.Context, f TxFilter) (TxPage, error) {
	q, err := s.filtered(ctx, f)
	if err != nil {
		return TxPage{}, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return TxPage{}, err
	}

	page := max(f.Page, 1)

	txs, err := q.OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).
		Limit(PageSize).Offset((page - 1) * PageSize).Get(ctx)
	if err != nil {
		return TxPage{}, err
	}

	rows, err := s.rowsOf(ctx, txs)

	return TxPage{Rows: rows, Total: total, Page: page, PageSize: PageSize}, err
}

// Rows adds the names, tags and transfer side the list shows to txs.
func (s *Service) Rows(ctx context.Context, txs []models.Transaction) ([]TxRow, error) {
	return s.rowsOf(ctx, txs)
}

func (s *Service) rowsOf(ctx context.Context, txs []models.Transaction) ([]TxRow, error) {
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

	ids := make([]int64, len(txs))
	for i := range txs {
		ids[i] = txs[i].ID
	}

	tags, err := s.tagsOf(ctx, ids)
	if err != nil {
		return nil, err
	}

	rows := make([]TxRow, len(txs))
	for i := range txs {
		t := &txs[i]
		rows[i] = TxRow{Transaction: *t, AccountName: an[t.AccountID], CategoryName: cn[t.CategoryID.V], Tags: tags[t.ID]}

		if rows[i].Tags == nil {
			rows[i].Tags = []TagRef{}
		}

		if t.CategoryID.Valid {
			rows[i].CategoryIDValue = &t.CategoryID.V
		}

		if t.TransferUID.Valid {
			rows[i].TransferID = &t.TransferUID.V
			rows[i].TransferAccountName = an[other[t.TransferUID.V][t.ID]]
			otherID := other[t.TransferUID.V][t.ID]
			rows[i].TransferAccountID = &otherID
		}
	}

	return rows, nil
}
