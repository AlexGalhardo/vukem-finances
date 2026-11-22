package finance

import (
	"context"
	"database/sql"
	"iter"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// exportChunk is how many rows one read of EachTransaction takes.
const exportChunk = 1000

// Now is the current time of the service clock, in UTC.
func (s *Service) Now() time.Time { return s.now().UTC() }

// CountTransactions counts the transactions a filter matches (the page size
// and the page number of the filter are ignored).
func (s *Service) CountTransactions(ctx context.Context, f TxFilter) (int64, error) {
	q, err := s.filtered(ctx, f)
	if err != nil {
		return 0, err
	}

	return q.Count(ctx)
}

// EachTransaction yields every transaction a filter matches, oldest first
// (date, then id), without loading them all: it reads exportChunk rows at a
// time, each read continuing after the last (date, id) seen. The continuation
// is a row value comparison: written as an OR of two conditions it made SQLite
// walk the index three times slower (QA-503). Nothing holds a
// cursor between the reads, so the one SQLite connection is free while the
// caller writes a slow response. A bad filter is the first and only yield.
func (s *Service) EachTransaction(ctx context.Context, f TxFilter) iter.Seq2[models.Transaction, error] {
	return func(yield func(models.Transaction, error) bool) {
		q, err := s.filtered(ctx, f)
		if err != nil {
			yield(models.Transaction{}, err)

			return
		}

		var (
			date string
			id   int64
		)

		for first := true; ; first = false {
			page := q
			if !first {
				page = page.WhereRaw("(occurred_on, id) > (?, ?)", date, id)
			}

			txs, err := page.OrderBy("occurred_on", orm.Asc).OrderBy("id", orm.Asc).Limit(exportChunk).Get(ctx)
			if err != nil {
				yield(models.Transaction{}, err)

				return
			}

			for _, t := range txs {
				if !yield(t, nil) {
					return
				}
			}

			if len(txs) < exportChunk {
				return
			}

			last := txs[len(txs)-1]
			date, id = last.OccurredOn, last.ID
		}
	}
}

// ExportRow is the part of a transaction the CSV export writes.
type ExportRow struct {
	ID          int64
	UID         string
	OccurredOn  string
	AccountID   int64
	CategoryID  sql.Null[int64]
	Kind        string
	AmountCents int64
	Description string
	TransferUID sql.Null[string]
	TagUIDs     string
}

// EachExportRow is EachTransaction for the CSV: the same rows in the same
// order and chunks, read as plain columns instead of whole models, which is
// most of the cost of a 50000 row download (QA-503).
func (s *Service) EachExportRow(ctx context.Context, f TxFilter) iter.Seq2[ExportRow, error] {
	return func(yield func(ExportRow, error) bool) {
		q, err := s.filtered(ctx, f)
		if err != nil {
			yield(ExportRow{}, err)

			return
		}

		q = q.Select("id", "uid", "occurred_on", "account_id", "category_id", "kind", "amount_cents", "description", "transfer_uid", "tag_uids").
			OrderBy("occurred_on", orm.Asc).OrderBy("id", orm.Asc).Limit(exportChunk)

		var (
			date string
			id   int64
		)

		for first := true; ; first = false {
			page := q
			if !first {
				page = page.WhereRaw("(occurred_on, id) > (?, ?)", date, id)
			}

			query, args, err := page.SQL()
			if err != nil {
				yield(ExportRow{}, err)

				return
			}

			chunk, err := s.readExportChunk(ctx, query, args)
			if err != nil {
				yield(ExportRow{}, err)

				return
			}

			for _, r := range chunk {
				if !yield(r, nil) {
					return
				}
			}

			if len(chunk) < exportChunk {
				return
			}

			last := chunk[len(chunk)-1]
			date, id = last.OccurredOn, last.ID
		}
	}
}

// readExportChunk reads one chunk and closes the cursor before it returns, so
// nothing holds a connection while the caller writes a slow response.
func (s *Service) readExportChunk(ctx context.Context, query string, args []any) ([]ExportRow, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	out := make([]ExportRow, 0, exportChunk)

	for rows.Next() {
		var r ExportRow

		err := rows.Scan(&r.ID, &r.UID, &r.OccurredOn, &r.AccountID, &r.CategoryID, &r.Kind, &r.AmountCents,
			&r.Description, &r.TransferUID, &r.TagUIDs)
		if err != nil {
			return nil, err
		}

		out = append(out, r)
	}

	return out, rows.Err()
}
