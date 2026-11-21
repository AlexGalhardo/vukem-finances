package billing

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/AlexGalhardo/vukem/contrib/stripe"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Sources of a billing event.
const (
	SourceStripe = "stripe"
	SourcePix    = "pix"
	SourceAdmin  = "admin"
)

// record writes a line of the billing audit trail of workspace id inside ex.
// It carries provider ids only: no card data, no personal data, no provider
// error text.
func record(ctx context.Context, ex database.Executor, id int64, source, kind, ref, summary string) error {
	return orm.Create(workspaces.WithTenant(ctx, id), ex, &models.BillingEvent{
		Source: source, Kind: kind, Ref: ref, Summary: summary,
	})
}

// providerError keeps only what is safe to log or return of a failed call to
// a payment provider: a status and a plain kind. The provider's own message
// can echo request data, so it is dropped.
type providerError struct {
	Provider string
	Status   int
	Kind     string
}

func (e *providerError) Error() string {
	return fmt.Sprintf("%s request failed: status %d (%s)", e.Provider, e.Status, e.Kind)
}

// sanitizeStripe turns an error of the Stripe client into a providerError.
func sanitizeStripe(err error) error {
	if err == nil {
		return nil
	}

	if se, ok := errors.AsType[*stripe.Error](err); ok {
		return &providerError{Provider: "stripe", Status: se.Status, Kind: se.Type}
	}

	return &providerError{Provider: "stripe", Kind: "transport"}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
