# Changelog

## v0.2.2 — 2026-09-04

- Installed pinned Gograph in release CI so exact Rulefloor reach can be
  evaluated before publication.
- Carried the authenticated private-asset and delimiter-ordering fixes from the
  unpublished v0.2.1 attempt into the corrective release.

## v0.2.1 — 2026-09-04

- Fixed authenticated Homebrew downloads from Achta's private GitHub release
  by publishing release-asset API URLs and validating their exact mapping.
- Rejected out-of-order and orphan derived-block end markers found by fuzzing.
- The release workflow stopped before publication because its tool bootstrap
  did not yet install Gograph.

## v0.2.0 — 2026-09-04

- Added deterministic decision insertion and native wiki status/derivation
  checks.
- Added explicit witness lifecycle recording and current-green validation.
- Added read-only landed-slice auditing.
- Added exact machine contracts for the v0.2.0 commands.
- Completed required human, integration, failure-mode, and fuzz coverage for
  the shipped command set.
- Added a reproducible `make test-fuzz` gate for all required parser families.
- Made command-local help workspace-independent, decision heading parsing
  fail closed, and ledger-header reconciliation explicitly fail closed.
- Reported the resolved Rulefloor executable and tightened Git discovery and
  refusal-versus-invalid exit classification.
- Made Rulefloor execute-mode enforcement part of `make verify`.
- Made release publication select only the matching versioned release-note
  section.
- Added verified Homebrew cask publication for authorized private-release
  users.

## v0.1.0 — 2026-09-04

- Added workspace-independent version and capability discovery.
- Added explicitly attested, atomic wiki repository pin updates.
- Added strict `gate-run.v1` witness summaries with freshness and timing.
- Added deterministic amendment declaration and accepted-witness rebasing.
- Added two-way reconciliation through Rulefloor's stable machine interface.
- Added opt-in end-to-end elapsed timing for human and JSON command output.
- Added a five-rule armed Rulefloor floor with measured mutation proofs for
  release-critical invariants.
- Added private GitHub release archives for macOS and Linux on amd64 and arm64.
