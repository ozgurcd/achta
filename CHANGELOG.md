# Changelog

## Unreleased

- Made each v2 named-script replacement claim cite the exact caller vocabulary
  and require that citation to match its digest-pinned replay record.
- Vendored the v0.5.6 replay and recorded `declared-route check` as replacing
  `wiki/tools/rulefloor-install-gate.sh` under declared-route-v2 after all 10
  selftest fixtures agreed, including the document-wide duplicate in fixture
  07. Raised the mutation-proved Rulefloor from 46 to 47.
- Corrected the README's counted-target vocabulary to count-v2 so date digits
  touching a dash cannot become claim counts, then recorded `count check` as
  replacing `wiki/tools/count-claim-check.sh` after all 17 fixture exits agreed.
  Recorded that fixture g reports target 22 instead of the script's 4 despite
  both exiting 1, and raised the floor from 47 to 48.

## v0.5.6 — 2026-09-06

- Added `release-notes-check` to `make verify` and made both release workflow
  stages invoke that same target, eliminating the local/CI extractor gap.
- Defined one release-note grammar: an optional `## Unreleased` section must
  contain substantive content, while an empty section is rejected. Raised the
  mutation-proved Rulefloor from 45 to 46.

## v0.5.5 — 2026-09-06

- Retracted the `count check` and `declared-route check` named-script
  replacement claims to `candidate`: their v0.5.4 manifest rows contradict the
  replay document they cite, so neither claim currently has checkable parity
  evidence.
- Added `achta.replacement-claims.v2` and digest-pinned,
  workspace-confined `achta.replacement-replay.v1` evidence. `replacement
  check` now opens the cited artifact and reconciles every fixture row as a
  closed set; the frozen v0.5.4 claims fail on all six contradictions.
- Required annotated release tags in the checked workflow; lightweight tags
  are refused before validation or publication. Raised the mutation-proved
  Rulefloor from 43 to 45 for replay evidence and tag type.

## v0.5.4 — 2026-09-06

- Closed all five measured `declared-route check` replay differences: comment-
  only workflows are evaluated as zero live scalars, and a designated-route
  workflow must contain the caller-named required key exactly once across the
  YAML document as well as once in the caller-selected scope.
- Recorded `declared-route check` as replacing
  `wiki/tools/rulefloor-install-gate.sh` after all 10 script selftest fixtures
  agreed under the contract-complete caller vocabulary. Raised the executable
  Rulefloor from 40 to 42 with separate mutation proofs.
- Added repeatable caller-owned `--claim-target-pattern` and
  `--claim-assertion-pattern` composition to `count check`. A paragraph with
  both selects its smallest positive counted target for citation enforcement.
- Recorded `count check` as replacing `wiki/tools/count-claim-check.sh` after
  all 17 selftest fixtures agreed, and raised the floor from 42 to 43 with a
  separate mutation proof.

## v0.5.3 — 2026-09-06

- Added `replacement check` and the canonical
  `achta.replacement-claims.v1` manifest. Caller-designated named-script
  replacement or retirement claims now require citations to the script's own
  selftest fixtures and a recorded dual replay, with matching script and verb
  exit codes for every fixture.
- Kept replacement vocabulary and prose interpretation outside Achta:
  repeatable `--claim-status` flags have no default, candidate entries make no
  claim, and citations remain explicit attestations rather than executed proof.
- Made `make verify` enforce the manifest, froze the three uncited v0.5.1
  replacement claims as a mandatory red fixture, and raised the executable
  Rulefloor from 39 to 40 with a mutation proof.

## v0.5.2 — 2026-09-06

- Closed the `count check` retirement gaps with additive repeatable claim
  patterns, caller-patterned claim/proof comparison, last-matching-section
  scope, and next-paragraph exemptions. Achta still refuses to infer claims or
  proof meaning; all vocabulary has no default.
- Extended `declared-route check` with deterministic direct workflow
  directories and opt-in `--route-cardinality per-file-any`. Zero or many
  routes are allowed independently per file while bans remain mandatory for
  every selected workflow and every route-using file still needs one scoped
  declaration.
- Added `mirror check --digest-name NAME` to select an exact digest record,
  including caller-owned path-shaped names. Basename selection remains the
  compatibility default and no normalization is introduced.
- Raised the executable Rulefloor from 36 to 39 with separate mutation proofs
  for count retirement parity, per-file route cardinality, and named digest
  record selection.

## v0.5.1 — 2026-09-06

- Added `declared-route check --file YAML... --route-pattern REGEX...
  --ban-pattern REGEX... --required-route-pattern REGEX --required-key KEY
  --required-scope JSON_POINTER`.
  Each selected workflow must contain exactly one configured route alternative,
  no configured banned scalar text, and, when the designated route is selected,
  exactly one required key in the caller-selected YAML mapping.
- YAML scope is resolved from a syntax tree, so a job-level key cannot satisfy
  a workflow-level `/env` requirement. YAML comments and comment-only lines
  inside scalar command blocks do not count.
- The command is read-only, workspace-confined, invokes neither shell nor Git,
  and uses the stable `achta.declared-route-check.v1` 0/1/2 contract.
- Added the narrowly scoped stable `go.yaml.in/yaml/v3` parser dependency and
  raised the executable Rulefloor from 35 to 36 with a mutation proof for true
  YAML scope.

- Added `count check --file MD... [--dir DIR]...` with caller-supplied regular
  expressions for stated totals, breakdown parts, count claims, and citations.
  Breakdown parts must sum to their stated total; each configured claim must
  have at least its count of distinct exact citation matches in the same
  blank-line-delimited paragraph.
- Count-bearing patterns expose exactly one named `count` capture. Decimal
  counts are intrinsic; caller vocabulary such as number words requires an
  explicit `--count-alias TOKEN=N`. Achta refuses to infer claims from prose or
  decide whether a matched citation proves a disposition.
- Explicit files are followed by explicitly selected directories; each
  directory contributes only its direct Markdown files in bytewise filename
  order. The command is read-only, workspace-confined, and uses the stable
  `achta.count-check.v1` interface with the 0/1/2 exit contract.
- Raised the executable Rulefloor from 34 to 35 with a mutation proof for
  breakdown and citation-count reconciliation.

- Added optional `mirror check --digest FILE` verification. Achta reads
  canonical lowercase `sha256  basename` records, selects exactly one record
  by the master's exact basename, and compares the recorded digest with the
  SHA-256 of the master's unnormalized bytes. Mirrors remain optional when a
  digest is supplied.
- Digest agreement reports the selected file, line, basename, recorded digest,
  and computed master digest. Disagreement is exit 1; an absent digest file,
  malformed record, missing basename, or duplicate applicable basename is
  `cannot_evaluate`, exit 2.
- Raised the executable Rulefloor from 33 to 34 with a mutation proof for
  recorded-digest selection and exact comparison.

## v0.5.0 — 2026-09-05

- Established the complete conversion-batch baseline with `recipe check`,
  `ledger census`, `floor census`, `mirror check`, `parts lock`, `parts verify`,
  and `ledger rows` available together for caller migrations.

- Added `parts lock --dir DIR --lock FILE [--bump]` and `parts verify --dir
  DIR --lock FILE`: Achta owns a strict `achta.parts-lock.v1` artifact, hashes
  direct `.txt` parts in bytewise filename order, reports its `VERSION`, and
  fails on an edited, absent, or unlocked part.
- Kept canonical-document heading references outside Achta because those names
  and their meaning belong to the caller.
- Raised the executable Rulefloor from 31 to 32 with a mutation proof for
  exact digest and closed-set verification.
- Added `ledger rows` with caller-supplied cell indexes and exact state,
  completion, exemption, condition, and quote markers. It reports open rows
  that explicitly claim completion and closed conditional rows that do not
  repeat a quoted string verbatim.
- Refused natural-language completion inference and condition-satisfaction
  judgement; the command evaluates only the literals named at its call site.
- Raised the executable Rulefloor from 32 to 33 with a mutation proof for the
  ledger-row structural checks.

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
