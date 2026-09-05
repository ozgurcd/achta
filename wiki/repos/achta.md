---
title: achta
category: repo
status: authoritative
sources: PROJECT_SPEC.md, README.md, Makefile, RULE-FLOOR.md, RELEASE_NOTES.md
updated: 2026-09-05
verified: 2026-09-05
co_versioned: true
---

# Achta

Achta is the private, fail-closed Go CLI for deterministic workspace
bookkeeping. It edits only explicitly selected canonical artifacts and does not
infer prose, execute generic gates, fetch, or mutate Git.

## Canonical facts

- Current release line: v0.4.0.
- Module: `github.com/ozgurcd/achta`; Go 1.27.1; `CGO_ENABLED=0` release builds.
- Stable command families cover version/capabilities, wiki pin/status/derive,
  decision insertion, witness recording/checking, reachability, toolchain
  parity, amendment workflows, and landed-slice checks.
- Private GitHub releases publish checksummed macOS and Linux archives for
  amd64 and arm64. The public Homebrew tap contains metadata only; downloading
  private assets requires authorized GitHub access.
- Achta-specific wiki material is co-versioned under `wiki/` in this
  repository and is not duplicated in the parent workspace wiki.

## Verification surface

- `make verify` is canonical: format, build, unit tests, vet, Staticcheck,
  govulncheck, module tidiness, precise Gograph, Rulefloor execute mode, and the
  repository-owned wiki check.
- `make test-fuzz` covers the five bounded parser families named by the project
  specification.
- `RULE-FLOOR.md` carries 23 armed, mutation-proved invariants on the current
  v0.4.0 line.

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
