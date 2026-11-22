---
title: "Recurrences and installments"
description: "Create rules that record transactions on their own, and split a purchase into installments with exact cents and month-end-safe dates."
---

## Recurring rules

A recurring rule makes a transaction again and again. It has a frequency (daily, weekly, monthly or yearly), an interval, a start date and optionally an end date or a maximum number of occurrences.

The date of each occurrence is computed from the start date and its number, never from the previous occurrence. So a monthly rule that starts on January 31 gives January 31, February 28 (or 29), March 31, April 30 and so on, and never drifts to the 28th for good.

A background job runs every hour and records the occurrences that are due; it also runs when you create or edit a rule. One rule is processed in one database transaction, and a rule that fails records its last error and is retried without blocking the others. A generated transaction is an ordinary one that remembers which rule and date made it, so deleting one occurrence keeps it deleted.

Deleting an account deletes its rules. Deleting a category moves the rules that used it.

The list of upcoming entries merges future-dated transactions with the occurrences the rules will make; those virtual occurrences are never stored.

## Installments

An installment purchase splits one amount into 2 to 60 parts on monthly dates. The split is exact: when the total does not divide evenly, the first parts get one extra cent each until the cents run out, so the parts always add up to the total. Every part is an ordinary transaction. On a credit card each one lands on the statement of its own month, and every part counts toward the monthly transaction quota.

## Everywhere

The same rule engine is implemented in Go on the server and in TypeScript in the web app and the mobile app, and both pass the same test vectors, so a phone and the server agree on every date.

Commands: [recurrences](/docs/integrate/cli/recurrences/). Endpoints: [recurrences](/docs/integrate/rest/recurrences/). The specification is `docs/specs/recurrence.md` in the repository.
