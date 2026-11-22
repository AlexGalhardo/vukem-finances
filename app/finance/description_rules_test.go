package finance

import "testing"

func TestNormalizeDescription(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"STARBUCKS #1234 SP", "starbucks sp"},
		{"Starbucks 99", "starbucks"},
		{"starbucks", "starbucks"},
		{"  Starbucks   Coffee  ", "starbucks coffee"},
		{"Padaria São João 12/11", "padaria sao joao"},
		{"PADARIA SAO JOAO 2026-11-12", "padaria sao joao"},
		{"Café Pérgola", "cafe pergola"},
		{"Açaí do Zé", "acai ze"},
		{"MERCADO LIVRE*2X9KD8Q1", "mercado livre"},
		{"UBER *TRIP 8XK2L9ZP", "uber trip"},
		{"7-Eleven 4411", "eleven"},
		{"Compra no cartão ZARA 12.11", "zara"},
		{"PIX ENVIADO Maria Silva", "pix enviado maria silva"},
		{"PIX ENVIADO Maria Silva Santos Lima", "pix enviado maria silva"},
		{"Pagamento de boleto 123456789012", "boleto"},
		{"Payment to Netflix.com", "netflix com"},
		{"AMAZON MKTPLACE PMTS AMZN.COM/BILL", "amazon mktplace pmts amzn"},
		{"Supermercado Pão de Açúcar 0345", "supermercado pao acucar"},
		{"El Corte Inglés", "corte ingles"},
		{"Panadería La Esquina 5/11", "panaderia esquina"},
		{"Niño's Café", "nino cafe"},
		{"Straße Cafe", "strasse cafe"},
		{"Rent - November 2026", "rent november"},
		{"Salary ACME Corp 2026-11-01", "salary acme corp"},
		{"Salário ACME 01/11/2026", "salario acme"},
		{"IFOOD *RESTAURANTE 3321", "ifood restaurante"},
		{"1234567890", ""},
		{"#### ---", ""},
		{"", ""},
		{"a b c", ""},
		{"Café", "cafe"},
		{"CAFÉ", "cafe"},
		{"Mixed idioma Spotify Premium Plan 12", "mixed idioma spotify premium"},
		{"Drogaria Pacheco Filial 12", "drogaria pacheco filial"},
		{"Farmácia 1.234,56", "farmacia"},
		{"Water bill 15-11", "water bill"},
	}

	for _, c := range cases {
		if got := NormalizeDescription(c.in); got != c.want {
			t.Errorf("NormalizeDescription(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if len(cases) < 30 {
		t.Fatalf("only %d fixtures", len(cases))
	}
}

func TestNormalizeDescriptionIsStable(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"STARBUCKS #1234", "Café 12/11 Pérgola", "x"} {
		once := NormalizeDescription(in)
		if twice := NormalizeDescription(once); twice != once {
			t.Errorf("not idempotent: %q -> %q -> %q", in, once, twice)
		}
	}
}
