package export

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// JSON export format name and version (docs/export-format.md).
const (
	JSONFormat  = "vukem-finances-export"
	JSONVersion = 1
)

// MaxJSONTransactions bounds one JSON export, like MaxCSVRows bounds a CSV.
const MaxJSONTransactions = 500000

// ErrTooManyTransactions is the answer to a workspace too big for one JSON export.
var ErrTooManyTransactions = errs.New(http.StatusRequestEntityTooLarge,
	"The workspace has more than 500000 transactions, which is more than one JSON export holds.")

// The types below are the document. Every field is named here on purpose:
// a column added to a model later reaches no export until it is added here
// and to docs/export-format.schema.json, whose additionalProperties is false.
// References between rows are uids; money is an integer in cents. Nothing of
// a user is exported but the display name and the role of a member.

type jsonMember struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type jsonWorkspace struct {
	Name     string       `json:"name"`
	Currency string       `json:"currency"`
	Timezone string       `json:"timezone"`
	Members  []jsonMember `json:"members"`
}

type jsonAccount struct {
	UID          string  `json:"uid"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	OpeningCents int64   `json:"opening_cents"`
	ArchivedAt   *string `json:"archived_at"`
	Position     int     `json:"position"`
}

type jsonCategory struct {
	UID        string  `json:"uid"`
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	ParentUID  *string `json:"parent_uid"`
	Color      string  `json:"color"`
	Icon       string  `json:"icon"`
	ArchivedAt *string `json:"archived_at"`
}

type jsonTag struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type jsonTransaction struct {
	UID                 string   `json:"uid"`
	AccountUID          string   `json:"account_uid"`
	CategoryUID         *string  `json:"category_uid"`
	Kind                string   `json:"kind"`
	AmountCents         int64    `json:"amount_cents"`
	Description         string   `json:"description"`
	OccurredOn          string   `json:"occurred_on"`
	TransferUID         *string  `json:"transfer_uid"`
	TagUIDs             []string `json:"tag_uids"`
	StatementRef        *string  `json:"statement_ref"`
	RecurrenceUID       *string  `json:"recurrence_uid"`
	OccurrenceOn        *string  `json:"occurrence_on"`
	InstallmentGroupUID *string  `json:"installment_group_uid"`
	InstallmentNo       *int64   `json:"installment_no"`
	InstallmentTotal    *int64   `json:"installment_total"`
}

type jsonBudget struct {
	UID         string  `json:"uid"`
	CategoryUID *string `json:"category_uid"`
	LimitCents  int64   `json:"limit_cents"`
	Rollover    bool    `json:"rollover"`
}

type jsonContribution struct {
	UID         string `json:"uid"`
	AmountCents int64  `json:"amount_cents"`
	OnDate      string `json:"on_date"`
	Note        string `json:"note"`
}

type jsonGoal struct {
	UID           string             `json:"uid"`
	Name          string             `json:"name"`
	TargetCents   int64              `json:"target_cents"`
	TargetDate    *string            `json:"target_date"`
	ArchivedAt    *string            `json:"archived_at"`
	Contributions []jsonContribution `json:"contributions"`
}

type jsonRecurrence struct {
	UID              string  `json:"uid"`
	AccountUID       string  `json:"account_uid"`
	CategoryUID      *string `json:"category_uid"`
	Kind             string  `json:"kind"`
	AmountCents      int64   `json:"amount_cents"`
	Description      string  `json:"description"`
	Frequency        string  `json:"frequency"`
	Interval         int     `json:"interval"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	MaxCount         *int64  `json:"max_count"`
	GeneratedThrough *string `json:"generated_through"`
	Active           bool    `json:"active"`
}

type jsonCard struct {
	UID               string  `json:"uid"`
	AccountUID        string  `json:"account_uid"`
	LimitCents        int64   `json:"limit_cents"`
	ClosingDay        int     `json:"closing_day"`
	DueDay            int     `json:"due_day"`
	PaymentAccountUID *string `json:"payment_account_uid"`
	Brand             string  `json:"brand"`
	Last4             string  `json:"last4"`
}

func str[T any](n sql.Null[T], f func(T) string) *string {
	if !n.Valid {
		return nil
	}

	s := f(n.V)

	return &s
}

func plain(s string) string { return s }

func stamp(n orm.NullTime) *string {
	if !n.Valid {
		return nil
	}

	s := n.V.UTC().Format(time.RFC3339)

	return &s
}

func num(n sql.Null[int64]) *int64 {
	if !n.Valid {
		return nil
	}

	v := n.V

	return &v
}

// jsonData is everything the document holds but the transactions, which stream.
type jsonData struct {
	workspace   jsonWorkspace
	accounts    []jsonAccount
	categories  []jsonCategory
	tags        []jsonTag
	budgets     []jsonBudget
	goals       []jsonGoal
	recurrences []jsonRecurrence
	cards       []jsonCard

	accountUID  map[int64]string
	categoryUID map[int64]string
}

// uidOf maps an id to a uid; a missing row gives nil.
func uidOf(ids map[int64]string, n sql.Null[int64]) *string {
	if !n.Valid {
		return nil
	}

	if u, ok := ids[n.V]; ok {
		return &u
	}

	return nil
}

// load reads the small tables of the workspace in ctx. Only live rows are
// exported, but a reference to a trashed account or category still resolves
// to its uid, so a transaction never loses where it belonged.
func load(ctx context.Context, fin *finance.Service) (*jsonData, error) {
	d := &jsonData{accountUID: map[int64]string{}, categoryUID: map[int64]string{}}

	ws, err := fin.WS.Current(ctx)
	if err != nil {
		return nil, err
	}

	members, err := fin.WS.Members(ctx)
	if err != nil {
		return nil, err
	}

	d.workspace = jsonWorkspace{Name: ws.Name, Currency: ws.Currency, Timezone: ws.Timezone, Members: []jsonMember{}}
	for _, m := range members {
		d.workspace.Members = append(d.workspace.Members, jsonMember{Name: m.Name, Role: m.Role})
	}

	accounts, err := workspaces.Scoped[models.Account](ctx, fin.DB).WithTrashed().OrderBy("position", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	d.accounts = []jsonAccount{}

	for _, a := range accounts {
		d.accountUID[a.ID] = a.UID

		if a.DeletedAt.Valid {
			continue
		}

		d.accounts = append(d.accounts, jsonAccount{
			UID: a.UID, Name: a.Name, Kind: a.Kind, OpeningCents: a.OpeningCents, ArchivedAt: stamp(a.ArchivedAt), Position: a.Position,
		})
	}

	cats, err := workspaces.Scoped[models.Category](ctx, fin.DB).WithTrashed().OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, c := range cats {
		d.categoryUID[c.ID] = c.UID
	}

	d.categories = []jsonCategory{}

	for _, c := range cats {
		if c.DeletedAt.Valid {
			continue
		}

		d.categories = append(d.categories, jsonCategory{
			UID: c.UID, Name: c.Name, Kind: c.Kind, ParentUID: uidOf(d.categoryUID, c.ParentID),
			Color: c.Color, Icon: c.Icon, ArchivedAt: stamp(c.ArchivedAt),
		})
	}

	tags, err := workspaces.Scoped[models.Tag](ctx, fin.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	d.tags = []jsonTag{}
	for _, t := range tags {
		d.tags = append(d.tags, jsonTag{UID: t.UID, Name: t.Name, Color: t.Color})
	}

	return d, d.loadRest(ctx, fin)
}

func (d *jsonData) loadRest(ctx context.Context, fin *finance.Service) error {
	budgets, err := workspaces.Scoped[models.Budget](ctx, fin.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return err
	}

	d.budgets = []jsonBudget{}
	for _, b := range budgets {
		d.budgets = append(d.budgets, jsonBudget{
			UID: b.UID, CategoryUID: uidOf(d.categoryUID, b.CategoryID), LimitCents: b.LimitCents, Rollover: b.Rollover,
		})
	}

	goals, err := workspaces.Scoped[models.Goal](ctx, fin.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return err
	}

	contributions, err := workspaces.Scoped[models.GoalContribution](ctx, fin.DB).
		OrderBy("on_date", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return err
	}

	byGoal := map[int64][]jsonContribution{}
	for _, c := range contributions {
		byGoal[c.GoalID] = append(byGoal[c.GoalID], jsonContribution{
			UID: c.UID, AmountCents: c.AmountCents, OnDate: c.OnDate, Note: c.Note,
		})
	}

	d.goals = []jsonGoal{}

	for _, g := range goals {
		list := byGoal[g.ID]
		if list == nil {
			list = []jsonContribution{}
		}

		d.goals = append(d.goals, jsonGoal{
			UID: g.UID, Name: g.Name, TargetCents: g.TargetCents, TargetDate: str(g.TargetDate, plain),
			ArchivedAt: stamp(g.ArchivedAt), Contributions: list,
		})
	}

	recs, err := workspaces.Scoped[models.Recurrence](ctx, fin.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return err
	}

	d.recurrences = []jsonRecurrence{}
	for _, r := range recs {
		d.recurrences = append(d.recurrences, jsonRecurrence{
			UID: r.UID, AccountUID: d.accountUID[r.AccountID], CategoryUID: uidOf(d.categoryUID, r.CategoryID), Kind: r.Kind,
			AmountCents: r.AmountCents, Description: r.Description, Frequency: r.Frequency, Interval: r.Interval,
			StartDate: r.StartDate, EndDate: str(r.EndDate, plain), MaxCount: num(r.MaxCount),
			GeneratedThrough: str(r.GeneratedThrough, plain), Active: r.Active,
		})
	}

	cards, err := workspaces.Scoped[models.CreditCard](ctx, fin.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return err
	}

	d.cards = []jsonCard{}
	for _, c := range cards {
		d.cards = append(d.cards, jsonCard{
			UID: c.UID, AccountUID: d.accountUID[c.AccountID], LimitCents: c.LimitCents, ClosingDay: c.ClosingDay, DueDay: c.DueDay,
			PaymentAccountUID: uidOf(d.accountUID, c.PaymentAccountID), Brand: c.Brand, Last4: c.Last4,
		})
	}

	return nil
}

// jsonWriter writes the object member by member, so the transactions never
// sit in memory together.
type jsonWriter struct {
	w   *bufio.Writer
	enc *json.Encoder
	err error
}

func (j *jsonWriter) raw(s string) {
	if j.err == nil {
		_, j.err = j.w.WriteString(s)
	}
}

func (j *jsonWriter) value(v any) {
	if j.err == nil {
		j.err = j.enc.Encode(v)
	}
}

func (j *jsonWriter) key(k string) {
	b, _ := json.Marshal(k)
	j.raw(string(b) + ":")
}

func (j *jsonWriter) field(k string, v any) {
	j.key(k)
	j.value(v)
	j.raw(",")
}

// WriteJSON writes the whole workspace of ctx to w: one JSON object with the transactions streamed in chunks. Only live rows are
// written. exportedAt is the time the export was taken.
func WriteJSON(ctx context.Context, w io.Writer, fin *finance.Service, exportedAt time.Time) error {
	d, err := load(ctx, fin)
	if err != nil {
		return err
	}

	n, err := fin.CountTransactions(ctx, finance.TxFilter{})
	if err != nil {
		return err
	}

	if n > MaxJSONTransactions {
		return ErrTooManyTransactions
	}

	tagUIDs := map[string]bool{}
	for _, t := range d.tags {
		tagUIDs[t.UID] = true
	}

	j := &jsonWriter{w: bufio.NewWriter(w)}
	j.enc = json.NewEncoder(j.w)
	j.enc.SetEscapeHTML(false)

	j.raw("{")
	j.field("format", JSONFormat)
	j.field("version", JSONVersion)
	j.field("exported_at", exportedAt.UTC().Format(time.RFC3339))
	j.field("workspace", d.workspace)
	j.field("accounts", d.accounts)
	j.field("categories", d.categories)
	j.field("tags", d.tags)

	j.key("transactions")
	j.raw("[")

	first := true

	for t, err := range fin.EachTransaction(ctx, finance.TxFilter{}) {
		if err != nil {
			return err
		}

		if !first {
			j.raw(",")
		}

		first = false

		j.value(d.transaction(t, tagUIDs))
	}

	j.raw("],")
	j.field("budgets", d.budgets)
	j.field("goals", d.goals)
	j.field("recurrences", d.recurrences)
	j.key("cards")
	j.value(d.cards)
	j.raw("}")

	if j.err != nil {
		return j.err
	}

	return j.w.Flush()
}

func (d *jsonData) transaction(t models.Transaction, known map[string]bool) jsonTransaction {
	tags := []string{}

	var uids []string
	if json.Unmarshal([]byte(t.TagUIDs), &uids) == nil {
		for _, u := range uids {
			if known[u] {
				tags = append(tags, u)
			}
		}
	}

	return jsonTransaction{
		UID: t.UID, AccountUID: d.accountUID[t.AccountID], CategoryUID: uidOf(d.categoryUID, t.CategoryID), Kind: t.Kind,
		AmountCents: t.AmountCents, Description: t.Description, OccurredOn: t.OccurredOn,
		TransferUID: str(t.TransferUID, plain), TagUIDs: tags, StatementRef: str(t.StatementRef, plain),
		RecurrenceUID: str(t.RecurrenceUID, plain), OccurrenceOn: str(t.OccurrenceOn, plain),
		InstallmentGroupUID: str(t.InstallmentGroupUID, plain), InstallmentNo: num(t.InstallmentNo),
		InstallmentTotal: num(t.InstallmentTotal),
	}
}
