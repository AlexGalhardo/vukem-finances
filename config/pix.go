package config

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// Limits of the PIX settings. MinPixCents is the smallest charge AbacatePay
// accepts (R$ 1,00); maxPixMonths keeps a mistyped option from selling years.
const (
	MinPixCents  = 100
	maxPixMonths = 36
	// maxPixTotal keeps every total far inside models.MaxCents (2^53 - 1).
	maxPixTotal = 1 << 40
)

// DefaultPixMonths are the months offered when PIX_MONTHS_OPTIONS is unset.
var DefaultPixMonths = []int{1, 3, 12}

// Pix configures the one-off PIX payment of the Pro plan (AbacatePay). PIX is
// optional: it is on when ABACATEPAY_API_KEY is set.
type Pix struct {
	// Key is the AbacatePay API key (ABACATEPAY_API_KEY) with the permissions
	// TRANSPARENT:CREATE and TRANSPARENT:READ.
	Key vconfig.Secret
	// WebhookSecret is the shared secret of the webhook (ABACATEPAY_WEBHOOK_SECRET).
	WebhookSecret vconfig.Secret
	// MonthlyCents is the price of one month in BRL cents
	// (PIX_PRICE_PRO_MONTHLY_CENTS): the amount of a charge is this times the
	// months, never a number from a request.
	MonthlyCents int64
	// Months are the periods offered (PIX_MONTHS_OPTIONS, default 1,3,12).
	Months []int

	err error
}

func loadPix(get func(string) string) Pix {
	p := Pix{
		Key: vconfig.NewSecret(get("ABACATEPAY_API_KEY")), WebhookSecret: vconfig.NewSecret(get("ABACATEPAY_WEBHOOK_SECRET")),
	}

	if v := get("PIX_PRICE_PRO_MONTHLY_CENTS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			p.err = errors.Join(p.err, errors.New("PIX_PRICE_PRO_MONTHLY_CENTS must be a whole number of cents"))
		}

		p.MonthlyCents = n
	}

	raw := get("PIX_MONTHS_OPTIONS")
	if raw == "" {
		p.Months = slices.Clone(DefaultPixMonths)
		return p
	}

	for s := range strings.SplitSeq(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 1 || n > maxPixMonths {
			p.err = errors.Join(p.err, errors.New("PIX_MONTHS_OPTIONS: each option must be a whole number from 1 to 36"))
			continue
		}

		if !slices.Contains(p.Months, n) {
			p.Months = append(p.Months, n)
		}
	}

	slices.Sort(p.Months)

	return p
}

// Enabled reports whether PIX is on.
func (p Pix) Enabled() bool { return !p.Key.IsZero() }

// validate reports what is wrong; it is called only when PIX is on.
func (p Pix) validate() error {
	var errs []error

	if p.err != nil {
		errs = append(errs, p.err)
	}

	if p.WebhookSecret.IsZero() {
		errs = append(errs, errors.New("ABACATEPAY_WEBHOOK_SECRET is not set"))
	}

	if p.MonthlyCents < MinPixCents {
		errs = append(errs, errors.New("PIX_PRICE_PRO_MONTHLY_CENTS must be at least 100"))
	}

	if len(p.Months) == 0 {
		errs = append(errs, errors.New("PIX_MONTHS_OPTIONS offers no month"))
	}

	return errors.Join(errs...)
}

// Amount is the price in cents of months, from the server's price. ok is false
// for a number of months that is not on offer.
func (p Pix) Amount(months int) (cents int64, ok bool) {
	if !slices.Contains(p.Months, months) || p.MonthlyCents < MinPixCents {
		return 0, false
	}

	total := p.MonthlyCents * int64(months)
	if p.MonthlyCents > maxPixTotal/maxPixMonths {
		return 0, false
	}

	return total, true
}
