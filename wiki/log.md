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
