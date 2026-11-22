package telegram

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
)

// TestUserTextIsEscaped checks that every message that carries text from a
// person or the database shows <b> and & escaped, in every language, so Telegram
// never reads it as markup.
func TestUserTextIsEscaped(t *testing.T) {
	t.Parallel()

	const evil = `<b>bold</b> & <a href="x">y</a>`

	const want = `&lt;b&gt;bold&lt;/b&gt; &amp; &lt;a href="x"&gt;y&lt;/a&gt;`

	for _, lg := range lang.Codes() {
		texts := map[string]string{
			"confirm": confirmText(lg, "expense", 1250, "USD", evil, "2026-11-21", evil),
			"saved":   savedText(lg, "income", 1250, "USD", evil, "2026-11-21", evil),
			"last":    lastLine(lg, finance.TxRow{AccountName: evil}, "USD"),
			"balance": balanceLine(lg, evil, 100, "USD"),
			"budget":  budgetLine(lg, finance.BudgetStatus{Category: evil}, "USD"),
		}

		for name, text := range texts {
			if !strings.Contains(text, want) {
				t.Errorf("%s/%s: the text is not escaped: %q", lg, name, text)
			}

			if strings.Contains(text, evil) || strings.Contains(text, "<a ") {
				t.Errorf("%s/%s: the raw markup reached the message: %q", lg, name, text)
			}
		}
	}

	if got := EscapeHTML("a & b <c>"); got != "a &amp; b &lt;c&gt;" {
		t.Errorf("EscapeHTML = %q", got)
	}
}
