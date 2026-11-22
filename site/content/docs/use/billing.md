---
title: "Billing for the SaaS edition"
description: "How the SaaS edition charges for the Pro plan with Stripe or a one-off PIX payment, who may manage it and how access is granted and revoked."
---

Billing exists only in the SaaS edition. In the personal edition nothing here is registered, shown or read: no billing page, no webhook, and no Stripe or AbacatePay setting is looked at. No hosted service is currently offered, so this page is for an operator who runs the SaaS edition on their own server; see [the editions](/docs/start/editions/).

## Plans

| | Free | Pro |
| --- | --- | --- |
| Accounts | 1 | unlimited |
| Transactions per month | 100 | unlimited |
| Members | 1 | unlimited |
| CSV import | no | yes |
| Budgets and alerts | no | yes |

The workspace pays, not the person: only its owner can start or manage billing. Admins, members and viewers see the plan and nothing else.

## Two ways to pay

- A **card subscription** with Stripe, renewed every month, with a 14 day trial on the first subscription of a workspace.
- A **one-off PIX payment** with AbacatePay for 1, 3 or 12 months, with no automatic renewal.

## Rules that never bend

1. Access changes only from a verified webhook, or from a re-read of the charge at the provider. Never from the page a checkout returns to and never from something the browser sent.
2. Amounts are integers in cents and come from the server. A request names only the number of months, and it must be one of the offered options.
3. A replayed, duplicated or out-of-order event changes nothing twice.
4. Provider error text and secrets are never logged or shown.

## Access states

Access is derived, never stored as a flag. A workspace is Pro when a Stripe subscription is active or in trial and not revoked, or when a PIX entitlement is not revoked and its end date is in the future. Otherwise it is Free. Both sources can hold at once. A full refund or a dispute revokes access, and a later active status from Stripe cannot give it back.

## Settings for the operator

| Variable | Meaning |
| --- | --- |
| `STRIPE_KEY` | a restricted Stripe key with only the billing permissions |
| `STRIPE_WEBHOOK_SECRET` | the signing secret of the Stripe webhook |
| `STRIPE_PRICE_PRO` | the recurring price id of the Pro plan |
| `STRIPE_CURRENCIES` | the currencies a checkout may ask for |
| `ABACATEPAY_API_KEY` | turns PIX on when set |
| `ABACATEPAY_WEBHOOK_SECRET` | the shared secret of the PIX webhook |
| `PIX_PRICE_PRO_MONTHLY_CENTS` | the price of one month in BRL cents, at least 100 |
| `PIX_MONTHS_OPTIONS` | the months that may be bought, `1,3,12` by default |

A production boot of the SaaS edition requires the three PIX settings only when `ABACATEPAY_API_KEY` is set. A job re-reads PIX charges that can still change every 6 hours.

Payment records follow a retention rule: after an account is deleted, only the amount, date, provider reference and a non-reversible subject reference are kept, for 5 years. The full design is `docs/billing.md` in the repository.
