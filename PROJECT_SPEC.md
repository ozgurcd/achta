# Achta Project Specification

Status: v0.2.0 released baseline; v0.2.5 current authenticated Homebrew patch

Project name: Achta

Executable: `achta`

Language: Go 1.27.1 or newer

Intended location: `/Users/odemir/Development/identuum/achta`

Git remote: `git@github.com:ozgurcd/achta.git`

Repository visibility: private

Go module path: `github.com/ozgurcd/achta`

Libraries: Go native libraries preferred over 3rd party.

UUID: If used, all UUIDs must be UUIDv7.

DB: If required, use SQLite without CGO.


Developer tools: use Gograph instead of Unix text-processing tools where
available.

## 1. Executive summary

Achta is a deterministic, workspace-local governance and evidence tool.

It replaces repeated ad hoc Python, Perl, `jq`, and shell manipulation of wiki
pins, decision records, amendment manifests, and gate-witness records with one
small, dependency-free Go binary. It owns the mechanical integrity of those
workspace artifacts. It does not decide whether their human claims are true.

Achta complements existing tools rather than absorbing them:

- Rulefloor owns invariant declarations, test/check bindings, red-proof
  observations, test fingerprints, covered-symbol bindings, and ledger drift.
- Gograph owns Go source structure and static graph evidence.
- Dedicated analyzers own their particular source or protocol comparisons.
- Achta owns workspace bookkeeping, declared-change reconciliation, witness
  record handling, and deterministic wiki maintenance.

Achta must not become a generic policy engine, task runner, requirements
system, documentation platform, source-code analyzer, CI server, or database.

## 2. Measured problem

Repeated agent work has concentrated in four mechanical operations:

1. Wiki verification pin updates occur many times across active work. Agents
   currently rewrite `verified_against:` and the corresponding derived
   repository-HEAD row with one-off regular expressions. This has contributed
   to working-directory drift, malformed edits, and hook false positives.
2. Adding a platform decision repeatedly requires allocating a decision ID and
   splicing prose at a structural marker by hand.
3. Amendment manifests are edited every slice to declare and later reconcile
   Rulefloor ledger changes. Hand editing has produced stale baselines and
   unnecessary failed verification runs.
4. Gate-witness records are repeatedly parsed by hand to total target elapsed
   times, compute wall time, find slow targets, and determine whether a record
   is current.

These operations are deterministic and format-sensitive. They are appropriate
for a tool. Their human meaning remains outside the tool's authority.

## 3. Product definition

Achta is:

- A local command-line tool.
- Scoped to one explicitly selected workspace.
- Deterministic for identical inputs, except for explicitly recorded current
  time or measured command duration.
- Conservative: ambiguous, stale, truncated, malformed, or unavailable
  evidence fails closed as `cannot_evaluate` where a safe result cannot be
  produced.
- A safe editor for a small closed set of canonical Markdown, JSON, and witness
  record formats.
- A reader and reconciler of machine output from tools that own other domains.
- Usable by humans, agents, Make targets, hooks, and CI.

Achta is not:

- A replacement for Rulefloor, Gograph, Git, Make, CI, or dedicated analyzers.
- A generic rule engine or arbitrary command framework.
- A system that judges the truth of prose, decisions, reasons, or attestations.
- A tool that silently derives a verification claim from repository state.
- A repository manager that commits, tags, pushes, creates releases, or rewrites
  history.
- A database-backed service, daemon, web service, or network control plane.
- An LLM-based interpreter, embedding store, semantic search engine, or memory
  system.

## 4. Core guarantees

Achta must provide these guarantees:

1. **No invented claims.** A repository HEAD can be measured, but a
   `verified_against` claim is written only when the caller explicitly attests
   that review occurred.
2. **No silent partial writes.** Every mutation is validated completely before
   replacement. A failed command leaves canonical files byte-identical.
3. **No implicit repository mutation.** Achta never invokes `git commit`,
   `git tag`, `git push`, `git reset`, `git clean`, or an equivalent operation.
4. **No shell execution.** External programs are invoked directly with argument
   vectors and bounded contexts.
5. **Path confinement.** Reads and writes stay within the canonical selected
   workspace and reject linked or special target files.
6. **Stable defaults.** Existing Markdown, amendment JSON, and witness record
   formats remain canonical and human-readable.
7. **Deterministic serialization.** JSON keys, arrays, manifest entries, and
   rendered Markdown changes use documented stable ordering.
8. **Bounded output and input.** Achta never emits whole unbounded records or
   prose fields as diagnostics.
9. **Explicit trust boundaries.** A syntactically valid attestation or witness
   record is not proof that its author was truthful or that commands ran.
10. **Compatibility before replacement.** An existing script is not retired
    until golden and end-to-end fixtures prove behavioral parity.

## 5. Ownership boundaries

| Capability | Owner | Achta relationship |
|---|---|---|
| Rule declaration and binding | Rulefloor | Consume stable machine output only |
| Rule sentence, proof, hash, and covered-symbol drift | Rulefloor | Reconcile declared amendments with `ledger-diff` output |
| Go symbol identity and reachability | Gograph | Never duplicate; consume only where a workspace gate explicitly needs it |
| Wiki frontmatter and derived blocks | Achta | Parse, validate, and update mechanically |
| Platform decision numbering and insertion | Achta | Parse and update mechanically; never choose decision content |
| Gate-witness records | Achta | Parse, validate, summarize, and eventually write |
| Amendment manifest lifecycle | Achta | Author explicit declarations, rebase, and reconcile |
| Product-specific wire, route, link, or clock checks | Dedicated analyzers | Keep external |
| Repository compilation and test suites | Make/CI | Achta may inspect records but must not become their runner |
| Agent prompt injection and source-navigation hooks | Agent harness | Keep external initially |

## 6. Canonical artifacts

Achta operates on existing artifacts. It does not replace them with private
storage.

### 6.1 Wiki repository pages

Repository pages live under `wiki/repos/` and include frontmatter fields such
as:

```text
updated: YYYY-MM-DD
verified: YYYY-MM-DD
verified_against: repository-name @ COMMIT (...optional existing prose...)
```

Some pages contain a generated block delimited by exact markers:

```text
<!-- BEGIN DERIVED: repository-name -->
...
<!-- END DERIVED -->
```

Within that block, a repository HEAD row is a measured fact. The frontmatter
verification pin is an attestation. Achta must preserve that distinction.

### 6.2 Decision register

The canonical decision register is Markdown containing stable decision IDs such
as `P-017`. Achta may allocate the next valid ID and insert caller-supplied
prose at the configured structural location. It must not synthesize the title,
reasoning, disposition, or consequences.

Before implementation, capture fixtures for the register's actual heading
order, insertion marker, and numbering rules. Do not infer ordering from a
single example.

### 6.3 Amendment manifest

The existing manifest remains JSON and human-readable:

```json
{
  "schema_version": "ledger-amendments.v1",
  "base_commit": "FULL_GIT_SHA",
  "changes": [
    {
      "rule_id": "RULE-ID",
      "change_class": "sentence_changed",
      "after_sentence_sha256": "FULL_SHA256",
      "reason": "A concise human-authored reason."
    }
  ]
}
```

Rule change classes mirror the current Rulefloor ledger-diff vocabulary:

- `rule_added`
- `rule_removed`
- `sentence_changed`
- `binding_changed`
- `proof_changed`
- `covered_symbols_changed`
- `test_fingerprint_changed`

Rulefloor header changes are separate evidence:

- `floor_changed`
- `red_proofs_changed`
- `repaired_fixtures_changed`

Header changes must never be silently converted into rule-level manifest
entries. The reconciliation result must report them explicitly.

### 6.4 Gate-witness records

Gate-witness records are line-oriented, human-readable records with a versioned
schema marker, repository identity, start and finish times, a declared plan,
per-target exit codes, optional elapsed measurements, a tree or commit tie, and
an overall result.

Achta must parse records strictly enough to distinguish:

- complete green;
- complete red;
- incomplete;
- stale against the selected repository;
- malformed;
- unsupported schema;
- cannot evaluate.

Missing timing information remains unknown. Achta must not fabricate durations,
timestamps, targets, or command execution.

## 7. Command-line design

### 7.1 General form

```text
achta [global options] COMMAND [SUBCOMMAND] [options]
```

Global options:

```text
--workspace PATH   Explicit workspace root.
--json             Emit the command's versioned JSON document where supported.
--quiet            Suppress non-problem human detail where supported.
--timing           Measure total command elapsed time and report it at the end.
--help             Show help without requiring a workspace.
```

`--help` and `-h` may appear before or after a command path. Both forms emit
the same workspace-independent top-level help and exit successfully.

`--quiet` suppresses successful human detail. It does not suppress evaluated
mismatches or error diagnostics, does not alter JSON documents, and does not
suppress the explicitly requested elapsed line when combined with `--timing`.

`--timing` starts a monotonic measurement immediately before global argument
parsing and stops after the command result has been constructed, just before
the timing value itself is rendered. Human success output ends with
`elapsed: Nms` on stdout; failed human output ends with that line on stderr.
For JSON output, the same single document gains an `elapsed_ms` integer field.
Without `--timing`, output remains byte-compatible and no elapsed field or line
is emitted. The timing measurement does not include writing the final timing
line or JSON field to the caller's output stream.

Workspace discovery may walk from the current directory toward the filesystem
root looking for the expected `wiki/` layout. Ambiguous discovery must fail and
request `--workspace`; it must never select a sibling or ancestor by guesswork.

The following commands must not require a workspace:

```text
achta version
achta version --json
achta capabilities
achta capabilities --json
```

### 7.2 Version

```text
achta version [--json]
```

The JSON schema is `achta.version.v1`. It reports both the release stamp and Go
toolchain module version, preserving disagreement rather than trusting a linker
flag alone.

Minimum fields:

```json
{
  "schema_version": "achta.version.v1",
  "version": "v0.1.0",
  "toolchain_version": "v0.1.0",
  "version_agreement": "pass"
}
```

Closed agreement values:

- `pass`
- `fail`
- `cannot_evaluate`

### 7.3 Capabilities

```text
achta capabilities [--json]
```

This is binary feature discovery only. It must not read the workspace, wiki,
Git repositories, manifests, or witness files.

The stable JSON schema is `achta.capabilities.v1`. It includes:

- schema and runtime version;
- supported machine interfaces;
- supported global options, including opt-in timing;
- public commands and subcommands;
- supported canonical artifact schemas;
- supported Rulefloor input schemas;
- whether commands read, write, execute an external program, or require Git;
- supported operating systems and architecture-independent limitations.

### 7.4 Wiki pin

```text
achta wiki pin REPOSITORY \
  --sha FULL_SHA \
  --verified YYYY-MM-DD \
  --attest-reviewed \
  [--check] [--json]
```

Behavior:

1. Resolve the canonical workspace, repository directory, wiki repository, and
   exactly one `wiki/repos/REPOSITORY.md` page.
2. Require `--attest-reviewed`. Its absence is a refusal, not an implicit yes.
3. Validate the full SHA and require it to equal the selected repository's HEAD
   by default.
4. Update both `verified:` and the SHA prefix of `verified_against:`.
5. Preserve any prose following the canonical `<repo> @ <sha>` prefix unless a
   separate future command explicitly replaces it.
6. Update the measured repository-HEAD value in the page's derived block using
   the same internal renderer used by `achta wiki derive`.
7. Validate the complete page after rendering.
8. Replace it atomically only if every check succeeds.
9. `--check` performs every validation and prints the proposed summary without
   writing.

The command must refuse:

- an abbreviated, malformed, missing, or non-HEAD SHA;
- a missing or duplicate frontmatter field;
- missing or duplicate derived markers;
- a repository/page identity mismatch;
- a linked page, linked parent component, or non-regular target;
- concurrent modification between read and replacement;
- dates not exactly `YYYY-MM-DD`;
- any attempt to append arbitrary prose as part of pinning.

The command proves only that an explicit attestation was recorded consistently.
It cannot prove that review actually occurred or that the page is true.

### 7.5 Wiki decision insertion

```text
achta decision add \
  --title TEXT \
  --body-file PATH \
  [--register wiki/platform/decisions.md] \
  [--prefix P] [--check] [--json]
```

Behavior:

1. Parse all decision headings using a closed structural grammar.
2. Allocate one more than the highest measured numeric ID for the selected
   prefix; physical order and numeric gaps do not change that result.
3. Require all measured headings for the prefix to belong to exactly one
   second-level section, then insert at that section's boundary.
4. Reject duplicates, malformed IDs, ambiguous sections, empty or multiline
   titles, and unsupported prefixes.
5. Read the body from one bounded, confined, non-secret regular file.
6. Insert one heading and body at the measured canonical location. The caller
   supplies the complete title, including any desired date; Achta does not add
   one.
7. Preserve unrelated bytes, LF or CRLF line endings, and final-newline
   behavior.
8. Validate the complete register and atomically replace it.
9. Return the allocated decision ID. `--check` returns exit 1 with
   `status=would_change` and leaves the register byte-identical.

Achta does not generate decision prose or decide whether a decision is correct.

### 7.6 Witness summary

```text
achta witness summarize \
  --repo PATH \
  --record PATH \
  [--require-head FULL_SHA] \
  [--slowest N] [--json]
```

Behavior:

- Parse and validate the witness schema before summarizing.
- Compare its repository/tree tie with the selected current repository.
- Report planned, recorded, passed, failed, and missing target counts.
- Report record start, finish, and wall elapsed when both valid timestamps are
  present.
- Report the sum of recorded target durations separately from wall elapsed.
- Report at most `N` slow targets; default 10, maximum 100.
- Detect impossible negative intervals and invalid or duplicate target timing.
- State freshness as `current`, `stale`, or `cannot_evaluate`.
- Never imply that a locally writable record proves commands actually ran.

The JSON schema is `achta.witness-summary.v1` and contains at minimum:

```json
{
  "schema_version": "achta.witness-summary.v1",
  "record_schema_version": "gate-run.v1",
  "status": "green",
  "completeness": "complete",
  "freshness": "current",
  "repository_head": "FULL_SHA",
  "recorded_head": "FULL_SHA",
  "planned_targets": 10,
  "recorded_targets": 10,
  "passed_targets": 10,
  "failed_targets": 0,
  "missing_targets": 0,
  "started_at": "RFC3339",
  "finished_at": "RFC3339",
  "wall_elapsed_ms": 1000,
  "recorded_target_elapsed_ms": 900,
  "slow_targets": []
}
```

Unknown time values must be omitted or explicitly null according to the final
schema fixture. They must never be written as zero if zero was not measured.

### 7.7 Amendment declaration

```text
achta amendments declare \
  --manifest PATH \
  --rule RULE-ID \
  --class CHANGE_CLASS \
  --reason-file PATH \
  [--after-sentence-sha256 FULL_SHA256] \
  [--check] [--json]
```

Behavior:

- Require an existing supported manifest with a full `base_commit`.
- Validate the closed change-class enum.
- Require a non-empty bounded one-line human reason.
- Require `after_sentence_sha256` exactly for `sentence_changed`.
- Require a lowercase 64-character SHA-256 value where applicable.
- Enforce uniqueness by `(rule_id, change_class)`.
- Sort entries deterministically by rule ID and then change class.
- Preserve the base commit and all unrelated valid entries.
- Serialize with stable indentation and a final newline.
- Refuse to infer declarations from the current ledger diff.

An optional future `--after-sentence-file` may compute the digest only after
its byte canonicalization is proven identical to Rulefloor's published parsed
sentence contract. Do not duplicate the Rulefloor Markdown parser merely to add
that convenience.

### 7.8 Amendment rebase

```text
achta amendments rebase \
  --manifest PATH \
  --repo PATH \
  --witness-log PATH \
  [--check] [--json]
```

Behavior:

- Reproduce the existing accepted-witness baseline-selection semantics exactly.
- Resolve the selected witness to a full commit SHA.
- Update only `base_commit`.
- Never add, remove, clear, or rewrite `changes` entries.
- Refuse an ambiguous, missing, unaccepted, malformed, or unreachable witness.
- Refuse a no-op unless `--check` is being used for confirmation.

Before replacing the current helper, capture real-history fixtures for first
commit after a witness, multiple witness commits, merges, malformed subjects,
and no-previous-witness cases.

### 7.9 Amendment reconciliation

```text
achta amendments reconcile \
  --manifest PATH \
  --repo PATH \
  [--rulefloor PATH] \
  [--json]
```

Achta must not parse `RULE-FLOOR.md` or recreate Rulefloor's logical diff.
Instead it must:

1. Execute `rulefloor capabilities --json` directly, without a shell.
2. Require the expected stable machine interface and sentence-digest feature.
3. Execute `rulefloor ledger-diff --base FULL_SHA --repo PATH --json`.
4. Capture stdout and stderr separately with strict byte limits and a timeout.
5. Require one complete `rulefloor.ledger-diff.v1` JSON document.
6. Reject `cannot_evaluate`, invalid JSON, schema mismatch, truncation, exit and
   status disagreement, or a resolved base differing from the manifest.
7. Reconcile in both directions:
   - every actual rule change must be declared;
   - every declared change must exist in the actual diff;
   - every sentence declaration must match `after_sentence_sha256`;
   - header changes must be reported and handled by explicit policy rather than
     silently ignored.

The reconciliation document reports the absolute selected Rulefloor executable
as `rulefloor_executable`. Executable discovery is completed before either
Rulefloor invocation; failures are `cannot_evaluate`.

Exit 0 means the manifest and measured logical ledger diff agree. It does not
mean the human reason is true or the ledger amendment is correct.

The stable JSON schema is `achta.amendments-reconciliation.v1`.

### 7.10 Clearing amendment declarations

There is no unconditional `amendments clear` command in the initial release.

Blind clearing could erase declarations before they are witnessed and turn an
undeclared ledger change into the next cycle's baseline. If a future command is
added, it must require evidence that:

- reconciliation passed;
- the referenced witness is complete and green;
- the witness carries the expected base and diff digest;
- the witness commit is reachable from the current history;
- no new ledger change appeared after that witness.

Until those checks exist, clearing remains an explicit reviewed edit.

### 7.11 Wiki status and derivation commands

```text
achta wiki freshness [--strict] [--repo NAME] [--json]
achta wiki derive [--check | --print REPOSITORY] [--json]
achta wiki unpushed [--repo PATH] [--json]
achta wiki check [--json]
```

`wiki freshness` compares each repository page's anchored canonical pin with
the corresponding local repository HEAD. It never fetches or writes. Without
`--strict`, measured drift is reported but does not fail the command;
`--strict` maps drift to exit 1. Unreadable or ambiguous evidence is
`cannot_evaluate` and exit 2.

`wiki derive --check` renders every marked derived block from local facts and
returns exit 1 when any page would change. Without `--check`, each page is
validated and replaced independently and atomically. `--print` renders exactly
one repository block without editing a page and is mutually exclusive with
`--check`.

`wiki unpushed` compares HEAD with the configured local upstream tracking ref.
It deliberately performs no fetch and records that limitation in JSON. Its
closed statuses are `published`, `unpushed`, `behind`, and `diverged`.

`wiki check` composes strict freshness and derived-block checks, exposes both
results separately, and preserves `cannot_evaluate`; a green aggregate never
hides a skipped or unevaluated component.

The stable schemas are `achta.wiki-freshness.v1`, `achta.wiki-derive.v1`,
`achta.wiki-unpushed.v1`, and `achta.wiki-check.v1`.

### 7.12 Witness lifecycle commands

Achta records explicitly supplied observations but does not execute gate
commands:

```text
achta witness init --repo PATH --record PATH --label TEXT --targets A,B
achta witness step --repo PATH --record PATH --target NAME --exit-code N \
  [--elapsed-ms N] [--evidence-file PATH]
achta witness finalize --repo PATH --record PATH [--commit-tie]
achta witness check --repo PATH --record PATH
```

`init` requires a repository clean except for the selected witness path and
records the exact HEAD and plan. When replacing an existing record, `init`
first requires that record to be a valid supported witness. `step` appends one planned target's observed
exit code, optional measured milliseconds, and bounded evidence lines; it never
runs that target. `finalize` writes green only when every planned target was
recorded with exit zero and binds the record to the current tree digest by
default. `--commit-tie` is allowed only for a completely clean repository.
`check` requires a complete, green, current record created from a clean
repository.

All lifecycle mutations refuse a stale recorded HEAD, repository changes
outside the witness path, duplicate or unplanned targets, and concurrent
record changes. The stable schemas are `achta.witness-operation.v1` and
`achta.witness-check.v1`.

There is intentionally no `witness run`. Make or CI owns command execution and
passes measured outcomes to `witness step`. Adding an arbitrary executor would
contradict Achta's no-shell, non-runner boundary and would make a locally
writable witness appear stronger than it is.

### 7.13 Slice check

```text
achta slice check --repo PATH [--commits N] [--entries N] [--ahead N] [--json]
```

The command audits local Git and wiki evidence for an already landed slice. It
checks a clean tree, upstream availability and ancestry, total commits ahead,
slice author and committer identity, absence of agent identities and
secret-like paths, module-boundary changes, expected `log.md` heading appends,
and a wiki pin equal to HEAD. `--commits` and `--entries` default to one;
`--ahead` optionally requires an exact total. It performs no fetch and no Git
mutation. Every component is returned separately using
`achta.slice-check.v1`; inability to measure a required fact is exit 2 rather
than a pass.

## 8. Exit codes

All commands use one central contract:

- `0`: requested operation or evaluation succeeded.
- `1`: evaluation completed and found a mismatch, refusal, stale record, red
  witness, unapproved change, or check-mode difference.
- `2`: invalid arguments, malformed or unsupported input, unavailable required
  tool, unsafe path, ambiguous repository, truncation, timeout, or another
  cannot-evaluate condition.

Commands must not silently reinterpret exit 2 as an evaluated mismatch.

For `--json`, stdout contains exactly one versioned JSON document for all three
exit classes. Human diagnostics go to stderr only when they cannot be expressed
in that document. Successful JSON commands write no prose to stderr.

## 9. Security and filesystem model

### 9.1 Workspace confinement

- Resolve the workspace with `filepath.Abs`, `filepath.Clean`, and
  `filepath.EvalSymlinks` before use.
- Require the canonical workspace to be a real directory.
- Every recognized input and output path must remain beneath it unless a
  command explicitly documents a read-only external executable path.
- Reject any write target with a linked path component, linked final entry,
  special file, directory target, or hard-to-classify filesystem state.
- Resolve repository roots through Git only after directory confinement.

### 9.2 Safe replacement

For every mutation:

1. Read a bounded regular file and record its identity, mode, size, and digest.
2. Parse and validate the complete document.
3. Render the complete replacement in memory with a maximum size.
4. Reparse and validate the rendered replacement.
5. Create a private temporary regular file in the same directory.
6. Write, sync, and preserve the intended file mode.
7. Recheck that the source file has not changed since step 1.
8. Rename within the same directory.
9. Sync the containing directory where supported.

No command should mutate more than one canonical file unless it implements a
documented transaction or can prove that the files are independently safe.

### 9.3 Subprocesses

- Never invoke a shell.
- Use `exec.CommandContext` with explicit arguments.
- Apply timeouts and bounded stdout/stderr buffers.
- Validate executable discovery and report which binary was selected without
  printing environment variables.
- Git discovery failures identify the requested Git executable, and bounded Git
  execution failures identify the selected absolute executable. Amendment
  reconciliation reports its selected absolute Rulefloor executable on success.
- Every Git invocation uses `--no-optional-locks` so read-only measurements do
  not refresh repository index metadata.
- Do not accept arbitrary command strings from manifests, wiki prose, or
  witness records.
- Git operations are read-only unless a future command separately documents a
  narrowly scoped mutation. No initial command needs mutating Git operations.

### 9.4 Secrets and diagnostics

- Never read `.env`, key, certificate, credential, token, cookie, license, or
  other secret-like files.
- Reject secret-like paths as amendment reason/body inputs.
- Bound and sanitize errors from external commands.
- Do not print entire Markdown documents, reasons, witness output, environment,
  or Git configuration in an error.
- JSON encoding must escape control characters and prevent terminal injection.

## 10. Determinism and concurrency

- Sort manifest changes by rule ID and change class.
- Sort summary target ties by target name after elapsed duration.
- Use RFC 3339 UTC for newly generated machine timestamps.
- Preserve existing timestamps rather than rewriting them opportunistically.
- Preserve source line endings unless the command explicitly owns the complete
  document format and migration tests approve normalization.
- Detect concurrent modification through the original file digest and metadata
  immediately before replacement.
- Never resolve a branch name where a command requires an exact immutable SHA.
- Human output ordering must be stable for identical semantic inputs.
- Elapsed output is intentionally nondeterministic and appears only when the
  caller explicitly supplies `--timing`.

## 11. Go architecture

The codebase should start small but keep command rendering separate from domain
logic:

```text
achta/
  cmd/achta/
  cmd/release-notes/
  internal/cli/
  internal/workspace/
  internal/safefile/
  internal/gitstate/
  internal/wiki/
  internal/decision/
  internal/witness/
  internal/amendments/
  internal/rulefloorclient/
  internal/releasenotes/
  internal/slicecheck/
  testdata/
    machine/
    wiki/
    witness/
    amendments/
```

Do not create empty packages. Introduce each package only when executable code
and tests need the boundary.

Responsibilities:

- `cmd/achta`: process entry point and runtime version stamp only.
- `cmd/release-notes`: exact extraction of one version section for release
  automation.
- `internal/cli`: argument parsing, help, exit mapping, and human/JSON rendering.
- `internal/workspace`: discovery and canonical workspace/repository selection.
- `internal/safefile`: confined regular-file reads and atomic replacement.
- `internal/gitstate`: read-only Git SHA, ancestry, upstream, and status facts.
- `internal/wiki`: frontmatter, derived-block, freshness, and page validation.
- `internal/decision`: decision ID allocation and structural insertion.
- `internal/witness`: strict record model, parser, validation, and summary.
- `internal/amendments`: manifest model, deterministic writes, and two-way
  reconciliation.
- `internal/rulefloorclient`: bounded direct execution and strict parsing of
  Rulefloor's stable machine interfaces.
- `internal/releasenotes`: bounded version-heading validation and extraction.
- `internal/slicecheck`: read-only landed-slice checks over Git and wiki facts.

Domain packages must return typed results and errors. They must not depend on
stdout, stderr, terminal formatting, or process exit codes.

Use only the Go standard library unless a dependency has a compelling measured
correctness or security benefit. Markdown operations should use narrow
format-specific parsers rather than a general Markdown rendering dependency.

## 12. Configuration

The initial release should prefer conventions plus explicit flags over a new
configuration format.

Default workspace layout:

```text
WORKSPACE/
  wiki/
    repos/
    platform/decisions.md
    tools/
  REPOSITORY-A/
  REPOSITORY-B/
```

If configuration later becomes necessary, use one bounded, versioned JSON file
at the workspace root. Do not introduce YAML solely for configuration, and do
not allow configuration to define arbitrary executable commands.

## 13. Compatibility and migration

### 13.1 Compatibility principles

- Existing wiki Markdown stays canonical.
- Existing `ledger-amendments.v1` stays canonical.
- Existing `gate-run.v1` records remain readable.
- Existing Make targets and scripts continue working during migration.
- Achta never requires a bulk rewrite merely to adopt the binary.
- No existing script is deleted in the same slice that first introduces its
  Achta replacement.

### 13.2 Script disposition

| Existing utility family | Disposition |
|---|---|
| Wiki freshness, derivation, and unpushed checks | Implemented in `achta wiki`; keep old scripts during measured shadow parity |
| New wiki pin editor | Implement directly in Achta |
| New decision insertion helper | Implement directly in Achta |
| Gate-witness parsing, summary, and writer | Implemented as summarize plus explicit init/step/finalize recording; external gates still execute commands |
| End-to-end witness freshness | Implemented by `achta witness check` |
| Slice postcheck | Implemented by `achta slice check`; no fetch or mutation |
| Close-condition, count-claim, ledger-claim, and model-mirror checks | Candidates for explicit `achta wiki check` components |
| Amendment gate and authoring | Migrate to `achta amendments` using Rulefloor machine output |
| Repository green build/test gate | Keep outside; Achta is not a generic test runner |
| Ledger census versus source graph | Keep as a composition gate; Rulefloor and Gograph retain ownership |
| Rulefloor installation-route gate | Keep outside; it audits workflow/distribution policy |
| Route, link, inert-parameter, clock, and wire analyzers | Keep dedicated |
| Prompt inclusion, prompt locks, commit hook, and source-navigation hook | Keep in the agent harness initially |

### 13.3 Parity procedure

For every migrated script:

1. Freeze representative green, red, malformed, missing, stale, and
   cannot-evaluate fixtures.
2. Run the old script and record exit code plus bounded normalized output.
3. Run Achta against the same fixtures.
4. Explain and approve every intentional difference.
5. Run both implementations in shadow mode in the workspace gate.
6. Replace callers only after repeated agreement.
7. Keep a temporary compatibility wrapper that calls Achta with direct
   arguments, not through a shell command string.
8. Remove the old implementation in a later explicit slice.

## 14. Machine-interface requirements

Every stable JSON interface must have an exact golden fixture under
`testdata/machine/`.

Requirements:

- One JSON document on stdout.
- No banners, progress, or human prose mixed into JSON stdout.
- Deterministic field and array ordering where semantically unordered.
- Closed enums validated on encode and decode.
- Bounded strings, arrays, and document size.
- Unsupported schema versions return exit 2.
- Optional additions must not change the meaning of existing fields.
- Breaking changes require a new schema version.
- Timestamps are omitted or explicitly unknown when unavailable; never
  fabricated.
- When global `--timing` is present, every JSON success or error document adds
  the optional non-negative integer field `elapsed_ms` without emitting a
  second document. Untimed documents remain byte-compatible.

Schemas shipped in v0.1.0:

- `achta.version.v1`
- `achta.capabilities.v1`
- `achta.wiki-pin.v1`
- `achta.witness-summary.v1`
- `achta.amendments-operation.v1`
- `achta.amendments-reconciliation.v1`

Schemas added for v0.2.0:

- `achta.decision-add.v1`
- `achta.wiki-freshness.v1`
- `achta.wiki-derive.v1`
- `achta.wiki-unpushed.v1`
- `achta.wiki-check.v1`
- `achta.witness-operation.v1`
- `achta.witness-check.v1`
- `achta.slice-check.v1`

Do not publish a schema until the corresponding implementation and conformance
tests are complete.

## 15. Testing requirements

### 15.1 Unit tests

Add focused tests for:

- workspace discovery and explicit-root precedence;
- ambiguous workspace discovery;
- repository name and path validation;
- regular-file and symlink confinement;
- path traversal and linked-parent rejection;
- concurrent modification refusal;
- atomic replacement and mode preservation;
- complete byte preservation outside owned regions;
- frontmatter missing, duplicated, reordered, and malformed fields;
- `verified_against` prefix replacement with suffix preservation;
- derived marker missing, duplicated, nested, and mismatched;
- full-SHA and date validation;
- required explicit review attestation;
- decision numbering, duplicates, gaps, and ambiguous insertion markers;
- decision body size, newline, and control-character handling;
- witness green, red, incomplete, stale, malformed, and unsupported records;
- missing and impossible witness timestamps;
- wall versus summed-target elapsed calculations;
- deterministic slow-target ordering and output bounds;
- every amendment change class;
- sentence digest required only for sentence changes;
- duplicate amendment keys and deterministic manifest order;
- rebase preserving the complete change array byte-for-byte semantically;
- two-way reconciliation, extra declarations, missing declarations, digest
  mismatch, header change, truncation, exit/status mismatch, and unavailable
  Rulefloor;
- strict single-document JSON and clean stderr on success;
- opt-in timing as the final human line and as `elapsed_ms` in the same JSON
  document, including failure output;
- version disagreement and unavailable Go build information;
- capabilities independence from the current directory and workspace files.
- exact golden encoding for every advertised stable machine interface.

### 15.2 Integration tests

Use temporary Git repositories and fixture workspaces. Tests must not access the
network, real user configuration, credentials, or production repositories.

Cover complete workflows:

1. Review-attested wiki pin dry run, write, freshness validation, and no-op
   detection.
2. Decision allocation and insertion with deterministic Git-diff-friendly
   output.
3. Witness parse, current summary, repository change, and stale summary.
4. Amendment declare, rebase, Rulefloor logical diff, successful reconciliation,
   undeclared change, and declared-but-absent change.
5. External binary missing, wrong schema, noisy stdout, oversized output,
   timeout, and nonzero exit behavior.
6. Failure after staging but before replacement leaves canonical bytes
   unchanged.

Use fake executables for Git and Rulefloor argument-vector tests where
appropriate, while retaining a small end-to-end suite against installed
compatible binaries.

Run the five required parser fuzz targets reproducibly with `make test-fuzz`;
`FUZZTIME` may be overridden without changing which targets execute.

### 15.3 Fuzz tests

Fuzz the narrow parsers for:

- wiki frontmatter;
- derived block delimiters;
- decision headings;
- gate-witness records;
- amendment JSON.

Fuzz assertions: no panic, no unbounded allocation, no path escape, and no
successful parse of structurally ambiguous input.

## 16. Performance requirements

Achta works on small text artifacts and should be fast without caching.

- Do not add a database or generated cache.
- Parse each selected canonical file at most once per command where practical.
- A wiki-wide read-only check should complete comfortably within one second on
  the current workspace, excluding external process time.
- Mutation commands should touch only their selected artifact.
- Witness summaries must remain linear in record size and bounded by the input
  size limit.
- Performance diagnostics, if later added, must be opt-in and must not alter
  normal deterministic output.

## 17. Validation and CI

The repository must provide a repo-local `Makefile` with at least:

```text
make verify
make install-tools
```

`make verify` must run:

```text
go version
go build ./...
go test ./... -count=1 -timeout=120s
go vet ./...
staticcheck ./...
govulncheck ./...
go mod tidy -diff
gograph build . --precise
rulefloor check --repo . --run-profile unit --timings
```

It must also fail on unformatted Go files. `RULE-FLOOR.md` is the canonical
repository-local invariant ledger. The default gate uses Rulefloor execute mode
so ledger syntax, bindings, tests, fingerprints, and red-proof requirements are
enforced together; a static-only check is diagnostic and is not a substitute
for `make verify`. New release-critical behavior must add or deliberately amend
an armed rule with a measured red proof. CI uses the Go version from `go.mod`
and runs the complete repository-local verification gate.

For every Go implementation slice:

- run current Gograph capability discovery;
- build a precise graph before structural work;
- use Gograph planning before edits and review after edits;
- report the Gograph session audit;
- run the complete `make verify` gate;
- run `git diff --check`;
- confirm no dependency, `replace`, or `go.work` was introduced unintentionally.

Unit tests must not make real network, DNS, Docker, browser, database, or
credential-backed calls.

## 18. Distribution

Achta is a private project. Its source, CI logs, release artifacts, checksums,
and documentation must remain within access-controlled locations. Nothing in
this specification authorizes making the repository or its artifacts public.

Initial development may use `go run` or a local build from the private
repository. Once the command and machine schemas are stable enough for shared
use by authorized workspace users:

- publish checksum-addressed binaries to private GitHub releases for Linux and
  macOS on amd64 and arm64;
- build with `CGO_ENABLED=0`;
- use GoReleaser or an equivalently reproducible checked-in release workflow;
- publish `checksums.txt`;
- stamp the release version while also reporting Go build metadata;
- publish an `achta` Homebrew cask to `ozgurcd/homebrew-tap` after the private
  GitHub release succeeds; installation still requires authorized access to
  the private release assets through `HOMEBREW_GITHUB_API_TOKEN`;
- allow Homebrew to evaluate the cask during update and discovery when the
  installer token is absent, without weakening authenticated asset downloads;
- rewrite every generated browser download URL to the unique numeric GitHub
  release-asset API URL assigned after publication, retain the binary Accept
  and bearer-token headers, and refuse missing, duplicate, or foreign URLs;
- use GoReleaser's current `homebrew_casks` support rather than its deprecated
  `brews` configuration, verify tap push access before creating the release,
  refuse tap downgrades, and verify the published cask.

The tool must remain buildable without third-party runtime services.

### 18.1 Release documentation

`RELEASE_NOTES.md` is the detailed, user-facing source for release bodies. It
contains newest-first sections whose headings are exactly
`## vMAJOR.MINOR.PATCH — YYYY-MM-DD`. Every change is recorded under the version
that first ships it; historical version sections are not rewritten to describe
later work.

`CHANGELOG.md` is the concise newest-first historical summary. It may omit
implementation detail but must not disagree with the release notes.

The checked-in `cmd/release-notes` helper validates an exact semantic version,
requires exactly one matching heading, and extracts only that section. The
release workflow uses the tag as the requested version and passes the extracted
temporary file to GoReleaser. A missing, duplicate, or malformed version
section fails the release before publication.

## 19. Delivery phases

Phase status is a measured project fact, not a roadmap implication. The
current v0.2.0 development state is:

| Phase | Status | Evidence |
|---|---|---|
| 0 | complete in v0.1.0 | foundations, safe-file checks, machine fixtures, and repository verification gate |
| 1 | complete in v0.1.0 | attested wiki pin and witness summary integration workflows, including dry-run/no-op/stale transitions and exact human and JSON fixtures |
| 2 | complete in v0.1.0 | amendment authoring/rebase, strict Rulefloor client, complete CLI reconciliation, bidirectional/header-change failures, and real-history witness selection tests |
| 3 | verified for v0.2.0 | on 2026-09-04, repeated real-workspace old/native cycles agreed on 10 fresh pages and six unchanged derived blocks; the latest local-unpushed facts agreed at 13; decision check allocated P-058 without a write; malformed-heading and derived-delimiter fuzzing, failure fixtures, and exact machine contracts pass |
| 4 | verified for v0.2.0 | two independent parser/writer lifecycle cycles and a landed-slice integration workflow pass; `witness run` is rejected by the product boundary |

### Phase 0: inventory and frozen fixtures

- Create the Go module and repository-local verification gate.
- Inventory exact current formats and script exit semantics.
- Capture sanitized golden fixtures from current wiki pages, manifests, and
  witness records.
- Implement workspace, safe-file, Git-state, CLI, version, and capabilities
  foundations.

Exit criterion: foundation tests and verification are green; no existing tool
or caller changes.

### Phase 1: highest-pain read/write paths

- Implement `achta wiki pin`.
- Implement `achta witness summarize`.
- Add exact human and JSON fixtures.
- Run in manual shadow mode without replacing existing scripts.

Exit criterion: repeated real-workspace dry runs agree with current facts and
no unrelated bytes move.

### Phase 2: amendments

- Implement `amendments declare` and `amendments rebase`.
- Implement the strict Rulefloor client and two-way reconciliation.
- Shadow the existing amendment gate.
- Do not implement unconditional clearing.

Exit criterion: green/red/cannot-evaluate parity across real-history fixtures
and at least two complete witness cycles.

### Phase 3: decision insertion and wiki checks

- Implement `decision add`.
- Migrate freshness, derived-block, and selected wiki structural checks behind
  explicit Achta subcommands.
- Preserve compatibility wrappers.

Exit criterion: old and new gates agree for multiple cycles, including their
failure fixtures.

### Phase 4: witness writer and slice checks

- Use the proven parser for init/step/finalize/check recording operations.
- Keep command execution in Make or CI; do not add `witness run`.
- Migrate slice-postcheck and end-to-end witness freshness to read-only Achta
  checks.
- Retire old scripts only in later explicit changes.

Exit criterion: no duplicate parser remains without a documented reason, and
canonical record bytes remain compatible.

## 20. MVP acceptance criteria

The first usable Achta release is accepted only when:

- `achta version --json` and `achta capabilities --json` work outside a
  workspace.
- `wiki pin` updates exactly the intended page regions and requires explicit
  review attestation.
- A failed pin leaves the page byte-identical.
- `witness summarize` removes all manual duration arithmetic and detects stale,
  incomplete, malformed, and red records.
- Amendment declaration and reconciliation eliminate direct `jq`/Python edits
  for the supported workflow without inventing change intent.
- Existing canonical files require no migration.
- Existing scripts remain available during the compatibility period.
- All machine outputs match exact fixtures and contain one JSON document.
- `--timing` reports total command elapsed time at the end without changing
  untimed output or producing a second JSON document.
- Missing Git or Rulefloor produces `cannot_evaluate`, never a crash or silent
  pass.
- The full Go verification gate and security-focused filesystem tests pass.
- Public documentation states precisely what Achta can and cannot establish.

## 21. Explicit deferred work

The following are not part of the initial implementation:

- A database, daemon, web UI, or remote service.
- Arbitrary plugins or user-defined policy expressions.
- Automatic prose, decision, reason, or amendment generation.
- Semantic evaluation of wiki truth.
- Automatic commits, tags, pushes, releases, or pull requests.
- Generic execution of repository test commands.
- Replacement of product-specific analyzers.
- Rulefloor ledger parsing inside Achta.
- Gograph source analysis inside Achta.
- Unconditional amendment clearing.
- A public embedding API.

## 22. v0.1.0 decisions

The v0.1.0 release records these choices explicitly:

1. `decision add` remains deferred to Phase 3. Its exact register allocation
   and insertion policy must be measured and approved before implementation;
   v0.1.0 does not advertise the command or its JSON schema.
2. `wiki pin` preserves the complete existing `verified_against` suffix and
   replaces only the canonical repository-and-SHA prefix.
3. Maximum inputs are 16 MiB for wiki pages, 8 MiB for witness records and
   external Rulefloor output, 4 MiB for amendment manifests, 64 KiB per
   witness line, and 4 KiB for a one-line amendment reason.
4. The machine schemas advertised by `achta capabilities` are stable in
   v0.1.0. Breaking changes require a new schema version.
5. Phase 2 parity uses Rulefloor v0.9.1 capability discovery and
   `rulefloor.ledger-diff.v1`, checked in both directions against explicit
   declarations.
6. Unconditional amendment clearing and post-witness finalization remain
   deferred; v0.1.0 cannot erase declaration intent.

## 23. v0.2.0 decisions

The v0.2.0 development line records these choices explicitly:

1. `decision add` is enabled with measured max-plus-one allocation and
   section-boundary insertion; its schema is first advertised in v0.2.0.
2. Wiki freshness, derivation, unpushed, and composed checks are native but do
   not authorize retiring compatibility scripts without repeated shadow
   agreement.
3. Witness init/step/finalize/check record caller-supplied observations. A
   `witness run` executor is rejected because Achta is not a generic task
   runner.
4. `slice check` is a read-only postcondition audit and performs no fetch or Git
   mutation.
5. `RULE-FLOOR.md` is enforced in execute mode by `make verify`.
6. Release bodies come only from the exactly matching version section in
   `RELEASE_NOTES.md`; `CHANGELOG.md` remains the concise history.
7. Homebrew uses the shared `ozgurcd/homebrew-tap` cask path. GoReleaser
   generates the cask after the private GitHub release; the workflow replaces
   browser download URLs with authenticated release-asset API URLs, verifies
   tap access before release creation, refuses a version downgrade, and
   authorized installers must retain access to the private release assets.

## 24. Success measure

Achta succeeds when agents and humans stop writing one-off scripts for the same
workspace bookkeeping, while every claim remains explicit and every canonical
artifact remains reviewable in Git.

The measure is not the number of scripts deleted. It is fewer malformed edits,
fewer stale witnesses and baselines, fewer working-directory mistakes, and no
loss of the fail-closed safeguards those scripts currently provide.
