package categorize

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// EvalRow is one line of testdata/categorization-eval.jsonl.
type EvalRow struct {
	Description string `json:"description"`
	Kind        string `json:"kind"`
	AmountCents int64  `json:"amount_cents"`
	Expected    string `json:"expected"`
}

// EvalResult is what an evaluation measured.
type EvalResult struct {
	Rows         int
	Answered     int
	Correct      int
	InputTokens  int
	OutputTokens int
}

// Line is the accuracy line the command prints.
func (r EvalResult) Line() string {
	pct := 0.0
	if r.Rows > 0 {
		pct = float64(r.Correct) * 100 / float64(r.Rows)
	}

	return fmt.Sprintf("accuracy: %d/%d (%.1f%%), answered %d, input tokens %d, output tokens %d",
		r.Correct, r.Rows, pct, r.Answered, r.InputTokens, r.OutputTokens)
}

// EvalCategories are the categories of the evaluation: the ones a new
// workspace starts with. The uid is made up from the name.
func EvalCategories() []Category {
	out := []Category{}

	for _, n := range []string{"Housing", "Groceries", "Transport", "Leisure", "Health"} {
		out = append(out, Category{UID: EvalUID(n), Name: n, Kind: models.KindExpense})
	}

	for _, n := range []string{"Salary", "Other income"} {
		out = append(out, Category{UID: EvalUID(n), Name: n, Kind: models.KindIncome})
	}

	return out
}

// EvalUID is the uid of an evaluation category.
func EvalUID(name string) string {
	return "eval-" + strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

// ReadEval reads the rows of the evaluation set.
func ReadEval(r io.Reader) ([]EvalRow, error) {
	var rows []EvalRow

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		var row EvalRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("categorize: evaluation line %d: %w", len(rows)+1, err)
		}

		rows = append(rows, row)
	}

	return rows, sc.Err()
}

// counting adds up the tokens of the calls that pass through it.
type counting struct {
	inner Completer
	in    int
	out   int
}

func (c *counting) Call(ctx context.Context, workspace int64, req ai.Request) (ai.Response, error) {
	resp, err := c.inner.Call(ctx, workspace, req)
	c.in += resp.InputTokens
	c.out += resp.OutputTokens

	return resp, err
}

// Evaluate runs rows through Batch (30 per call, one kind per call) and
// compares the category chosen with the expected one. A failed batch counts as
// unanswered, not as an abort, so one provider hiccup does not hide the rest;
// the first error is returned with the result.
func Evaluate(ctx context.Context, c Completer, workspace int64, rows []EvalRow) (EvalResult, error) {
	cats := EvalCategories()
	counter := &counting{inner: c}
	res := EvalResult{Rows: len(rows)}

	var firstErr error

	for _, kind := range []string{models.KindExpense, models.KindIncome} {
		var ofKind []EvalRow

		for _, r := range rows {
			if r.Kind == kind {
				ofKind = append(ofKind, r)
			}
		}

		for len(ofKind) > 0 {
			n := min(BatchSize, len(ofKind))
			batch := ofKind[:n]
			ofKind = ofKind[n:]

			items := make([]Item, n)
			for i, r := range batch {
				items[i] = Item{ID: itemID(i), Description: r.Description, Kind: r.Kind, AmountCents: r.AmountCents}
			}

			proposals, err := Batch(ctx, counter, workspace, kind, items, cats)
			if err != nil && firstErr == nil {
				firstErr = err
			}

			for _, p := range proposals {
				res.Answered++

				for i := range items {
					if items[i].ID == p.ItemID && EvalUID(batch[i].Expected) == p.CategoryUID {
						res.Correct++
					}
				}
			}
		}
	}

	res.InputTokens, res.OutputTokens = counter.in, counter.out

	return res, firstErr
}
