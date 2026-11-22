package categorize_test

import (
	"os"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/ai/fake"
)

func TestEvaluateWithTheFakeProvider(t *testing.T) {
	t.Parallel()

	f, err := os.Open("../../../testdata/categorization-eval.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rows, err := categorize.ReadEval(f)
	if err != nil || len(rows) != 100 {
		t.Fatalf("rows = %d, %v", len(rows), err)
	}

	res, err := categorize.Evaluate(t.Context(), fake.DefaultClassifier(), 0, rows)
	if err != nil {
		t.Fatal(err)
	}

	if res.Rows != 100 || res.Correct < 90 || res.InputTokens == 0 {
		t.Errorf("result = %+v", res)
	}

	if !strings.HasPrefix(res.Line(), "accuracy: ") {
		t.Errorf("line = %q", res.Line())
	}
}

func TestReadEvalRefusesBadLines(t *testing.T) {
	t.Parallel()

	if _, err := categorize.ReadEval(strings.NewReader("{not json}\n")); err == nil {
		t.Error("a bad line must be an error")
	}
}
