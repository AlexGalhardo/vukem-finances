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

Prompts and answers are not stored. No price table is kept, so the page shows
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
5. Saving a new key revokes the previous active one. The owner can delete the
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
  "the provider is unavailable"); the log gets the status code and a redacted,
  truncated excerpt.

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
