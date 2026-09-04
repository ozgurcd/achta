---
title: Achta Decisions
category: platform
status: authoritative
updated: 2026-09-05
---

# Achta Decisions

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
