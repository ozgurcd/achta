# Achta

Achta is a deterministic, workspace-local governance and evidence CLI. It
updates a closed set of canonical artifacts, summarizes gate witnesses, and
reconciles explicit ledger amendments with Rulefloor's logical diff. It does
not decide whether human claims are true.

Achta is public. Release archives and checksums can be downloaded without
GitHub credentials.

## Install

From an authorized checkout:

```sh
go install ./cmd/achta
```

## Commands

`version`, `capabilities`, `claim`, and the Bash PreToolUse hook are
workspace-independent:

```sh
achta version --json
achta capabilities --json
achta hook cd
```

`--help` and `-h` are also workspace-independent before or after a command
path; command-local forms emit the same stable top-level help.

Add global `--timing` to report total command elapsed time at the end:

```sh
achta --timing version
achta --timing capabilities --json
```

Human output ends with `elapsed: Nms`. JSON output remains one document and
adds the optional `elapsed_ms` integer field. Untimed output is unchanged.

Workspace-bound commands accept either an explicit root, its exact `wiki/`
directory, or discover exactly one ancestor with `wiki/repos/` and
`wiki/platform/decisions.md`. `--workspace` and `--wiki-dir` are mutually
exclusive. A repository may own that layout directly; when a parent workspace
also has a wiki, pass `--wiki-dir ./wiki` so the intended authority is explicit:

```sh
achta --workspace /path/to/workspace wiki pin REPOSITORY \
  --sha FULL_HEAD_SHA --verified YYYY-MM-DD --attest-reviewed --check

achta --wiki-dir /path/to/repository/wiki wiki check --json
achta --wiki-dir /path/to/repository/wiki wiki check --only freshness --json

achta --workspace /path/to/workspace decision add \
  --title 'complete decision title' --body-file decision-body.md --check

achta --workspace /path/to/workspace wiki freshness --json
achta --workspace /path/to/workspace wiki derive --check --json
achta --workspace /path/to/workspace wiki unpushed --repo REPOSITORY --json
achta --workspace /path/to/workspace wiki check --json

achta --workspace /path/to/workspace witness summarize \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt --json

achta --workspace /path/to/workspace witness init \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt \
  --label 'make verify' --targets build,test
achta --workspace /path/to/workspace witness step \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt \
  --target build --exit-code 0 --elapsed-ms 125
achta --workspace /path/to/workspace witness finalize \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt \
  --sibling SIBLING=SIBLING
achta --workspace /path/to/workspace witness check \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt \
  --sibling SIBLING=SIBLING
achta --workspace /path/to/workspace witness earned \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt --json

achta --workspace /path/to/workspace reachability classify \
  --repo REPOSITORY --base FULL_SHA \
  --no-reach 'docs/**=documentation only' --json

achta --workspace /path/to/workspace toolchain check \
  --repo REPOSITORY --manifest REPOSITORY/toolchain-pins.json \
  --workflow REPOSITORY/.github/workflows/verify.yml --json

achta --workspace /path/to/workspace slice check \
  --repo REPOSITORY --log-dir wiki/log --commits 1 --entries 1 --json

achta --workspace /path/to/workspace amendments declare \
  --manifest REPOSITORY/ledger-amendments.json \
  --rule RULE-ID --class rule_added --reason-file REASON.txt

achta --workspace /path/to/workspace amendments rebase \
  --manifest REPOSITORY/ledger-amendments.json --repo REPOSITORY \
  --witness-log REPOSITORY/GATE-RUN.txt

achta --workspace /path/to/workspace amendments reconcile \
  --manifest REPOSITORY/ledger-amendments.json --repo REPOSITORY --json

achta --workspace /path/to/workspace recipe check \
  --makefile wiki/Makefile --target check --expect-file wiki/contracts/check-recipe.txt --json
achta --workspace /path/to/workspace ledger census \
  --file wiki/contracts/retirement-ledger.md --dir wiki/tools \
  --tree GO=identuum-idp-oss/tools --ext .go --json
achta --workspace /path/to/workspace ledger rows \
  --file wiki/contracts/to-do-queue.MD --id-cell 1 --prose-cell 2 \
  --open-marker '**' --closed-marker 'DONE **' --identity-end-marker '**' \
  --completion-marker 'DONE ' --completion-marker '**SATISFIED' \
  --exempt-marker 'RULED-OPEN' --exempt-marker 'STAYS OPEN' \
  --condition-marker 'Close condition.' --condition-marker 'Close condition:' \
  --quote-marker 'CLOSE-CONDITION-MET:' --json
achta --workspace /path/to/workspace floor census \
  --file wiki/contracts/floor-completeness-census.md --repo identuum-idp-oss \
  --bucket COVERED,OOS-L,OOS-O,OOS-P,OOS-T --covered COVERED --fence-heading '## Full list' \
  --marker '~' --count-sum '^## Totals \(sum = (\d+)' --allow-prefix '- ALLOW-COVERS-ABSENCE:' --json

achta --wiki-dir /path/to/workspace/wiki mirror check \
  --master AGENTS.md --mirror wiki/contracts/AGENTS.md \
  --mirror wiki/tools/AGENTS.md --json
achta --wiki-dir /path/to/workspace/wiki mirror check \
  --master wiki/tools/gate-witness.sh \
  --digest wiki/contracts/gate-witness.master.sha256 \
  --digest-name tools/gate-witness.sh --json

achta --wiki-dir /path/to/workspace/wiki parts lock \
  --dir wiki/prompt/parts --lock wiki/prompt/parts.lock --bump
achta --wiki-dir /path/to/workspace/wiki parts verify \
  --dir wiki/prompt/parts --lock wiki/prompt/parts.lock --json

achta --wiki-dir /path/to/workspace/wiki count check \
  --file wiki/log.md --dir wiki/log --file wiki/contracts/to-do-queue.MD \
  --total-pattern 'TOTAL[[:space:]]+(?P<count>[0-9]+)[[:space:]]*[|]' \
  --part-pattern '(VACUOUS|HAS-CONTROL|SOUND)[[:space:]]+(?P<count>[0-9]+)' \
  --claim-pattern 'claims[[:space:]]+(?P<count>[0-9]+)[[:space:]]+fixed' \
  --claim-pattern '(?P<count>[0-9]+)[[:space:]]+sites fixed' \
  --claim-target-pattern '(?i)(?:^|[^-0-9])(?P<count>[0-9]{1,3}|Four|Five|One)(?:[^-0-9])[^.\n]{0,60}?\b(sites?|tests?|fences?|statements?|assertions?|controls?|gaps?|callers?|routes?|guards?|probes?|seeds?|mutations?)\b' \
  --claim-assertion-pattern '(?i)(closed|repaired|red-proved|now[[:space:]]+asserts?|is[[:space:]]+asserted|are[[:space:]]+asserted)' \
  --claim-section-pattern '^[[]20[0-9]{2}-[0-9]{2}-[0-9]{2}[]]' \
  --exempt-pattern 'count-claim-exempt:' \
  --citation-pattern '[[:alnum:]_./-]+[.](go|md):[0-9]+' --json

achta --workspace /path/to/workspace declared-route check \
  --dir identuum-idp-oss/.github/workflows \
  --dir identuum-ui/.github/workflows \
  --route-cardinality per-file-any \
  --route-pattern 'rulefloor/archive/refs/tags/[$][{]RULEFLOOR_VERSION[}]' \
  --required-route-pattern 'rulefloor/archive/refs/tags/[$][{]RULEFLOOR_VERSION[}]' \
  --ban-pattern 'brew install[^#]*rulefloor' \
  --ban-pattern 'go install[^#]*rulefloor' \
  --ban-pattern 'rulefloor/archive/refs/tags/v[0-9]' \
  --ban-pattern '-X main[.]version=([$][{]RULEFLOOR_VERSION[^}]|[$][{][^R]|[$][^{]|[^$])' \
  --required-key RULEFLOOR_VERSION --required-scope /env --json

achta --wiki-dir /path/to/repository/wiki replacement check \
  --file replacement-claims.json \
  --claim-status replaces --claim-status retired --json

go run ./cmd/achta --wiki-dir ./wiki wiki check --json
```

`hook cd` reads one PreToolUse JSON document from stdin and evaluates
`tool_input.command`. It removes heredoc bodies, splits statements on newline,
`;`, `&&`, and `||`, and keeps pipelines together. A command containing a
writing statement is allowed only when its first statement is an exact `cd` to
an absolute, `~`, `$HOME`, or `${HOME}` path before any assignment, or when
every writing statement contains `-C` with one of those absolute path forms.
Relative `cd` and `-C` paths do not count. Denial is exit 2 and names the
statement number, classified verb, and both fixes. Input that cannot be parsed
warns on stderr and allows with exit 0; unknown verbs are read-only by contract.
An unquoted `>` or `>>` is a write only when its next token is a file target.
Targets beginning with `&` are file-descriptor duplication, and the exact
target `/dev/null` is a sink; neither is a filesystem write. Numeric or
combined prefixes do not weaken a real file redirect: `2>err.log`, `&>out`,
and `1>>log` remain writes.

The committed writer vocabulary is:

- Git: `add`, `am`, `apply`, `bisect`, `branch`, `checkout`, `cherry-pick`,
  `clean`, `clone`, `commit`, `config`, `fetch`, `gc`, `init`, `merge`, `mv`,
  `notes`, `pull`, `push`, `rebase`, `remote`, `reset`, `restore`, `revert`,
  `rm`, `stash`, `submodule`, `switch`, `tag`, and `worktree`.
- Direct writers: `make`, `mv`, `cp`, `rm`, `mkdir`, `touch`, `tee`, file-
  targeting output redirection with `>` or `>>`, `gofmt -w`, `sed -i`, `go mod`, `go get`,
  `go install`, `go generate`, and `rulefloor rehash`.
- Achta writers: `amendments declare`, `amendments rebase`, `decision add`,
  `claim take`, `claim release`, `parts lock`, `wiki derive --write`, `wiki pin`,
  `witness finalize`, `witness init`, and `witness step`.

Git config reads (`--get`, `--get-all`, `--get-regexp`, `--list`/`-l`, and
a single key without a value) need no selector. Read modifiers such as
`--show-origin` retain that behavior. Setting a value, adding, replacing,
unsetting, editing, renaming or removing a section still requires a selector.
Achta `--help`/`-h` flags are reads; an option value spelled `--help` is data.
`wiki derive` previews are reads unless `--write` is present. Redirects and
other commands in a pipeline or chain retain their own write checks.
For example, `git config --show-origin user.name` and
`achta decision add --help` pass; `git config user.name Example` needs a selector.

The stable hook contract is `achta.hook-cd.v1`. A Claude Code owner can add
this object to the existing `hooks.PreToolUse` array in the machine-local
settings file:

```json
{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "command": "achta hook cd"
    }
  ]
}
```

`wiki pin` writes only after the caller explicitly attests that review
occurred. It checks the supplied full SHA against repository HEAD, preserves
the existing `verified_against` suffix, lead text and DERIVED bytes, and moves
`verified:` and `updated:` together with the pin (adding `updated:` if absent). It
validates the complete rendered page and atomically replaces only that page.
Repository-owned pages instead use `co_versioned: true`: they omit the
impossible self-referential SHA pin and are committed atomically with source.

`decision add` allocates one more than the highest measured ID for a prefix and
inserts caller-written prose at the measured section boundary. The caller owns
the complete title; Achta invents neither dates nor content.

`--prefix` selects the ID series and the section that already holds it; this is
not limited to Platform decisions. For a register whose D series is in Identity
and P series is in Platform:

```sh
achta decision add --prefix D --title "Identity choice" --body-file - < identity-body.md
achta decision add --prefix P --title "Platform choice" --body-file platform-body.md
```

`--body-file -` reads stdin. File paths must stay inside the workspace and pass
the existing secret-like filename checks; stdin has no filename. Both forms use
the same body validation: non-whitespace text, at most 65536 bytes, no NUL or
carriage return, and no second- or third-level register headings. Empty stdin is
refused. `--check` validates without writing and exits 1 for `would_change`;
a successful write exits 0, and invalid input exits 2. Prefixes and sections
come from the register, not a built-in mapping. `capabilities --json` advertises
both forms in the decision-add command's `input_forms` field.

`wiki freshness`, `wiki derive`, `wiki unpushed`, and `wiki check` reproduce the
workspace's mechanical page checks from local Git and filesystem facts.
`wiki unpushed` deliberately uses only the local upstream tracking ref and does
not fetch. The composed check reports freshness and derivation separately;
`--only freshness` (or `derive`, or both) evaluates and reports exactly the
named checks under the unchanged exit contract, so a gate can enforce pin
freshness on a working tree it has legitimately dirtied. An empty or unknown
name is exit 2 with nothing evaluated. The JSON document names the resolved
`wiki_dir` so a pass against the wrong wiki cannot be silent. Repeatable
`--repo NAME` scopes repository pages; repeatable `--exclude NAME=REASON` names
an exclusion with its required reason. Both checks honor that scope, report every
excluded repository as NOT judged, and keep all selected judgements unchanged.
Unknown, duplicate or conflicting selectors fail before evaluation.

`wiki freshness --repo NAME` now enforces drift by default (exit 1). `--strict`
remains accepted; use `--report-only` for an explicitly labelled report with exit
0 for evaluated drift. Unavailable evidence still exits 2 in either mode.

`wiki derive` and `wiki derive --check` preview before/after content per changed
page without writing (exit 1 when stale). Apply with `wiki derive --write`, which
names updated pages. `--print NAME` prints a block. The explicit modes are
mutually exclusive. Generated headers identify Achta and its version. `wiki check`
reports header-only generator version drift as a note rather than a failure;
any other derived content drift still fails. Derive previews remain byte-exact,
and `--write` updates the header to the running version. Pinning
does not regenerate blocks and reports that a new front lead is the PM's to write.
All selectors are resolved from global `--wiki-dir WORKSPACE/wiki`, which must
precede the command, for both reads and writes; the working directory is not a
second authority.

`recipe check` reads one Makefile target's recipe as text and refuses
neutralizers — a `-` prefix whose exit make ignores, a pipe, a trailing `&`, a
swallowed exit, and with `--forbid-noop` a command that cannot fail — while
`--expect-line` and `--expect-file` demand byte-exact lines, so anything
appended fails. Add `--expect-order` to require an ordered subsequence, matching
each occurrence once (`--expect-line` values first, then `--expect-file` lines).
Reordering identical membership then fails; other lines are still checked.
`ledger census` recounts a markdown ledger table against its
totals rows and against the files (`--dir`) or directories (`--tree --ext`) on
disk: a present row must exist at exactly its stated size, a RETIRED row must
be gone, and every entry on disk must have a row. Both are read-only,
workspace-confined, and exit 2 when they cannot evaluate.

`ledger rows` applies two bounded structural rules to explicitly selected
Markdown cells. The caller supplies one-based ID and prose cells, exact open
and closed ID prefixes, the identity suffix, repeatable completion, exemption,
and condition markers, and the closure quote marker; none has a default. An
open row carrying a configured completion marker without an exemption is a
violation. A closed row carrying a configured condition marker must have one
double-quoted string after the quote marker, and those bytes must appear
verbatim elsewhere in the same prose cell. The stable
`achta.ledger-rows.v1` result names every violation's line, rule, row identity,
and text. Achta deliberately does not infer completion from synonyms, normalize
quoted text, or claim that a repeated condition was actually satisfied.

`floor census` recounts a fenced completeness census against itself and
against `rulefloor covers --json`, executed as an argument vector: uncited
covered rows, citations of unknown rules, repeated citations, bucket-led prose
outside the fence, stated counts the table contradicts, mutation-proven pairs
no row cites, and absences not on the frozen allowlist. Every piece of the
census's vocabulary is a flag; nothing is built in. It refuses to say whether a
cited rule is armed — that is RULE-FLOOR.md's column, Rulefloor's format — and
says so in its document.

`mirror check` computes SHA-256 over one master's exact bytes and compares each
explicit `--mirror` in caller order. Optional `--digest FILE` also compares the
computed master digest with one canonical lowercase `sha256  name` record.
Selection defaults to the master's exact basename; `--digest-name NAME`
selects an exact caller-owned record such as `tools/gate-witness.sh` without
normalizing it. The digest result names the file, line, record, recorded hex,
and computed hex. Zero or duplicate selected records cannot evaluate. No
newline, whitespace, case, path, content, or semantic normalization is
performed. An absent mirror or valid digest disagreement is exit 1; an absent
master or digest file and malformed or ambiguous evidence is exit 2. Paths are
read-only and workspace-confined. With `--wiki-dir WORKSPACE/wiki`, a master at
the workspace root remains addressable because the selector retains
`WORKSPACE` as its root.

`parts lock` writes Achta's strict `achta.parts-lock.v1` artifact for every
direct `.txt` file under `--dir`, in bytewise filename order. A new lock starts
at `VERSION v1`; an existing lock keeps its version unless `--bump` explicitly
increments it. `parts verify` recomputes exact SHA-256 bytes, reports the lock
version, and fails by name when a locked part is edited or absent or when an
unlocked `.txt` file appears beside the set. Missing, malformed, linked,
oversized, or concurrently changing evidence is `cannot_evaluate`, never a
pass. Neither command invokes a shell or Git; only `parts lock` writes, and it
atomically replaces the explicitly named lock. The separate question “does
each part name a live canonical-document section?” remains caller-owned because
heading vocabulary and document meaning are not properties of a byte lock.

`count check` reconciles caller-declared count relationships across repeatable
`--file` inputs and the direct Markdown files in repeatable `--dir` inputs.
Explicit files come first; directories retain caller order and contribute files
in bytewise filename order. `--total-pattern` and `--part-pattern` form the
breakdown pair; repeatable `--claim-pattern` plus `--citation-pattern` form the
citation relationship. `--claim-section-pattern` limits it to the last matching
`## ` section and repeatable `--exempt-pattern` markers exempt the next nonblank
paragraph. Repeatable `--proof-claim-pattern` and `--proof-pattern`, with
`--proof-within-lines`, compare a claim count with the nearest bounded proof
count. Every count-bearing expression has one named `count` capture. Decimal
tokens are built in; aliases are explicit. Achta owns only those exact
mechanics. Repeatable `--claim-target-pattern` expressions classify counted
targets, while repeatable `--claim-assertion-pattern` expressions classify an
assertion elsewhere in the same blank-line-delimited paragraph. When both
families match, the smallest positive target count is one citation-backed
claim. The patterns have no defaults: Achta does not infer English claims or
decide that proof prose or a citation is true or relevant. Disagreement is
exit 1 and unavailable, unsafe, malformed, or incomplete evidence is exit 2.
The command reads only, invokes neither a shell nor Git, and performs no
content normalization.

The replacement vocabulary above is named `count-v2`. Its count token may not
touch a digit or dash, matching the retired script's refusal to treat date
fragments as claim counts. Exit-code parity is not diagnostic parity: on replay
fixture g, count-v2 reports counted target 22 while count-v1 and the script name
4; all three are exit 1. A caller retiring the script accepts that changed human
diagnostic while preserving the gate verdict.

`declared-route check` parses every explicit file, plus direct `.yml` and
`.yaml` files from explicit directories, as one YAML document. The default
requires exactly one configured route per file; `--route-cardinality
per-file-any` permits zero or more per file while still scanning every file for
every configured ban.
The caller designates one route pattern that needs a scoped key, then names
that exact key and its YAML mapping with a JSON Pointer such as `/env`. The
key is required exactly once in that mapping and exactly once across the whole
document. This is syntax-tree scope, so
a job-level `/jobs/verify/env` declaration cannot stand in for workflow-level
`/env` or coexist with it as a second version declaration.
YAML comments and comment-only lines within scalar command blocks do not count.
An otherwise comment-only or whitespace-only workflow is still an evaluated
zero-live-scalar document: it may pass under `per-file-any`, while the default
exact-one mode reports an evaluated route-count failure. There are no route,
ban, key, or scope defaults. Missing, malformed, multi-document, unsafe, or
ambiguous input is `cannot_evaluate`, exit 2. The command reads only
workspace-confined regular files and invokes neither shell nor Git.

`replacement check` makes named-script parity an explicit repository gate.
The caller supplies the exact statuses that mean replacement or retirement;
there are no vocabulary defaults and Achta does not infer claims from prose.
Every matching `achta.replacement-claims.v2` entry must cite the named script's
own selftest fixtures, cite the exact caller vocabulary used by the replay,
name a workspace-confined `achta.replacement-replay.v1` artifact, pin that
artifact's exact SHA-256, and list the complete fixture set. The vocabulary
citation must match the replay record byte-for-byte, so parity under one flag
set cannot be presented as parity under another.
Achta opens the cited artifact and reconciles every script/verb exit pair in
both directions. A digest, row, citation, or closed-set disagreement is exit 1;
malformed or unavailable evidence is exit 2. Candidate entries make no claim.
Legacy v1 claim rows are explicit attestations and cannot establish
replacement. Achta still executes neither implementation.

`witness summarize` treats wall elapsed time and summed target elapsed time as
different measurements. Missing timing remains absent. Green, red,
incomplete, malformed, unsupported, and stale records remain distinguishable.

`witness init`, `step`, `finalize`, and `check` maintain the same canonical
record format. Optional sibling pins are parsed and singleton-guarded. A changed
repository may pass only as `proven_no_reach` when every exact changed path is
covered by an explicit non-catch-all declaration. Optional CI provenance is
accepted only for a green, complete, clean commit tie on local HEAD ancestry;
the check performs no fetch or network request. `witness earned` refuses cycles
containing only witness machinery paths. `step` records an explicitly supplied
result; it does not run a command. Make or CI remains the gate runner. Achta
intentionally has no `witness run` command.

`reachability classify` emits the complete path inventory and exclusion reasons
as its audit record; it does not write a self-invalidating marker.
`toolchain check` compares bounded JSON version and script-digest declarations
with top-level CI workflow pins and never accepts an executable command.

`slice check` audits an already landed slice using local Git and wiki facts. A
repository-owned `wiki/log.md` takes precedence over a root `log.md`. The
optional, repo-relative `--log-dir` has no default: Achta reads its regular files
recursively in deterministic filename order after the frozen log file, requires
old filenames and per-file headings to remain append-only prefixes, and counts
new headings across both sources. An explicit missing or linked directory is
`cannot_evaluate`; only absence of both an automatically discovered log file and
the flag skips the check. The directory name stays caller-owned, consistent with
the flag-only vocabulary decision for `floor census`. The command invokes no
shell and does not fetch, stage, commit, or otherwise mutate Git.

`amendments declare` records human intent; it never infers a declaration.
`amendments rebase` selects the newest reachable commit with the accepted
`Witness: make verify green at ` subject that touched the named witness record.
`amendments reconcile` invokes Rulefloor directly with argument vectors,
requires its advertised stable interfaces, and checks measured and declared
changes in both directions. Header changes are reported separately, and output
identifies the absolute Rulefloor executable selected before invocation.

The v0.2.0 release adds decision insertion, native wiki status and
derivation, explicit witness lifecycle recording, landed-slice checks, and a
Homebrew cask. Unconditional amendment clearing remains deferred.

The v0.3.0 release adds cross-repository witness pins, no-reach-aware
staleness, CI provenance, earned-cycle refusal, and declarative toolchain parity.
Published-fix vulnerability policy remains in dedicated analyzers.

The v0.4.0 release makes repository-owned wiki authority directly selectable
with the fail-closed global `--wiki-dir` option.

Install a released build without a GitHub token:

```sh
brew install --cask ozgurcd/tap/achta
```

The cask downloads public GitHub release archives without Authorization or
Accept headers. The release check refuses API asset URLs and installer-token
lookups, and verifies the cask version and archive hashes against checksums.txt.
Tap publication uses the repository secret `HOMEBREW_TAP_GITHUB_TOKEN`;
the workflow verifies its tap push permission before creating the release.
Release tags are annotated objects; the workflow refuses a lightweight tag
before building or publishing artifacts.

## Rulefloor

[`RULE-FLOOR.md`](RULE-FLOOR.md) is Achta's canonical repository-local ledger.
Its floor contains armed, mutation-proved release invariants. `make verify`
validates the ledger, runs the same release-note extractor target as release
CI, enforces `replacement-claims.json`, and executes its Go-test bindings on
every slice:

```sh
rulefloor check --repo . --run-profile unit --timings
```

`make rulefloor-static` is available for diagnosis, but it is not the complete
verification gate.

## Repository wiki

[`wiki/index.md`](wiki/index.md) is the entry point for Achta-specific current
facts, decisions, and history. This wiki is co-versioned with the source and is
the only home for that information; the parent workspace wiki must not
duplicate it. `make verify` runs the repository-owned wiki check.

## Checkout claims

Claims let cooperating slices announce that they are writing one checkout.
They are advisory: they do not prevent an editor or Git from writing files.
They have no expiry; a crashed slice leaves its claim visible until release.
The explicit `--repo` must be an absolute checkout root. No wiki is required.

```sh
achta claim take --repo /work/project --slice schema-change --note 'database work'
achta claim check --repo /work/project --slice docs-change
achta claim status --repo /work/project --json
achta claim release --repo /work/project --slice schema-change
achta claim take --repo /work/project --slice docs-change
```

The second command refuses and names `schema-change` and its age. Repeating
`take` as the holder succeeds without renewing its time or changing its note.
`check` passes for the holder, or for an unclaimed checkout with no tracked
changes and no commits ahead of its locally available upstream. Without a
resolvable upstream, unclaimed `check` refuses; `status` reports the commit
count as unknown. Neither command fetches. Untracked files are not counted.
Changed tracked paths include staged changes, unstaged changes and both names
of a rename. Local commit paths include changes later reverted by another local
commit; they are not just the net diff. Output contains filenames, not contents.

All four commands accept `--json` (`achta.claim.v1`). Evaluated refusals exit 1;
invalid inputs or unavailable evidence exit 2. Status lists changes and commits
whether or not a claim exists. `release --slice docs-change --force --reason
'owner handoff'` can release another holder; the private history retains the
previous holder, releasing slice, time, force flag and reason. The slice name
is a caller assertion, not an authenticated identity.

The state is `achta-claim.json` in the checkout's Git directory, with mode 0600;
a linked worktree uses its own Git directory, not the shared one. The adjacent
`achta-claim.lock` serializes writers without waiting and survives as a private
lock file. Status and check create no files. State replacement is atomic and
bounded to 1 MiB, including release history. Notes and reasons allow at most
4096 bytes, slice names 128 bytes. Secret-like words, URL-shaped input, control
characters, malformed state, linked metadata files and Git checkout override
environment variables are refused without echoing the input. Claims coordinate
one local checkout only; they do not coordinate separate clones or machines.

## Gate logs

Run a long gate without flooding the terminal:

```sh
achta gate run --repo /work/project -- make verify-parallel
achta gate run --repo /work/project --json -- make verify
```

The command runs directly in that checkout, with its inherited environment.
Achta adds no shell and makes no gate plan. Only run commands you authorize.
The child may change files or use the network just as a direct invocation can.
It has no automatic timeout; interrupt or termination cancels its process group.
Child arguments after `--`, including `--json` and `--help`, stay child arguments.

Combined stdout and stderr go to a mode-0600 log under the checkout Git directory:
`achta/logs/<UTC time>-<command>-<unique suffix>.log`. Linked worktrees use their
own Git directory. The directories are mode 0700. The newest 20 logs remain;
another run in the same checkout refuses immediately while the log lock is held.
Logs contain full raw output: keep them private and inspect before sharing.

The summary has at most 24 lines: executable (arguments withheld), exit code,
duration, log path, and up to 20 failure lines. Changed `GATE-RUN*.txt` records
take precedence: only evidence for nonzero targets is selected. Unchanged
records are ignored. Otherwise the last error/failure/refusal lines in the last
64 KiB of the log are selected. Oversized or secret-shaped lines, URLs and
assignment-bearing lines are withheld with a pointer to the private log.
No environment is dumped. Gate arguments are never echoed.
`--quiet` and `--timing` do not alter this already bounded gate summary.

`--json` emits one `achta.gate-run.v1` document with `command`, `exit_code`,
`duration_ms`, `log_path` and `failures`. The exit code is the child's code;
signals use 128 plus the signal number. Failure to start is 127. Invalid inputs
or unavailable log storage are exit 2. This command is not a witness operation.

## Exit contract

- `0`: operation or evaluation succeeded.
- `1`: evaluated mismatch, refusal, stale/red/incomplete evidence, or a
  check-mode difference.
- `2`: invalid input, unsafe path, malformed/unsupported/truncated evidence,
  timeout, unavailable tool, or another cannot-evaluate condition.

JSON mode writes exactly one versioned document to stdout. Successful JSON
commands write no prose to stderr.
`gate run` is the explicit child-exit exception described above.
`--quiet` suppresses successful human detail while preserving failures, JSON,
and an explicitly requested `--timing` line.

## Security and trust boundaries

Outside the explicit `gate run` child, Achta never invokes a shell and never
performs Git mutations. Its Git and
Rulefloor subprocesses are bounded, timed, and invoked with explicit argument
vectors. Executables are resolved before invocation; Rulefloor reconciliation
reports the selected absolute binary, while Git failures identify discovery or
the selected binary without exposing environment variables. Every Git call uses
`--no-optional-locks` so status checks cannot refresh repository index metadata.
Mutation inputs and
targets must be bounded regular files inside the
canonical workspace (claims explicitly select a checkout's Git directory);
linked target components, traversal, concurrent changes,
and secret-like input paths are refused. Writes use a same-directory temporary
file, sync, identity recheck, rename, and directory sync.

The initial limits are 16 MiB for wiki pages, 8 MiB for witness records and
external Rulefloor output, 4 MiB for amendment manifests, 64 KiB per witness
line, and 4 KiB for a one-line amendment reason. Slow-target output is capped
at 100 entries.

A syntactically valid attestation or witness remains locally writable evidence.
Achta proves structural consistency and freshness, not author honesty, command
execution, or the truth of prose.

## Development

Go 1.27.1 or newer is required. Install pinned verification tools and run the
complete repository gate:

```sh
make install-tools
make verify
make test-fuzz
```

`make test-fuzz` runs every narrow parser fuzz target for five seconds by
default; set `FUZZTIME` to change the per-target duration.

`RULE-FLOOR.md` is Achta's own Rulefloor ledger. A target repository's
`ledger-amendments.json` is a single-use declaration manifest consumed by
Achta; it is not Achta's ledger.

## Release documentation

Detailed release bodies live in `RELEASE_NOTES.md`, newest version first.
`CHANGELOG.md` is the concise historical summary. Release automation requires
one exact `## vMAJOR.MINOR.PATCH — YYYY-MM-DD` heading and publishes only the
section matching the release tag. One optional leading `## Unreleased` section
must contain content; remove the heading after folding its entries. Local
`make verify` and both release stages run the same `release-notes-check` target.

Before releasing, run `make release-check` and `make verify`. The release-check
recipe runs `go run github.com/goreleaser/goreleaser/v2@v2.17.0 check`, at the
exact version pinned by the release workflow. It needs Go, not a globally
installed `goreleaser`. The workflow runs the same target before building
release artifacts, and a regression test refuses a mismatched tool version.

## Short status after compaction

Run `achta journal brief --workspace /work` to read current claims, HEADs,
local-origin ahead/behind counts, dirty counts, gate results, witness commits,
today's UTC decision IDs and the last three owner notes per claimed checkout.
The text has seven category lines; `--json` emits achta.journal-brief.v1.
Unavailable evidence stays explicit. The command does not fetch or judge gates.

Record an answer with `achta journal note --workspace /work --slice change-name
'Owner approved the revised scope'` (one shell line; flags before the text).
Without --workspace, it discovers the enclosing workspace. The note goes to
each claimed checkout's Git metadata, never its working tree. Notes are private,
limited to 1024 bytes, reject secret-like input, and retain the newest 200 lines.
Each append is atomic; a later repository failure does not undo earlier notes.
Never put credentials or sensitive values in notes.

An optional SessionStart command, `achta hook journal`, adds this context only
for compact or resume. It is silent on errors and always exits zero, with a
two-second work bound. Register it yourself with matcher compact|resume and a
five-second harness timeout. Achta does not change your harness settings.
