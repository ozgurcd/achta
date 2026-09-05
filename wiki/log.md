# Achta Wiki Log

## [2026-09-04] release | achta-v0.2.0

Achta `3c4fc7a` completes the v0.2.0 native workspace workflows, expands the
repository floor to 13 armed mutation-proved rules, and adds authenticated
Homebrew Cask publication after private GitHub releases.


## [2026-09-04] fix | achta-v0.2.1

Achta `9349056` replaces private release browser URLs with authenticated GitHub
asset API URLs before publishing its Homebrew Cask, arms the invariant at a
14-rule floor, and rejects invalid derived-marker ordering found by fuzzing.


## [2026-09-04] fix | achta-v0.2.2-ci

Achta `af08a03` installs pinned Gograph v1.6.10 in release CI so Rulefloor can
evaluate the authenticated asset invariant's exact structural reach before
publishing v0.2.2.


## [2026-09-04] fix | achta-v0.2.3-ci

Achta `7b0e9ff` corrects the pinned Gograph installer to
`github.com/ozgurcd/gograph/cmd/gograph@v1.6.10`, matching the executable
package validated in isolation before the v0.2.3 tag.


## [2026-09-04] fix | achta-v0.2.4-ci

Achta `29e5c49` makes canonical verification build a current precise Gograph
index before Rulefloor exact-reach evaluation; an isolated clean-clone tool
installation plus `make verify` passed before tagging v0.2.4.


## [2026-09-04] release | achta-v0.2.4

Achta v0.2.4 published five private release assets and an authenticated
Homebrew Cask whose four API asset URLs and checksums match the GitHub release;
the release workflow completed successfully.


## [2026-09-04] fix | achta-v0.2.5-homebrew-evaluation

Achta `a23e0c9` makes Cask evaluation non-throwing when
`HOMEBREW_GITHUB_API_TOKEN` is absent while retaining the bearer header for
authenticated private release downloads; the Rulefloor rises to 15 armed,
mutation-proved invariants.


## [2026-09-04] release | achta-v0.2.5

Achta v0.2.5 published five private release assets and a verified Cask at
`ozgurcd/homebrew-tap`; token-less Ruby evaluation succeeds and all four
download checksums match the release manifest.


## [2026-09-04] tooling-adoption | achta-wiki-close

The wiki half of the close ran on hand-written heredocs. Under the owner ruling
that Achta v0.2.5 is the tool, each step was measured against its Achta command
on the same tree before anything moved.

Freshness agreed exactly: eleven fresh, none behind, none unpinned, none
unreadable, from both. Derive agreed: six pages scanned, none changed, from
both. Unpushed agreed on the counts and the commits, and Achta additionally
reports a behind counter ours never had.

The pin moved, and the measurement is the reason. The fear was that a canonical
writer would flatten the narrative these pin lines carry, since that line is
the durable record of what a slice proved. It does not. Run against a page whose
pin is a full paragraph, Achta changed exactly one token — the short sha to the
full forty-character one — and left every word intact. It is also stricter than
the heredoc it replaces: it refuses a sha that is not the repository HEAD, and
it requires an explicit attestation flag, so a pin cannot move just because a
command knew the sha. Our freshness gate accepts its form, Achta's agrees, and
a second run reports unchanged.

What stayed, stayed for reasons rather than inertia. The narrative sentence has
no Achta equivalent and is now written with the Edit tool instead of a heredoc.
Ledger rows, log entries and queue rows have no Achta command at all. The
freshness and derive scripts are not merely called but WIRED as named gate
targets, one of them inside another repo's verify, and replacing a gate is not
adopting a tool.

Deleted: the unpushed check. Its own header said it was wired into nothing, and
that was true — zero callers and no rule binding it. One stale mention survives
in a comment in the idp-oss Makefile, named rather than fixed because that repo
is out of scope here and a comment is not worth an appliance mint.

Two refusals, both recorded as findings rather than worked around. Achta
rejects a body file outside the workspace, which is a containment rule worth
having and was satisfied by moving the file inside. And it resolves that path
from the workspace ROOT, not the wiki, which cost a second attempt. Witness and
amendments were left alone by ruling: Achta's record carries no cross-repo pin
and our ledger rules are armed to our own tools, so adopting either would drop
the two-repo tie or strand a rule. This decision is P-058, and Achta wrote it.


## [2026-09-04] release | achta-v0.3.0

Achta `2a545cd` publishes v0.3.0 with cross-repository witness pins,
reachability-aware staleness, CI provenance, earned-cycle refusal, and
declarative toolchain parity; the GitHub release, Homebrew Cask, and installed
binary were verified at v0.3.0.


## [2026-09-05] governance | repository-owned-wiki

Achta-specific current facts, decisions, history, and release records moved
from the parent workspace wiki into this co-versioned repository wiki. The
parent governance now routes repositories with owned wikis locally and forbids
duplicate central records. Two mutation-proved rules raise the Rulefloor from
20 to 22 for co-versioned freshness and repository-owned log precedence.


## [2026-09-05] release | achta-v0.4.0

Achta v0.4.0 adds the explicit `--wiki-dir PATH` selector for repository-owned
wiki authority, rejects conflicts with `--workspace`, and raises the executable
Rulefloor from 22 to 23 with a measured parser mutation proof. The release
workflow publishes the matching private archives and Homebrew Cask metadata.


## [2026-09-05] release-fix | v0.4.0-source-version

The first v0.4.0 tag run stopped before publication because the source version
and exact machine fixtures still reported v0.3.0. They now advance together to
v0.4.0 so the tag, release binary, and toolchain version agree.


## [2026-09-05] feature | wiki-check-selection

A caller measured against v0.4.0 that no command enforces pin freshness on a
dirty working tree: `wiki check` couples freshness with derive, whose block
records `Working tree vs HEAD`, and `wiki freshness` only reports. `wiki check
--only NAME[,NAME...]` now evaluates and reports exactly the selected checks in
canonical order under the unchanged exit contract; an empty, unknown, or
repeated name is exit 2 with nothing evaluated, and naming every check is the
composed check. Selection is a set, not a per-check exit code (P-061).
Independently, `achta.wiki-check.v1` carries `wiki_dir` and the text output
prints it first, so a pass against the wrong wiki is no longer silent. Both
behaviors are armed with mutation proofs (WIKI-CHECK-SELECTION-1,
WIKI-CHECK-WIKI-DIR-1), raising the executable floor from 23 to 25. The
decisions register gained its missing `## Decisions` section because `decision
add` refused the page without one.


## [2026-09-05] release | achta-v0.4.1

The source version, exact machine contracts, human goldens, current release
facts, changelog, and release notes advance together to v0.4.1. The wiki-check
selection and resolved-wiki reporting entries move unchanged in substance from
Unreleased to the exact v0.4.1 heading. The executable Rulefloor remains 25.


## [2026-09-05] release-fix | structured-homebrew-postflight

Achta's cask publisher now fail-closed converts GoReleaser's deprecated
`postflight` hook into structured `postflight_steps` while retaining private
asset authentication and macOS quarantine handling. A mutation-proved rule
raises the executable Rulefloor from 25 to 26.


## [2026-09-05] feature | recipe-check-and-ledger-census

Owner ruling: new workspace tooling is an Achta command, never a new script
(P-062). Two identuum wiki gates were shell only because Achta had no verb for
them. `recipe check` reads one Makefile target's recipe as text and refuses
neutralizers — a `-` prefix whose exit make ignores, a pipe, a trailing `&`, a
swallowed exit, and with `--forbid-noop` a command that cannot fail — and
matches `--expect-line` / `--expect-file` by byte-for-byte equality, because a
prefix match let anything appended pass. The measured cause behind it is
macOS make 3.81 silently ignoring `.SHELLFLAGS := -o pipefail -c`. `ledger
census` recounts a markdown ledger table against its totals rows and against
files (`--dir`) or directories (`--tree --ext`) on disk: a row added with the
totals untouched, a stale size, a RETIRED subject still present, or an
unrowed entry all fail; the measured cause is exactly that first case passing
every gate a workspace had. Both are read-only, workspace-confined, and exit 2
when they cannot evaluate. Refused as outside the boundary: a rule that every
wiki script implementing `--selftest` has a check entry (it inspects other
scripts for a workspace convention) and gate-witness entry semantics (an
identuum Makefile convention) — `--expect-file` covers the latter byte-exactly.
RECIPE-CHECK-1 and LEDGER-CENSUS-1 are armed with mutation proofs; the
executable Rulefloor rises from 26 to 28. Both machine interfaces have exact
fixtures and the capabilities documents advertise them.


## [2026-09-05] release | achta-v0.4.2

The source version, exact machine contracts, human goldens, current release
facts, changelog, and release notes advance together to v0.4.2. The `recipe
check` and `ledger census` entries move unchanged in substance from Unreleased
to the exact v0.4.2 heading. The release workflow will publish the matching
Homebrew cask after the tag passes its independent verification job. The
executable Rulefloor remains 28.


## [2026-09-05] feature | floor-census

`floor census` gives the identuum wiki's last non-witness shell gate a verb.
It recounts a fenced completeness census against itself — an out-of-scope row
naming a rule, a covered row with no citation or citing a rule the covers
document does not know, a citation repeated in one row, a bucket-led line
outside the fence, a stated count the table contradicts — and against
`rulefloor covers --json`, executed as an argument vector the way `amendments
reconcile` executes Rulefloor: a mutation-proven pair no census row cites, an
unqualified covers entry, an absence not on the frozen allowlist. Rulefloor
absent is `cannot_evaluate`, exit 2, naming the resolved executable;
`rulefloor.covers.v1` joins the consumed input schemas.

The vocabulary decision (P-063): every piece of the census's vocabulary is a
flag — buckets, covered bucket, fence heading, marker, count patterns,
allowlist prefix — with no defaults; a canonical Achta-owned format was
rejected because it would make Achta the owner of a taxonomy it does not judge
and force a 1285-row rewrite; hardcoding identuum's tokens was rejected as the
same ownership mistake. Refused, and reported in the document: whether a cited
rule is ARMED, which lives in RULE-FLOOR.md's columns — Rulefloor's format.

Measured live against identuum's census with the real rulefloor: 1285 rows
(COVERED 227, OOS-L 9, OOS-O 6, OOS-P 953, OOS-T 90), frozen 26, plain 201,
covers 252 rules with 157 mapped, 45 absences all allowlisted, zero
violations — the same numbers the shell gate prints. FLOOR-CENSUS-1 is armed
with a mutation proof; the executable Rulefloor rises from 28 to 29.


## [2026-09-05] release | achta-v0.4.3

The source version, exact machine contracts, human goldens, current release
facts, changelog, and release notes advance together to v0.4.3. The `floor
census` entries move unchanged in substance from Unreleased to the exact
v0.4.3 heading; the executable Rulefloor remains 29.

Owner standing rule P-064 now makes the release boundary explicit: a release
is not complete at a local commit or green build. Source push, tag and private
release publication, Homebrew tap commit and push, and installation through
Homebrew are all required, with the installed binary's exact version and new
capability as the final proof.


## [2026-09-05] feature | slice-log-directory

`slice check --log-dir RELATIVE_PATH` combines the discovered frozen log file
with regular files recursively read from an explicit repository-relative
directory. The file is first and directory paths follow in deterministic
bytewise filename order; old filenames must remain a prefix, and old headings
must remain a prefix inside each existing file. A lexically earlier file,
removal or reordering, or a heading inserted before an old heading therefore
fails. `--entries` counts appends across both sources. Explicit missing, linked,
unreadable, or overlapping directories are `cannot_evaluate`, exit 2; only no
log file plus no flag skips. The verb keeps direct argument-vector Git reads and
does not invoke a shell or mutate Git.

Decision P-065 keeps the directory name caller-owned, parallel to P-063's census
vocabulary: auto-detecting `log/` would make Achta own an Identuum migration
convention and could silently widen another workspace's evidence. The new
SLICE-LOG-DIRECTORY-1 invariant is mutation-proved: disabling the filename-order
guard made a lexically earlier file pass, so the bound test failed; restoring
the guard returned it green. The executable Rulefloor rises from 29 to 30.


## [2026-09-05] release | achta-v0.4.4

The source version, exact machine contracts, human goldens, current release
facts, changelog, and release notes advance together to v0.4.4. The `slice check
--log-dir` entries move unchanged in substance from Unreleased to the exact
v0.4.4 heading; the executable Rulefloor remains 30. A lexically later tracked
directory fixture joins this release commit, allowing the installed Homebrew
binary to prove that it counts the release's frozen-file heading plus the
directory heading and fails when an additional expected heading is nowhere.
