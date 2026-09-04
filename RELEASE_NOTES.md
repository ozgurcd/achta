# Achta release notes

Release changes are recorded under their exact version, newest first.

## v0.1.0 — 2026-09-04

Initial private MVP release.

### Added

- Workspace-independent `version` and `capabilities` commands with stable,
  versioned JSON interfaces.
- Opt-in global `--timing`, rendered as the final `elapsed: Nms` human line or
  as `elapsed_ms` in the same single JSON document.
- Review-attested `wiki pin` updates with full-HEAD validation, suffix
  preservation, confined atomic replacement, and check mode.
- Strict `gate-run.v1` witness summaries with completeness, freshness, wall
  time, recorded-target time, and deterministic slow-target reporting.
- Deterministic amendment declaration and accepted-witness rebasing.
- Two-way amendment reconciliation through Rulefloor capability discovery and
  `rulefloor.ledger-diff.v1`.
- Exact machine fixtures, fuzz seeds, integration tests, a repository-local
  verification gate, and private release archives for macOS and Linux on amd64
  and arm64.
- Five armed Rulefloor invariants with measured mutation proofs covering
  workspace ambiguity, timed JSON, wiki review attestation, amendment
  reconciliation, and concurrent-write refusal.

### Security and compatibility

- Filesystem mutations are confined to the selected workspace and use
  same-directory atomic replacement with concurrent-change detection.
- Git and Rulefloor are invoked directly without a shell, with bounded output
  and execution time.
- Existing canonical wiki, witness, and amendment formats require no migration.

### Deferred

- Phase 3 `decision add` and broader wiki checks.
- Amendment clearing or automatic intent generation.
- Homebrew configuration, tap publication, and installation.
