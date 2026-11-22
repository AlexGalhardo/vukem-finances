package finance

import (
	"context"
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
// time, each read continuing after the last (date, id) seen. Nothing holds a
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
				page = page.WhereRaw("(occurred_on > ? OR (occurred_on = ? AND id > ?))", date, date, id)
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
