# Achta release notes

Release changes are recorded under their exact version, newest first.

## Unreleased

## v0.5.5 — 2026-09-06

### Corrected

- Reverted `count check` and `declared-route check` from `replaces` to
  `candidate`. The v0.5.4 manifest recorded equal exits for six rows whose
  cited replay records unequal exits; a citation that contradicts a claim is
  not parity evidence.

### Added

- `achta.replacement-claims.v2` turns replay citations into checkable inputs.
  Each replacement or retirement claim names a workspace-confined
  `achta.replacement-replay.v1` artifact, pins its SHA-256, and must match its
  verb/script identity, selftest citation, and complete fixture set row for
  row. Legacy v1 claim rows fail as uncheckable attestations.
- The committed replay evidence is vendored with its source path, source commit,
  and source digest as provenance. The frozen v0.5.4 manifest now fails on the
  six rows it contradicted: count `g` and declared-route `02`, `03`, `04`, `07`,
  and `10`. No replay was rerun and neither candidate was promoted.
- Release validation now requires an annotated tag object and still requires
  its dereferenced commit to equal HEAD. v0.5.4 remains untouched; future
  releases cannot repeat its lightweight-tag inconsistency.
- Two mutation-proved rules raise the executable floor from 43 to 45.

## v0.5.4 — 2026-09-06

### Changed

- `declared-route check` now treats a comment-only or whitespace-only workflow
  as an evaluated document with zero live scalars. With `per-file-any` it can
  pass; with the default exact-one route cardinality it is an evaluated exit 1.
- A workflow using the designated route must contain the exact caller-named
  required key once across the whole YAML document and once in the selected
  scope. A second job-level declaration can no longer hide beside a valid
  workflow-level declaration.
- `replacement-claims.json` records full 10-of-10 parity with
  `wiki/tools/rulefloor-install-gate.sh`, citing the script selftest and the
  frozen dual-replay table. Two mutation-proved rules raise the floor to 42.
- `count check` accepts repeatable caller-owned `--claim-target-pattern` and
  `--claim-assertion-pattern`. When both match within one blank-line-delimited
  paragraph, the smallest positive target count is checked against the exact
  distinct citations in that paragraph. This captures split assertion prose
  without inferring its vocabulary.
- `replacement-claims.json` records full 17-of-17 parity with
  `wiki/tools/count-claim-check.sh`, citing its selftest and the frozen replay
  table. COUNT-PARAGRAPH-COMPOSITION-1 raises the floor from 42 to 43.

## v0.5.3 — 2026-09-06

### Added

- `replacement check --file MANIFEST --claim-status STATUS
  [--claim-status STATUS]... [--json]` makes principle 10 executable. The
  caller declares which exact statuses assert replacement or retirement; for
  every matching verb-to-named-script entry Achta requires the script's own
  selftest-fixture citation, a dual-replay citation, and agreeing script and
  verb exit codes for every recorded fixture.
- `achta.replacement-claims.v1` is the canonical replacement-claim artifact.
  Candidate entries carry no replacement claim. Missing required evidence or
  exit disagreement is exit 1; malformed or unavailable evidence is exit 2.
  Achta does not run either implementation or infer replacement claims from
  prose.
- `make verify` now checks `replacement-claims.json`. A frozen v0.5.1 manifest
  names `count check`, `declared-route check`, and `mirror check --digest` as
  replacements without script-side replay; it fails with nine evidence
  violations and prevents regression to the two-release parity gap.
- `achta.replacement-check.v1` is advertised in capabilities, and
  REPLACEMENT-PARITY-CITATION-1 raises the executable Rulefloor from 39 to 40
  with a mutation proof.

## v0.5.2 — 2026-09-06

### Added

- `count check` now accepts repeatable `--claim-pattern`, repeatable
  `--proof-claim-pattern` and `--proof-pattern` with
  `--proof-within-lines`, optional `--claim-section-pattern`, and repeatable
  `--exempt-pattern`. These reproduce caller-classified CLAIM-vs-PROOF,
  last-matching-`## `-section, and next-nonblank-paragraph exemption mechanics
  without giving Achta ownership of the caller's prose vocabulary or proof
  meaning.
- `declared-route check` now accepts repeatable `--dir` and
  `--route-cardinality per-file-any`. Direct lowercase `.yml` and `.yaml`
  files are bytewise ordered. Under `per-file-any`, each workflow may contain
  zero or many configured routes, every workflow is still ban-scanned, and
  every file using the designated route must declare its required key exactly
  once at the selected YAML scope.
- `mirror check --digest-name NAME` selects one canonical digest record by its
  exact caller-supplied name, so records such as
  `tools/rulefloor-install-gate.sh` are supported. Omitting the flag preserves
  exact-basename selection. Record names and master bytes are never normalized.
- Three independent mutation proofs raise the executable Rulefloor from 36 to
  39. The stable machine schemas and 0/1/2 exit contracts are unchanged.

## v0.5.1 — 2026-09-06

### Added

- `declared-route check --file WORKFLOW [--file WORKFLOW]...
  --route-pattern REGEX [--route-pattern REGEX]...
  --ban-pattern REGEX [--ban-pattern REGEX]...
  --required-route-pattern REGEX --required-key KEY
  --required-scope JSON_POINTER [--json]` evaluates an
  explicit workflow set. Exactly one configured route alternative must occur
  per file and configured banned patterns must occur zero times. When the
  designated route is selected, its required key must occur exactly once in
  the selected YAML mapping.
- `--required-scope` is a caller-owned JSON Pointer such as `/env`, resolved
  through a real YAML syntax tree. A key under `/jobs/verify/env` does not
  satisfy `/env`; malformed, multi-document, missing, unsafe, or structurally
  ambiguous YAML is `cannot_evaluate`, exit 2. YAML comments and comment-only
  scalar lines are excluded from route and ban counts.
- `achta.declared-route-check.v1` reports each file's per-pattern counts and
  lines, the structurally scoped key count, and every 0/1 disagreement.
  Route, designated-route, banned-command, key, and scope vocabulary have no defaults. The
  command reads only confined regular files and invokes neither shell nor Git.
- A measured correctness requirement adds the stable
  `go.yaml.in/yaml/v3 v3.0.5` AST parser. A mutation-proved Rulefloor
  invariant raises the executable floor from 35 to 36.

- `count check --file MD [--file MD]... [--dir DIR]...
  [--total-pattern REGEX --part-pattern REGEX]
  [--claim-pattern REGEX --citation-pattern REGEX]
  [--count-alias TOKEN=N]... [--json]` checks two caller-declared structural
  relationships. Breakdown parts must sum to the stated total on their line;
  a configured count claim needs at least that many distinct exact citation
  matches in its blank-line-delimited paragraph. At least one complete pattern
  pair is required, and each count-bearing pattern exposes exactly one named
  `count` capture.
- Explicit files are evaluated before explicit directories; direct `.md` or
  `.MD` directory entries are evaluated in bytewise filename order. Achta owns
  decimal arithmetic, paragraph scope, and exact distinct-match counting. The
  caller owns every token and regular expression; non-decimal count tokens need
  explicit `--count-alias TOKEN=N` declarations.
- `achta.count-check.v1` reports every violation's file, line, rule,
  claimed count, observed count, and text. Achta refuses to infer that
  unconfigured prose makes a claim or that a matched citation proves a
  disposition. Exit 0 is clean, exit 1 is a structural disagreement, and
  unsafe, absent, malformed, ambiguous, or incomplete input is exit 2.
- A mutation-proved Rulefloor invariant raises the executable floor from 34 to
  35.

- `mirror check --master PATH --digest FILE [--mirror PATH]...` verifies that
  the SHA-256 recorded for the master's exact basename equals the digest of
  the master's exact bytes. Its additive `achta.mirror-check.v1` digest result
  names the digest file, selected line and basename, recorded digest, computed
  master digest, and `match` or `differs` status.
- Achta accepts a multi-record digest file only when exactly one canonical
  lowercase `sha256  name` record names the master's basename byte-for-byte.
  It refuses whitespace, line-ending, case, path, or content normalization.
  Digest disagreement is exit 1; absent, malformed, or ambiguous evidence is
  `cannot_evaluate`, exit 2.
- A mutation-proved Rulefloor invariant raises the executable floor from 33 to
  34.

## v0.5.0 — 2026-09-05

### Added

- v0.5.0 is the conversion-batch baseline intended for caller migrations. Its
  released capability surface includes `recipe check`, `ledger census`, `floor
  census`, `mirror check`, `parts lock`, `parts verify`, and `ledger rows`.
- `parts lock --dir DIR --lock FILE [--bump]` writes the canonical
  `achta.parts-lock.v1` artifact from direct `.txt` files, preserving its
  `VERSION` unless `--bump` explicitly increments it. `parts verify --dir DIR
  --lock FILE` recomputes exact SHA-256 bytes, requires every locked name at
  its digest, and rejects every unlocked `.txt` neighbor. Its stable
  `achta.parts-verify.v1` result names each `match`, `differs`, `absent`, or
  `unlocked` part; drift is exit 1 and unsafe, missing, malformed, or
  concurrently changing evidence is `cannot_evaluate`, exit 2.
- Achta owns the lock grammar because it writes the artifact: schema marker,
  `VERSION vN`, then unique bytewise filename-sorted `name sha256` records.
  Caller-specific section-heading references remain outside Achta.
- A mutation-proved Rulefloor invariant raises the executable floor from 31 to
  32.
- `ledger rows --file MD --id-cell N --prose-cell N --open-marker TEXT
  --closed-marker TEXT --identity-end-marker TEXT --completion-marker TEXT...
  --condition-marker TEXT... --quote-marker TEXT [--exempt-marker TEXT]...`
  checks caller-shaped Markdown ledger rows. An open ID cell conflicts with a
  configured completion literal unless an explicit exemption literal is
  present. A closed row containing a configured condition literal must carry
  exactly one quote marker followed by a double-quoted string whose bytes also
  appear elsewhere in the selected prose cell. `achta.ledger-rows.v1` reports
  line, rule, row identity, and text per violation under the 0/1/2 contract.
- Every vocabulary token is an explicit flag with no default. Achta refuses to
  infer that prose means completion, normalize a quote, or assert that a
  repeated condition was actually satisfied. The local implementation invokes
  no shell, performs no Git operation, and writes nothing.
- A mutation-proved Rulefloor invariant raises the executable floor from 32 to
  33.

## v0.4.5 — 2026-09-05

### Added

- `mirror check --master PATH --mirror PATH [--mirror PATH]...` replaces
  repeated file-copy comparisons with one workspace-confined, read-only verb.
  It computes SHA-256 over the exact bytes of the master and every mirror in
  caller order, reporting `match`, `differs` with both digests, or `absent` per
  mirror through the stable `achta.mirror-check.v1` interface. It never
  normalizes newlines, whitespace, encoding, or document meaning and invokes no
  shell or Git command. A master at the workspace root remains selectable when
  `--wiki-dir WORKSPACE/wiki` is used because that selector retains the parent
  workspace as the confinement root.
- An absent mirror is an evaluated non-identity and returns exit 1. An absent
  master has no reference bytes, is always `cannot_evaluate`, and returns exit
  2; no allowance flag preserves the former fail-open behavior.
- A mutation-proved Rulefloor invariant raises the executable floor from 30 to
  31.

## v0.4.4 — 2026-09-05

### Added

- `slice check --log-dir RELATIVE_PATH` optionally combines the automatically
  discovered frozen log file with regular files recursively found under an
  explicit repository-relative directory. Sources are ordered with the log file
  first and directory files in bytewise filename order. Existing source names
  must remain an exact prefix, and every existing file's headings must retain
  their prior sequence as a prefix, so lexically earlier additions, removals,
  reordering, and mid-file heading insertion fail. `--entries` is the exact sum
  of appended headings across both sources. The flag has no default because the
  caller owns the log-directory vocabulary; an explicitly missing, linked,
  unreadable, or overlapping directory is `cannot_evaluate`, exit 2. The verb
  retains its direct argument-vector Git reads and performs no shell invocation
  or Git mutation.
- A mutation-proved Rulefloor invariant, raising the executable floor from 29
  to 30.

## v0.4.3 — 2026-09-05

### Added

- `floor census --file CENSUS --repo REPO --bucket A,B,… --covered NAME
  --fence-heading TEXT [--marker C] [--count-sum RE] [--count-frozen RE]
  [--count-plain RE] [--allow-prefix TEXT] [--covers FILE] [--rulefloor PATH]`
  recounts a fenced completeness census. Against itself: an out-of-scope row
  naming a rule, a covered row without a citation, a covered row citing a rule
  the covers document does not know, a citation repeated in one row, a
  bucket-led line outside the fence, and a stated count the table
  contradicts. Against `rulefloor covers --json --repo REPO`, executed as an
  argument vector with no shell: a mutation-proven `file:Symbol` whose census
  rows never cite the rule, an unqualified covers entry, and a covers absence
  not on the frozen allowlist. Rulefloor absent or non-JSON is
  `cannot_evaluate`, exit 2, naming the resolved executable. The census
  vocabulary is the caller's: no bucket names, headings, markers or count
  phrases are built in. REFUSED, and reported in the document: whether a cited
  rule is ARMED — that needs RULE-FLOOR.md column parsing, Rulefloor's format.
- `rulefloor.covers.v1` joins the consumed Rulefloor input schemas.
- A mutation-proved Rulefloor invariant, raising the executable floor from 28
  to 29.

## v0.4.2 — 2026-09-05

### Added

- `recipe check --makefile PATH --target NAME [--expect-line S]...
  [--expect-file PATH] [--forbid-noop]` reads one target's recipe as text and
  refuses neutralizers: a line beginning `-` (before or after `@`) whose exit
  make ignores, a pipe (a `||` is not a pipe), a line ending `&`, a swallowed
  exit (`|| true`, `|| :`, `; true`, `exit 0`), and with `--forbid-noop` a
  command that is `true`, `:`, `echo` or `printf`. Expectations are equality,
  byte for byte, on the physical line without its leading tab — anything
  appended fails. Measured cause: macOS make 3.81 silently ignores
  `.SHELLFLAGS := -o pipefail -c`, so a static rule is the only guard there.
- `ledger census --file MD --dir [LABEL=]PATH... --tree [LABEL=]PATH... --ext E`
  recounts a markdown table against itself and against disk: per-mark counts
  and sums against the totals rows (`LABEL-MARK`, `LABEL-total`), the total row
  against every row summed, unique subjects, a present-marked row existing at
  exactly the stated size, a RETIRED row absent, and every file or subdirectory
  under a source rowed. Measured cause: a row added with the totals untouched
  passed every gate a workspace had.
- Two mutation-proved Rulefloor invariants, raising the executable floor from
  26 to 28.

## v0.4.1 — 2026-09-05

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
