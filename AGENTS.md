# AGENTS.md — Achta

The workspace-level `AGENTS.md` remains in force. These rules add the
repository-owned documentation contract for Achta.

Before non-trivial work, read:

- `wiki/WIKI.md`
- `wiki/index.md`
- `wiki/agent-rules.md`
- `wiki/repos/achta.md`
- `PROJECT_SPEC.md` when behavior or interfaces may change

Achta-specific wiki facts, decisions, history, and release records live only
under this repository's `wiki/` directory. Do not create or update an Achta
page, decision, or activity entry in the parent workspace wiki.

Every Achta commit must update the co-versioned wiki in the same commit:

- append one row to `wiki/repos/achta.md`, using `co-versioned` rather than an
  impossible self-SHA for that row's commit value;
- append one entry to `wiki/log.md` at the end of the file;
- update `verified:` only after re-reading the affected page against the code.

The repository page uses `co_versioned: true` and deliberately omits
`verified_against:`. A commit cannot contain its own SHA; the page and its new
ledger row are already versioned atomically with the source they describe.

Run `make verify` from the repository root after changes. For Go source
changes, follow the workspace Gograph session, plan, review, and audit rules.
