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


## [2026-09-05] feature | mirror-check

`mirror check --master PATH --mirror PATH...` replaces three repeated
master/copy comparisons with one read-only, workspace-confined surface. It
computes SHA-256 over exact bytes in caller order and reports `match`, `differs`
with both digests, or `absent` per mirror. No newline, whitespace, encoding, or
semantic normalization exists. With `--wiki-dir WORKSPACE/wiki`, the retained
workspace root permits a root-level master and wiki-contained mirrors.

Decision P-066 makes the old fail-open explicit and rejects it: an absent master
has no reference bytes and is always `cannot_evaluate`, exit 2, without an
allowance flag. An absent mirror is an evaluated non-identity, exit 1. The verb
invokes no shell or Git command and mutates nothing. MIRROR-BYTE-IDENTITY-1 is
armed after inversion of the digest-equality branch made identical inputs fail
both the domain and CLI tests; restoration returns green. The executable
Rulefloor rises from 30 to 31.


## [2026-09-05] release | achta-v0.4.5

The source version, exact machine and human fixtures, current wiki facts,
changelog, and release notes advance together to v0.4.5. The `mirror check`
entries move unchanged in substance from Unreleased to the exact v0.4.5
heading; the executable Rulefloor remains 31. The committed master, identical,
and one-byte-changed fixtures support final proof through the Homebrew-installed
binary after the authorized source, tag, release, and tap publication steps.


## [2026-09-05] feature | parts-lock-and-verify

`parts lock --dir DIR --lock FILE [--bump]` writes the canonical
`achta.parts-lock.v1` schema marker, `VERSION vN`, and one exact SHA-256 record
per direct `.txt` part in bytewise filename order. `parts verify` recomputes the
closed set, names edited, absent, and unlocked files, returns exit 1 for drift,
and preserves exit 2 for missing, malformed, unsafe, or concurrently changing
evidence. Neither command invokes a shell or Git.

P-067 records the ownership boundary: Achta owns the format it writes, while
the canonical document and heading vocabulary behind section references remain
the caller's semantic analyzer. PARTS-LOCK-1's mutation disabled unlocked-file
detection; the bound CLI test then observed an extra part passing at exit 0.
Restoration returned green and raised the executable floor from 31 to 32. The
source remains v0.4.5 and the changelog and release-note entries stay under
Unreleased; no release or remote action belongs to this slice.


## [2026-09-05] feature | ledger-row-rules

`ledger rows` reads caller-selected ID and prose cells using exact, required
open, closed, identity-end, completion, condition, and quote markers plus
optional exemption markers. An open row with an explicit completion marker and
no exemption fails. A closed conditional row fails unless the quote marker is
followed by a non-empty double-quoted string whose bytes occur elsewhere in the
same prose cell. Violations name line, rule, row identity, and text under
`achta.ledger-rows.v1`; invalid, unsafe, ambiguous, or vocabulary-free input is
`cannot_evaluate`, exit 2.

P-068 records the line Achta will not cross: no natural-language completion
inference, quote normalization, or claim that a repeated condition was actually
satisfied. The command invokes no shell or Git process and writes nothing.
LEDGER-ROWS-1 is armed after disabling the open-completion branch made the
real-ledger-shaped completion fixture pass at exit 0; the CLI and domain tests
both went red, and restoration returned them green. The executable floor rises
from 32 to 33. Version remains v0.4.5, entries stay under Unreleased, and no
remote or release action belongs to this slice.

## [2026-09-05] release | achta-v0.5.0

The source version, exact machine and human fixtures, current wiki facts,
changelog, and release notes advance together to v0.5.0. The two unreleased
conversion entries move unchanged in substance under the exact v0.5.0 heading,
and the complete caller-migration baseline names `recipe check`, `ledger
census`, `floor census`, `mirror check`, `parts lock`, `parts verify`, and
`ledger rows`. The executable Rulefloor remains 33.


## [2026-09-06] feature | mirror-recorded-digest

`mirror check --digest FILE` compares the SHA-256 of the master's exact bytes
with a canonical lowercase `sha256  name` record. It reports the digest file,
selected line and exact basename, recorded hex, and computed master hex. A
valid disagreement is exit 1; absent, malformed, or ambiguous digest evidence
is `cannot_evaluate`, exit 2. Mirrors are optional when the digest is present,
and the command remains workspace-confined, read-only, and free of shell or Git
invocation.

P-069 binds the record to the master's exact basename. Multi-record files are
accepted only when every line is canonical and exactly one record applies;
differently named or duplicate applicable records cannot evaluate. Achta
refuses case, whitespace, line-ending, path, and content normalization. The
new MIRROR-RECORDED-DIGEST-1 invariant is mutation-proved and raises the
executable Rulefloor from 33 to 34. The source remains v0.5.0, the release notes
stay under Unreleased, and no remote or release action belongs to this slice.

## [2026-09-06] feature | count-relationship-check

`count check` accepts explicit Markdown files and caller-named directories of
direct Markdown files. Caller-supplied total and part patterns drive
line-scoped breakdown arithmetic; caller-supplied claim and citation patterns
drive distinct exact citation counts within blank-line-delimited paragraphs.
Count-bearing patterns expose one named `count` capture, and explicit aliases
are the only non-decimal vocabulary.

P-070 accepts both structural halves while refusing the semantic leap: Achta
does not infer that unconfigured prose claims a fix or that a matched citation
proves a disposition. Bad sums and citation undercounts fail, missing or
ambiguous inputs cannot evaluate, and correct fixtures remain clean. The
CLAIM-COUNT-RELATIONSHIPS-1 mutation proof raises the executable Rulefloor from
34 to 35. The source remains v0.5.0, all release entries stay under Unreleased,
and no remote or release action belongs to this slice.

## [2026-09-06] feature | declared-route-check

`declared-route check` reads explicit workflow YAML files, counts exactly one
caller-patterned route alternative, rejects caller-patterned banned text, and
when the caller-designated route is selected requires an exact caller-named key
once in a caller-selected JSON-Pointer mapping scope. YAML comments and
comment-only scalar lines do not count.
Malformed, multi-document, absent, unsafe, or ambiguous input cannot evaluate.

P-071 accepts YAML syntax because mapping scope is portable structure, not
caller meaning. The stable `go.yaml.in/yaml/v3` node parser is the measured
correctness exception to the standard-library preference; line-level scope
inference and all default route vocabulary are refused. A mutation that let a
nested job key satisfy absent workflow `/env` made both domain and CLI tests
red because the job-level fixture passed at exit 0. Restoration returned green,
and DECLARED-ROUTE-YAML-SCOPE-1 raises the executable floor from 35 to 36. The
source remains v0.5.0, entries stay under Unreleased, and no remote or release
action belongs to this slice.

## [2026-09-06] release | achta-v0.5.1

The source version, exact machine and human fixtures, current wiki facts,
changelog, release notes, and project specification advance together to
v0.5.1. The recorded-digest, count-check, and declared-route entries move
unchanged in substance from Unreleased to the exact v0.5.1 heading. The
specification records `go.yaml.in/yaml/v3 v3.0.5` as Achta's first external Go
module dependency and the measured correctness exception for true YAML scope.
The executable Rulefloor remains 36.

## [2026-09-06] release | achta-v0.5.2-retirement-gaps

Three measured script-retirement failures are covered without importing caller
meaning. `count check` gains additive claim patterns, explicit claim/proof
comparison, last-matching-section scope, and caller-patterned next-paragraph
exemptions. `declared-route check` gains deterministic direct workflow
directories and opt-in per-file-any cardinality while retaining all-file ban
scanning and one scoped declaration in every route-using file. `mirror check
--digest-name` selects one exact opaque record name, including path-shaped
names, without normalization.

P-072 through P-074 record the boundaries. Achta still refuses to infer prose
claims, proof truth or relevance, route vocabulary, and normalized mirror
identity. Separate mutations made each new rule's focused test red; byte-exact
restoration returned the focused suite green. COUNT-RETIREMENT-PARITY-1,
DECLARED-ROUTE-PER-FILE-ANY-1, and MIRROR-DIGEST-NAMED-RECORD-1 raise the
executable Rulefloor from 36 to 39. Source, exact fixtures, changelog, release
notes, current wiki facts, and the release line advance together to v0.5.2.

## [2026-09-06] release | achta-v0.5.3-replacement-parity-gate

Principle 10 is now executable. `replacement check` reads the canonical
`achta.replacement-claims.v1` manifest and requires every caller-designated
verb-to-named-script replacement or retirement claim to cite the script's own
selftest fixtures, cite a recorded replay of both implementations, and record
agreeing exit codes for each fixture. Achta refuses natural-language claim
inference; exact claim statuses are caller flags with no default and candidate
entries make no claim.

The frozen v0.5.1-shaped claims for `count check`, `declared-route check`, and
`mirror check --digest` fail with nine missing-evidence violations. Disabling
the selftest citation branch made the focused rule test red with two violations
instead of three; restoration returned green. P-075 records the boundary,
REPLACEMENT-PARITY-CITATION-1 raises the executable floor from 39 to 40, and
source, fixtures, release documents, and current wiki facts advance together
to v0.5.3.

## [2026-09-06] feature | declared-route-full-replay-parity

The frozen replay at
`../wiki/contracts/replacement-replay-2026-09-06.md:117-128` now agrees on all
10 `rulefloor-install-gate.sh` selftest fixtures. Comment-only workflows are
evaluated as zero live scalars, so `per-file-any` passes them while exact-one
reports an evaluated mismatch. A designated-route workflow must carry the
caller-named key exactly once across its whole YAML document and exactly once
in the selected scope, closing the job-level second-declaration gap.

Both mechanics went red independently: restoring the empty-document refusal
made `TestCheckTreatsCommentOnlyDocumentsAsEmpty` fail with `empty YAML
document`; disabling the document-wide disagreement made
`TestCheckRejectsDocumentKeyDuplicates` pass the two-version fixture. Exact
restoration returned both tests green. DECLARED-ROUTE-EMPTY-DOC-1 and
DECLARED-ROUTE-DOCUMENT-KEY-1 raise the executable floor from 40 to 42, and
`replacement-claims.json` flips only `declared-route check` to `replaces`.

Mirror remains a candidate. An absent digest is missing required evidence and
therefore exit 2, while a valid disagreeing record is exit 1. A path-form
record name is opaque caller vocabulary and requires exact `--digest-name`;
there is no path normalization fallback. Two vendored-copy cases remain
unreplayed, so no mirror replacement claim is made.

## [2026-09-06] feature | count-full-replay-parity

`count check` now composes two caller-owned, repeatable pattern families: a
count-bearing target and an assertion anywhere in the same blank-line-delimited
paragraph. When both occur, the smallest positive target count becomes one
citation-backed claim. This is a generic mechanic with no target-noun or
assertion-verb defaults; it does not infer prose meaning and does not use a
paragraph-spanning expression fitted to replay fixture g.

The frozen table at
`../wiki/contracts/replacement-replay-2026-09-06.md:53-71` consequently agrees
on all 17 `count-claim-check.sh` selftest fixtures. Disabling assertion
recognition made both domain and CLI tests red: the four-target paragraph
returned pass with zero claims. Restoration returned green.
COUNT-PARAGRAPH-COMPOSITION-1 raises the executable floor from 42 to 43, and
`replacement-claims.json` flips `count check` to `replaces`. Declared-route
remains `replaces`; mirror remains `candidate`.

## [2026-09-06] release | achta-v0.5.4-claim-flip

Source, exact version fixtures, project status, changelog, release notes, and
current wiki facts advance together to v0.5.4. The declared-route and count
parity entries move unchanged in substance from Unreleased to the exact
v0.5.4 headings. The release is a patch because it closes measured gaps in two
existing verbs and adds caller-opted pattern vocabulary without removing or
renaming a command, changing a default, or changing a stable machine schema.
The executable Rulefloor remains 43.

## [2026-09-06] correction | retract-contradicted-replacement-claims

`replacement-claims.json` returns `count check` and `declared-route check` to
`candidate`. Their v0.5.4 rows asserted equal exits where the cited committed
replay records six differences, so neither named-script replacement claim is
currently supportable. Current specification and repository facts are corrected
without rewriting the historical v0.5.4 release record.

## [2026-09-06] feature | check-replacement-citations

`replacement check` now requires v2 claims to cite a workspace-confined
`achta.replacement-replay.v1` artifact, pin its exact SHA-256, and match its
complete verb/script fixture record. The committed v0.5.3 replay table is
vendored without rerunning either implementation; the frozen v0.5.4 claim state
fails on count `g` and declared-route `02`, `03`, `04`, `07`, and `10`. Legacy
v1 claims are explicitly attestations, both live dispositions remain
`candidate`, and a second armed rule makes future release workflows reject
lightweight tags. The executable Rulefloor rises from 43 to 45.

## [2026-09-06] release | achta-v0.5.5-checkable-replay

Source and exact version fixtures advance to v0.5.5. The correction and
checkable replay-evidence entries move unchanged in substance under the exact
release headings, while both named-script dispositions remain `candidate`.
The release workflow now requires an annotated tag, and the executable
Rulefloor remains 45.

## [2026-09-06] release | achta-v0.5.6-release-notes-parity

`make verify` and both release workflow stages now invoke one
`release-notes-check` target backed by the checked-in extractor. An optional
Unreleased section must be non-empty, the v0.5.5 tag remains untouched, source
and exact fixtures advance to v0.5.6, and RELEASE-NOTES-LOCAL-PARITY-1 raises
the executable Rulefloor from 45 to 46.

## [2026-09-06] feature | declared-route-earned-flip

`replacement check` now requires every claimed v2 row to name the exact caller
vocabulary and match that citation against the digest-pinned replay record.
The v0.5.6 replay is vendored and `declared-route check` becomes `replaces`
only under declared-route-v2 at the cited source lines, where all 10 selftest
fixtures agree and fixture 07 preserves THE-SECOND-INSTALL. Count and mirror
remain candidates. REPLACEMENT-VOCAB-CITATION-1 raises the executable floor
from 46 to 47.

## [2026-09-06] feature | count-check-earned-flip

The README's counted-target pattern advances to count-v2, excluding decimal
digits that touch another digit or a dash. The pinned v0.5.6 replay records all
17 `count-claim-check.sh` selftest exits agreeing under that vocabulary, so only
that named claim becomes `replaces`. Fixture g remains an explicit human-output
difference: count-v2 reports 22 where the script reports 4, while both exit 1.
COUNT-REPLACEMENT-VOCAB-1 raises the executable floor from 47 to 48; mirror
remains candidate.

## [2026-09-06] release | achta-v0.5.7-earned-replacements

Source, exact version fixtures, project status, changelog, release notes, and
current wiki facts advance together to v0.5.7. The declared-route and count
claims ship separately earned on their named replay vocabularies; mirror stays
candidate, the fixture-g 22-versus-4 diagnostic difference remains explicit,
and the executable Rulefloor remains 48.

## [2026-09-08] release | achta-v0.5.8-cd-hook

`achta hook cd` adds the workspace-independent `achta.hook-cd.v1` Bash
PreToolUse contract. It strips heredoc bodies, preserves pipelines, splits only
the declared statement operators, and denies committed-table writes without an
initial absolute or home `cd` or an absolute or home `-C` on every writer.
Unknown verbs are non-writes and unparseable input warns and allows. The
46-case verdict table covers the complete committed vocabulary;
HOOK-CD-WORKDIR-1 is mutation-proved by removing `make`, raising the executable
Rulefloor from 48 to 49. Source, exact fixtures, release documents, and current
co-versioned wiki facts advance together to v0.5.8.

## [2026-09-08] release | achta-v0.5.9-cd-hook-redirect

The v0.5.8 hook classified every output-redirection token as a file write,
blocking descriptor duplication and `/dev/null` sinks in ordinary read-only
commands. Eight new verdict cases reproduce the shipped defect before the
target-aware correction: `>&N`, `N>&M`, and exact `/dev/null` output now allow,
while numeric and combined redirects to real files still deny. The leading
absolute `cd` rule is unchanged. HOOK-CD-WORKDIR-1 was rehashed without a new
declaration, so Rulefloor remains 49/49; source, exact fixtures, release
documents, and current co-versioned wiki facts advance to v0.5.9.


## [2026-09-17] change | THE-WIKI-TOOL-THAT-SAYS-WHAT-IT-DOES

Measured the v0.5.9 source at 6a6e7df using fixture workspaces. Added explicit
repository selection/exclusion with reasons, aligned freshness enforcement,
made derive preview read-only by default with explicit writes and versioned
headers, confined pin edits to the pin and both dates, and added opt-in ordered
recipe expectations. Updated command help, specification and consumer migration
directions. No consumer repository was modified and no release is made here.

The brief overstates three claims at this head: derive already has --check;
pin already preserves lead text; --wiki-dir routing already selects the canonical
wiki for writes. Regression fixtures retain those truths. Existing expectations
for rewriting DERIVED Repo HEAD and naming a retired script are replaced with
byte-preservation and current-producer assertions. Detailed red/green evidence,
assertion counts, and open boundaries are recorded in docs/wiki-behavior.md.


## [2026-09-17] release | v0.5.10

The owner authorized the next patch from b56021a: annotated tag, fast-forward
main/tag push, repository Release workflow, archive checksum verification and
Homebrew upgrade with installed-byte comparison. Source and exact version
fixtures advance to v0.5.10; the Unreleased notes become this release's notes.
The existing workflow publishes and verifies only Casks/achta.rb in the tap,
including its single commit and ordinary main push. The release requires full
local validation before publication and comparison of the installed executable
with its published archive. No consumer repository is part of this release.


## [2026-10-05] release preparation | ACHTA-0.5.11

Prepare bounded stdin decision bodies, prefix help, and capability input forms
for v0.5.11. The baseline de3395b returned a missing-file error for stdin;
new regressions reproduce it and pass with the change. A disabled stdin branch
turns four success/check cases red again; restoration is green. Rulefloor owns
the new DECISION-STDIN-1 row (50 armed, mutation-proved rules). Owner RESUME 2 authorizes the release testVersion and four version/capability
fixtures to advance to v0.5.11, and only the decision-help case to check the
global help prefix and required flags. Other existing test cases stay unchanged.
The first full verification failed on these now-authorized expectations; this
is retained as a measured failure, not a successful verification. Source Version
is v0.5.11.

See docs/achta-0.5.11.md for measurements, invocation corrections and the complete
read-only consumer-reference census. No consumer pin is moved. Filing those
references in the read-only parent queue is declined because this brief permits
writes only in Achta; the open list remains in that evidence document. The
existing outside-path and filename restrictions remain; stdin shares the existing
body validator, not a new secret-content classifier. The stale v0.5.9 specification
status is corrected to this release. Full repository verification precedes the
commit, annotated tag, single tag-triggered Release workflow and cask installation.


Owner RESUME 3 authorized one compliant retry after the identity-read hook
refusal; it succeeded. Verification after RESUME 2 exited 0, including all
50 Rulefloor bindings and owned wiki checks. The parent-only postcheck is not
applicable to this independently owned wiki: its required parent page/log would
violate Achta's AGENTS.md, so no duplicate record is created. Release completion
still requires published-archive and installed-binary evidence in the final report.

## [2026-10-05] release | ACHTA-0.5.12 public Homebrew downloads

Owner-authorized v0.5.12 replaces private API download rewriting with public
release URLs and removes installer headers. An anonymous v0.5.11 checksum
download returned HTTP 200. The helper remains necessary for structured
postflight conversion and now refuses headers, API URLs, and missing, duplicate
or foreign downloads. Existing checksum, version, downgrade, idempotency and
remote-copy publication checks remain.

The public contract tests failed on the previous configuration and passed after
the change. A real GoReleaser snapshot initially exposed a configuration gap:
its version was 0.5.11-SNAPSHOT-3f2bae2 but its URLs used the literal v0.5.11 tag.
Owner RESUME 1 authorized repair. An explicit url.template now preserves Ruby's
version interpolation; the rebuilt snapshot passed all four URL/checksum pairs,
and the old token cask remained refused. Rulefloor amendments preserve all 50
rows and supersede changed proofs with measured observations.

Source version, the CLI test constant and four version/capability fixtures move
only from v0.5.11 to v0.5.12. Live private-distribution claims are corrected;
historical entries remain. Gograph individual reviews passed; whole-change
review cannot traverse deleted declarations, so its complete historical change
census records those deletions. Hook refusals were resumed or retried only in
owner-authorized compliant forms. Full make verify precedes the commit; the
annotated tag triggers one release workflow, whose publication and local brew
installation evidence belong in the final report. No adjacent item is deferred.

## [2026-10-05] release | ACHTA-0.5.13 generator header comparison

The CLI fixture fails on 602769e v0.5.12 with exit=1 want=0 and derive fail
for LF and CRLF version-only drift. The check-only wrapper reports both versions
as an explicit note; fact, wording, spacing and invalid-version controls remain
red, while exact derive previews and running-version writes are preserved.
WIKI-DERIVE-HEADER-VERSION-1 adds the observed red proof; existing tests change
only their release-version constant. Full verification precedes publication.
The source and four version/capability fixtures advance to v0.5.13.

The parent wiki is read-only. Its make check records GATE-RUN.txt on a clean
tree, so its proof requires the existing dirty-tree non-minting path or an
owner-authorized read-only alternative. No parent wiki content is changed here.
Lictor consumer pins are a separately authorized follow-up after installation.

## [2026-10-06] release | ACHTA-CLAIM

Baseline `5407b0a` was clean, equal to origin/main, and reported v0.5.13.
The new CLI fixture first failed with `claim check exit=2 want=0: achta:
unknown command "claim"`; help lacked the explicit checkout selector. The
implemented take, status, check and release lifecycle now passes. Fixed-clock
fixtures cover holder/age refusal, retained forced-release actor and reason,
tracked paths with and without claims, reverted local commits, missing upstream,
worktree isolation, concurrent takes and safe refusal without state changes.

New safety tests first found `malformed state accepted`, `linked Git directory
accepted for claim writes`, and `filename changed: ["leading newline.txt"]`.
The corrected decoder, metadata check and NUL-delimited diff-tree reader pass.
Disabling the other-holder guard made TestOwnershipAtFixedClock fail with
`take: status=pass error=<nil> reason=`; restoration passed. Rulefloor records
CLAIM-OWNERSHIP-1, increasing its armed, proved rows from 51 to 52. The existing
advertised-writer test exposed missing take/release entries in the hook's table;
adding those entries passes without altering the existing rules.

The source version, CLI test constant and four exact version/capability fixtures
advance from v0.5.13 to v0.5.14. Capability fixtures additionally enumerate the
four new commands and their two schemas. The old human fixture failed with
`want: achta v0.5.13` versus `got: achta v0.5.14`; revised fixtures pass.
Race-enabled claim and CLI tests pass. Complete verification precedes the release
commit; the annotated tag triggers publication once. The installed-binary proof
reuses the lifecycle and input-refusal tests through ACHTA_CLAIM_BINARY.

Claims are advisory, have no expiry and remain in private per-worktree Git
metadata. A missing upstream is unknown and refuses an unclaimed check; no fetch
or tracked-file write is performed. Release history is retained within the
bounded state artifact. All claim writes in this work use temporary fixtures.
No consumer gate or parent wiki is changed. The brief's baseline is correct;
no adjacent item is deferred. The release pin check's existing v0.4.4 consumer
warning is outside this task's scope; filing a duplicate parent queue entry is
declined because the parent wiki is read-only here.

The first full verification passed tests and vet, then staticcheck reported
`internal/gitstate/checkout.go:28:15: error strings should not be capitalized
(ST1005)`. Lowercasing that new error message is an in-scope correction under
P-076; staticcheck and the full gate are rerun before publication.

## [2026-10-06] release | ACHTA-0.5.15

Baseline `9eee610` matched origin/main and was clean; installed Achta was
v0.5.14 with module agreement pass. The new read-only fixture failed on the
old classifier: `verdict=deny want=allow; statement 1 (git config)` for reads,
with equivalent denials for Achta help and derive previews. All 54 original
verdict-table cases, including its 30 denial cases, remain unchanged. The
advertised-writer fixture now supplies `wiki derive --write` so it continues
to exercise the actual writer. Git config mutations, redirects, pipelines,
chained writers, and help-looking option values remain refused without a
selector. The additional preserved-write fixture passed on the old logic too.

The old release command failed with `zsh:2: command not found: goreleaser`.
TestPinnedReleaseCheck then failed with `No rule to make target release-check`.
The new shared Make target runs GoReleaser v2.17.0 through Go, matching
release.yml:12, and the workflow invokes it before artifact creation.
The real invocation reported `1 configuration file(s) validated`.
HOOK-CD-READS-1 and RELEASE-PINNED-CHECK-1 record these red observations;
Rulefloor's armed floor increases from 52 to 54 without removing any rule.

The source version, CLI test constant and four exact fixtures change only
v0.5.14 to v0.5.15: testdata/human/version.txt, human/capabilities.txt,
machine/version.json and machine/capabilities.json. The release recipe,
README, specification and current local wiki claims describe the new behavior.
The README and specification writer lists had omitted claim take/release,
although hookcd.go already classified both; their live lists are corrected.

Gograph source calls initially omitted the active session's required intention;
each returned `requires an intention. Please supply the --intention (-i) flag
stating your technical rationale.` Corrected calls succeeded. No permission,
owner or PreToolUse hook refused an action. Whole-change review and session
audit precede full verification. The authorized annotated tag triggers one
release workflow; published artifact, cask and installed-hook proofs are
reported at the close. Lictor's pin update is separately authorized; no claims
are taken and no other checkout is written. The parent pin check passed with
its existing v0.4.4-behind-v0.4.5 warning. Filing that warning again is declined:
the parent wiki is read-only and the consumer pins are outside this brief.

Brief detail: the existing repository has no release-check target or documented
bare goreleaser recipe; the PATH-dependent check was the measured prior manual
command. The new Make target and documented release procedure close that gap.

## [2026-10-09] release | ACHTA-GATE-RUN

Prepare v0.5.16 with explicit gate output capture. Make and CI keep their plans
and verdicts; gate run keeps the full combined log inside checkout Git metadata,
retains 20 logs, preserves child exit codes and returns one bounded summary or
one versioned JSON document. Arguments and potentially sensitive lines are
withheld. The full log remains private raw output. Linked worktrees retain their
own logs, and concurrent runs refuse rather than delete an active log.

Red at cd1bfef: TestGateRunFullLogAndExit (exit 2 instead of 7),
TestGateRunSummaryAndRetention (exit 2 instead of 3),
TestGateRunChildFlagsAndWorktree and TestGateRunHelpAndCapabilities failed.
The focused tests then passed, including fresh-record preference, stale-record
fallback and linked-directory refusal. GATE-RUN-LOG-1 records the observed red.
The baseline hook package passed 171 tests, including 54 original verdict cases.
No original hook verdict changes; the newly advertised gate writer is guarded.

Brief correction: the exact absolute-cd && claim command already passes on
v0.5.15, as does its newline form. Regression coverage preserves both; there
is no justified parser relaxation. A separate help-only rulefloor rehash call
was denied by achta hook cd; its named absolute-cd retry was used.

Decisions: arguments are withheld rather than echoed, log directories are 0700
and logs 0600, summaries ignore unchanged records, and a nonblocking checkout
lock protects active logs during retention. The explicit child may have the
effects the caller authorizes; the wrapper does not mint a witness.

Version-only fixtures: internal/cli/run_test.go testVersion,
testdata/human/version.txt and testdata/machine/version.json. The two capability
fixtures also add gate run and its schema and clarify the explicit-child limit.
Release publication, cask verification and installed proofs remain required
and are reported separately after the authorized annotated tag.
