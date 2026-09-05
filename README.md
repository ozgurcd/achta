# Achta

Achta is a deterministic, workspace-local governance and evidence CLI. It
updates a closed set of canonical artifacts, summarizes gate witnesses, and
reconciles explicit ledger amendments with Rulefloor's logical diff. It does
not decide whether human claims are true.

This is a private project. Source, releases, checksums, and CI output are for
authorized users only.

## Install

From an authorized checkout:

```sh
go install ./cmd/achta
```

## Commands

`version` and `capabilities` are workspace-independent:

```sh
achta version --json
achta capabilities --json
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

achta --workspace /path/to/workspace wiki freshness --strict --json
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
achta --workspace /path/to/workspace floor census \
  --file wiki/contracts/floor-completeness-census.md --repo identuum-idp-oss \
  --bucket COVERED,OOS-L,OOS-O,OOS-P,OOS-T --covered COVERED --fence-heading '## Full list' \
  --marker '~' --count-sum '^## Totals \(sum = (\d+)' --allow-prefix '- ALLOW-COVERS-ABSENCE:' --json

achta --wiki-dir /path/to/workspace/wiki mirror check \
  --master AGENTS.md --mirror wiki/contracts/AGENTS.md \
  --mirror wiki/tools/AGENTS.md --json

achta --wiki-dir /path/to/workspace/wiki parts lock \
  --dir wiki/prompt/parts --lock wiki/prompt/parts.lock --bump
achta --wiki-dir /path/to/workspace/wiki parts verify \
  --dir wiki/prompt/parts --lock wiki/prompt/parts.lock --json

go run ./cmd/achta --wiki-dir ./wiki wiki check --json
```

`wiki pin` writes only after the caller explicitly attests that review
occurred. It checks the supplied full SHA against repository HEAD, preserves
the existing `verified_against` suffix, updates the derived `Repo HEAD` row,
validates the complete rendered page, and atomically replaces only that page.
Repository-owned pages instead use `co_versioned: true`: they omit the
impossible self-referential SHA pin and are committed atomically with source.

`decision add` allocates one more than the highest measured ID for a prefix and
inserts caller-written prose at the measured section boundary. The caller owns
the complete title; Achta invents neither dates nor content.

`wiki freshness`, `wiki derive`, `wiki unpushed`, and `wiki check` reproduce the
workspace's mechanical page checks from local Git and filesystem facts.
`wiki unpushed` deliberately uses only the local upstream tracking ref and does
not fetch. The composed check reports freshness and derivation separately;
`--only freshness` (or `derive`, or both) evaluates and reports exactly the
named checks under the unchanged exit contract, so a gate can enforce pin
freshness on a working tree it has legitimately dirtied. An empty or unknown
name is exit 2 with nothing evaluated. The JSON document names the resolved
`wiki_dir` so a pass against the wrong wiki cannot be silent.

`recipe check` reads one Makefile target's recipe as text and refuses
neutralizers — a `-` prefix whose exit make ignores, a pipe, a trailing `&`, a
swallowed exit, and with `--forbid-noop` a command that cannot fail — while
`--expect-line` and `--expect-file` demand byte-exact lines, so anything
appended fails. `ledger census` recounts a markdown ledger table against its
totals rows and against the files (`--dir`) or directories (`--tree --ext`) on
disk: a present row must exist at exactly its stated size, a RETIRED row must
be gone, and every entry on disk must have a row. Both are read-only,
workspace-confined, and exit 2 when they cannot evaluate.

`floor census` recounts a fenced completeness census against itself and
against `rulefloor covers --json`, executed as an argument vector: uncited
covered rows, citations of unknown rules, repeated citations, bucket-led prose
outside the fence, stated counts the table contradicts, mutation-proven pairs
no row cites, and absences not on the frozen allowlist. Every piece of the
census's vocabulary is a flag; nothing is built in. It refuses to say whether a
cited rule is armed — that is RULE-FLOOR.md's column, Rulefloor's format — and
says so in its document.

`mirror check` computes SHA-256 over one master's exact bytes and compares each
explicit `--mirror` in caller order. It performs no newline, whitespace, or
semantic normalization: identical bytes match and any byte change differs.
Each result carries both digests when comparison is possible. An absent mirror
is an evaluated mismatch (exit 1); an absent master cannot establish the
reference digest and is `cannot_evaluate` (exit 2). Paths are read-only and
workspace-confined. With `--wiki-dir WORKSPACE/wiki`, a master at the workspace
root remains addressable because the selector retains `WORKSPACE` as its root.

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

Authorized users can install a released build with:

```sh
HOMEBREW_GITHUB_API_TOKEN="$(gh auth token)" brew install --cask ozgurcd/tap/achta
```

Achta releases remain private. Set `HOMEBREW_GITHUB_API_TOKEN` to a GitHub token
that can read `ozgurcd/achta` before installation. Tap publication separately
uses the repository secret `HOMEBREW_TAP_GITHUB_TOKEN`; neither token is stored
in the generated Cask. Configure the repository secret before pushing a release
tag; the workflow verifies its tap push permission before creating the release.

## Rulefloor

[`RULE-FLOOR.md`](RULE-FLOOR.md) is Achta's canonical repository-local ledger.
Its floor contains armed, mutation-proved release invariants. `make verify`
validates the ledger and executes its Go-test bindings on every slice:

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

## Exit contract

- `0`: operation or evaluation succeeded.
- `1`: evaluated mismatch, refusal, stale/red/incomplete evidence, or a
  check-mode difference.
- `2`: invalid input, unsafe path, malformed/unsupported/truncated evidence,
  timeout, unavailable tool, or another cannot-evaluate condition.

JSON mode writes exactly one versioned document to stdout. Successful JSON
commands write no prose to stderr.
`--quiet` suppresses successful human detail while preserving failures, JSON,
and an explicitly requested `--timing` line.

## Security and trust boundaries

Achta never invokes a shell and never performs Git mutations. Its Git and
Rulefloor subprocesses are bounded, timed, and invoked with explicit argument
vectors. Executables are resolved before invocation; Rulefloor reconciliation
reports the selected absolute binary, while Git failures identify discovery or
the selected binary without exposing environment variables. Every Git call uses
`--no-optional-locks` so status checks cannot refresh repository index metadata.
Mutation inputs and
targets must be bounded regular files inside the
canonical workspace; linked target components, traversal, concurrent changes,
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
section matching the release tag.
