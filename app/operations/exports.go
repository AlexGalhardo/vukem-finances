package operations

import (
	"context"
	"io"
	"net/http"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/export"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errDelimiter = errs.New(http.StatusUnprocessableEntity, "The delimiter must be comma or semicolon.")

// Download is a file an operation produces. The operation has refused what it
// must refuse (a result that is too large) before it returns one; Write then
// streams the file, so a surface decides where the bytes go.
type Download struct {
	Filename    string                `json:"filename"     example:"vukem-finances-transactions-2026-11-21.csv"`
	ContentType string                `json:"content_type" example:"text/csv; charset=utf-8"`
	Write       func(io.Writer) error `json:"-"`
}

// ExportTransactionsIn is the query of export.transactions: the filters of
// transactions.list, and the spreadsheet quirks the person can ask for.
type ExportTransactionsIn struct {
	From      string `json:"from"      validate:"max=10" doc:"First date, YYYY-MM-DD" example:"2026-11-01"`
	To        string `json:"to"        validate:"max=10" doc:"Last date, YYYY-MM-DD" example:"2026-11-30"`
	Account   string `json:"account"   validate:"max=26" doc:"Only this account (uid)"`
	Category  string `json:"category"  validate:"max=26" doc:"Only this category and its subcategories (uid)"`
	Tag       string `json:"tag"       validate:"max=26" doc:"Only transactions with this tag (uid)"`
	Kind      string `json:"kind"      validate:"max=20" doc:"income or expense"`
	Q         string `json:"q"         validate:"max=200" doc:"Text of the description, case-insensitive"`
	MinCents  *int64 `json:"min_cents" doc:"Smallest absolute amount, in cents"`
	MaxCents  *int64 `json:"max_cents" doc:"Largest absolute amount, in cents"`
	Delimiter string `json:"delimiter" validate:"max=10" doc:"comma (the default) or semicolon, for locales whose decimal separator is a comma"`
	BOM       bool   `json:"bom"       doc:"Start the file with a byte order mark, for old versions of Excel"`
}

// ExportWorkspaceIn is the input of export.workspace: none.
type ExportWorkspaceIn struct{}

// mcpFile is why a download has no MCP tool: a tool answers with JSON, and a file
// of unbounded size has no place in the context of a model.
var mcpFile = map[Surface]string{SurfaceMCP: "it streams a file, and a tool answers with JSON that fits a model context; the reads (transactions.list, reports.*) cover it"} //nolint:gochecknoglobals // a constant table

type exportOps struct {
	Transactions *Op[ExportTransactionsIn, Download]
	Workspace    *Op[ExportWorkspaceIn, Download]
}

func (c *Catalog) defineExports() {
	c.Exports = exportOps{
		Transactions: Define(c.Registry, Operation[ExportTransactionsIn, Download]{
			Name: "export.transactions", Summary: "Download the transactions that match the filters as a CSV file",
			Scope: ScopeExportRead, Kind: KindRead,
			OptOut: mcpFile,
			Run: func(ctx context.Context, _ Actor, in ExportTransactionsIn) (Download, error) {
				return c.exportTransactions(ctx, in)
			},
		}),
		// The whole workspace holds what a viewer's CSV does not (budgets, goals,
		// rules and member names), so the JSON keeps its owner and admin rule.
		Workspace: Define(c.Registry, Operation[ExportWorkspaceIn, Download]{
			Name: "export.workspace", Summary: "Download the whole workspace as one JSON document (owner and admins)",
			Scope: ScopeExportRead, Ability: workspaces.AbilityManageData, Kind: KindRead,
			OptOut: mcpFile,
			Run: func(ctx context.Context, _ Actor, _ ExportWorkspaceIn) (Download, error) {
				return c.exportWorkspace(ctx)
			},
		}),
	}
}

func (c *Catalog) exportTransactions(ctx context.Context, in ExportTransactionsIn) (Download, error) {
	var opt export.CSVOptions

	switch in.Delimiter {
	case "", "comma":
	case "semicolon":
		opt.Semicolon = true
	default:
		return Download{}, errDelimiter
	}

	opt.BOM = in.BOM

	f := finance.TxFilter{From: in.From, To: in.To, Kind: in.Kind, Q: in.Q, Min: centsFilter(in.MinCents), Max: centsFilter(in.MaxCents)}

	var err error

	if in.Account != "" {
		if f.AccountID, err = c.accountID(ctx, in.Account); err != nil {
			return Download{}, err
		}
	}

	if f.CategoryID, err = c.categoryID(ctx, in.Category); err != nil {
		return Download{}, err
	}

	if in.Tag != "" {
		tag, err := c.tagByUID(ctx, in.Tag)
		if err != nil {
			return Download{}, err
		}

		f.TagIDs = []int64{tag.ID}
	}

	n, err := c.Fin.CountTransactions(ctx, f)
	if err != nil {
		return Download{}, err
	}

	if n > export.MaxCSVRows {
		return Download{}, export.ErrTooManyRows
	}

	rows, err := export.Transactions(ctx, c.Fin, f)
	if err != nil {
		return Download{}, err
	}

	return Download{
		Filename:    "vukem-finances-transactions-" + c.Fin.Today(ctx) + ".csv",
		ContentType: "text/csv; charset=utf-8",
		Write: func(w io.Writer) error {
			_, err := export.WriteCSV(w, rows, opt)

			return err
		},
	}, nil
}

func (c *Catalog) exportWorkspace(ctx context.Context) (Download, error) {
	return Download{
		Filename:    "vukem-finances-export-" + c.Fin.Today(ctx) + ".json",
		ContentType: "application/json; charset=utf-8",
		Write:       func(w io.Writer) error { return export.WriteJSON(ctx, w, c.Fin, c.Fin.Now()) },
	}, nil
}
