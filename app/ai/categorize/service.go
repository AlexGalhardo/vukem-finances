package categorize

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	// RunCap is the most transactions one run looks at: it bounds the cost.
	RunCap = 300
	// AutoApplyMin is the confidence, in percent, from which the opt-in applies a suggestion.
	AutoApplyMin = 90
	// UndoWindow is how long an auto-applied suggestion can be reversed.
	UndoWindow = 7 * 24 * time.Hour
	// listCap bounds the suggestions one page shows.
	listCap = 500
)

var (
	errNotPending = errs.New(http.StatusConflict, "That suggestion was already handled.")
	errNoUndo     = errs.New(http.StatusConflict, "That suggestion can no longer be undone.")
)

// Service makes and handles the suggestions of the workspace in the context.
type Service struct {
	DB  *database.DB
	AI  *ai.Client
	Fin *finance.Service
	Log *slog.Logger

	now func() time.Time
}

// New builds the service; now defaults to the wall clock.
func New(db *database.DB, client *ai.Client, fin *finance.Service, log *slog.Logger, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	if log == nil {
		log = slog.Default()
	}

	return &Service{DB: db, AI: client, Fin: fin, Log: log, now: now}
}

// Options narrows a run: an import, one transaction, or (zero) every
// uncategorized transaction without a suggestion. Cap defaults to RunCap and
// never exceeds it.
type Options struct {
	Cap            int
	ImportID       int64
	TransactionUID string
}

// Result is what a run did.
type Result struct {
	Considered int  `json:"considered"`
	FromRules  int  `json:"fromRules"`
	FromAI     int  `json:"fromAi"`
	Applied    int  `json:"applied"`
	AIFailed   bool `json:"aiFailed"`
}

// Run proposes categories for the uncategorized transactions of the workspace
// of ctx: the memory of descriptions first (always, no AI), then the model for
// the rest when the feature is on. It records suggestions and, only when the
// owner opted in, applies those of at least AutoApplyMin percent.
func (s *Service) Run(ctx context.Context, opts Options) (Result, error) {
	var res Result

	workspace, ok := workspaces.TenantID(ctx)
	if !ok {
		return res, tenancy.ErrNoTenant
	}

	limit := opts.Cap
	if limit <= 0 || limit > RunCap {
		limit = RunCap
	}

	settings, err := s.AI.Settings.Get(ctx)
	if err != nil {
		return res, err
	}

	rows, err := s.candidates(ctx, opts, limit)
	if err != nil {
		return res, err
	}

	res.Considered = len(rows)
	autoApply := settings.FeatureOn(ai.FeatureCategorize) && settings.AutoApplySuggestions

	var rest []models.Transaction

	for _, t := range rows {
		cat, hit, err := s.Fin.RuleCategory(ctx, s.DB, t.Description, t.Kind)
		if err != nil {
			return res, err
		}

		if !hit {
			rest = append(rest, t)

			continue
		}

		if made, applied := s.record(ctx, t, cat.UID, 100, models.SuggestionRule, autoApply); made {
			res.FromRules++

			if applied {
				res.Applied++
			}
		}
	}

	if !settings.FeatureOn(ai.FeatureCategorize) || len(rest) == 0 {
		return res, nil
	}

	cats, err := s.allowed(ctx)
	if err != nil {
		return res, err
	}

	for _, kind := range []string{models.KindExpense, models.KindIncome} {
		var ofKind []models.Transaction

		for _, t := range rest {
			if t.Kind == kind {
				ofKind = append(ofKind, t)
			}
		}

		for len(ofKind) > 0 {
			n := min(BatchSize, len(ofKind))
			batch := ofKind[:n]
			ofKind = ofKind[n:]

			stop, made, applied := s.aiBatch(ctx, workspace, kind, batch, cats, autoApply)
			res.FromAI += made
			res.Applied += applied

			if stop {
				res.AIFailed = true

				return res, nil
			}
		}
	}

	return res, nil
}

// aiBatch runs one batch. stop is true when the run should end: a refusal of
// the client (switch, key, rate, monthly limit) or a provider failure would
// only repeat. A model that answers with unusable text costs the batch, not the run.
func (s *Service) aiBatch(
	ctx context.Context, workspace int64, kind string, batch []models.Transaction, cats []Category, autoApply bool,
) (stop bool, made, applied int) {
	items := make([]Item, len(batch))
	byID := make(map[string]models.Transaction, len(batch))

	for i, t := range batch {
		id := itemID(i)
		items[i] = Item{ID: id, Description: t.Description, Kind: t.Kind, AmountCents: t.AmountCents}
		byID[id] = t
	}

	proposals, err := Batch(ctx, s.AI, workspace, kind, items, cats)
	if errors.Is(err, ErrBadOutput) {
		s.Log.WarnContext(ctx, "ai categorize: unusable output", "workspace", workspace)

		return false, 0, 0
	}

	if err != nil {
		s.Log.WarnContext(ctx, "ai categorize: call failed", "workspace", workspace, "class", err.Error())

		return true, 0, 0
	}

	for _, p := range proposals {
		if ok, did := s.record(ctx, byID[p.ItemID], p.CategoryUID, p.Confidence, models.SuggestionAI, autoApply); ok {
			made++

			if did {
				applied++
			}
		}
	}

	return false, made, applied
}

// candidates are the live, uncategorized, non-transfer transactions that have
// no suggestion yet (a rejected one is never proposed again), newest first.
func (s *Service) candidates(ctx context.Context, opts Options, limit int) ([]models.Transaction, error) {
	ws, _ := workspaces.TenantID(ctx)

	q := workspaces.Scoped[models.Transaction](ctx, s.DB).WhereNull("category_id").WhereNull("transfer_uid").
		WhereRaw("uid NOT IN (SELECT transaction_uid FROM suggestions WHERE workspace_id = ?)", ws)

	if opts.ImportID > 0 {
		q = q.Where("import_id", opts.ImportID)
	}

	if opts.TransactionUID != "" {
		q = q.Where("uid", opts.TransactionUID)
	}

	return q.OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).Limit(limit).Get(ctx)
}

// allowed lists the categories the model may choose: live and not archived.
func (s *Service) allowed(ctx context.Context) ([]Category, error) {
	rows, err := s.Fin.ActiveCategories(ctx)
	if err != nil {
		return nil, err
	}

	names := make(map[int64]string, len(rows))
	for _, c := range rows {
		names[c.ID] = c.Name
	}

	out := make([]Category, 0, len(rows))
	for _, c := range rows {
		cat := Category{UID: c.UID, Name: c.Name, Kind: c.Kind}
		if c.ParentID.Valid {
			cat.Parent = names[c.ParentID.V]
		}

		out = append(out, cat)
	}

	return out, nil
}

// record stores a suggestion and, when allowed, applies it. It reports whether
// a suggestion row was made. A suggestion that cannot be stored (the row
// exists already, or the transaction changed meanwhile) is skipped.
func (s *Service) record(ctx context.Context, t models.Transaction, categoryUID string, confidence int, source string, autoApply bool) (made, applied bool) {
	row := models.Suggestion{
		TransactionUID: t.UID, CategoryUID: categoryUID, Confidence: confidence, Source: source,
		Status: models.SuggestionPending,
	}

	if err := orm.Create(ctx, s.DB, &row); err != nil {
		if !s.DB.UniqueViolation(err) {
			s.Log.WarnContext(ctx, "categorize: could not store a suggestion", "error", err.Error())
		}

		return false, false
	}

	if autoApply && confidence >= AutoApplyMin {
		return true, s.apply(ctx, row)
	}

	return true, false
}

// apply is the opt-in path: it gives the category and marks the suggestion
// accepted with the time, which opens the undo window. It does not teach the
// memory: only a person's choice does.
func (s *Service) apply(ctx context.Context, row models.Suggestion) bool {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := s.Fin.AssignCategory(ctx, tx, row.TransactionUID, row.CategoryUID); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.Suggestion](ctx, tx).Where("id", row.ID).
			Where("status", models.SuggestionPending).
			Update(ctx, orm.Set{"status": models.SuggestionAccepted, "applied_at": s.now().UTC()})

		return err
	})
	if err != nil {
		s.Log.InfoContext(ctx, "categorize: a suggestion was not applied", "suggestion", row.ID)
	}

	return err == nil
}

// Pending is one row of the review list.
type Pending struct {
	ID             int64  `json:"id"`
	TransactionUID string `json:"transactionUid"`
	Description    string `json:"description"`
	Kind           string `json:"kind"`
	AmountCents    int64  `json:"amountCents"`
	Date           string `json:"date"`
	CategoryUID    string `json:"categoryUid"`
	CategoryName   string `json:"categoryName"`
	Confidence     int    `json:"confidence"`
	Source         string `json:"source"`
	// AppliedAt is set on an auto-applied suggestion still inside its undo window.
	AppliedAt string `json:"appliedAt"`
}

// List returns the pending suggestions (best first) and the auto-applied ones
// that can still be undone. Suggestions whose transaction or category is gone,
// or that stopped being pending-worthy, are left out.
func (s *Service) List(ctx context.Context) (pending, applied []Pending, err error) {
	pending, applied = []Pending{}, []Pending{}

	rows, err := workspaces.Scoped[models.Suggestion](ctx, s.DB).Where("status", models.SuggestionPending).
		OrderBy("confidence", orm.Desc).OrderBy("id", orm.Asc).Limit(listCap).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	recent, err := workspaces.Scoped[models.Suggestion](ctx, s.DB).Where("status", models.SuggestionAccepted).
		WhereNotNull("applied_at").WhereNull("undone_at").
		Where("applied_at", ">=", s.now().UTC().Add(-UndoWindow)).OrderBy("id", orm.Desc).Limit(listCap).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	rows = append(rows, recent...)

	txUIDs := make([]any, 0, len(rows))
	catUIDs := make([]any, 0, len(rows))

	for _, r := range rows {
		txUIDs = append(txUIDs, r.TransactionUID)
		catUIDs = append(catUIDs, r.CategoryUID)
	}

	if len(rows) == 0 {
		return pending, applied, nil
	}

	txs, err := workspaces.Scoped[models.Transaction](ctx, s.DB).WhereIn("uid", txUIDs...).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	cats, err := workspaces.Scoped[models.Category](ctx, s.DB).WhereIn("uid", catUIDs...).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	txBy := make(map[string]models.Transaction, len(txs))
	for _, t := range txs {
		txBy[t.UID] = t
	}

	catBy := make(map[string]models.Category, len(cats))
	for _, c := range cats {
		catBy[c.UID] = c
	}

	for _, r := range rows {
		t, tok := txBy[r.TransactionUID]
		c, cok := catBy[r.CategoryUID]

		if !tok || !cok {
			continue
		}

		item := Pending{
			ID: r.ID, TransactionUID: t.UID, Description: t.Description, Kind: t.Kind, AmountCents: t.AmountCents,
			Date: t.OccurredOn, CategoryUID: c.UID, CategoryName: c.Name, Confidence: r.Confidence, Source: r.Source,
		}

		if r.Status == models.SuggestionPending {
			if t.CategoryID.Valid || c.ArchivedAt.Valid {
				continue
			}

			pending = append(pending, item)

			continue
		}

		item.AppliedAt = r.AppliedAt.V.UTC().Format("2006-01-02 15:04")
		applied = append(applied, item)
	}

	return pending, applied, nil
}

// Accept applies the suggestion the person confirmed and teaches the memory.
// A suggestion whose transaction or category changed since is closed as
// rejected and the caller gets the reason.
func (s *Service) Accept(ctx context.Context, id int64) error {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		ws, _ := workspaces.TenantID(ctx)
		if _, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, ws); err != nil {
			return err
		}

		row, err := workspaces.Scoped[models.Suggestion](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		if row.Status != models.SuggestionPending {
			return errNotPending
		}

		t, err := s.Fin.AssignCategory(ctx, tx, row.TransactionUID, row.CategoryUID)
		if err != nil {
			return err
		}

		if err := s.Fin.LearnRule(ctx, tx, t.Description, t.CategoryID.V, models.RuleSourceAcceptedSuggestion); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Suggestion](ctx, tx).Where("id", id).Update(ctx, orm.Set{"status": models.SuggestionAccepted})

		return err
	})

	if _, stale := errors.AsType[*errs.HTTPError](err); stale && !errors.Is(err, errNotPending) {
		_, _ = workspaces.Scoped[models.Suggestion](ctx, s.DB).Where("id", id).Where("status", models.SuggestionPending).
			Update(ctx, orm.Set{"status": models.SuggestionRejected})
	}

	return err
}

// Reject closes a pending suggestion without changing the transaction.
func (s *Service) Reject(ctx context.Context, id int64) error {
	n, err := workspaces.Scoped[models.Suggestion](ctx, s.DB).Where("id", id).Where("status", models.SuggestionPending).
		Update(ctx, orm.Set{"status": models.SuggestionRejected})
	if err != nil {
		return err
	}

	if n == 0 {
		if _, err := workspaces.Scoped[models.Suggestion](ctx, s.DB).Find(ctx, id); err != nil {
			return err
		}

		return errNotPending
	}

	return nil
}

// AcceptAll accepts the pending suggestions of at least min percent and returns
// how many were applied; one that went stale is skipped.
func (s *Service) AcceptAll(ctx context.Context, minConfidence int) (int, error) {
	rows, err := workspaces.Scoped[models.Suggestion](ctx, s.DB).Where("status", models.SuggestionPending).
		Where("confidence", ">=", max(0, min(100, minConfidence))).OrderBy("id", orm.Asc).Limit(listCap).Get(ctx)
	if err != nil {
		return 0, err
	}

	done := 0

	for _, r := range rows {
		switch err := s.Accept(ctx, r.ID); {
		case err == nil:
			done++
		case isHTTP(err):
		default:
			return done, err
		}
	}

	return done, nil
}

func isHTTP(err error) bool {
	_, ok := errors.AsType[*errs.HTTPError](err)

	return ok
}

// Undo reverses an auto-applied suggestion within UndoWindow: the category is
// taken back only if it is still the one that was applied, and the suggestion
// is closed as rejected so it is not proposed again.
func (s *Service) Undo(ctx context.Context, id int64) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		ws, _ := workspaces.TenantID(ctx)
		if _, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, ws); err != nil {
			return err
		}

		row, err := workspaces.Scoped[models.Suggestion](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		if row.Status != models.SuggestionAccepted || !row.AppliedAt.Valid || row.UndoneAt.Valid ||
			s.now().UTC().Sub(row.AppliedAt.V) > UndoWindow {
			return errNoUndo
		}

		if _, err := s.Fin.ClearCategory(ctx, tx, row.TransactionUID, row.CategoryUID); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Suggestion](ctx, tx).Where("id", id).
			Update(ctx, orm.Set{"status": models.SuggestionRejected, "undone_at": s.now().UTC()})

		return err
	})
}
