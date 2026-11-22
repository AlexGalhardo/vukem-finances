---
title: "MCP: resources and prompts"
description: "The read-only resources, resource templates and prompts of the vukem-finances MCP server."
order: 1
---

> Generated from `docs/reference/mcp.md`. Do not edit this page: change the source and run `bun run reference` in `site/`.

## Resources

| Name | Description |
| --- | --- |
| `finances://accounts` | The accounts with their balances. |
| `finances://categories` | The categories. |
| `finances://workspace` | The workspace overview: currency, this month and the net worth. |

## Resource templates

| Name | Description |
| --- | --- |
| `finances://budgets/{month}` | The budgets of a month (YYYY-MM) with what the month spent. |
| `finances://reports/{kind}/{period}` | A report: kind is cashflow, categories, accounts, tags, networth, compare; period is this_month, last_month, last_3_months, last_12_months, this_year or FROM_TO as 2026-01-01_2026-03-31. |

## Prompts

| Name | Description |
| --- | --- |
| `budget-plan` | Propose monthly budgets from the recent spending. |
| `categorize-uncategorized` | Find transactions with no category and propose one for each. |
| `find-unusual-spending` | Look for spending that stands out from the usual. |
| `import-statement` | Turn a pasted bank statement into transactions, after you approve them. |
| `monthly-review` | Review a month: where the money went and what changed. |
