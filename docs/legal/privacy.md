# Privacy notice

Draft, pending review by the author. This text states facts about the code of this repository; it is not final legal text.

## This website

The website at finances.vukem.dev is static. It sets no cookies, runs no analytics, has no forms and loads no script, font, image or stylesheet from another site. It does not know who reads it. GitHub Pages serves the files and may keep its own server logs under its own policy.

## The software

vukem-finances is software that a person or an organization runs on their own computer or server. The people who run it are responsible for the data it stores; the project does not receive that data and operates no hosted service.

What the software stores, in the database of the machine that runs it (SQLite by default):

- Your account: name, email address and a password hash, and the workspaces and roles you belong to.
- Your finances: accounts, categories, tags, transactions, budgets, goals, recurring entries and credit card statements.
- Settings and notification preferences.
- The key you provide for AI features, stored encrypted, only when you turn those features on.
- The link between your account and a Telegram chat, only when you link one.

You can export your transactions as CSV and your workspace as JSON at any time.

## Optional connections

- AI features are off by default. When on, the text sent to the provider you chose goes through a redaction step first (see the AI documentation in the repository).
- The Telegram bot is off until the operator configures it. Messages pass through Telegram, whose servers are not end-to-end encrypted. See the Telegram documentation in the repository.

## Questions

Open an issue in the [repository](https://github.com/AlexGalhardo/vukem-finances).
