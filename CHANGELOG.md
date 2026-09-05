# Changelog

## Unreleased

- Added `parts lock --dir DIR --lock FILE [--bump]` and `parts verify --dir
  DIR --lock FILE`: Achta owns a strict `achta.parts-lock.v1` artifact, hashes
  direct `.txt` parts in bytewise filename order, reports its `VERSION`, and
  fails on an edited, absent, or unlocked part.
- Kept canonical-document heading references outside Achta because those names
  and their meaning belong to the caller.
- Raised the executable Rulefloor from 31 to 32 with a mutation proof for
  exact digest and closed-set verification.

## v0.4.5 — 2026-09-05

- Added `mirror check --master PATH --mirror PATH...`: SHA-256 is computed over
  the exact master and mirror bytes, with per-mirror `match`, `differs`, or
  `absent` results and both digests for comparable files. No normalization is
  permitted.
- An absent mirror is an evaluated failure; an absent master is always
  `cannot_evaluate`, exit 2. Workspace-root masters work when `--wiki-dir`
  selects `WORKSPACE/wiki`.
- Raised the executable Rulefloor from 30 to 31 with a mutation proof for exact
  digest equality.

## v0.4.4 — 2026-09-05

- Added opt-in `slice check --log-dir RELATIVE_PATH`. The frozen log file is
  followed by recursively discovered regular directory files in deterministic
  filename order; old filenames and each file's old headings remain strict
  append-only prefixes, and `--entries` counts new headings across both.
- Kept the log-directory layout caller-owned: there is no conventional default,
  and an explicitly missing, linked, or overlapping directory is
  `cannot_evaluate`.
- Raised the executable Rulefloor from 29 to 30 with a mutation proof for
  filename-order enforcement.

## v0.4.3 — 2026-09-05

- Added `floor census`: a fenced completeness census is recounted against
  itself (citations, repeated tokens, bucket-led prose leaks, stated counts)
  and against `rulefloor covers --json` (uncited mutation-proven pairs, new
  absences vs a frozen allowlist). The census vocabulary is the caller's —
  `--bucket`, `--covered`, `--fence-heading`, `--marker`, `--count-*`,
  `--allow-prefix` — never a built-in default. Whether a cited rule is ARMED
  is refused and reported as refused.
- Added `rulefloor.covers.v1` to the consumed Rulefloor input schemas.
- Raised the executable Rulefloor from 28 to 29 with a mutation proof for
  stated-count recounting.

## v0.4.2 — 2026-09-05

- Added `recipe check`: a Makefile target's recipe is read as text and
  neutralizers are refused — a `-` prefix, a pipe, a trailing `&`, a swallowed
  exit, and with `--forbid-noop` a `true`/`:`/`echo`/`printf` command;
  `--expect-line` and `--expect-file` are byte-exact equality.
- Added `ledger census`: a markdown ledger table is recounted against its
  totals rows and against files (`--dir`) or directories (`--tree --ext`) on
  disk; a present row must exist at exactly its stated size, a RETIRED row must
  be gone, and every entry on disk must have a row.
- Raised the executable Rulefloor from 26 to 28 with mutation proofs for
  byte-exact recipe expectations and totals recounting.

## v0.4.1 — 2026-09-05

- Added `wiki check --only NAME[,NAME...]` to evaluate and report exactly the
  selected checks under the unchanged exit contract; empty, unknown, or
  repeated names are exit 2 with nothing evaluated.
- Added `wiki_dir` to `achta.wiki-check.v1` (and a text line) naming the wiki
  directory Achta resolved.
- Raised the executable Rulefloor from 23 to 25 with mutation proofs for check
  selection and resolved-wiki reporting.

## v0.4.0 — 2026-09-05

- Moved Achta-specific wiki ownership into the Achta repository.
- Added co-versioned repository-page freshness and owned-wiki log selection.
- Added fail-closed `--wiki-dir` selection for repository-owned wiki authority.
- Raised the executable Rulefloor from 20 to 23 with mutation proofs for both
  ownership behaviors and direct wiki selection.

## v0.3.0 — 2026-09-04

- Added parsed sibling witness pins, reachability-aware staleness, CI
  provenance, and record-only earned-cycle refusal.
- Added declarative workspace/CI version and script-digest parity.
- Kept witness command execution and vulnerability-fix policy outside Achta's
  product boundary.

## v0.2.5 — 2026-09-04

- Made token-less Homebrew Cask evaluation safe during `brew update` while
  retaining bearer authentication for private release downloads.
- Added a release contract that prevents strict installer-token lookup from
  returning.

## v0.2.4 — 2026-09-04

- Built a current precise Gograph index inside the canonical verification gate
  before Rulefloor exact-reach evaluation.
- Carried the unpublished authenticated private-asset and delimiter-ordering
  fixes forward.

## v0.2.3 — 2026-09-04

- Corrected the pinned Gograph installer to its `cmd/gograph` executable
  package and carried the unpublished corrective changes forward.
- The release workflow stopped before publication because its clean checkout
  had no precise graph for Rulefloor exact-reach evaluation.

## v0.2.2 — 2026-09-04

- Installed pinned Gograph in release CI so exact Rulefloor reach can be
  evaluated before publication.
- Carried the authenticated private-asset and delimiter-ordering fixes from the
  unpublished v0.2.1 attempt into the corrective release.
- The release workflow stopped before publication because the Gograph module
  root is not an installable command package.

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
