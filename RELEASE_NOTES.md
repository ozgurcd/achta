# Achta release notes

Release changes are recorded under their exact version, newest first.

## Unreleased

### Added

- `wiki check --only NAME[,NAME...]` selects which of `freshness` and `derive`
  to evaluate. The selected checks are the only ones evaluated and the only
  ones reported, in canonical order; the exit contract is unchanged and applies
  to the selection. An empty, unknown, or repeated name is invalid input:
  exit 2, nothing evaluated. This gives callers an enforcing freshness check
  on a working tree they have legitimately dirtied, which no command offered.
- `achta.wiki-check.v1` carries `wiki_dir`, the wiki directory Achta resolved,
  and the text output prints it first, so a pass against the wrong wiki is no
  longer silent.
- Two mutation-proved Rulefloor invariants for check selection and resolved
  wiki reporting, raising the executable floor from 23 to 25.

## v0.4.0 — 2026-09-05

### Added

- A repository-owned wiki under `wiki/`, including current facts, durable
  decisions, release history, and local maintenance rules.
- `co_versioned: true` freshness semantics for a wiki committed in the
  repository it describes, avoiding an impossible self-referential Git SHA.
- Two mutation-proved Rulefloor invariants for co-versioned freshness and
  repository-owned log precedence, raising the floor from 20 to 22.
- A global `--wiki-dir PATH` selector for directly choosing a repository-owned
  wiki. It accepts only the canonical direct `wiki` child with the required
  layout and is mutually exclusive with `--workspace`.
- A mutation-proved Rulefloor invariant for direct wiki selection, raising the
  executable floor from 22 to 23.

### Changed

- Slice checks prefer a repository-owned `wiki/log.md` over a root `log.md`.
- Achta-specific wiki information is no longer owned or duplicated by the
  parent workspace wiki.
- Capability and help output advertise `--wiki-dir` as a stable global option.
- Source version advanced to v0.4.0.

## v0.3.0 — 2026-09-04

### Added

- Parsed, name-singleton sibling repository pins in `gate-run.v1`, with clean
  HEAD and tree-digest capture during finalization and explicit repository
  mappings during checks.
- Reachability-aware witness staleness and `reachability classify`. A fully
  declared no-reach diff is reported as `proven_no_reach`; unmatched paths fail
  closed as `REQUIRED`, catch-all declarations are rejected, and machine output
  records every changed and excluded path.
- CI provenance fields for run URL, attempt, and SHA. Witness checks require a
  green, complete, clean commit tie on local HEAD ancestry and perform no fetch
  or network request.
- `witness earned`, which refuses a new witness cycle when the committed delta
  since the newest accepted witness contains only witness machinery paths.
- Declarative toolchain parity for workspace version pins, CI environment pins,
  and confined script SHA-256 values. Manifests cannot define commands.
- Exact machine fixtures and unit/integration coverage for all new interfaces.

### Changed

- Source version advanced to v0.3.0.
- Capability output now authoritatively lists `witness summarize`, the new
  reachability, earned-cycle, and toolchain commands, and their schemas.
- Existing shell witness implementations remain in compatibility shadow mode;
  retirement requires a later explicit parity-backed slice.

### Deferred

- Published-fix vulnerability policy remains in dedicated vulnerability
  analyzers because it is product- and ecosystem-specific rather than workspace
  bookkeeping.

## v0.2.5 — 2026-09-04

### Fixed

- Homebrew can evaluate the Achta Cask during `brew update` when
  `HOMEBREW_GITHUB_API_TOKEN` is absent. Private release downloads remain
  authenticated when the token is supplied to the install or upgrade command.

### Changed

- Source version advanced to v0.2.5.

## v0.2.4 — 2026-09-04

### Fixed

- The canonical verification gate now builds a current precise Gograph index
  before Rulefloor evaluates exact structural reach in a clean checkout.
- The authenticated Homebrew asset-API rewrite and derived-marker ordering fix
  are carried forward from the fail-closed, unpublished patch tags.

### Changed

- Source version advanced to v0.2.4.

## v0.2.3 — 2026-09-04

The v0.2.3 tag installed Gograph successfully but stopped at verification
before any GitHub release or Homebrew update was published because the clean
checkout had no precise graph yet.

### Fixed

- Release CI now installs Gograph from its executable package at
  `github.com/ozgurcd/gograph/cmd/gograph@v1.6.10`.
- The authenticated Homebrew asset-API rewrite and derived-marker ordering fix
  are carried forward from the two tags whose fail-closed verification gates
  stopped before publication.

### Changed

- Source version advanced to v0.2.3.

## v0.2.2 — 2026-09-04

The v0.2.2 tag stopped during tool installation before any GitHub release or
Homebrew update was published because the Gograph module root is not an
installable command package.

### Fixed

- The release runner now installs pinned Gograph v1.6.10 before Rulefloor
  evaluates exact structural reach, allowing the authenticated private-asset
  invariant to run in CI.
- The authenticated Homebrew asset-API rewrite and derived-marker ordering fix
  first tagged in v0.2.1 are carried into the publishable patch release.

### Changed

- Source version advanced to v0.2.2.

## v0.2.1 — 2026-09-04

The v0.2.1 tag stopped at its verification gate before any GitHub release or
Homebrew update was published because Gograph was absent from the CI tool
bootstrap.

### Fixed

- Homebrew downloads for the private Achta repository now use GitHub's
  authenticated release-asset API URLs instead of browser download URLs that
  return `404` for private assets.
- Release publication now fails closed unless all four generated archive URLs
  map exactly once to numeric GitHub asset API endpoints, retain the required
  authentication headers, and match the published checksums.
- Derived-block validation now rejects end-before-start and orphan end markers,
  closing a reproducible delimiter fuzz failure before publication.

### Changed

- Source version advanced to v0.2.1.

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
