---
title: Achta Decisions
category: platform
status: authoritative
updated: 2026-09-05
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
