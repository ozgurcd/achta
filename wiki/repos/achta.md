---
title: achta
category: repo
status: authoritative
sources: PROJECT_SPEC.md, README.md, Makefile, RULE-FLOOR.md, RELEASE_NOTES.md
updated: 2026-09-06
verified: 2026-09-06
co_versioned: true
---

# Achta

Achta is the private, fail-closed Go CLI for deterministic workspace
bookkeeping. It edits only explicitly selected canonical artifacts and does not
infer prose, execute generic gates, fetch, or mutate Git.

## Canonical facts

- Current release line: v0.5.6.
- Module: `github.com/ozgurcd/achta`; Go 1.27.1; `CGO_ENABLED=0` release builds.
- Stable command families cover version/capabilities, wiki pin/status/derive,
  decision insertion, witness recording/checking, reachability, toolchain
  parity, amendment workflows, and landed-slice checks.
- Private GitHub releases publish checksummed macOS and Linux archives for
  amd64 and arm64. The public Homebrew tap contains metadata only; downloading
  private assets requires authorized GitHub access.
- A release is complete only after source push, tag and release publication,
  Homebrew cask commit and push, and installation through Homebrew prove the
  installed version and newly shipped capability. Release tags are annotated;
  the workflow rejects lightweight tags before publication.
- Achta-specific wiki material is co-versioned under `wiki/` in this
  repository and is not duplicated in the parent workspace wiki.
- `wiki check --only NAME[,NAME...]` evaluates and reports exactly the selected
  checks under the unchanged exit contract; an empty, unknown, or repeated name
  is exit 2 with nothing evaluated. `achta.wiki-check.v1` names the resolved
  `wiki_dir`.
- `recipe check` refuses neutralized make recipe lines (a `-` prefix, a pipe, a
  trailing `&`, a swallowed exit, with `--forbid-noop` a no-op command) and
  matches `--expect-line` / `--expect-file` byte for byte. `ledger census`
  recounts a markdown ledger table against its totals and against files
  (`--dir`) or directories (`--tree --ext`) on disk. Both read only, both
  workspace-confined, both `cannot_evaluate` on unreadable input.
- `floor census` recounts a fenced completeness census against itself and
  against `rulefloor covers --json` (executed as an argument vector); the
  census vocabulary is the caller's, every piece a flag, nothing built in. It
  refuses to judge whether a cited rule is armed and says so in its document.
- `mirror check` compares exact SHA-256 bytes from one master to explicit
  mirrors and, optionally, to the unique canonical digest record selected by
  exact master basename or explicit `--digest-name`. It reports both digest
  hexes and the selected record; disagreements fail while absent or ambiguous
  evidence is `cannot_evaluate`.
- `parts lock` writes the Achta-owned `achta.parts-lock.v1` artifact for a
  direct `.txt` set, preserving or explicitly bumping `VERSION`; `parts verify`
  names edited, absent, and unlocked parts under the 0/1/2 exit contract. Live
  canonical-document heading references remain caller-owned.
- `ledger rows` applies caller-supplied cell indexes and exact state, identity,
  completion, exemption, condition, and quote markers. It catches open rows
  with explicit completion literals and closed conditional rows without a
  verbatim repeated quote, while refusing prose-meaning and satisfaction
  judgements.
- `count check` reconciles caller-patterned breakdown sums, repeatable claim
  and citation counts, caller-selected last-section scope and exemptions, and
  bounded claim/proof counts. Caller-patterned counted targets and assertions
  can compose within one paragraph using the smallest positive target count.
  Its named-script disposition is `candidate`: the prior 17-of-17 claim was
  retracted because the cited replay records fixture `g` as script 1 / verb 0.
  Achta still refuses prose-claim and proof meaning.
- `declared-route check` parses explicit files and deterministic direct workflow
  directories. Its compatibility default requires one route per file; opt-in
  per-file-any permits zero or many while every file remains ban-scanned and
  every designated-route user needs the caller-named key exactly once in the
  selected scope and once across the document. Comment-only files are evaluated
  as zero live scalars. It replaces `wiki/tools/rulefloor-install-gate.sh`
  under the README's declared-route-v2 four-ban vocabulary: the digest-pinned
  v0.5.6 replay records all 10 selftest fixtures agreeing, including the
  document-wide duplicate in fixture 07.
- `replacement check` validates the canonical `replacement-claims.json` gate.
  A v2 replacement or retirement status requires a selftest citation plus a
  matching exact-vocabulary citation and a workspace-confined, SHA-256-pinned
  `achta.replacement-replay.v1` artifact; verb/script identity, citations, and
  the complete fixture set must agree row-for-row. Legacy v1 claims are
  attestations and candidates make no claim.

## Verification surface

- `make verify` is canonical: release-note extraction, format, build, unit
  tests, vet, Staticcheck, govulncheck, module tidiness, precise Gograph,
  Rulefloor execute mode, and the repository-owned wiki check. Both release
  workflow stages invoke the same `release-notes-check` target; an optional
  `## Unreleased` section is rejected when empty.
- `make test-fuzz` covers the five bounded parser families named by the project
  specification.
- `slice check` can combine its discovered log file with an explicit,
  caller-named directory in deterministic filename order while preserving
  per-file append-only heading history.
- `RULE-FLOOR.md` carries 47 armed, mutation-proved invariants on the current
  source line, including digest-pinned closed-set replay evidence and annotated
  release tags.

## Known limitations

- Achta establishes structural consistency and freshness, not prose truth or
  author honesty.
- It does not run arbitrary commands, create product commits, fetch remotes, or
  replace product-specific analyzers.
- Homebrew installation requires `HOMEBREW_GITHUB_API_TOKEN` with read access
  to the private repository.
- Co-versioned pages have no external SHA pin; the containing checkout is their
  version boundary, while `verified:` records the last claim review.

## Commit ledger

| Date | Commit | Slice |
|---|---|---|
| 2026-09-04 | `3c4fc7a` | Complete v0.2.0 native workflows, release contracts, executable Rulefloor floor, and authenticated Homebrew Cask publication. |
| 2026-09-04 | `9349056` | Route private Homebrew downloads through authenticated GitHub asset API URLs and reject invalid derived-marker ordering. |
| 2026-09-04 | `af08a03` | Install pinned Gograph in release CI so exact Rulefloor structural reach can be evaluated before publication. |
| 2026-09-04 | `7b0e9ff` | Correct the pinned Gograph installer to the module's executable `cmd/gograph` package. |
| 2026-09-04 | `29e5c49` | Build a current precise graph in canonical verification before Rulefloor exact-reach evaluation. |
| 2026-09-04 | `a23e0c9` | Make token-less Homebrew Cask evaluation safe while retaining authenticated private asset downloads. |
| 2026-09-04 | `2a545cd` | Add v0.3.0 cross-repository witness pins, reachability-aware staleness, CI provenance, earned-cycle refusal, and toolchain parity. |
| 2026-09-05 | `co-versioned` | Move all Achta-specific wiki information into this repository and enforce local wiki freshness and history selection with two new armed invariants. |
| 2026-09-05 | `co-versioned` | Add explicit, fail-closed repository wiki selection and prepare the v0.4.0 release and Homebrew update. |
| 2026-09-05 | `co-versioned` | Advance the source and exact machine version contracts to v0.4.0 after the release gate rejected the stale v0.3.0 pin. |
| 2026-09-05 | `co-versioned` | Add fail-closed `wiki check --only` selection and the resolved `wiki_dir` field, both mutation-proved, raising the executable floor from 23 to 25. |
| 2026-09-05 | `co-versioned` | Advance the release surfaces to v0.4.1 and move the wiki-check selection notes under their exact release heading without changing the 25-rule floor. |
| 2026-09-05 | `co-versioned` | Publish only structured Homebrew postflight steps while preserving authenticated private-asset rewriting and macOS quarantine handling. |
| 2026-09-05 | `co-versioned` | Add `recipe check` (byte-exact recipe expectations, neutralizer refusal) and `ledger census` (a ledger table recounted against its totals and disk), both mutation-proved, raising the executable floor from 26 to 28. |
| 2026-09-05 | `co-versioned` | Advance source, exact version fixtures, release notes, current wiki facts, and Homebrew publication metadata together for v0.4.2 without changing the 28-rule floor. |
| 2026-09-05 | `co-versioned` | Add `floor census` (a fenced completeness census recounted against itself and rulefloor covers, vocabulary by flags, armed state refused), mutation-proved, raising the executable floor from 28 to 29. |
| 2026-09-05 | `co-versioned` | Prepare v0.4.3 and record that every release completes only after source, tag, release, and tap publication culminate in an installed Homebrew binary proving its version and new capability. |
| 2026-09-05 | `co-versioned` | Add explicit caller-owned `slice check --log-dir`, deterministic filename ordering, and strict per-file append-only headings, mutation-proved, raising the executable floor from 29 to 30. |
| 2026-09-05 | `co-versioned` | Advance source, exact machine and human fixtures, current wiki facts, changelog, and release notes together for v0.4.4, retaining the 30-rule floor and adding the installed directory-proof fixture. |
| 2026-09-05 | `co-versioned` | Add workspace-confined, exact-byte `mirror check`, make absent master fail closed, add its stable machine contract and mutation proof, and raise the executable floor from 30 to 31. |
| 2026-09-05 | `co-versioned` | Advance source, exact machine and human fixtures, current wiki facts, changelog, and release notes together for v0.4.5, retaining the 31-rule floor and adding installed exact-byte proof fixtures. |
| 2026-09-05 | `co-versioned` | Add Achta-owned `parts lock` and exact closed-set `parts verify`, keep section-reference semantics caller-owned, and raise the mutation-proved floor from 31 to 32 without releasing. |
| 2026-09-05 | `co-versioned` | Add caller-shaped `ledger rows`, enforce exact completion and closure-quote structure while refusing prose meaning, and raise the mutation-proved floor from 32 to 33 without releasing. |
| 2026-09-05 | `co-versioned` | Release the complete conversion batch as v0.5.0, advancing source and exact fixtures while retaining the 33-rule floor for caller migration. |
| 2026-09-06 | `co-versioned` | Add exact recorded-master-digest verification to `mirror check`, refuse absent or ambiguous evidence, and raise the mutation-proved floor from 33 to 34 without releasing. |
| 2026-09-06 | `co-versioned` | Add caller-patterned breakdown and citation-count reconciliation, refuse implicit prose and proof meaning, and raise the mutation-proved floor from 34 to 35 without releasing. |
| 2026-09-06 | `co-versioned` | Add true-YAML `declared-route check` with caller-owned route, ban, key, and scope flags, and raise the mutation-proved floor from 35 to 36 without releasing. |
| 2026-09-06 | `co-versioned` | Release the second additive conversion batch as v0.5.1, advance exact version fixtures, record Achta's first external Go module dependency, and retain the 36-rule floor. |
| 2026-09-06 | `co-versioned` | Close the three measured script-retirement gaps, add exact caller-shaped count, route-cardinality, and digest-record controls, release v0.5.2, and raise the mutation-proved floor from 36 to 39. |
| 2026-09-06 | `co-versioned` | Make principle 10 executable with a structured named-script replacement-claim gate, freeze the uncited v0.5.1 state as red, release v0.5.3, and raise the mutation-proved floor from 39 to 40. |
| 2026-09-06 | `co-versioned` | Close all five measured declared-route replay differences, record 10-of-10 script parity and the mirror refusal boundary, flip declared-route to replaces, and raise the mutation-proved floor from 40 to 42. |
| 2026-09-06 | `co-versioned` | Add generic counted-target plus assertion paragraph composition, close fixture g, record 17-of-17 script parity, flip count check to replaces, and raise the mutation-proved floor from 42 to 43. |
| 2026-09-06 | `co-versioned` | Advance source and exact version fixtures, fold the two parity entries under v0.5.4, and prepare the patch release without changing the 43-rule floor. |
| 2026-09-06 | `co-versioned` | Retract the count and declared-route named-script replacement claims to candidate because their cited replay contradicts six manifest rows. |
| 2026-09-06 | `co-versioned` | Make replacement citations checkable through digest-pinned vendored replay evidence, reject all six v0.5.4 contradictions, require annotated release tags, and raise the floor from 43 to 45. |
| 2026-09-06 | `co-versioned` | Release v0.5.5 with checked replay evidence, retracted replacement claims, annotated-tag enforcement, and the 45-rule floor. |
| 2026-09-06 | `co-versioned` | Fix forward to v0.5.6 by making local and release validation share the release-note extractor, forbidding empty Unreleased sections, and raising the floor from 45 to 46. |
| 2026-09-06 | `co-versioned` | Require exact caller-vocabulary citations in replacement claims, vendor the v0.5.6 replay, flip only declared-route under its documented 10-of-10 vocabulary, and raise the floor from 46 to 47. |
