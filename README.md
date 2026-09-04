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

achta --workspace /path/to/workspace witness summarize \
  --repo REPOSITORY --record REPOSITORY/GATE-RUN.txt --json

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

`witness summarize` treats wall elapsed time and summed target elapsed time as
different measurements. Missing timing remains absent. Green, red,
incomplete, malformed, unsupported, and stale records remain distinguishable.

`amendments declare` records human intent; it never infers a declaration.
`amendments rebase` selects the newest reachable commit with the accepted
`Witness: make verify green at ` subject that touched the named witness record.
`amendments reconcile` invokes Rulefloor directly with argument vectors,
requires its advertised stable interfaces, and checks measured and declared
changes in both directions. Header changes are reported separately.

The v0.1.0 release does not advertise Phase 3 `decision add`, broader wiki
checks, amendment clearing, or Homebrew distribution. Those workflows remain
explicitly deferred.

## Rulefloor

[`RULE-FLOOR.md`](RULE-FLOOR.md) is Achta's canonical repository-local ledger.
Its initial floor contains five armed, mutation-proved invariants covering
workspace discovery, timed JSON output, reviewed wiki pinning, amendment
reconciliation, and concurrent-safe replacement. `make verify` checks the
ledger on every slice; execute the bound tests with:

```sh
rulefloor check --repo . --run-profile unit --timings
```

## Exit contract

- `0`: operation or evaluation succeeded.
- `1`: evaluated mismatch, refusal, stale/red/incomplete evidence, or a
  check-mode difference.
- `2`: invalid input, unsafe path, malformed/unsupported/truncated evidence,
  timeout, unavailable tool, or another cannot-evaluate condition.

JSON mode writes exactly one versioned document to stdout. Successful JSON
commands write no prose to stderr.

## Security and trust boundaries

Achta never invokes a shell and never performs Git mutations. Its Git and
Rulefloor subprocesses are bounded, timed, and invoked with explicit argument
vectors. Mutation inputs and targets must be bounded regular files inside the
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
```

`RULE-FLOOR.md` is Achta's own Rulefloor ledger. A target repository's
`ledger-amendments.json` is a single-use declaration manifest consumed by
Achta; it is not Achta's ledger.
