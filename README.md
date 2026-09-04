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

Workspace-bound commands accept an explicit root or discover exactly one
ancestor with `wiki/repos/` and `wiki/platform/decisions.md`:

```sh
achta --workspace /path/to/workspace wiki pin REPOSITORY \
  --sha FULL_HEAD_SHA --verified YYYY-MM-DD --attest-reviewed --check

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
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt
achta --workspace /path/to/workspace witness check \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt

achta --workspace /path/to/workspace slice check \
  --repo REPOSITORY --commits 1 --entries 1 --json

achta --workspace /path/to/workspace amendments declare \
  --manifest REPOSITORY/ledger-amendments.json \
  --rule RULE-ID --class rule_added --reason-file REASON.txt

achta --workspace /path/to/workspace amendments rebase \
  --manifest REPOSITORY/ledger-amendments.json --repo REPOSITORY \
  --witness-log REPOSITORY/GATE-RUN.txt

achta --workspace /path/to/workspace amendments reconcile \
  --manifest REPOSITORY/ledger-amendments.json --repo REPOSITORY --json
```

`wiki pin` writes only after the caller explicitly attests that review
occurred. It checks the supplied full SHA against repository HEAD, preserves
the existing `verified_against` suffix, updates the derived `Repo HEAD` row,
validates the complete rendered page, and atomically replaces only that page.

`decision add` allocates one more than the highest measured ID for a prefix and
inserts caller-written prose at the measured section boundary. The caller owns
the complete title; Achta invents neither dates nor content.

`wiki freshness`, `wiki derive`, `wiki unpushed`, and `wiki check` reproduce the
workspace's mechanical page checks from local Git and filesystem facts.
`wiki unpushed` deliberately uses only the local upstream tracking ref and does
not fetch. The composed check reports freshness and derivation separately.

`witness summarize` treats wall elapsed time and summed target elapsed time as
different measurements. Missing timing remains absent. Green, red,
incomplete, malformed, unsupported, and stale records remain distinguishable.

`witness init`, `step`, `finalize`, and `check` maintain the same canonical
record format. `step` records an explicitly supplied result; it does not run a
command. Make or CI remains the gate runner. Achta intentionally has no
`witness run` command.

`slice check` audits an already landed slice using local Git and wiki facts. It
does not fetch, stage, commit, or otherwise mutate Git.

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

The v0.2.4 patch routes authenticated Homebrew downloads through GitHub's
private release-asset API.

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
