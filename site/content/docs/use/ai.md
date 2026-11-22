---
title: "AI with your own key"
description: "Optional AI features that use a provider key you own: what is sent, what is stored, how the key is protected and which limits apply."
---

The application can use an AI model to suggest categories, draft an entry from a sentence and answer questions about your spending. AI is bring your own key: the service never pays for it. The workspace owner connects a key from their own provider, turns AI on and chooses which features may use it. AI is off by default.

## Offline

AI needs a model, and a hosted model needs the internet. With no network a call fails with a clear message and everything else keeps working. A model server on your own computer or local network needs no internet: in the personal edition, set `AI_ALLOW_PRIVATE_HOSTS=true` to let the OpenAI-compatible provider reach it.

## What is sent

Only when AI is on, the feature is on and a key is saved. A feature sends the description, amount, date and category names of the transactions it needs, and nothing else. Never sent: names of people, email addresses, account numbers, the key, or anything from another workspace.

Every free-text field passes a redaction step first. It removes email addresses, phone numbers, runs of nine or more digits (cards, tax ids, account numbers), IBAN-like strings and key-like tokens. Plain amounts are left alone. The dialog that turns AI on states this and asks you to tick a box.

## What is stored

| Stored | What it holds |
| --- | --- |
| The credential | provider, label, the key encrypted, the last four characters, an optional base URL and default model |
| The settings | whether AI is on, the switch of each feature and the monthly call limit |
| The usage | one row per call with the feature, provider, model, token counts, status and month |

Prompts and raw answers are not stored. The assistant keeps its sanitized questions and result tables for 90 days. No prices are kept, so the usage page shows calls and tokens, never money: your provider's billing page is the source of truth.

## The key

The key is pasted in a password field, encrypted with `APP_KEY` and shown afterwards only as a short hint. It never appears in a response, a log, an audit entry or an export. A workspace has one active key. After you rotate `APP_KEY`, keep the old one in `APP_PREVIOUS_KEYS` until the daily re-sealing job has run.

Only the workspace owner can save or delete the key, turn AI on or off and change the features and the limit.

## Limits

- 30 second timeout, 1 MiB response cap and no redirects.
- One retry on a 429 or 5xx answer.
- 20 calls per minute per workspace.
- A monthly call limit per workspace, 2000 by default.

## The features

- **Category suggestions.** The app first answers from what it learned from your own categorizations. Only then it asks the model, and the result is a row for you to review, never a saved change.
- **Entries from a sentence.** A sentence becomes a draft that you confirm. The amount is read by the same parser as every other amount.
- **The assistant.** It answers questions with the figures of the app's own reports and never writes data.

Text from your transactions is treated as data and never as an instruction to the model. The full description is `docs/ai.md` in the repository.
