---
title: Achta Wiki Agent Rules
category: meta
status: authoritative
updated: 2026-09-05
---

# Achta Wiki Agent Rules

The workspace agent contract remains authoritative for Git, secret, Gograph,
validation, and reporting rules. This page owns only the repository-local wiki
rules.

1. Read `WIKI.md`, `index.md`, and `repos/achta.md` before changing Achta.
2. Keep all Achta-specific wiki information under this `wiki/` directory.
3. Never create or update an Achta page, decision, or log entry in the parent
   workspace wiki.
4. Commit code and its wiki update together. Append one commit-ledger row using
   `co-versioned` as its commit value, plus one `log.md` entry, in the same
   slice. The containing Git commit is the exact anchor.
5. Keep `co_versioned: true` on the repository page and omit
   `verified_against:`; same-repository versioning is the freshness tie.
6. Update `verified:` only after checking every affected current-state claim
   against code on disk.
7. Run `make verify`, including the repository-owned wiki check.
