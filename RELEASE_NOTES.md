# Achta release notes

Release changes are recorded under their exact version, newest first.

## v0.2.0 — 2026-09-04

### Added

- Deterministic `decision add` allocation and section-boundary insertion with
  check mode and the `achta.decision-add.v1` interface.
- Native wiki freshness, derived-block, local-upstream, and composed checks
  with four stable machine interfaces.
- Explicit witness init, step, finalize, and freshness-check operations while
  keeping command execution in Make or CI.
- Read-only landed-slice checks for Git ancestry, commit identity, changed-path
  safety, module boundaries, log appends, and wiki pins.
- Exact machine fixtures for every v0.2.0 interface.
- Exact Phase 1 human fixtures for wiki-pin and witness-summary output, plus
  completion coverage for pin dry-run/no-op, stale witnesses, amendment
  reconciliation, decision headings, and derived delimiters.
- A repository-local `make test-fuzz` target covering all five narrow parser
  families required by the specification.
- A GoReleaser-generated Homebrew cask published to `ozgurcd/homebrew-tap`
  after the private GitHub release succeeds.

### Changed

- `make verify` now runs Rulefloor in execute mode so armed Go-test bindings
  are executed as part of the repository gate.
- Release automation extracts and publishes only the exact version section
  matching the release tag.
- Source version advanced to v0.2.0.
- Command-local `--help` and `-h` now succeed without workspace discovery and
  emit the same stable top-level help as the global form.
- The advertised global `--quiet` option now suppresses successful human detail
  while preserving failures, JSON documents, and explicit timing output.
- Amendment reconciliation now resolves Rulefloor before invocation and reports
  the selected absolute executable in human and machine output.

### Security and compatibility

- Witness writers refuse stale HEADs, dirty initialization, unrelated working
  tree changes, and concurrent record replacement.
- Native wiki status checks use local Git facts only and never fetch.
- Decision insertion now rejects malformed decision-like headings across the
  register instead of silently ignoring them.
- Ledger-header differences now fail amendment reconciliation while remaining
  separately reported in machine output.
- Canonical mutations reparse their complete rendered document before atomic
  replacement, and workspace confinement rejects linked path components.
- Witness initialization validates an existing canonical record before atomic
  replacement and leaves malformed existing bytes unchanged.
- Existing `gate-run.v1`, wiki Markdown, and amendment formats remain
  compatible.
- Latest real-workspace shadow checks agree with the retained scripts: 10 fresh
  pages, six unchanged derived blocks, and 13 local wiki commits ahead.
- Capability limitations now state the precise boundary—no Git mutation—while
  canonical artifact edits remain explicitly advertised per command.
- Lifecycle and amendment commands now preserve the central exit contract by
  distinguishing evaluated refusals from malformed or unsupported input.
- Git subprocesses validate executable discovery first and identify the selected
  executable in bounded failure diagnostics.
- Git measurements pass `--no-optional-locks`, preventing read-only workspace
  checks from refreshing sibling repository index metadata.
- Homebrew publication refuses version downgrades and verifies the tap copy;
  download authentication is supplied at install time through
  `HOMEBREW_GITHUB_API_TOKEN`, so installation remains limited to users
  authorized for the private release. Tap credentials are checked for push
  access before GitHub release creation, preventing a known partial release.

### Deferred

- Unconditional amendment clearing.

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
