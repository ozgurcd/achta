---
title: Achta Decisions
category: platform
status: authoritative
updated: 2026-09-06
---

# Achta Decisions

## Decisions

### P-058 — The wiki close runs through Achta

THE-ACHTA-WIKI-CLOSE (owner ruling: Achta v0.2.5 is the tool). The wiki half of
the close ritual was hand-written — python heredocs rewriting pin lines,
appending ledger rows, log entries and decisions — and that class of command
also trips the gograph-first hook whenever the prose it writes names a Go path.
This decision records what moved to Achta, what did not, and the measurement
behind each.

MEASURED, NOT ASSUMED, on one tree. Freshness: Achta reports `11 fresh, 0 behind,
0 unpinned, 0 unreadable`; ours reports the same four counters with the same
verdict on all eleven pages. Derive: Achta `6 scanned, 0 changed`; ours `6
page-block(s) scanned; 0 rewritten; 0 stale-or-broken` over the same six pages.
Unpushed: Achta `2 ahead, 0 behind origin/main` for the wiki and `4 ahead` for
identuum-idp-oss; ours names the same counts and the same commits, and does NOT
report a behind counter, so Achta is strictly more informative there.

THE PIN MOVED, and the measurement is why. The fear was that a canonical pin
writer would flatten the narrative these pages carry — the pin line is the
durable record of what a slice proved. It does not: `achta wiki pin` is
surgical. Run against a page whose narrative is a full paragraph, it changed
exactly one token, the short sha to the full forty-character sha, and left every
word of the prose intact. It is also stricter than the heredoc it replaces in
two ways worth having: it REFUSES a sha that is not the repository HEAD, and it
requires an explicit `--attest-reviewed`, so a pin cannot be moved by a command
that merely knew the sha. Our own freshness gate accepts the full-sha form,
Achta's agrees, and a second run reports `unchanged`, so the two tools now hold
the same fixed point.

WHAT DID NOT MOVE, and why each stays. The narrative prose itself has no Achta
equivalent: Achta writes the sha and the verified date, not the sentence about
what was proved. That sentence is now written with the Edit tool rather than a
heredoc, which satisfies the no-heredoc rule without inventing a script. Ledger
rows, log entries and queue rows have no Achta command at all, so they stay out
of scope by the brief's own rule. `wiki-freshness.sh` and `wiki-derive.sh` stay
because they are not merely called but WIRED: the wiki check runs each as a
named gate target, and identuum-idp-oss's verify runs `wiki-freshness.sh`
through its wiki-fresh target. Replacing a gate is not adopting a tool, and the
brief forbids weakening one.

DELETED: `wiki-unpushed-check.sh`. Its own header said it was wired into
nothing, and that was true — zero callers, no Rulefloor rule binding it, and
`achta wiki unpushed` does the same job with one counter more. It leaves one
stale mention behind, in a comment in identuum-idp-oss's Makefile, which is
named here rather than fixed because that repo is out of this slice's scope and
a comment is not worth a twelve-minute appliance mint.

OUT OF SCOPE BY RULING: witness and amendments. Achta has both, and both would
cost something real — its witness record carries no cross-repo pin, so adopting
it would drop the two-repo tie the e2e mint depends on, and our ledger rules are
armed to our own tools, so switching would strand a rule. Named, not attempted.

### P-059 — Achta-specific wiki information is repository-owned

The owner requires all Achta-specific wiki facts, decisions, history, and
release records to live only under this repository's `wiki/` directory. The
parent workspace wiki must contain none of that material.

The local repository page is marked `co_versioned: true` and deliberately has
no `verified_against:` SHA. Code and wiki are committed atomically, and a Git
commit cannot embed its own final hash. `verified:` records the human review
date; the containing checkout supplies the version boundary.

Generic workspace governance may route any repository to an owned wiki when
that repository carries one. It must not special-case this project by name or
recreate a central page, log entry, or decision.

### P-060 — Direct wiki selection preserves workspace semantics

The global `--wiki-dir PATH` option directly selects repository-owned wiki
authority without relying on ambiguous ancestor discovery. The path must
canonicalize to the direct `wiki` child of its workspace root and contain both
`repos/` and `platform/decisions.md`.

`--wiki-dir` and `--workspace` are mutually exclusive. Achta still retains the
wiki's parent as the workspace root because wiki pinning, freshness, derivation,
slice checks, and related operations may resolve repositories beside the wiki.
An arbitrary detached content directory would silently change those contracts
and is therefore rejected.

### P-061 — Wiki check selection is a set under the single exit contract
`wiki check --only NAME[,NAME...]` selects which of `freshness` and `derive`
are evaluated. The selected checks are the only ones evaluated and the only
ones reported, in canonical order, and the existing three-way exit contract
applies to that selection unchanged. Selection is a set: Achta does not encode
per-check outcomes in the exit status, because a per-check code would have to
encode a set — a bitmask, not an exit status — and the process contract is one
verdict per invocation.

An empty, unknown, or repeated name is invalid input: exit 2 and nothing is
evaluated. Without `--only`, every check runs, which is the prior behaviour.

The need was measured by a caller against v0.4.0. `wiki check` was the only
enforcing wiki command and it coupled freshness with derive; derive records
`Working tree vs HEAD`, so it fails on a tree the caller legitimately dirtied,
and `make verify` dirties its own gate record by definition. `wiki freshness`
only reports. So "are the pins fresh on a dirty tree" had no enforcing command,
and the caller parsed `checks[]` out of the JSON in a Makefile.

Independently, `achta.wiki-check.v1` carries `wiki_dir`, the wiki directory
Achta actually resolved, and the text output prints it first. Callers pass
`--wiki-dir` because a pass against the wrong wiki was silent; the document now
says which wiki it judged.

### P-062 — New workspace tooling is an Achta command, never a new script
Owner ruling, 2026-09-05: new workspace tooling is an Achta command, never a
new shell script. Two identuum wiki gates existed as shell only because Achta
had no verb for them; both are read-only, workspace-local, deterministic, and
use the standard 0/1/2 exit contract.

`recipe check --makefile PATH --target NAME [--expect-line S]... [--expect-file
PATH] [--forbid-noop]` reads one target's recipe as text and refuses
neutralizers — edits that keep a gate looking present while make ignores its
exit: a line beginning `-` (before or after `@`), a pipe (a `||` is not a pipe),
a line ending `&`, a swallowed exit (`|| true`, `|| :`, `; true`, `exit 0`), and
with `--forbid-noop` a command that is `true`, `:`, `echo` or `printf`.
`--expect-line` and `--expect-file` are EQUALITY, byte for byte, on the physical
recipe line without its leading tab: substring matching is the bug this exists
for, so anything appended must fail. Measured cause: macOS make 3.81 silently
ignores `.SHELLFLAGS := -o pipefail -c` (GNU Make 4.3 honours it), so a static
rule is the only guard on that toolchain.

`ledger census --file MD --dir [LABEL=]PATH... --tree [LABEL=]PATH... --ext E`
recounts a markdown table against itself and against disk: per-mark counts and
line sums against the totals rows, the total row against every row summed,
unique subjects, a present-marked row existing at exactly the stated size, a
RETIRED row absent, and every file (or subdirectory, for --tree) under a source
rowed. Measured cause: a row added with the totals untouched passed every gate
the workspace had.

Out of Achta's boundary and refused: rule 5 of the retired shell (every wiki
script that implements `--selftest` must have a `check` entry) — it inspects
other scripts' contents for a workspace convention — and gate-witness entry
semantics (`name=command`), which are the identuum wiki's Makefile convention,
not a make concept; `--expect-file` covers them byte-exactly instead.

### P-063 — Census vocabulary is the caller's: flags, never a canonical Achta format
The census vocabulary belongs to the workspace that wrote the census, not to
Achta. `floor census` therefore takes every piece of it as a flag — the bucket
tokens (`--bucket`), the covered bucket (`--covered`), the heading that
introduces the fenced table (`--fence-heading`), the frozen marker
(`--marker`), the stated-count patterns (`--count-sum`, `--count-frozen`,
`--count-plain`) and the allowlist line prefix (`--allow-prefix`) — and refuses
to run with the required ones absent. Nothing is defaulted to identuum's
names.

Considered and rejected: a canonical census format owned by Achta. It would
force the workspace to rewrite a 1285-row page into Achta's vocabulary, make
Achta the owner of a classification scheme whose meaning it does not judge,
and turn every future bucket into an Achta release. Also rejected: hardcoding
identuum's tokens, which is the same ownership mistake with worse manners.
Flags keep the contract visible at the call site, the way `--sibling` and
`--only` do, and let a second workspace with a different taxonomy use the same
verb unchanged.

What Achta does own here is the mechanics: a fenced row is `BUCKET N Symbol
file:line reason…`; a citation is a token in Rulefloor's rule-ID grammar,
optionally carrying the marker; a stated count is one capture group; the
covers document is `rulefloor covers --json`, executed as an argument vector,
consumed like ledger-diff.v1. And what it refuses: whether a cited rule is
ARMED, because that lives in RULE-FLOOR.md's columns — Rulefloor's format —
and Achta consumes Rulefloor's machine output, never its ledger's text.

### P-064 — A release ends at the installed Homebrew binary

Owner ruling, 2026-09-05: every Achta release is committed and pushed, tagged
and published, made available by committing and pushing its matching Homebrew
cask, and then installed through Homebrew. A local release commit, tag, release
asset, tap update, or green build is an intermediate state, not a completed
release.

Completion evidence is the installed `/opt/homebrew/bin/achta` reporting the
exact release version with version agreement `pass` and advertising the newly
shipped capability. This sequence applies to every future release.

### P-065 — A log directory is caller vocabulary, selected explicitly

`slice check` accepts a repository-relative `--log-dir` with no default. This is
the same ownership boundary as P-063: Achta owns deterministic ordering and
append-only mechanics, but `log/`, `wiki/log/`, or another directory name is a
workspace layout decision. Auto-detecting one conventional name would make
Achta own Identuum's migration vocabulary and could silently change the evidence
set in an unrelated workspace.

When selected, the discovered frozen log file is ordered first and recursive
directory files follow in bytewise filename order. Existing filenames and each
existing file's headings remain prefixes, so a lexically earlier file cannot
reorder the evidence and an inserted heading cannot masquerade as an append.
Explicit absence, links, unreadable inputs, or overlap with the discovered log
file are `cannot_evaluate`; only absence of both the file and the flag skips.

### P-066 — Mirror identity is exact bytes and an absent master cannot evaluate

`mirror check` computes SHA-256 over the exact bytes of one explicitly selected
master and every explicitly selected mirror. Newline, whitespace, encoding, and
semantic normalization are refused: these artifacts are copies, so anything
other than identical bytes is drift. Mirrors retain caller order and each
reports `match`, `differs` with both digests, or `absent`.

An absent master is always `cannot_evaluate`, exit 2. Without reference bytes,
equality has no truth value. An `--allow-absent-master` flag was considered and
rejected because it would preserve the old fail-open and widen a small command
solely to encode permission not to compare. A caller intentionally lacking a
master should not invoke the comparison. An absent mirror is different: once
the master exists, the named copy is conclusively not identical, so that is an
evaluated failure, exit 1.

Per P-060, `--wiki-dir WORKSPACE/wiki` retains `WORKSPACE` as the confinement
root; a root-level master and wiki-contained mirrors therefore share one safe
boundary. The verb reads bounded regular non-linked files, invokes no shell or
Git command, and performs no mutation. Existing comparison scripts remain until
their callers explicitly prove parity and retire them in a later slice.

### P-067 — Achta owns the prompt-part lock, not the referenced heading vocabulary

`parts lock --dir DIR --lock FILE [--bump]` writes a canonical
`achta.parts-lock.v1` artifact and `parts verify --dir DIR --lock FILE` is its
only semantic reader. Achta owns this format because it creates the fact being
enforced, as it owns `gate-run.v1`: one schema marker, one `VERSION vN`, and one
unique bytewise filename-sorted `name sha256` record for every direct `.txt`
part. Making those tokens flags would let callers redefine the artifact the
writer claims to own and would weaken cross-caller interoperability. A caller
with the former hand-written lock migrates it when adopting these commands;
Achta does not silently accept unversioned comment dialects as the canonical
format.

The answer on “each part names a live section of a canonical document” is NO.
A content lock can establish exact part identity and closed-set membership, but
the canonical document, its heading grammar, and what a reference means are
caller vocabulary. Folding that assertion into Achta would repeat the P-063
ownership error and turn a workspace-specific naming convention into a product
contract. The caller keeps that one semantic analyzer line.

Verification is fail closed: a matching closed set is exit 0; an edited,
absent, or unlocked part is evaluated drift and exit 1; malformed, missing,
unsafe, oversized, or concurrently changing evidence is `cannot_evaluate` and
exit 2. The commands invoke no shell and perform no Git operation.

### P-068 — Ledger rows are structural, and every vocabulary token is explicit

`ledger rows` takes the two mechanics and refuses the semantic leap between
them. The caller selects one-based ID and prose cells and supplies exact open
and closed ID prefixes, an identity terminator, repeatable completion,
exemption, and condition markers, and a closure quote marker. None has a
default. This is the same ownership line as P-063, P-065, and P-067: Identuum's
words do not become Achta's universal vocabulary merely because Achta can
compare their placement.

The completion half is deliberately literal. An open row violates only when
its selected prose contains a caller-supplied completion marker and no supplied
exemption marker. Achta says NO to deciding whether `landed`, `verified`, or any
other unconfigured prose means completion. This avoids a regex synonym list
whose false positives and false negatives would be undocumented policy
decisions.

The close-condition half is also mechanical. A closed row carrying a supplied
condition marker must contain exactly one supplied quote marker followed by a
non-empty double-quoted string, and those exact bytes must appear elsewhere in
the same prose cell. Achta does not normalize or decide that the quoted
condition was actually satisfied. Repetition proves that the condition was
read and carried into the closure; truth remains human review.

Malformed, ambiguous, unsafe, or vocabulary-free input is
`cannot_evaluate`, exit 2, never a pass. The verb reads one confined regular
file, invokes no shell or Git process, mutates nothing, and names its semantic
refusals in `achta.ledger-rows.v1`.

### P-069 — Recorded mirror digests bind to one exact basename

`mirror check --digest FILE` reads canonical lowercase
`sha256<two ASCII spaces>name` records and selects a record only when `name`
equals the master's basename byte-for-byte. A differently named record is not
evidence about the selected master, so a file with no applicable record is
`cannot_evaluate`, exit 2, rather than an evaluated digest mismatch.

A multi-record file is accepted only when every record is canonical and exactly
one record names that basename. This makes the selection unambiguous without
silently discarding malformed evidence; duplicate applicable names and absent
applicable names are exit 2. Achta reports the digest file, selected line and
name, recorded digest, and computed master digest.

A valid recorded digest that differs from the SHA-256 of the master's exact
bytes is an evaluated failure, exit 1. Achta refuses case folding, whitespace
repair, line-ending conversion, path normalization, and master-content
normalization. An absent digest file is `cannot_evaluate`, matching the absent
master ruling in P-066. The command remains read-only, workspace-confined, and
invokes neither shell nor Git.

### P-070 — Count claims become mechanics only after caller classification

Achta accepts both the arithmetic and citation-count halves, but not an English
claim detector. `count check` requires the caller to provide each total, part,
claim, and citation regular expression with no default. Count-bearing patterns
expose one named `count` capture; exact non-decimal tokens require explicit
aliases. Once the caller has classified the text, Achta owns the mechanical
questions: do line-scoped parts sum to the stated total, and does the claim's
blank-line-delimited paragraph contain at least the stated number of distinct
exact citation matches?

This keeps the same boundary as P-063, P-065, and P-067. Syntax and arithmetic
are portable; whether prose claims a fix and whether a citation proves a
disposition are caller meaning. The result therefore names both refusals. It
does not build in fix verbs, dates, latest-section rules, exemptions, file:line
grammar, or proof relevance. A caller can reproduce its chosen structure with
flags without making that vocabulary Achta policy.

Inputs are also explicit: repeatable files followed by repeatable direct
Markdown directories, each directory in caller order and its files in bytewise
filename order. Achta does not auto-detect `log/`. This avoids owning a caller's
layout while still covering a growing split-log set without shell expansion.

### P-071 — Declared-route checks own YAML syntax, not route vocabulary

Achta takes YAML structure into its boundary because mapping scope is portable
document mechanics: workflow-level `/env` and job-level
`/jobs/verify/env` are different syntax-tree locations regardless of what a
key means. A line counter cannot preserve that distinction and would silently
pass the measured wrong-scope failure. `declared-route check` therefore uses
the stable `go.yaml.in/yaml/v3` node parser; this is the measured correctness
exception to the standard-library preference. The maintained v3 API is chosen
over the current v4 release candidate.

The caller still owns every policy word. Files, route alternatives, banned
patterns, designated route requiring the key, required key, and JSON-Pointer
scope are explicit flags with no defaults, consistent with P-063, P-065,
P-067, and P-070. The scoped key is required only when that designated route
is selected. Achta owns only exact match counts, YAML mapping resolution, and
comment exclusion. It refuses
line-level scope approximation, shell interpretation, route-name defaults, and
any claim that an install route is operationally correct. Malformed or
ambiguous YAML cannot evaluate; it never passes.

### P-072 — Count retirement parity is caller-classified structure

Achta covers CLAIM-vs-PROOF, last-section scope, exemptions, and multiple claim
forms only after the caller supplies every classifier. Repeatable claim,
proof-claim, proof, and exemption patterns, a section pattern, and a positive
proof distance have no defaults. The mechanics are deterministic: citation
claims are limited to the last matching `## ` section, an exemption marker
applies to the next nonblank paragraph, and a proof claim is compared with the
nearest proof count inside the caller's line distance.

This extends P-070 without crossing its boundary. Achta can compare explicit
counts and locations; it still refuses to decide that prose makes a claim or
that a nearby proof line is true, relevant, or sufficient. Repeated patterns
are additive and never silently overwrite one another.

### P-073 — Route cardinality belongs to each workflow file

The compatibility default remains exactly one configured route per workflow.
Explicit `--route-cardinality per-file-any` permits zero or many route uses in
each file independently, matching repositories where CI has multiple derived
installs and another workflow has none. Every selected workflow is still
scanned for every caller-supplied ban, and each file using the designated route
must contain exactly one required key at the selected YAML mapping scope.

Repeatable explicit directories add only their direct lowercase `.yml` and
`.yaml` regular files in bytewise filename order. Achta owns that selection and
cardinality mechanic, not directory conventions, route names, bans, keys, or
scope vocabulary. No route-free file may escape the ban scan.

### P-074 — Digest record names are exact opaque caller vocabulary

`mirror check --digest-name NAME` selects the unique canonical digest record
whose name equals `NAME` byte-for-byte. This covers path-shaped records such as
`tools/rulefloor-install-gate.sh` without interpreting the name as a path.
Omitting the flag preserves exact master-basename selection for compatibility.

Zero or duplicate exact matches cannot evaluate. Achta refuses basename
fallback after an explicit name, path cleanup, separator conversion, case or
whitespace repair, and every form of master-content normalization.

### P-075 — Named-script replacement claims are explicit artifacts

Achta refuses to decide that prose in a specification, changelog, release
note, README, capability description, or wiki means a verb replaces or retires
a script. That classification is caller meaning, consistent with P-063.
Capabilities continue to advertise only command and schema existence.

The authoritative claim is instead a record in
`replacement-claims.json` using the Achta-owned
`achta.replacement-claims.v1` shape. The caller declares exact claim statuses
through repeatable `--claim-status` flags with no default. Each matching
verb-to-named-script entry must cite the script's own selftest fixtures, cite a
recorded replay of both implementations, and record agreeing script and verb
exit codes for every named fixture. Candidate entries are explicitly not
claims.

Achta owns the bounded artifact grammar, required-field presence, and numeric
exit agreement. It does not execute the cited replay or judge whether a
citation is truthful. `make verify` runs the manifest check, and the frozen
uncited v0.5.1 claims remain a red fixture so the former silent pass cannot
recur.

### P-076 — Replacement evidence is vendored, digest-pinned, and reconciled

An opaque citation into a sibling workspace cannot prove parity inside Achta's
repository gate. The v0.5.4 manifest demonstrated the defect: it recorded equal
exits for six rows while the cited committed replay recorded different exits,
and `replacement check` passed because it never opened the citation.

Achta therefore owns `achta.replacement-claims.v2` and
`achta.replacement-replay.v1`. A claim status must name a workspace-confined
replay artifact, pin its exact SHA-256, and match one exact verb/script record,
its selftest citation, and its complete fixture set in both directions. Digest,
row, missing-fixture, and extra-fixture disagreements are evaluated failures.
Missing, unsafe, malformed, or ambiguous evidence cannot evaluate. A v1 claim
status is explicitly an attestation and cannot establish replacement.

The vendored artifact records the sibling document's path, commit, and digest
as provenance, but Achta does not read outside its workspace or rerun either
implementation. It proves that the canonical claim agrees with reviewed,
co-versioned replay bytes; it still cannot prove the replay author was honest.
Both count and declared-route stay candidates until a caller-owned replay
records actual parity and a later Achta slice vendors that new evidence.

### P-077 — Release tags are annotated objects

Future release tags are annotated, not lightweight. The release workflow checks
the pushed ref with `git cat-file -t` and refuses unless it is a tag object,
before source-version validation, build, release publication, or tap mutation.
It still dereferences the annotated object and requires its commit to equal the
checked-out HEAD.

Published v0.5.4 remains untouched: moving or replacing that lightweight tag
would rewrite a published release identity. Pinning the type prospectively
removes operator ambiguity without altering existing history.

### P-078 — Release notes have one local-and-CI grammar

An optional leading `## Unreleased` section is accepted only while it contains
nonblank content. Once those entries are folded into an exact version section,
the empty heading is removed. This preserves the working convention without
letting a syntactically present but content-free section pass locally.

`make verify` runs `release-notes-check`, and both release workflow stages call
that same Make target. The target invokes the checked-in `cmd/release-notes`
entry point, which defaults to the source version; there is no copied heading
parser or second CI-only command. The v0.5.5 annotated tag remains immutable,
and the failed publication is corrected forward as v0.5.6.

### P-079 — A replacement claim is earned on one named vocabulary

Exit parity is meaningful only for the exact caller flag vocabulary used by
the replay. Every v2 claim status therefore carries a non-empty
`vocabulary_citation`, and the selected digest-pinned replay record must carry
the same bytes. Missing or different vocabulary is an evaluated failure; it
cannot borrow parity measured under another flag set.

The v0.5.6 replay establishes `declared-route check` parity on all 10
`rulefloor-install-gate.sh` selftest fixtures only under declared-route-v2, the
README's four-ban vocabulary. The manifest cites
`../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:124-156`; fixture 07
is red on both sides through `required-key-document expected=1 observed=2`.
That claim becomes `replaces`. `count check` and `mirror check --digest` remain
candidate at this commit.

### P-080 — Count replacement uses dash-excluding count-v2

The README's replacement command adopts count-v2: a decimal count may not touch
another digit or a dash. This is a vocabulary correction, not a fixture-shaped
exception. A date fragment is not a claim count, and
`count-claim-check.sh` has always refused digits touching a dash. Under the
v0.5.6 replay, count-v2 agrees with all 17 script selftest exits and the manifest
cites `../wiki/contracts/replacement-replay-2026-09-06-v0.5.6.md:57-107`.
`count check` therefore becomes `replaces` on exactly that vocabulary.

The retirement accepts one diagnostic difference rather than hiding it. In
fixture g, count-v2's consuming boundary makes the verb report target 22;
count-v1 and the script report 4. All are red at exit 1. Exit-code parity is the
replacement contract recorded by the replay; it is not diagnostic parity, and
the project specification names the human-visible difference.
