# Names of folders, files and ids

1. Product code keeps the Vukem layout: `app/`, `bootstrap/`, `cmd/`, `config/`, `database/`, `resources/`, `routes/`, `storage/`, `tests/`. New surfaces are `mobile/`, `desktop/` and `site/`. This layout must stay portable, because the app becomes the Vukem finance scaffold (epic 25).
2. Files and folders outside the framework's conventions are kebab-case, lowercase and ASCII, except canonical files (`README.md`, `AGENTS.md`, `CLAUDE.md`, `CHANGELOG.md`, `LICENSE`). Go and TypeScript files follow their language and the scaffold (React pages are PascalCase).
3. Ids are unique and never reused: `D-NNN` (decisions, in the roadmap), `VF-NNN` (Vukem feedback), `FB-NNN` (product feedback). Epics are numbered by their file, and tasks by position.
4. The team's agents are named by role in `.claude/agents/`: `senior-a` to `senior-e`, `intern`, `specialist`.
5. Branches and commits follow `git-and-branches.md` and `commits.md`: `epic/NN-<slug>` and `fix/FB-NNN-<slug>`, from the integration branch.
