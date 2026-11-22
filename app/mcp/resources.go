package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	vmcp "github.com/AlexGalhardo/vukem/mcp"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
)

const (
	uriWorkspace = "finances://workspace"
	uriAccounts  = "finances://accounts"
	uriCats      = "finances://categories"
	uriBudgets   = "finances://budgets/{month}"
	uriReports   = "finances://reports/{kind}/{period}"
)

var (
	monthRe  = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
	customRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})_(\d{4}-\d{2}-\d{2})$`)

	reportKinds   = []string{"cashflow", "categories", "accounts", "tags", "networth", "compare"}
	periodPresets = []string{"this_month", "last_month", "last_3_months", "last_12_months", "this_year"}
)

// readOp runs a read operation as the caller and returns its JSON. Every
// resource and every completion goes through here, so it passes the same hooks
// as a tool call (the rate limit of the grant and the audit line) and has the
// scope, role and isolation of the tool of the same operation. A refusal looks
// like a missing resource, except a rate limit, which says so: it reveals
// nothing about the resource.
func (h *host) readOp(ctx context.Context, op string, input any) ([]byte, error) {
	m, ok := h.cat.Named(op)
	dyn, found := h.cat.Dynamic(op)

	if !ok || !found {
		return nil, vmcp.ErrNotFound
	}

	finish := func(err error) {
		if h.done != nil {
			h.done(ctx, m, err)
		}
	}

	refuse := func(err error) ([]byte, error) {
		finish(err)

		if operations.IsCode(err, operations.CodeRateLimited) {
			return nil, operations.ToolError(err)
		}

		return nil, vmcp.ErrNotFound
	}

	actor, err := h.actor(ctx)
	if err != nil {
		return refuse(err)
	}

	if h.before != nil {
		if err := h.before(ctx, m, actor); err != nil {
			return refuse(err)
		}
	}

	body, err := json.Marshal(input)
	if err != nil {
		finish(err)

		return nil, err
	}

	res, err := dyn.CallJSON(ctx, actor, body)
	finish(err)

	if err != nil {
		if e, ok := errors.AsType[*operations.Error](err); ok && e.Code != operations.CodeInternal {
			return nil, vmcp.ErrNotFound
		}

		return nil, err
	}

	return json.Marshal(res.Value)
}

func (h *host) contents(uri string, body []byte) []vmcp.ResourceContents {
	return []vmcp.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(body)}}
}

func (h *host) addResources(srv *vmcp.Server) {
	fixed := []struct {
		uri, name, desc, op string
		in                  any
	}{
		{uriWorkspace, "workspace", "The workspace overview: currency, this month and the net worth.", "summary.show", operations.MonthIn{}},
		{uriAccounts, "accounts", "The accounts with their balances.", "accounts.list", struct{}{}},
		{uriCats, "categories", "The categories.", "categories.list", struct{}{}},
	}

	for _, f := range fixed {
		if !h.canRead(f.op) {
			continue
		}

		vmcp.AddResource(srv, vmcp.Resource{URI: f.uri, Name: f.name, Description: f.desc, MIMEType: "application/json"},
			func(ctx context.Context) ([]vmcp.ResourceContents, error) {
				body, err := h.readOp(ctx, f.op, f.in)
				if err != nil {
					return nil, err
				}

				return h.contents(f.uri, body), nil
			})
	}

	if h.canRead("budgets.list") {
		vmcp.AddResourceTemplate(srv, vmcp.ResourceTemplate{
			URITemplate: uriBudgets, Name: "budgets", MIMEType: "application/json",
			Description: "The budgets of a month (YYYY-MM) with what the month spent.",
		}, h.readBudgets, h.completeResource)
	}

	vmcp.AddResourceTemplate(srv, vmcp.ResourceTemplate{
		URITemplate: uriReports, Name: "reports", MIMEType: "application/json",
		Description: "A report: kind is " + strings.Join(reportKinds, ", ") +
			"; period is " + strings.Join(periodPresets, ", ") + " or FROM_TO as 2026-01-01_2026-03-31.",
	}, h.readReport, h.completeResource)
}

// canRead reports whether the registry has the read operation on this surface.
func (h *host) canRead(op string) bool {
	m, ok := h.cat.Named(op)

	return ok && m.Kind == operations.KindRead && m.Offered(operations.SurfaceMCP)
}

func (h *host) readBudgets(ctx context.Context, uri string, vars map[string]string) ([]vmcp.ResourceContents, error) {
	if !monthRe.MatchString(vars["month"]) {
		return nil, vmcp.ErrNotFound
	}

	body, err := h.readOp(ctx, "budgets.list", operations.MonthIn{Month: vars["month"]})
	if err != nil {
		return nil, err
	}

	return h.contents(uri, body), nil
}

func (h *host) readReport(ctx context.Context, uri string, vars map[string]string) ([]vmcp.ResourceContents, error) {
	kind, period := vars["kind"], vars["period"]
	if !slices.Contains(reportKinds, kind) {
		return nil, vmcp.ErrNotFound
	}

	in := operations.PeriodIn{Period: period}

	switch m := customRe.FindStringSubmatch(period); {
	case m != nil:
		in = operations.PeriodIn{Period: "custom", From: m[1], To: m[2]}
	case !slices.Contains(periodPresets, period):
		return nil, vmcp.ErrNotFound
	}

	var input any = in
	if kind == "categories" {
		input = operations.CategoryReportIn{Period: in.Period, From: in.From, To: in.To}
	}

	body, err := h.readOp(ctx, "reports."+kind, input)
	if err != nil {
		return nil, err
	}

	return h.contents(uri, body), nil
}

// completeResource suggests the variables of the two templates.
func (h *host) completeResource(_ context.Context, arg, value string, _ map[string]string) (vmcp.Completion, error) {
	var all []string

	switch arg {
	case "month":
		all = recentMonths(timeNow(), 12)
	case "kind":
		all = reportKinds
	case "period":
		all = periodPresets
	}

	return vmcp.Completion{Values: withPrefix(all, value)}, nil
}

func recentMonths(now time.Time, n int) []string {
	out := make([]string, 0, n)

	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		out = append(out, timeutil.AddMonths(first, -i).Format("2006-01"))
	}

	return out
}

func withPrefix(all []string, prefix string) []string {
	prefix = strings.ToLower(prefix)
	out := []string{}

	for _, v := range all {
		if strings.HasPrefix(strings.ToLower(v), prefix) {
			out = append(out, v)
		}
	}

	return out
}
