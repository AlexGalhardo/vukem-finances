---
title: "Budgets and goals"
description: "Set monthly budgets per category or overall, with rollover, and track savings goals with their progress, monthly amount and projected date."
---

## Monthly budgets

A budget is a monthly limit on one expense category. A category budget counts the spending of its subcategories too. A budget with no category is the overall budget, and a workspace has one overall budget at a time.

Spending in a budget is the sum of the expenses of the month in that category. Transfers and deleted entries never count. Amounts are added as exact integers.

**Rollover.** A budget can carry over last month's unspent part. The carried part is never negative and never chains from month to month, and it only applies to a budget that already existed the month before.

In the SaaS edition, budgets and alerts are part of the Pro plan.

## Goals

A savings goal has a target amount, an optional target date and its contributions. Contributions are manual: adding one records progress toward the goal and does not move money between accounts.

From the contributions the app derives the progress, the monthly amount still needed to reach the target by its date, and a projected date at your current pace. You can archive a goal that is done or paused, and delete one.

## Alerts

The app evaluates alerts from your current data, for example a budget close to its limit. An alert is created once per subject and shows in the app for everyone; email goes to owners and admins and never to viewers. The Telegram bot can deliver alerts too, when a person opts in. The write path never sends mail; an hourly job catches up.

## Elsewhere

Commands: [budgets](/docs/integrate/cli/budgets/), [goals](/docs/integrate/cli/goals/) and [summary](/docs/integrate/cli/summary/). Endpoints: [budgets](/docs/integrate/rest/budgets/) and [goals](/docs/integrate/rest/goals/). The Telegram bot shows budget progress with `/budgets`.
