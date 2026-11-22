package sync

import (
	"context"
	"maps"
	"slices"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// checkTransfers looks at the whole batch before anything is written: the two
// legs of a transfer (two transactions that share transfer_uid) are applied
// together or rejected together. It returns the mutations that must be
// rejected, by index.
//
// A group of legs is complete when, at the end of the batch, exactly two live
// legs hold its transfer_uid (the ones already stored plus the ones the batch
// creates, edits or restores), they are one income and one expense of the same
// amount and date on two accounts, and neither has a category. A restore counts
// the stored deleted legs too, because restoring one leg brings the other back.
// Deleting one leg deletes both (the finance service does it), so a group with
// a delete in the batch is not checked.
func (s *Service) checkTransfers(ctx context.Context, tx *database.Tx, ws int64, muts []MutationIn) (map[int]*mutationError, error) {
	broken := map[int]*mutationError{}

	ent, ok := changes.Lookup("transaction")
	if !ok {
		return broken, nil
	}

	var uids []string

	for _, m := range muts {
		if m.Entity == "transaction" && changes.ValidUID(m.UID, s.now()) == nil {
			uids = append(uids, m.UID)
		}
	}

	stored, err := rowsByUID(ctx, tx, ent, ws, uids)
	if err != nil {
		return nil, err
	}

	type member struct {
		idx int
		mut MutationIn
	}

	groups := map[string][]member{}
	affected := map[string]bool{}
	deleted := map[string]bool{}
	restores := map[string]bool{}
	left := map[string][]int{}

	for i, m := range muts {
		if m.Entity != "transaction" || changes.ValidUID(m.UID, s.now()) != nil {
			continue
		}

		row, exists := stored[m.UID]

		var was string
		if exists {
			was, _ = row.Fields["transfer_uid"].(string)
		}

		switch m.Op {
		case "delete":
			if was != "" {
				deleted[was] = true
			}
		case "upsert", "restore":
			now := was
			if v, set := m.Fields["transfer_uid"]; set {
				now, _ = v.(string)
			}

			if now != "" {
				groups[now] = append(groups[now], member{i, m})
				affected[now] = true

				if m.Op == "restore" {
					restores[now] = true
				}
			}

			if was != "" && was != now {
				affected[was] = true
				left[was] = append(left[was], i)
			}
		}
	}

	for _, t := range slices.Sorted(maps.Keys(affected)) {
		// A leg that leaves in the same batch is still checked: otherwise it
		// would survive alone while the delete of its partner finds no pair.
		if deleted[t] && len(left[t]) == 0 {
			continue
		}

		legs, err := selectRows(ctx, tx, ent, ws, "t.transfer_uid = ?", []any{t}, "t.id", 0)
		if err != nil {
			return nil, err
		}

		// The legs after the batch, by uid, with the values the batch gives them.
		final := map[string]map[string]any{}

		for _, l := range legs {
			if l.Deleted && !restores[t] {
				continue
			}

			final[l.UID] = maps.Clone(l.Fields)
		}

		for _, g := range groups[t] {
			base, ok := final[g.mut.UID]
			if !ok {
				base = map[string]any{}
				if r, exists := stored[g.mut.UID]; exists {
					base = maps.Clone(r.Fields)
				}
			}

			maps.Copy(base, g.mut.Fields)
			final[g.mut.UID] = base
		}

		for _, i := range left[t] {
			delete(final, muts[i].UID)
		}

		var problem *mutationError

		switch {
		case len(final) != 2:
			problem = reject(CodeTransferBroken, "transfer_uid", "A transfer needs exactly two legs: both are sent in the same push or are already stored.")
		case !consistent(slices.Collect(maps.Values(final))):
			problem = reject(CodeTransferBad, "transfer_uid", "The legs of a transfer are one expense and one income of the same amount and date on two accounts, with no category.")
		}

		if problem == nil {
			continue
		}

		for _, g := range groups[t] {
			broken[g.idx] = problem
		}

		for _, i := range left[t] {
			broken[i] = problem
		}
	}

	return broken, nil
}

// consistent checks the two legs of a transfer against each other; a value a
// leg does not have yet is left to the field validation.
func consistent(legs []map[string]any) bool {
	a, b := legs[0], legs[1]

	if a["category_uid"] != nil || b["category_uid"] != nil {
		return false
	}

	if ka, kb := a["kind"], b["kind"]; ka != nil && kb != nil && (ka == kb || (ka != models.KindIncome && ka != models.KindExpense)) {
		return false
	}

	if aa, ab := a["amount_cents"], b["amount_cents"]; aa != nil && ab != nil {
		na, oka := number(aa)
		nb, okb := number(ab)

		if !oka || !okb || na != nb {
			return false
		}
	}

	if da, db := a["occurred_on"], b["occurred_on"]; da != nil && db != nil && da != db {
		return false
	}

	if xa, xb := a["account_uid"], b["account_uid"]; xa != nil && xb != nil && xa == xb {
		return false
	}

	return true
}
