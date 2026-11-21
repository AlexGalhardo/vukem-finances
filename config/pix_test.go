package config

import (
	"strings"
	"testing"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

func billingFrom(env map[string]string) Billing { return LoadBilling(vconfig.MapSource(env)) }

var stripeOnly = map[string]string{
	"STRIPE_KEY": "rk_test_x", "STRIPE_WEBHOOK_SECRET": "whsec_x", "STRIPE_PRICE_PRO": "price_x",
}

func with(extra map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range stripeOnly {
		out[k] = v
	}

	for k, v := range extra {
		out[k] = v
	}

	return out
}

func TestPixIsOptionalAndRequiresItsSettingsOnceOn(t *testing.T) {
	t.Parallel()

	if err := billingFrom(stripeOnly).Validate(); err != nil {
		t.Fatalf("Stripe alone must be valid, got %v", err)
	}

	err := billingFrom(with(map[string]string{"ABACATEPAY_API_KEY": "abc_dev_x"})).Validate()
	if err == nil {
		t.Fatal("PIX on without its secret and price must fail")
	}

	for _, want := range []string{"ABACATEPAY_WEBHOOK_SECRET", "PIX_PRICE_PRO_MONTHLY_CENTS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}

	ok := with(map[string]string{
		"ABACATEPAY_API_KEY": "abc_dev_x", "ABACATEPAY_WEBHOOK_SECRET": "s", "PIX_PRICE_PRO_MONTHLY_CENTS": "1990",
	})
	if err := billingFrom(ok).Validate(); err != nil {
		t.Errorf("a complete PIX setup was refused: %v", err)
	}
}

func TestPixAmountIsServerSideInteger(t *testing.T) {
	t.Parallel()

	p := billingFrom(with(map[string]string{"PIX_PRICE_PRO_MONTHLY_CENTS": "1990"})).Pix

	for months, want := range map[int]int64{1: 1990, 3: 5970, 12: 23880} {
		if got, ok := p.Amount(months); !ok || got != want {
			t.Errorf("Amount(%d) = %d, %v, want %d", months, got, ok, want)
		}
	}

	for _, months := range []int{0, 2, -3, 13} {
		if _, ok := p.Amount(months); ok {
			t.Errorf("Amount(%d) was offered", months)
		}
	}

	custom := billingFrom(with(map[string]string{"PIX_PRICE_PRO_MONTHLY_CENTS": "500", "PIX_MONTHS_OPTIONS": "6, 1,6"})).Pix
	if len(custom.Months) != 2 || custom.Months[0] != 1 || custom.Months[1] != 6 {
		t.Errorf("months = %v, want [1 6]", custom.Months)
	}

	bad := billingFrom(with(map[string]string{
		"ABACATEPAY_API_KEY": "k", "ABACATEPAY_WEBHOOK_SECRET": "s", "PIX_PRICE_PRO_MONTHLY_CENTS": "50",
		"PIX_MONTHS_OPTIONS": "0,x,99",
	}))
	if bad.Validate() == nil {
		t.Error("a price under 100 and bad month options must fail")
	}
}
