package finance

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// statementVectorsPath is the file the mobile app (epic 19) reads, so the two
// implementations of the card statement rules cannot drift.
const statementVectorsPath = "../../testdata/statement-vectors.json"

type closingVector struct {
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	ClosingDay int    `json:"closing_day"`
	Expect     string `json:"expect"`
}

type statementForVector struct {
	Date       string `json:"date"`
	ClosingDay int    `json:"closing_day"`
	Expect     string `json:"expect"`
}

type dueVector struct {
	Closing string `json:"closing"`
	DueDay  int    `json:"due_day"`
	Expect  string `json:"expect"`
}

type statusVector struct {
	Today      string `json:"today"`
	Closing    string `json:"closing"`
	Due        string `json:"due"`
	TotalCents int64  `json:"total_cents"`
	PaidCents  int64  `json:"paid_cents"`
	Expect     string `json:"expect"`
}

type statementVectors struct {
	ClosingDate  []closingVector      `json:"closing_date"`
	StatementFor []statementForVector `json:"statement_for"`
	DueDate      []dueVector          `json:"due_date"`
	Status       []statusVector       `json:"status"`
}

// buildStatementVectors runs the Go functions over fixed grids of inputs that
// cover month ends, leap years, the year turn and the month overflow.
func buildStatementVectors() statementVectors {
	var v statementVectors

	days := []int{1, 10, 15, 28, 29, 30, 31}

	for _, year := range []int{2027, 2028, 2100} {
		for month := 1; month <= 12; month++ {
			for _, d := range days {
				v.ClosingDate = append(v.ClosingDate, closingVector{
					Year: year, Month: month, ClosingDay: d,
					Expect: closingDate(year, time.Month(month), d).Format(isoDate),
				})
			}
		}
	}

	for _, d := range days {
		v.ClosingDate = append(v.ClosingDate, closingVector{
			Year: 2027, Month: 13, ClosingDay: d, Expect: closingDate(2027, time.Month(13), d).Format(isoDate),
		})
	}

	windows := [][2]string{
		{"2026-12-24", "2027-01-06"},
		{"2027-01-26", "2027-03-04"},
		{"2028-02-20", "2028-03-04"},
		{"2027-04-28", "2027-05-02"},
	}

	for _, w := range windows {
		for d := day(w[0]); !d.After(day(w[1])); d = d.AddDate(0, 0, 1) {
			for _, c := range days {
				v.StatementFor = append(v.StatementFor, statementForVector{
					Date: d.Format(isoDate), ClosingDay: c, Expect: statementFor(d, c).Format(isoDate),
				})
			}
		}
	}

	closings := []string{
		"2026-11-10", "2026-12-20", "2026-12-31", "2027-01-10", "2027-01-31", "2027-02-01",
		"2027-02-28", "2027-03-31", "2028-02-29", "2028-12-31",
	}

	for _, c := range closings {
		for _, due := range days {
			v.DueDate = append(v.DueDate, dueVector{Closing: c, DueDay: due, Expect: dueDate(day(c), due).Format(isoDate)})
		}
	}

	amounts := [][2]int64{{100, 0}, {100, 40}, {100, 99}, {100, 100}, {100, 150}, {0, 0}, {-50, 0}, {9007199254740991, 9007199254740990}}

	for _, today := range []string{"2026-11-05", "2026-11-10", "2026-11-11", "2026-11-20", "2026-11-21"} {
		for _, a := range amounts {
			v.Status = append(v.Status, statusVector{
				Today: today, Closing: "2026-11-10", Due: "2026-11-20", TotalCents: a[0], PaidCents: a[1],
				Expect: statementStatus(today, "2026-11-10", "2026-11-20", a[0], a[1]),
			})
		}
	}

	return v
}

// TestStatementVectorsExport writes the vectors of the statement rules to
// testdata/statement-vectors.json, which the mobile tests read. With
// UPDATE_STATEMENT_VECTORS=1 it rewrites the file; without it, the test fails
// when the file is not what the Go functions produce.
func TestStatementVectorsExport(t *testing.T) {
	t.Parallel()

	out, err := json.MarshalIndent(buildStatementVectors(), "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	out = append(out, '\n')

	if os.Getenv("UPDATE_STATEMENT_VECTORS") == "1" {
		if err := os.WriteFile(statementVectorsPath, out, 0o644); err != nil { //nolint:gosec // a test fixture, read by everyone
			t.Fatal(err)
		}

		return
	}

	have, err := os.ReadFile(statementVectorsPath)
	if err != nil {
		t.Fatalf("%v (run UPDATE_STATEMENT_VECTORS=1 go test ./app/finance/ -run TestStatementVectorsExport)", err)
	}

	if !bytes.Equal(bytes.ReplaceAll(have, []byte("\r\n"), []byte("\n")), out) {
		t.Fatal("testdata/statement-vectors.json is stale: run UPDATE_STATEMENT_VECTORS=1 go test ./app/finance/ -run TestStatementVectorsExport")
	}
}
