package ledger

import (
	"context"
	"io"
	"net/http"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func (h *Handlers) importPage(c *web.Ctx, preview *finance.Preview) error {
	accounts, err := h.Fin.Accounts(c.Context())
	if err != nil {
		return err
	}

	imports, err := h.Fin.Imports(c.Context())
	if err != nil {
		return err
	}

	lim, err := h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	return pages.Import.Render(c, pages.ImportProps{
		Accounts: accounts, Imports: imports, Preview: preview,
		DateFormats: finance.DateFormats, Enabled: lim.CSVImport,
	})
}

// Import shows the upload form and the latest imports.
func (h *Handlers) Import(c *web.Ctx) error { return h.importPage(c, nil) }

// UploadImport reads the statement (CSV or OFX) and shows its columns to map,
// or, for an OFX file, the rows as they will be stored.
func (h *Handlers) UploadImport(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	file, header, err := c.FormFile("file")
	if err != nil {
		return errs.New(http.StatusUnprocessableEntity, "Choose a CSV or OFX file.")
	}

	defer file.Close()

	if err := finance.CheckUpload(header.Filename, header.Header.Get("Content-Type")); err != nil {
		return back(c, "/import", err)
	}

	if header.Size > finance.MaxImportBytes {
		return back(c, "/import", finance.ErrImportTooBig)
	}

	content, err := io.ReadAll(io.LimitReader(file, finance.MaxImportBytes+1))
	if err != nil {
		return err
	}

	preview, err := h.Fin.StartImport(c.Context(), id(c.FormValue("accountId")), header.Filename, content)
	if err != nil {
		return back(c, "/import", err)
	}

	return h.importPage(c, &preview)
}

func mappingOf(in requests.ImportMapping) finance.Mapping {
	return finance.Mapping{
		Date: in.Date, Description: in.Description, Amount: in.Amount, Category: in.Category,
		DateFormat: in.DateFormat, Invert: in.Invert == "true", AllowDuplicates: in.AllowDuplicates == "true",
	}
}

// PreviewImport shows what an import would do with a column mapping and the
// duplicate switch, without creating anything.
func (h *Handlers) PreviewImport(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	iid, err := c.ParamInt("import")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.ImportMappingSchema)
	if err != nil {
		return err
	}

	preview, err := h.Fin.PreviewImport(c.Context(), iid, mappingOf(in))
	if err != nil {
		return back(c, "/import", err)
	}

	return h.importPage(c, &preview)
}

// ConfirmImport queues the job that creates the transactions.
func (h *Handlers) ConfirmImport(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	iid, err := c.ParamInt("import")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.ImportMappingSchema)
	if err != nil {
		return err
	}

	err = h.Fin.ConfirmImport(c.Context(), iid, mappingOf(in), func(ctx context.Context, ws, imp int64) error {
		return h.Fin.Queue.Dispatch(ctx, jobs.ImportCSV{WorkspaceID: ws, ImportID: imp})
	})
	if err != nil {
		return back(c, "/import", err)
	}

	return done(c, "/import", "Import queued: it appears below when it finishes.")
}
