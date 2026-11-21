# Recurrence and installment rules

This is the language-neutral specification of the rule engine of recurring transactions and of the split of an installment purchase (decisions D-004, D-005, D-008). The Go engine (`app/finance/recurrence.go`, `app/finance/installments.go`) and the later mobile engine must produce the same output for the same input. The contract is `testdata/recurrence-vectors.json`: both engines run every vector.

Dates are calendar dates (ISO `YYYY-MM-DD`), never instants. No time of day and no time zone enter the engine; the workspace time zone only decides which date is "today" before the engine is called.

## The rule

| Field | Meaning |
| --- | --- |
| `frequency` | `daily`, `weekly`, `monthly` or `yearly` |
| `interval` | integer, at least 1: every `interval` days, weeks, months or years |
| `start_date` | the date of the first occurrence (k = 0) |
| `end_date` | optional, inclusive: no occurrence is later than it |
| `max_count` | optional, at least 1: at most this many occurrences, counted from the start |

A rule with an `end_date` before its `start_date` has no occurrence.

## The k-th occurrence

The occurrence number `k` (0, 1, 2, ...) is computed from `start_date` and `k` alone, never from the previous occurrence, so a clamped date never drifts:

- `daily`: `start_date + k * interval` days.
- `weekly`: `start_date + 7 * k * interval` days.
- `monthly`: `AddMonths(start_date, k * interval)`.
- `yearly`: `AddMonths(start_date, 12 * k * interval)`.

`AddMonths(date, n)` moves the date by `n` months and keeps the day of the month, except that the day is clamped to the last day of the target month. A monthly rule starting on January 31 gives January 31, February 28 (29 in a leap year), March 31, April 30, May 31. A yearly rule starting on February 29 gives February 28 in a common year and February 29 in a leap year. Adding days to the first of the month and then overflowing (the behaviour of `time.AddDate`) is wrong: January 31 plus one month must never be a March date.

## Which occurrences are listed

`Occurrences(rule, from, to)` lists, in ascending order, the dates of the occurrences `k` such that:

1. `k < max_count`, when `max_count` is set;
2. the date is not after `end_date`, when `end_date` is set;
3. `from <= date <= to` (both inclusive).

The index `k` always counts from `start_date`, not from `from`: `max_count: 4` with a window that starts after the third occurrence yields at most one date. An empty window (`to < from`) is empty. The engine stops at the year 9999.

## Vector file

`testdata/recurrence-vectors.json` is an array of objects:

```json
{
  "name": "monthly from Jan 31, common February",
  "rule": { "frequency": "monthly", "interval": 1, "start_date": "2027-01-31", "end_date": null, "max_count": null },
  "from": "2027-01-01",
  "to": "2027-06-30",
  "expect": ["2027-01-31", "2027-02-28", "2027-03-31", "2027-04-30", "2027-05-31", "2027-06-30"]
}
```

The expected values were written by an independent reference implementation, not by the Go engine, and the key cases are checked by hand. A vector is changed only after the rule it exercises is agreed again.

## Generation

A stored rule remembers `generated_through`, the date up to which its occurrences were considered. Generation lists the occurrences from the day after `generated_through` (or `start_date` when it is empty) to today, in the time zone of the workspace, and creates a transaction for each, up to 500 per rule per run.

- A generated transaction carries `recurrence_uid` (the `uid` of the rule) and `occurrence_on` (the ISO date of the occurrence). `(workspace, recurrence_uid, occurrence_on)` is unique among all rows of the table, soft-deleted ones included: two runs, or two devices, that generate the same occurrence produce one row, and an occurrence the person deleted is never created again.
- `generated_through` moves forward in the same database transaction as the occurrences it covers, so it also protects a deleted occurrence after its tombstone is purged by compaction. When two copies of a rule meet in a sync, the later watermark wins.
- A failure of one occurrence (an archived account, the monthly quota of the Free plan) is stored in `last_error` of the rule; the watermark stops before that occurrence and the next run retries it. Other rules are not blocked.
- Changing a rule affects the occurrences that are not generated yet. Generated transactions keep their values. Moving `start_date` or the frequency does not rewrite history.

## Upcoming

`Upcoming(days)` merges the transactions already dated in the future (installment parts) with the virtual occurrences of the active rules between tomorrow and today plus `days`: dates the rule would generate but has not yet. A virtual occurrence is computed, never stored, and is dropped when a generated transaction (even a deleted one) holds its `(recurrence_uid, occurrence_on)` or when it is at or before `generated_through`.

## Installments

`parts` is 2 to 60 and the total is at least `parts` cents, so every part is at least one cent.

- Amounts: `base = total / parts` (integer division) and the first `total % parts` parts get one extra cent. 100.00 in 3 parts is 33.34, 33.33, 33.33. The parts always sum to the total.
- Dates: part `i` (0-based) is on `AddMonths(first_date, i)`, so a first date of January 31 gives January 31, February 28, March 31.
- On a credit card account each part gets its statement from the closing day of the card, as a purchase on that date would.
- The description of part `i` (1-based) shows as `Description (i/parts)`. The stored description is the plain one; `installment_no` and `installment_total` carry the numbers.
