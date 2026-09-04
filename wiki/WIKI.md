---
title: Achta Wiki — Schema and Conventions
category: meta
status: authoritative
updated: 2026-09-05
---

# Achta Wiki

This directory is the sole home of Achta-specific wiki information. It is
tracked in the same Git repository as the source it describes.

## Ownership

- `repos/achta.md` owns current product facts and the commit ledger.
- `platform/decisions.md` owns durable Achta decisions.
- `log.md` is the append-only Achta activity and release history.
- `index.md` is the complete page catalog.
- `agent-rules.md` defines the local maintenance contract.

The parent Identuum wiki may contain generic workspace governance and tooling,
but it must not duplicate Achta facts, decisions, release history, or a
repository page.

## Co-versioned repository pages

`repos/achta.md` carries `co_versioned: true` and a review date. It omits
`verified_against:` because the page and the source are committed together; a
commit cannot embed its own final SHA. Freshness means the page is present in
the same checkout as the source. Truth still requires review, recorded by
`verified:`. New commit-ledger rows use `co-versioned` in the commit column;
the containing Git commit supplies their exact identity.

## Maintenance

- State current facts once, in present tense.
- Append history; do not rewrite prior log entries.
- Keep the repository page bounded and move narrative into decisions or log.
- Run `make verify`; its wiki check must pass before the work is complete.
