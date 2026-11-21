# Billing

Billing exists only in the `saas` edition (decision D-003). In the personal
edition nothing here is registered, shown or read: no billing page, no
`/stripe/webhook`, no `/abacatepay/webhook`, no `/admin/revenue`, and no
Stripe or AbacatePay setting is looked at. The code is in `app/billing/`.

A workspace owner can pay for the Pro plan in two ways (decision D-006):

- a card subscription with Stripe, renewed by Stripe every month, with a
  14-day trial on the first subscription of a workspace;
- a one-off PIX payment with AbacatePay for 1, 3 or 12 months, with no
  automatic renewal.

Only the owner manages billing. The check is on the server (`billing.manage`);
admins, members and viewers get a 403 and can only see the plan. A workspace
never reads or changes the billing data of another one.

## The rules that never bend

1. Access changes only from a verified webhook (or from a re-read of the
   charge at the provider), never from a redirect back from a checkout and
   never from something the browser sent.
2. Amounts are `int64` cents and come from the server: the plan table
   (`PIX_PRICE_PRO_MONTHLY_CENTS * months`, whole integers, no floats) or the
   Stripe price. The request only names the number of months, and it must be
   one of `PIX_MONTHS_OPTIONS`.
3. A replayed, duplicated or out-of-order event changes nothing twice.
4. Provider error text and secrets are never logged, shown or written to an
   audit entry. A provider failure is recorded with a status code and a plain
   kind only.

## States of access

Access is derived, never stored as a flag. `workspaces.Entitled(ctx,
workspaceID, now)` is the only place that decides, and `Limits` uses it:

| Source | The workspace is Pro when |
| --- | --- |
| Stripe | a subscription row is `active` or `trialing` and `access_revoked_at` is empty |
| PIX | the entitlement is not revoked and `valid_until` is after now |

Otherwise it is Free. Both sources can hold at the same time; one being
revoked does not touch the other.

The billing page shows one of: trialing, active, ends on a date, not renewing
(PIX), or revoked with the reason (`refund`, `dispute`, `dispute_lost`,
`admin`).

## Stripe: revocation

`workspace_subscriptions` has `access_revoked_at`, `revoked_reason` and
`revoked_ref`. `SyncSubscription` never writes them, so a later `active`
status from Stripe cannot give access back. `Valid()` is false whenever
`access_revoked_at` is set.

Every handler below reads the current state from the Stripe API instead of
trusting the order of events, and writes with a conditional `UPDATE`, so a
second delivery finds nothing to change.

| Event | Effect |
| --- | --- |
| `charge.refunded` | a charge of a subscription invoice that is fully refunded (`amount_refunded == amount`) revokes with reason `refund` and cancels the subscription at Stripe at once (`DELETE /v1/subscriptions/{id}`, no proration, no final invoice). A partial refund keeps access and writes a `billing_events` row. |
| `charge.dispute.created` | revokes with reason `dispute`; `revoked_ref` is the charge id |
| `charge.dispute.closed`, `won` (or `warning_closed`) | clears the revocation only when the reason is `dispute`, `revoked_ref` is this dispute's charge and the subscription is still `active`; a refund is never undone by a won dispute |
| `charge.dispute.closed`, `lost` | stays revoked, reason becomes `dispute_lost`, the subscription is cancelled |

The owners get an email saying why access ended. The local revoke is the
source of truth and happens first. If the cancel call to Stripe fails, the
webhook answers 500 and Stripe delivers the event again; the retry finds the
row already revoked and only repeats the cancel. A new checkout after a
revocation is allowed and creates a new subscription row.

## PIX flow

Table `pix_payments`: `public_ref` (a ULID used in URLs and logs),
`provider_ref` (unique), `status` (`pending`, `paid`, `expired`, `refunded`,
`disputed`), `amount_cents`, `currency` (`BRL`), `months`, the copy-and-paste
`br_code`, the QR image, `expires_at`, `paid_at`, `created_by`.

1. The owner picks a number of months. The server computes the amount and
   creates a charge at AbacatePay (`POST /transparents/create`, method `PIX`,
   `expiresIn` 1800 seconds).
2. The dialog shows the QR code and the copy-and-paste code and polls
   `GET /billing/pix/{public_ref}` every 3 seconds (rate limited, stops at
   paid or expired).
3. Payment is learned from the webhook or from a poll. Either way the service
   first re-reads the charge from the provider (`GET /transparents/check`) and
   believes only that answer, never the body of the request.
4. A charge becomes `expired` only after the provider says so, never from the
   local clock. A payment the provider confirms just after the QR timer still
   activates the plan. A charge the provider had already reported expired and
   later reports paid (a late payment) also activates: the money was
   received.

### Idempotent activation

In one database transaction the service runs
`UPDATE pix_payments SET status='paid', paid_at=? WHERE id=? AND status IN
('pending','expired')` and goes on only when `RowsAffected() == 1`, extending
the entitlement in the same transaction. Every repeat (webhook retry, poll plus
webhook, 50 deliveries at once) finds 0 rows and returns success with no
effect. One payment never adds its months twice.

### Entitlement and months

`pix_entitlements` (one row per workspace): `anchor_at`, `paid_months`,
`valid_until = AddMonths(anchor_at, paid_months)`, `revoked_at`,
`revoked_reason`.

- A payment while still valid adds to `paid_months`.
- A payment after expiry or revocation resets `anchor_at` to now and
  `paid_months` to the new months.

Because the end is always computed from the anchor, there is no drift: paying
one month on Jan 31 and again while valid ends on Mar 31, not Mar 28 or Mar 3.
`timeutil.AddMonths` clamps the end of a month.

### Refunds and disputes of a PIX payment

`RevokePixPayment` sets the payment to `refunded` and subtracts its months from
`paid_months`, so access ends only when no paid period remains. The owners are
told. A dispute (`UNDER_DISPUTE`) marks the payment `disputed` and subtracts
the months the same way; if the provider later says `PAID` again the dispute
was won and the months come back, and if it says `REFUNDED` the payment is
refunded (the months are not subtracted twice).

### The provider documentation used

Read from the AbacatePay documentation (docs.abacatepay.com) when the epic
was built:

- API base `https://api.abacatepay.com/v2`, `Authorization: Bearer <key>`.
- `POST /transparents/create` body `{ method: "PIX", data: { amount,
  description, expiresIn, metadata } }`: `amount` in cents (minimum 100),
  `expiresIn` in seconds (minimum 300). Response `data`: `id`, `amount`,
  `status`, `brCode`, `brCodeBase64` (a data URL of a PNG), `expiresAt`.
- `GET /transparents/check?id=` answers `data.status`: `PENDING`, `EXPIRED`,
  `CANCELLED`, `PAID`, `UNDER_DISPUTE`, `REFUNDED`, `REDEEMED`, `APPROVED`,
  `FAILED`.
- `POST /transparents/simulate-payment?id=` pays a charge in Test Mode (keys
  that start with `abc_dev_`).
- Events: `transparent.completed`, `transparent.refunded`,
  `transparent.disputed` (status `UNDER_DISPUTE`, then `PAID` if won or
  `REFUNDED` if lost) and a listed `transparent.lost`. The envelope is `{ id,
  event, apiVersion, devMode, data }` and the charge id is
  `data.transparent.id`. Delivery retries up to 7 times over 18 hours.
- Authentication of a delivery: the secret registered with the webhook, sent as
  `?webhookSecret=` and in the `X-Webhook-Secret` header; and
  `X-Webhook-Signature`, an HMAC-SHA256 (base64) of the raw body made with a
  public key that is the same for every account, so it proves integrity only.
  The signature carries no timestamp, so there is no age check to make: the
  protection against replay is the idempotent activation above.

The design does not depend on the event names: the reconciliation job below
finds refunds by re-reading the charge.

## Webhook endpoint

`/abacatepay/webhook` is registered ahead of the session and CSRF middleware,
because the caller is a server.

- The shared secret (`ABACATEPAY_WEBHOOK_SECRET`) is compared in constant time.
  It is read from the `X-Webhook-Secret` header or from a path segment
  (`/abacatepay/webhook/{secret}`, declared with `SecretParams`). The query
  string is never read: a request whose only credential is the query string is
  refused with 401. Register the URL without a query; AbacatePay appends
  `?webhookSecret=` on its own, so keep access logs of this path off or
  redacted at the proxy.
- `X-Webhook-Signature` must match the HMAC of the raw body (constant time).
- The body is capped (413 above 64 KiB) and decoded with a strict struct.
- The body is never trusted for activation: a "paid" event for a charge the
  provider still reports pending activates nothing.
- Unknown events answer 200 and are logged by name only.

Stripe's endpoint is `/stripe/webhook`; its signature check (constant time,
5 minutes of timestamp tolerance) and the remembering of event ids come from
the `contrib/stripe` handler.

## Reconciliation

The job `billing_reconcile` runs every 6 hours. For each PIX payment paid in
the last 35 days it re-reads the charge. When the provider says refunded it
calls `RevokePixPayment`; when it says under dispute it records the dispute.
For each pending charge past its expiry it re-reads the charge and expires it
only when the provider confirms. Running it twice changes nothing the second
time.

## What the admin can do

- `/admin/revenue` sums active Stripe subscriptions (their monthly price) and
  the PIX monthly equivalent (`amount_cents / months` of the valid, not
  revoked entitlements), leaving out the revoked ones.
- A support admin can mark a PIX payment as refunded from the admin panel with
  the same effect as `RevokePixPayment`. Each such action writes a
  `billing_events` row with source `admin`.
- The panel is read-only for subscriptions: it cannot grant access.

## Audit and retention

`billing_events` (`workspace_id`, `source`, `kind`, `ref`, `summary`) records
what happened to a workspace's plan, with provider ids only: no card data and
no personal data.

Payment records follow decision D-022: after a deletion only the amount, the
date, the provider reference and a non-reversible subject reference are kept,
for 5 years (`PAYMENT_RECORD_RETENTION_YEARS`, default 5). That default has
had no legal review.

Billing tables are server-only data. None of them has a `uid` column, so the
sync guard does not treat them as synced entities and they never reach a
device; all of them carry `workspace_id` with a leading index and join
`models.TenantModels`.

## Settings

| Variable | Meaning |
| --- | --- |
| `STRIPE_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PRICE_PRO`, `STRIPE_CURRENCIES` | Stripe, as before |
| `ABACATEPAY_API_KEY` | enables PIX when set; use a key with `TRANSPARENT:CREATE` and `TRANSPARENT:READ` |
| `ABACATEPAY_WEBHOOK_SECRET` | the shared secret of the webhook (required with the API key in production) |
| `PIX_PRICE_PRO_MONTHLY_CENTS` | price of one month in BRL cents (at least 100) |
| `PIX_MONTHS_OPTIONS` | months offered, default `1,3,12` |

A production boot in saas requires the three PIX settings only when
`ABACATEPAY_API_KEY` is set: PIX is optional.

What the author must create at the providers: a Stripe restricted key, price
and webhook endpoint listening to the events in the table above plus the
subscription and invoice events; an AbacatePay API key, and a webhook that
points at `https://<host>/abacatepay/webhook/<secret>` for the transparent
events.
