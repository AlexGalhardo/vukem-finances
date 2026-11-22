package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

// Classifier is a deterministic stand-in for a model in the evaluation command:
// it answers the categorization schema from keywords in the description. It
// needs no key and opens no socket. It is not a measure of any real model.
type Classifier struct {
	// Keywords maps a lower-case word to the name of a category.
	Keywords map[string]string
}

// DefaultClassifier knows a few words of English and Portuguese per category.
func DefaultClassifier() *Classifier {
	k := map[string]string{}
	add := func(category string, words ...string) {
		for _, w := range words {
			k[w] = category
		}
	}

	add("Housing", "rent", "aluguel", "condomínio", "electric", "luz", "water", "internet", "insurance", "seguro", "iptu", "plumber", "gas", "mortgage", "encanamento", "property")
	add("Groceries", "supermercado", "grocery", "padaria", "mercado", "carrefour", "whole", "hortifruti", "açougue", "trader", "atacadão", "costco", "fruit", "supermarket", "feira", "sam's")
	add("Transport", "uber", "pop", "shell", "posto", "metro", "bilhete", "parking", "estacionamento", "lyft", "car", "pedágio", "bus", "bike", "combustível", "train")
	add("Leisure", "netflix", "spotify", "cinemark", "steam", "restaurante", "bar", "concert", "livraria", "disney+", "pizza", "ifood", "bowling", "prime", "teatro", "hotel", "airbnb")
	add("Health", "farmácia", "cvs", "dentista", "unimed", "gym", "laboratório", "dr.", "eye", "óptica", "physiotherapy", "hospital", "vitamin", "psicóloga", "amil", "pharmacy", "médica")
	add("Salary", "salary", "salário", "payroll", "pagamento", "paycheck", "folha", "deposit", "13o", "holerite")
	add("Other income", "freelance", "venda", "refund", "restituição", "dividend", "cashback", "gift", "reembolso", "interest", "rendimento")

	return &Classifier{Keywords: k}
}

// Call implements the Completer of the categorizer: it reads the user message
// of the request and answers every transaction.
func (c *Classifier) Call(_ context.Context, _ int64, req ai.Request) (ai.Response, error) {
	var doc struct {
		Categories []struct{ UID, Name string } `json:"categories"`
		Items      []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"transactions"`
	}

	if len(req.Messages) == 0 || json.Unmarshal([]byte(req.Messages[0].Content), &doc) != nil {
		return ai.Response{Text: "not json"}, nil
	}

	uids := map[string]string{}
	for _, cat := range doc.Categories {
		uids[cat.Name] = cat.UID
	}

	items := make([]string, 0, len(doc.Items))

	for _, it := range doc.Items {
		uid, conf := "null", 0.0

		for _, word := range strings.Fields(strings.ToLower(it.Description)) {
			if name, ok := c.Keywords[word]; ok && uids[name] != "" {
				uid, conf = fmt.Sprintf("%q", uids[name]), 0.9

				break
			}
		}

		items = append(items, fmt.Sprintf(`{"id":%q,"category_uid":%s,"confidence":%v}`, it.ID, uid, conf))
	}

	text := `{"items":[` + strings.Join(items, ",") + `]}`

	return ai.Response{Text: text, InputTokens: len(req.Messages[0].Content) / 4, OutputTokens: len(text) / 4, Model: "fake-keywords", Provider: "fake"}, nil
}
