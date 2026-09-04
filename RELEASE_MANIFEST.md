# Achta v0.1.0 first-publish manifest

This manifest classifies every repository path family considered for the first
publication. Only explicit, surgical staging is permitted.

## Include

- `.github/workflows/verify.yml` — runs the repository verification gate in CI.
- `.github/workflows/release.yml` — validates a version tag and publishes GitHub
  release archives through GoReleaser.
- `.gitignore` — prevents local caches, binaries, coverage, databases, and editor
  artifacts from entering the release.
- `.goreleaser.yaml` — defines deterministic Darwin and Linux release archives and
  their checksum file.
- `Makefile` — owns the complete repository-local build and verification commands.
- `go.mod` — declares the module identity and required Go toolchain version.
- `cmd/achta/` and `internal/` — contain the CLI entry point and product code.
- `testdata/machine/` — contains exact stable machine-interface fixtures.
- `README.md`, `CHANGELOG.md`, `RELEASE_NOTES.md`, and `PROJECT_SPEC.md` — document
  the supported release, its versioned changes, scope, and contract.
- `RULE-FLOOR.md` — is the canonical repository-local Rulefloor ledger.
- `RELEASE_MANIFEST.md` — records this first-publication classification.

## Exclude

- `.git/` — repository metadata is not release content.
- `.gograph/` — generated local analysis output is never published.
- `.cache/` — generated build, test, and analyzer caches are never published.
- `bin/`, `dist/`, `coverage.out`, and `*.test` — local executable or generated
  artifacts are recreated by CI and are never staged.
- `*.db`, `*.sqlite`, and `*.sqlite3` — local databases are not product inputs.
- `.DS_Store` and editor metadata — machine-local noise is not product content.
- `.env*`, keys, tokens, credentials, cookies, licenses, and recovery material —
  secret or local authentication material is never read, staged, or published.
- `Formula/` and tap publication — Homebrew work is explicitly deferred by the
  owner for this slice.
- `wiki/` and every sibling repository under `identuum/` — the owner restricted all
  workspace writes for this task to `identuum/achta`.

## Owner decisions

- The v0.1.0 machine schemas advertised by `achta capabilities` are stable;
  breaking changes require a new schema version.
- Phase 3 `decision add` is deferred and unadvertised until its register
  allocation and insertion policy is measured and approved.
- Homebrew configuration, tap publication, and installation are deferred for
  this slice.
