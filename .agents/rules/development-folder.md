# The development folder

`development/` holds the files that exist only to develop the project locally. It is git-ignored and docker-ignored, and it is never published.

1. Never commit a file from it, and never reference it from product code, tests, CI, the site or public docs. Only the agent guides (`AGENTS.md`, `CLAUDE.md`, `.agents/`, `.claude/agents/`) name it.
2. Its files are written in Brazilian Portuguese, because the author reads them. Identifiers, commands, code and quoted errors stay in English.
3. It is kept in step with the project: what a change makes stale is updated or deleted in the same change.
4. No secrets, real financial data, builds or logs go in it.

The layout is fixed:

```text
development/
├── STATE.md       current state: branch, pull request, what each agent is doing, epics done and blocked, what waits for the author
├── QUESTIONS.md   what the agents need from the author, with options, the recommendation and what was done meanwhile
└── feedback.md    the open product problems (FB-NNN)
```

Decisions live in `../plans/00-roadmap.md`, and Vukem frictions in `../vukem-feedbacks.md`.
