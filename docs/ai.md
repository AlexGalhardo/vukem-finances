# AI

The application can use an AI model for features such as categorizing
transactions and answering questions about spending. The model is bring your
own key (decision D-007): the service never pays for AI. The workspace owner
connects a key from their own provider, turns AI on, and decides which
features may use it. AI is off by default.

This page describes what is sent, what is stored, how the key is handled and
which limits apply. The code is in `app/ai/`.

## What is sent

Only when AI is on for the workspace, the feature is on, and a key is saved.
A feature sends the description, amount, date and category names of the
transactions it needs, and nothing else.

Never sent: names of people, email addresses, account numbers, the key, or
anything from another workspace. Every free-text field goes through
`ai.Redact` before it leaves the server. It removes email addresses, phone
numbers, runs of nine or more digits (cards, tax ids such as CPF and CNPJ,
account numbers), IBAN-like strings and key-like tokens (`sk-`, `sk-ant-`).
Plain amounts such as `12.50` are left alone.

The enable dialog states this in plain words and the owner must tick a
checkbox before AI is turned on.

## What is stored

| Table | Holds |
| --- | --- |
| `ai_credentials` | provider, label, the key encrypted, the last four characters of the key (`key_hint`), optional base URL, default model, who saved it, `last_used_at`, `revoked_at` |
| `workspace_ai_settings` | `enabled` (default false), the per-feature switches, the monthly call limit, who enabled it and when |
| `ai_usage` | one row per call: feature, provider, model, input and output tokens, status, month |

Prompts and raw answers are not stored; the assistant keeps the sanitized
questions and its result tables for 90 days (see "Stored by the features").
No price table is kept, so the page shows
calls and tokens, never money; the provider's own billing page is the source
of truth for cost.

## The key lifecycle

1. The owner pastes the key in Settings, AI. The field is a password field,
   `autocomplete="off"`, and is cleared after saving.
2. The server seals it with the app key (`APP_KEY`, AES-GCM through
   `crypt.Encrypter`). The additional data is
   `ai-key:<workspace id>:<credential id>`, so a ciphertext copied to another
   row does not open.
3. After saving, every response, page and log shows only that a key exists and
   its `key_hint`. The `ai.Key` type prints as `[redacted]` in `String`,
   `GoString`, JSON and logs.
4. The key exists in clear only inside the function that builds one outbound
   request.
5. A workspace has one active key: saving a new one (for any provider) revokes the previous one. The owner can delete the
   key; deleting only marks it revoked and clears the ciphertext.
6. After an `APP_KEY` rotation, a daily job (`ai_reseal`) re-seals every
   credential whose ciphertext needs rotation. Keep the old key in
   `APP_PREVIOUS_KEYS` until the job has run.

The credentials table is not part of the sync registry, the JSON export or any
API scope.

Only the workspace owner (ability `ai.manage`) can save or delete the key,
enable or disable AI, change the features and the limit. Enabling, disabling,
saving and deleting a key are written to the audit log, without the key.

## Limits

- 30 second timeout, 1 MiB response size cap, no redirects followed (a redirect
  could carry the key to another host).
- One retry, with jitter, on 429 and 5xx. Never on another 4xx.
- 20 calls per minute per workspace.
- A monthly call limit per workspace (default 2000), counted in the workspace
  timezone. The check and the reservation of the call are one SQL statement,
  so concurrent calls cannot exceed it.
- A provider's error text is never shown to the browser or written to the log
  as is. The browser gets a generic message ("the provider rejected the key",
  "the provider is unavailable"); the log gets the provider, the feature, the
  status code and the provider's error type when it has a plain shape
  (`authentication_error`), and never the upstream message or body, because a
  provider can echo the key or the prompt in it.

## Custom base URL and local models

The `openai_compatible` provider takes a base URL for any server that speaks
the OpenAI Chat Completions format. The URL must be `https`. Calls go through
`httpclient.Safe`, which refuses loopback, private and link-local addresses
(including the cloud metadata address) at connect time, after DNS resolution,
and on every redirect.

In the personal edition only, the host administrator may set
`AI_ALLOW_PRIVATE_HOSTS=true` to reach a model server on the same machine or
network. `http` is then accepted for private hosts only. In the saas edition
the variable is ignored.

## Pages and routes

`GET /ai` (every member reads the status; only the owner receives the saved key
and its hint), `PUT /ai/settings`, `POST /ai/key`, `DELETE /ai/key/{credential}`
and `POST /ai/check` (the "Test key" button; one 16-token request, not counted
against the monthly limit). Every write checks `ai.manage` on the server.

## Features

Two features use the model. Both are off until the owner turns them on, and
both go through the same door (`ai.Client.Call`).

### Data sent by each feature

Every free-text field below goes through `ai.Sanitize` first: control and
Unicode space characters are normalized to one space, then `ai.Redact` runs,
then the text is clipped. Normalizing before redacting is deliberate: a card
number split by double spaces, tabs or non-breaking spaces is joined first, so
the digit rules catch it.

| Feature | Sent to the provider | Never sent |
| --- | --- | --- |
| Suggest categories (`categorize`) | per transaction: a short local id (`t1`..`t30`), the redacted description (120 characters at most), the kind and the amount in cents; the allowed categories of the matching kind (uid, name, parent name) | dates, account names, tags, names of people, anything of another workspace |
| Sentence to draft (`categorize`) | the typed sentence (redacted, 300 characters at most), today's date, the categories (uid, name, kind) | account names and uids: an account is matched locally on the server against the sentence, or left for the person to pick |
| Assistant (`assistant`), planning | the redacted question (500 characters at most), today's date, the currency, the names of the allowed reports, the categories (uid, name, kind) | account and tag names, balances, descriptions |
| Assistant, answer | the redacted question and the aggregated results of the queries the server ran: category names with totals, months with income and expense, **opaque labels** (`Account 1`, `Tag 2`) with totals for accounts and tags, the **net worth** per month (the sum over all accounts, never a balance per account), and, only if the plan asked for the `transactions` report, up to 200 redacted descriptions with amounts | real account and tag names (the server maps the labels back in the tables and in the answer text, locally) |

Memory first, AI second: a transaction whose description the workspace
categorized before gets its suggestion from the memory (`description_rules`),
with no call at all. That works with AI off, because it uses only the
workspace's own data and nothing leaves the server.

### Stored by the features

| Table | Holds | Kept |
| --- | --- | --- |
| `description_rules` | the normalized description and the category the people of the workspace chose | until the category goes |
| `suggestions` | a proposed category per transaction (by uid), its confidence, status and, for an auto-applied one, when | with the transaction |
| `assistant_conversations`, `assistant_messages` | the **sanitized** question (never the raw text: a card number typed in a question is redacted before it is stored), the model's answer text and the result tables the server computed | 90 days after the last message, then deleted by the daily job `assistant_prune`; the person can delete them in one click at any time |

Conversations are private to the person who asked, inside the workspace: no
route returns another person's conversation, and the owner and the admins are
no exception. They are never synced, never exported (not by the workspace JSON
export either) and never reachable by the API. Each question is answered on its
own: earlier turns are not sent to the provider.

### Prompt injection posture

A description, a category name, a question or a sentence is untrusted text. It
travels only as a value inside a JSON document, never in the system
instruction, and the instruction says it is data. What comes back is untrusted
too:

- The answers are accepted only as the JSON of a schema, only for ids the
  server sent and category uids the server listed, and of the right kind;
  anything else is dropped, and text instead of JSON yields nothing.
- A model never writes. A category suggestion is a row to review; nothing is
  applied unless the owner turned on "apply suggestions with confidence at
  least 90 %", off by default. Applied ones are listed and reversible for 7
  days. That switch trusts the model's confidence: a model that is wrong and
  sure of itself is applied until it is undone.
- The assistant never writes SQL and never names a table. Its plan is checked
  against an allowlist of seven reports, at most four queries, a range of 5
  years and 200 rows, and runs with the report functions of the workspace.
- The figures a person reads come from the server's own results. The model's
  sentence sits next to them, and the interface says the tables are
  authoritative and warns when the sentence quotes a number the tables lack.
- A sentence becomes a draft only: the amount is read by `finance.ParseAmount`
  (a cents value from the model is ignored), the date against `today`,
  `yesterday`, an ISO date or `dd/mm`. It is saved only when the person
  confirms it.
- The text of an answer is shown as plain text, never as HTML, Markdown or
  links.

### Limits of the features

A categorization run looks at 300 transactions at most, in batches of 30
transactions per call; the monthly limit and the rate of the workspace apply to
every call. The on-demand run is bounded to 25 seconds; what finished is kept.

### Evaluation

`testdata/categorization-eval.jsonl` holds 100 synthetic rows (English and
Portuguese descriptions with the expected category name). The command

```sh
go run ./cmd/app ai:eval --provider=fake
go run ./cmd/app ai:eval --workspace=<id>
```

runs them and prints the accuracy and the tokens used. With `--workspace` it
uses the key of that workspace, so it spends real calls: run it by hand, never
in CI. The `fake` provider is deterministic and needs no key.

## Add a provider

1. Add `app/ai/<name>.go` with a type that implements `ai.Provider`
   (`Complete` and `Check`), built from `ai.ProviderConfig`.
2. Use `ai.NewHTTP` for the outbound call: it applies the safe client, the
   limits, the retry and the error redaction. Never build an `http.Client`.
3. Register the name in `ai.Kinds` and in `ai.NewProvider`.
4. Test it against a local `httptest` server shaped like the documented API.
   No test may call a real provider or need a real key.
5. Cite the documentation page and the date read in the commit message.

Provider shapes were read on 2026-11-21 from
<https://platform.claude.com/docs/en/api/messages> (Anthropic Messages) and
<https://developers.openai.com/api/reference/resources/chat> (OpenAI Chat
Completions, still documented as current).
