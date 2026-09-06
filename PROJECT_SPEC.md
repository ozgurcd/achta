# Achta Project Specification

Status: v0.5.5 release specification

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
small Go binary. It owns the mechanical integrity of those
workspace artifacts. It does not decide whether their human claims are true.

Achta complements existing tools rather than absorbing them:

- Rulefloor owns invariant declarations, test/check bindings, red-proof
  observations, test fingerprints, covered-symbol bindings, and ledger drift.
- Gograph owns Go source structure and static graph evidence.
- Dedicated analyzers own their particular source or protocol comparisons.
- Achta owns workspace bookkeeping, declared-change reconciliation, witness
  record handling, exact document-mirror comparison, and deterministic wiki
  maintenance.

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
    until its own selftest fixtures have been replayed against both the script
    and the proposed Achta replacement, with matching exit codes recorded for
    every fixture in a workspace-confined replay artifact whose exact SHA-256
    is pinned by the claim. The gate opens that artifact and reconciles its
    complete fixture set; an opaque citation string alone is an attestation,
    not proof. A named replacement or retirement claim without checkable
    evidence is an evaluated defect.

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

Repository pages live under `wiki/repos/`. A page owned by a separate workspace
wiki includes an anchored repository SHA:

```text
updated: YYYY-MM-DD
verified: YYYY-MM-DD
verified_against: repository-name @ COMMIT (...optional existing prose...)
```

A page committed in the repository it describes instead uses:

```text
updated: YYYY-MM-DD
verified: YYYY-MM-DD
co_versioned: true
```

It must omit `verified_against:`. A Git commit cannot contain its own final SHA;
the code and page already share one atomic commit boundary. `verified:` remains
the explicit claim-review date. A co-versioned marker outside the repository's
own wiki is malformed and must fail closed.

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
per-target exit codes, optional elapsed measurements, a tree or commit tie,
optional parsed sibling-repository pins, optional CI provenance, and an overall
result.

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

### 6.5 Prompt-part lock

Achta owns the lock artifact it writes:

```text
# achta.parts-lock.v1
VERSION vN

name.txt LOWERCASE_SHA256
```

The artifact contains one unique record for every direct `.txt` part, ordered
bytewise by filename. The schema marker, version grammar, filename grammar,
single-space separator, lowercase digest, LF line endings, ordering, and final
newline are canonical rather than caller-configurable. A workspace adopting
the command migrates its former comment dialect to this format.

The lock establishes exact bytes and closed-set membership. Whether a part
names a live section of some canonical document is not part of this artifact:
the document selection, heading grammar, and reference semantics belong to the
caller.

### 6.6 Replacement-claim manifest

`replacement-claims.json` is the canonical place where this repository may
claim that an Achta verb replaces or retires a named script. Achta owns its
strict `achta.replacement-claims.v2` JSON shape: a schema marker and ordered
`claims`, each naming a `verb`, `script`, and caller-chosen `status`.

The words that assert replacement or retirement remain caller vocabulary and
are passed to the checker as repeatable `--claim-status` values with no
default. An entry carrying one of those statuses must also contain non-empty
`selftest_citation` and `replay_citation` fields and at least one named fixture
with both `script_exit` and `verb_exit`. `replay_citation` names a confined
`achta.replacement-replay.v1` artifact and `replay_sha256` pins its exact bytes.
The two manifest exit codes must agree for every fixture, and the manifest's
complete fixture set, exits, verb/script identity, and selftest citation must
match the selected record inside the replay artifact. Candidate entries need
no evidence and make no replacement claim.

The replay artifact is reviewed, co-versioned evidence. It records the
read-only source document's path, commit, and SHA-256 as provenance, followed
by strict verb/script records and per-fixture exits. Achta cannot read a sibling
workspace at repository verification time, so it does not pretend that an
external path is a gate. It verifies the vendored bytes and their claim pin;
it still does not execute either implementation or prove that the replay's
author ran what the artifact says. Legacy `achta.replacement-claims.v1` remains
parseable for candidates, but a v1 replacement status fails as an uncheckable
attestation and must migrate to v2.

## 7. Command-line design

### 7.1 General form

```text
achta [global options] COMMAND [SUBCOMMAND] [options]
```

Global options:

```text
--workspace PATH   Explicit workspace root.
--wiki-dir PATH    Explicit workspace wiki directory.
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
request an explicit selector; it must never select a sibling or ancestor by
guesswork. `--workspace PATH` selects the workspace root. `--wiki-dir PATH`
selects only the canonical direct `wiki` child of that root and requires the
same `repos/` and `platform/decisions.md` layout because commands may resolve
repositories beside the wiki. The two selectors are mutually exclusive. A
repository-owned wiki uses the same layout at the repository root. When both
the repository and a parent are candidates, callers can select the repository
directly with `--wiki-dir ./wiki`.

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

`wiki pin` applies to externally pinned repository pages. A co-versioned page
does not carry a SHA field to rewrite; it is reviewed and committed with the
source it describes.

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
achta wiki check [--only NAME[,NAME...]] [--json]
```

`wiki freshness` compares each externally owned repository page's anchored
canonical pin with the corresponding local repository HEAD. A self-owned page
is fresh only when it carries `co_versioned: true`, omits `verified_against:`,
has a review date, and its selected workspace root is the exact Git repository
root. It never fetches or writes. Without `--strict`, measured drift is reported
but does not fail the command;
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
hides a skipped or unevaluated component. `--only NAME[,NAME...]` selects the
checks to evaluate from `freshness` and `derive`: the selected checks are the
only ones evaluated and the only ones in `checks[]`, in canonical order, and
the exit contract applies unchanged to that selection. The selection is a set,
never a per-check exit code. An empty, unknown, or repeated name is invalid
input: exit 2 and nothing is evaluated. Without `--only`, every check runs. The
JSON document also carries `wiki_dir`, the wiki directory Achta actually
resolved, because a pass against the wrong wiki would otherwise be silent.

The stable schemas are `achta.wiki-freshness.v1`, `achta.wiki-derive.v1`,
`achta.wiki-unpushed.v1`, and `achta.wiki-check.v1`.

### 7.12 Witness lifecycle commands

Achta records explicitly supplied observations but does not execute gate
commands:

```text
achta witness init --repo PATH --record PATH --label TEXT --targets A,B
achta witness step --repo PATH --record PATH --target NAME --exit-code N \
  [--elapsed-ms N] [--evidence-file PATH]
achta witness finalize --repo PATH --record PATH [--commit-tie] \
  [--sibling NAME=REPOSITORY]... \
  [--ci-run URL --ci-attempt N --ci-sha FULL_SHA]
achta witness check --repo PATH --record PATH \
  [--sibling NAME=REPOSITORY]... [--no-reach PATTERN=WHY]... \
  [--sibling-no-reach NAME:PATTERN=WHY]...
achta witness earned --repo PATH --record PATH
```

`init` requires a repository clean except for the selected witness path and
records the exact HEAD and plan. When replacing an existing record, `init`
first requires that record to be a valid supported witness. `step` appends one planned target's observed
exit code, optional measured milliseconds, and bounded evidence lines; it never
runs that target. `finalize` writes green only when every planned target was
recorded with exit zero and binds the record to the current tree digest by
default. `--commit-tie` is allowed only for a completely clean repository.
Repeated `--sibling` flags add parsed, name-singleton, clean sibling pins with
full HEAD and complete tree SHA-256 values. CI provenance requires all four of
`--commit-tie`, `--ci-run`, `--ci-attempt`, and `--ci-sha`; its SHA must equal
the commit tie.

`check` requires a complete green record created from a clean repository. A
record is `current` when its original tie still matches. If a pinned repository
has advanced along the recorded commit's ancestry, exact changed paths may be
classified against explicit no-reach declarations. A completely excluded diff
is `proven_no_reach`, neither stale nor current, and may pass; any unmatched or
unknown path is `REQUIRED` and the witness remains stale. Catch-all declarations
such as `*`, `**`, and `**/*` are invalid. The JSON result records every changed
path and the exact declaration and reason used for each exclusion.

A CI-provenance record passes only when it is green, complete, commit-tied,
clean at initialization and finalization, tied to the recorded CI SHA, and that
commit remains on local HEAD ancestry. CI validation performs no fetch or
network request and states that limitation. `witness earned` compares HEAD with
the newest accepted witness commit and returns `REFUSE` when the intervening
committed diff consists only of `GATE-RUN*.txt`, `ledger-amendments.json`, and
`MINT-STATE.json` record paths.

All lifecycle mutations refuse a stale recorded HEAD, repository changes
outside the witness path, duplicate or unplanned targets, and concurrent
record changes. The stable schemas are `achta.witness-operation.v1` and
`achta.witness-check.v1`. Earned-cycle output uses
`achta.witness-earned.v1`.

There is intentionally no `witness run`. Make or CI owns command execution and
passes measured outcomes to `witness step`. Adding an arbitrary executor would
contradict Achta's no-shell, non-runner boundary and would make a locally
writable witness appear stronger than it is.

### 7.13 Slice check

```text
achta slice check --repo PATH [--log-dir RELATIVE_PATH] \
  [--commits N] [--entries N] [--ahead N] [--json]
```

The command audits local Git and wiki evidence for an already landed slice. It
checks a clean tree, upstream availability and ancestry, total commits ahead,
slice author and committer identity, absence of agent identities and
secret-like paths, module-boundary changes, expected log heading appends, and
wiki freshness. A repository-owned `wiki/log.md` takes precedence over a root
`log.md`. The optional `--log-dir` has no default and names a repository-relative,
non-linked directory whose regular files are read recursively in bytewise
filename order after the discovered log file. Existing filenames must remain an
exact prefix of that order, and the prior headings in every existing file must
remain an exact prefix within that file; inserting a lexically earlier file,
removing or reordering a file, or inserting a heading before a prior heading
fails. Appended headings across the file and directory together must equal
`--entries`. An explicitly selected directory that is absent, linked, unreadable,
or overlaps the discovered log file is `cannot_evaluate`; the log check skips
only when no log file exists and `--log-dir` was not supplied. The caller owns
the directory name and layout vocabulary, so Achta does not auto-detect a
conventional `log/` directory. A co-versioned wiki page is tied by the containing
commit rather than an impossible self-SHA. `--commits` and `--entries` default to
one; `--ahead` optionally requires an exact total. It performs no fetch, shell
invocation, or Git mutation. Every component is returned separately using
`achta.slice-check.v1`; inability to measure a required fact is exit 2 rather
than a pass.

### 7.14 Reachability classification

```text
achta reachability classify --repo PATH --base COMMIT \
  [--no-reach PATTERN=WHY]... [--json]
```

The command resolves the base locally, requires it on HEAD ancestry, inventories
the exact committed path diff, and returns `REQUIRED` when any path lacks a
declared no-reach match. `SKIPPABLE` is returned only when every changed path is
excluded. The result is the audit record; the command writes no marker file, so
committing its output cannot make an internal state marker self-invalidating.
Exit 0 means `SKIPPABLE`, exit 1 means `REQUIRED`, and exit 2 means the decision
could not be safely evaluated.

### 7.15 Toolchain parity

```text
achta toolchain check --repo PATH --manifest PATH --workflow PATH [--json]
```

The bounded `achta.toolchain-manifest.v1` JSON document declares version pins
and SHA-256 script pins using explicit CI environment keys. The check compares
each declaration with the workflow's top-level `env` value and recomputes every
declared script digest from a confined, non-secret regular file. Empty manifests,
missing CI pins, duplicate names or environment keys, digest mismatches, and
paths outside the repository fail closed. The manifest cannot define commands;
the check executes no external process. Output uses
`achta.toolchain-parity.v1`.

### 7.16 Recipe check

```text
achta recipe check --makefile PATH --target NAME [--expect-line S]... [--expect-file PATH] [--forbid-noop] [--json]
```

`recipe check` reads one Makefile target's recipe as TEXT — the physical lines
that begin with a tab after `NAME:` up to the next non-recipe line, comment
lines skipped — and refuses NEUTRALIZERS: edits that keep a gate looking
present while make ignores its exit. A line beginning `-` after the tab,
before or after `@`, is `ignored-exit`; a pipe is `pipe` (a `||` is not a
pipe); a line ending `&` is `background`; `|| true`, `|| :`, `; true` or
`exit 0` is `swallowed-exit`; and with `--forbid-noop` a command whose first
word is `true`, `:`, `echo` or `printf` is `noop`. Every `--expect-line` and
every line of `--expect-file` must EQUAL some recipe line byte for byte,
without the leading tab; substring or prefix matching is the defect this
command exists for, so anything appended must fail. The measured cause is
that macOS make 3.81 silently ignores `.SHELLFLAGS := -o pipefail -c`
(GNU Make 4.3 honours it), leaving a static rule as the only guard on that
toolchain. Exit 1 on any violation or absent expectation; a missing or
duplicated target, an empty recipe, or an unreadable makefile is
`cannot_evaluate` and exit 2. The makefile and expect-file are confined to
the workspace. Output uses `achta.recipe-check.v1`.

### 7.17 Ledger census

```text
achta ledger census --file PATH [--dir [LABEL=]PATH]... [--tree [LABEL=]PATH]... [--ext EXT] [--json]
```

`ledger census` recounts a markdown ledger table against ITSELF and against
disk. A row is any table line whose first cell is an integer row number and
whose third cell is an integer size; its second cell names the subject and
its last cell's first word is the mark. A totals row is any three-cell line
whose label is not a number and whose other two cells are integers; the key
is the label's first word, `MARK` or `total`, prefixed `LABEL-` for a
labelled source. Per source, per mark, the row count and size sum must equal
the totals row; the `total` row must equal every row summed; a totals row for
a mark with no rows must read zero; subjects are unique. Against disk, a
`--dir` source censuses every regular file directly under PATH (size = line
count) and a `--tree` source every subdirectory (size = summed lines of files
with `--ext` under it, recursive); a present-marked row must exist at EXACTLY
the stated size, a `RETIRED` row must NOT exist, and every entry on disk must
have a row. Rows are assigned to a `--dir` source when their subject is a bare
file name and to a `--tree` source when it starts with the source directory's
base name and a slash; a row matching no source is a violation. The measured
cause is a row added with the totals untouched that passed every gate a
workspace had. Exit 1 on any violation; no sources, an unreadable source, or
no rows at all is `cannot_evaluate` and exit 2. Nothing is written. Output
uses `achta.ledger-census.v1`.

### 7.18 Floor census

```text
achta floor census --file PATH --repo PATH --bucket A,B,... --covered NAME --fence-heading TEXT
                   [--marker C] [--count-sum RE] [--count-frozen RE] [--count-plain RE]
                   [--allow-prefix TEXT] [--covers PATH] [--rulefloor PATH] [--json]
```

`floor census` recounts a fenced completeness census against itself and
against Rulefloor's covers map. The census vocabulary is the CALLER'S: the
bucket tokens that lead a row, the bucket whose rows must cite a rule, the
heading under which the fenced table sits, an optional single-character
marker that a citation may carry to mark the frozen arm, optional stated-count
patterns (each a regexp with exactly one `(\d+)` capture) and an optional
allowlist line prefix. None of these has a default; a required one missing is
invalid input, exit 2. Achta owns only the mechanics: a row is
`BUCKET N Symbol file:line reason…` inside the fence; a citation is a token in
Rulefloor's rule-ID grammar, optionally suffixed by the marker; a covers
document is `rulefloor covers --json --repo PATH` (rulefloor.covers.v1),
executed as an argument vector with no shell, exactly as `amendments reconcile`
executes Rulefloor, or read from `--covers PATH` for fixtures and replay.

Violations, by class: (1) a row in a non-covered bucket whose reason names a
rule the covers document knows; (2) a covered row citing a rule the covers
document does not know; (3) a covered row with no citation; (4) a citation
repeated within one row; (5) a line outside the fence that begins with a bucket
token; (6) a stated count the table contradicts — `- BUCKET: N` per bucket,
and the sum, frozen and plain counts where a pattern was given; (7) a
qualified covers entry whose census rows at that `(file, symbol)` identity
never cite the rule, or an unqualified covers entry; (8) a covers absence — no
census row at that identity — not on the allowlist. Rulefloor absent, exiting
non-zero, or answering non-JSON is `cannot_evaluate`, exit 2, and the resolved
executable is named. REFUSED, and reported in the `refused` field: whether a
cited rule is ARMED — that needs RULE-FLOOR.md column parsing, which is
Rulefloor's format; Achta consumes Rulefloor's machine output only. Output
uses `achta.floor-census.v1`: bucket counts, frozen/plain split, covers stats
(rules, mapped, absences, allowlisted, new), the executable, violations
`[{line, class, text}]`, and the refused list.

### 7.19 Mirror check

```text
achta mirror check --master PATH [--digest FILE [--digest-name NAME]]
  [--mirror PATH]... [--json]
```

`mirror check` reads one master plus at least one explicitly selected mirror or
one explicit digest file within the resolved workspace. It computes SHA-256
over exact file bytes and evaluates mirrors in caller-supplied order. Each
mirror reports `match`, `differs`, or `absent`; comparable files report both
the master and mirror digests.

When `--digest FILE` is present, the file uses canonical lowercase
`sha256<two ASCII spaces>name` records. By default Achta selects the one record
whose name equals the master's basename byte-for-byte. `--digest-name NAME`
instead selects that exact record name, including a caller-owned path-shaped
name such as `tools/gate.sh`; it neither resolves nor normalizes the name.
Achta reports the digest file, line, name, recorded hex, computed master hex,
and `match` or `differs`. Multiple records are permitted only when exactly one
matches the selected name; zero or duplicate matches are ambiguous and
`cannot_evaluate`.

There is no newline, whitespace, digest-case, path, encoding, content, or
semantic normalization.
An absent mirror or a well-formed digest disagreement is an evaluated mismatch,
exit 1. Any absent master or digest file, malformed or ambiguous digest
evidence, or linked or unsafe input provides no trustworthy comparison and is
always `cannot_evaluate`, exit 2.

The global `--wiki-dir WORKSPACE/wiki` selector retains `WORKSPACE` as the
confinement root, so the master may be a workspace-root document while mirrors
sit under the wiki. Linked, special, oversized, unreadable, or escaping paths
are `cannot_evaluate`. The command invokes no external process and performs no
Git or filesystem mutation. Output uses `achta.mirror-check.v1`.

### 7.20 Parts lock and verification

```text
achta parts lock   --dir DIR --lock FILE [--bump] [--json]
achta parts verify --dir DIR --lock FILE [--json]
```

`parts lock` reads every direct `.txt` regular file beneath the explicit,
workspace-confined directory and writes the explicit lock artifact atomically.
Names are unique and bytewise filename-sorted; SHA-256 is computed over exact
bytes without newline, whitespace, encoding, or semantic normalization. A new
lock starts at `VERSION v1`. Rewriting an existing canonical lock preserves its
version; `--bump` requires an existing lock and increments the positive decimal
version exactly once. An unchanged render does not replace the file.

`parts verify` parses only canonical `achta.parts-lock.v1`, recomputes the same
exact digests, and reports every locked name as `match`, `differs`, or `absent`,
followed by every neighboring unlocked `.txt` file as `unlocked`. It reports
the lock version and both locked and on-disk counts. Exit 0 means the complete
set matches. An edited, absent, or unlocked part is evaluated drift, exit 1.
A missing or malformed lock, missing or unsafe directory, linked or special
part, bound violation, or concurrently changing lock/directory is
`cannot_evaluate`, exit 2.

The commands enumerate only direct `.txt` files, accept no format-shaping
flags, invoke no external process, perform no Git operation, and never inspect
part prose for canonical-document section references. Output uses
`achta.parts-operation.v1` for writes and `achta.parts-verify.v1` for checks;
the artifact schema is `achta.parts-lock.v1`.

### 7.21 Ledger row rules

```text
achta ledger rows --file MD --id-cell N --prose-cell N
  --open-marker TEXT --closed-marker TEXT --identity-end-marker TEXT
  --completion-marker TEXT [--completion-marker TEXT]...
  [--exempt-marker TEXT]...
  --condition-marker TEXT [--condition-marker TEXT]...
  --quote-marker TEXT [--json]
```

`ledger rows` evaluates two bounded structural relationships across selected
cells in Markdown table rows. The ID and prose cells are explicit, positive,
one-based indexes. The open and closed markers are distinct exact prefixes of
the trimmed ID cell; the exact identity end marker terminates the row identity.
Every marker is caller-supplied and has no default.

First, an open row whose selected prose contains any exact
`--completion-marker` is a violation unless that prose also contains an exact
`--exempt-marker`. Achta does not infer synonyms or decide that prose means the
work is complete. Second, a closed row whose prose contains any exact
`--condition-marker` must carry exactly one `--quote-marker`, followed by one
non-empty double-quoted string. The quoted bytes must appear verbatim elsewhere
in that same prose cell after the marker and its quote are removed. No case,
whitespace, punctuation, Unicode, or semantic normalization is performed.

Each violation reports `line`, closed rule name, row `identity`, and bounded
`text`. Exit 0 means no structural violations; exit 1 means at least one rule
fired; missing vocabulary, malformed or duplicate identities, no matching
rows, unsafe or unreadable input, and ambiguous markers are
`cannot_evaluate`, exit 2. The result explicitly refuses natural-language
completion inference and condition-satisfaction truth. It uses
`achta.ledger-rows.v1`, reads only the selected workspace-confined regular
file, invokes no external process, performs no Git operation, and writes
nothing.

### 7.22 Count relationship check

```text
achta count check --file MD [--file MD]... [--dir DIR]...
  [--total-pattern REGEX --part-pattern REGEX]
  [--claim-pattern REGEX]... [--citation-pattern REGEX]
  [--claim-target-pattern REGEX]...
  [--claim-assertion-pattern REGEX]...
  [--claim-section-pattern REGEX] [--exempt-pattern REGEX]...
  [--proof-claim-pattern REGEX]... [--proof-pattern REGEX]...
  [--proof-within-lines N]
  [--count-alias TOKEN=N]... [--json]
```

`count check` evaluates caller-classified structural relationships over an
explicit Markdown document set. At least one complete relationship is
required. Every total, part, claim, proof-claim, and proof pattern has exactly
one named `count` capture; other captures are permitted. Claim and proof
patterns are repeatable and additive. Unsigned decimal tokens are intrinsic.
Every non-decimal count token must be declared byte-for-byte by a repeatable
`--count-alias TOKEN=N`; no claim, proof, citation, section, exemption, or
breakdown vocabulary has a default.

For the breakdown pair, each line with exactly one total match and at least one
part match is a breakdown. The part counts must sum without overflow to the
stated total. More than one total on a line cannot evaluate. For the citation
relationship, Markdown blank lines delimit paragraphs. Every configured claim
match in a paragraph must have at least its count of distinct exact full
matches of the citation expression in that paragraph. Citation text is masked
before claim matching so digits inside citations do not create claims.

The optional paragraph-composition relationship requires both repeatable
`--claim-target-pattern` and repeatable `--claim-assertion-pattern` families.
Every target pattern has one named `count` capture; assertion patterns have no
count capture requirement. If at least one expression from each family matches
the same blank-line-delimited paragraph, the smallest positive target count in
that paragraph becomes one citation-backed claim. Zero targets are ignored.
This is generic composition over caller vocabulary, not inference: target
nouns and assertion verbs have no defaults, and a pattern fitted to one fixture
is not part of the contract.

`--claim-section-pattern` restricts citation claims to the last `## ` section
whose heading matches that caller expression; no matching section yields zero
citation claims. Each `--exempt-pattern` classifies one marker line and exempts
the next nonblank paragraph from citation and proof-claim checks. The proof
relationship compares each caller-classified proof claim with the nearest
caller-classified proof count within the positive `--proof-within-lines`
distance; a tie keeps the earlier proof, and no nearby proof leaves the claim
unevaluated rather than inventing evidence. No case, whitespace, punctuation,
Unicode, path, citation, proof, or document-content normalization occurs.

Repeatable `--file` inputs are evaluated first in caller order. Repeatable
`--dir` inputs then contribute their direct `.md` or `.MD` regular files in
bytewise filename order, preserving directory flag order. Empty, unsafe,
linked, oversized, or unreadable selections, repeated documents, incomplete or
ambiguous patterns, and invalid aliases are `cannot_evaluate`, exit 2. A
structural disagreement is exit 1 and a clean evaluation is exit 0. The stable
`achta.count-check.v1` result reports file and aggregate counts, then every
violation's file, line, rule, claimed value, observed value, and bounded text.

Accepted: decimal arithmetic, exact configured matching, caller-selected last
section scope, caller-classified exemption, nearest bounded proof comparison,
same-paragraph target/assertion composition, smallest-positive target
selection, paragraph scope, and distinct citation counting. Refused and reported:
deciding whether unconfigured prose makes a count claim, whether a matched
citation proves a disposition, or whether matched proof prose actually proves
a claim. The command reads only confined regular files, invokes no external
process or shell, performs no Git operation, and writes nothing.

### 7.23 Declared-route check

```text
achta declared-route check [--file WORKFLOW]... [--dir DIR]...
  --route-pattern REGEX [--route-pattern REGEX]...
  --ban-pattern REGEX [--ban-pattern REGEX]...
  --required-route-pattern REGEX --required-key KEY
  --required-scope JSON_POINTER
  [--route-cardinality per-file-one|per-file-any] [--json]
```

`declared-route check` evaluates an explicit set of single-document YAML
workflow files. Repeatable files retain caller order; repeatable directories
contribute their direct lowercase `.yml` and `.yaml` regular files in bytewise
filename order. By default, each file must contain exactly one match occurrence
across all caller-supplied route alternatives. Opt-in `--route-cardinality
per-file-any` permits zero or more route occurrences independently in each
file, while every selected file is still scanned for every banned pattern.
Matching is performed on parsed scalar values; YAML comments and scalar lines
whose first non-space byte is `#` are excluded. Patterns
must not be empty, repeated, invalid, capable of matching empty text, or longer
than 4 KiB; each pattern family is capped at 64 expressions and 4,096 matches
per document. No route or banned-command vocabulary is built in.

`--required-route-pattern` must equal exactly one configured route alternative.
When that alternative occurs at least once, `--required-scope` is an RFC 6901
JSON Pointer to a YAML mapping and `--required-key` is the exact caller-owned
key that must appear once in that mapping and exactly once across the entire
YAML document. Other route alternatives do not require the key. Scope is
resolved from YAML mapping nodes, never indentation or a line-level
approximation. Thus `/jobs/verify/env` is structurally distinct from
workflow-level `/env`, and a second job-level declaration cannot coexist with
the required workflow-level declaration. A missing or repeated conditionally
required key is an evaluated violation.

A comment-only or whitespace-only YAML file is a valid evaluated document with
zero live scalars. It can pass under `per-file-any`; under the default
`per-file-one` it fails the route-count rule at exit 1. Malformed or
multi-document YAML, a non-mapping or ambiguously repeated scope, unsafe input,
or incomplete vocabulary is `cannot_evaluate`, exit 2.

The stable `achta.declared-route-check.v1` result preserves file order and
reports each route and banned pattern's count and source lines, the selected
scope and exact key count, and every violation's file, line, rule, expected
count, observed count, and bounded text. Exit 0 means all relationships hold
and exit 1 means at least one evaluated disagreement. The command reads only
workspace-confined regular files, invokes no external process or shell,
performs no Git operation, and writes nothing.

Accepted: YAML syntax and mapping scope, deterministic direct-directory
selection, exact caller-pattern counting, explicit per-file cardinality, and
comment exclusion. Refused and reported: default route, banned-command, token,
or scope vocabulary and line-level inference of YAML scope.

### 7.24 Replacement parity check

```text
achta replacement check --file MANIFEST
  --claim-status STATUS [--claim-status STATUS]... [--json]
```

`replacement check` parses canonical `achta.replacement-claims.v2` and legacy
v1, then evaluates entries whose exact status matches a caller-supplied claim
status. For every v2 verb-to-named-script claim it requires a selftest
citation, a confined `achta.replacement-replay.v1` artifact, its exact SHA-256,
and matching script and verb exit codes for the complete fixture set. The gate
opens the artifact, selects exactly one matching verb/script record, verifies
the selftest citation, and reconciles every row in both directions. Missing
evidence, a digest disagreement, an exit disagreement, a contradictory row,
or an open fixture set is exit 1. A v1 claim status is also exit 1 because its
opaque citation is only an attestation. Missing files, malformed or unsupported
JSON, repeated identities or fixtures, and invalid exit codes are
`cannot_evaluate`, exit 2. Clean evidence is exit 0.

Achta refuses to classify natural-language release prose as a replacement
claim, consistent with P-063: that is meaning inference. Instead, the
structured v2 manifest is the only authoritative replacement/retirement claim
surface and `make verify` checks it against co-versioned replay evidence. The command is read-only,
workspace-confined, invokes neither shell nor Git, and writes nothing.

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
  internal/earned/
  internal/reachability/
  internal/toolchain/
  internal/amendments/
  internal/rulefloorclient/
  internal/releasenotes/
  internal/slicecheck/
  internal/mirror/
  internal/parts/
  internal/ledgerrows/
  internal/countcheck/
  internal/declaredroute/
  internal/replacement/
  testdata/
    machine/
    wiki/
    witness/
    amendments/
    ledgerrows/
    countcheck/
    declaredroute/
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
- `internal/earned`: pure record-only versus substantive witness-cycle
  classification.
- `internal/reachability`: pure fail-closed path/no-reach classification.
- `internal/toolchain`: bounded manifest, workflow-pin, and script-digest
  parity checks.
- `internal/amendments`: manifest model, deterministic writes, and two-way
  reconciliation.
- `internal/rulefloorclient`: bounded direct execution and strict parsing of
  Rulefloor's stable machine interfaces.
- `internal/releasenotes`: bounded version-heading validation and extraction.
- `internal/slicecheck`: read-only landed-slice checks over Git and wiki facts.
- `internal/mirror`: pure exact-byte SHA-256 comparison, strict recorded-digest
  selection, and deterministic per-mirror and digest results.
- `internal/parts`: canonical prompt-part lock parsing/rendering, version
  transitions, exact-byte digests, and closed-set verification.
- `internal/ledgerrows`: caller-shaped Markdown row parsing, exact literal
  state/prose relationships, and verbatim closure-quote verification.
- `internal/countcheck`: caller-shaped count and proof patterns, exact decimal
  and alias parsing, breakdown arithmetic, caller-selected last-section scope,
  exemptions, bounded proof comparison, and paragraph-scoped citation counts.
- `internal/declaredroute`: real YAML mapping-scope resolution plus exact
  caller-pattern route, ban, per-file cardinality, and required-key counts.
- `internal/replacement`: strict claim and replay artifact parsing, SHA-256
  evidence pinning, and closed-set per-fixture exit reconciliation.

Domain packages must return typed results and errors. They must not depend on
stdout, stderr, terminal formatting, or process exit codes.

Use only the Go standard library unless a dependency has a compelling measured
correctness or security benefit. True YAML scope is that measured exception:
use the stable `go.yaml.in/yaml/v3` node parser rather than reproducing YAML
with indentation heuristics. `go.yaml.in/yaml/v3 v3.0.5` is Achta's first
external Go module dependency; before it, the module was standard-library-only.
Markdown operations should use narrow
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

Repository-owned wiki layout:

```text
REPOSITORY/
  wiki/
    repos/REPOSITORY.md  # co_versioned: true; no verified_against
    platform/decisions.md
    log.md
```

Repository-specific facts must have exactly one wiki owner. A local page and a
central page for the same repository are a configuration error, not mirrors.

If configuration later becomes necessary, use a bounded, versioned JSON file.
The repo-local `achta.toolchain-manifest.v1` is the first such format. Do not
introduce YAML solely for configuration, and do not allow configuration to
define arbitrary executable commands.

## 13. Compatibility and migration

### 13.1 Compatibility principles

- Existing wiki Markdown stays canonical until ownership is explicitly moved.
- A repository ownership move transfers its page, decisions, and history; it
  does not leave a central mirror.
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
| Gate-witness parsing, summary, and writer | Implemented as summarize plus explicit init/step/finalize recording, parsed sibling pins, and CI provenance; external gates still execute commands |
| End-to-end witness freshness | Implemented by `achta witness check`, including explicit no-reach classification |
| Gate-witness shell copies | Keep during measured shadow parity; retire only in a later explicit slice after callers and failure semantics agree |
| Slice postcheck | Implemented by `achta slice check`; no fetch or mutation |
| Close-condition and ledger-claim checks | Candidate implementation: `achta ledger rows`; no named-script replacement or retirement claim is recorded |
| Count-claim and breakdown-sum checks | Candidate implementation: `achta count check`; the prior replacement claim was retracted because its cited replay records fixture `g` as script 1 / verb 0 |
| Byte-identical master/mirror and recorded-master-digest checks | Candidate implementation: `achta mirror check`; `replacement-claims.json` records no replacement claim until script-side parity evidence exists |
| Prompt-part lock writing and exact closed-set verification | Implemented by `achta parts lock` and `achta parts verify`; caller migrates its lock and gates in a later explicit parity slice |
| Amendment gate and authoring | Migrate to `achta amendments` using Rulefloor machine output |
| Repository green build/test gate | Keep outside; Achta is not a generic test runner |
| Ledger census versus source graph | Keep as a composition gate; Rulefloor and Gograph retain ownership |
| Rulefloor installation-route gate | Candidate implementation: `achta declared-route check`; the prior replacement claim was retracted because its cited replay records five exit-code differences |
| Vulnerability fix-availability policy | Keep in a dedicated vulnerability analyzer; Achta does not own advisory or fix semantics |
| Route, link, inert-parameter, clock, and wire analyzers | Keep dedicated |
| Prompt inclusion, canonical-section reference checking, commit hook, and source-navigation hook | Keep in the agent harness; the caller owns heading vocabulary and hook wiring |

Replacement language can be written in `PROJECT_SPEC.md` (principle 10 and
this disposition table), `CHANGELOG.md`, `RELEASE_NOTES.md` and its generated
GitHub release body, `README.md`, and the repository wiki's current facts,
decisions, and append-only log. Those prose surfaces are explanatory and may
describe features or migration candidates, but they are not machine-classified
because doing so would infer meaning. `capabilities` advertises command and
schema existence only and has no replacement/parity field, so it is not a
replacement-claim surface. Only a matching-status record in
`replacement-claims.json` is an authoritative named-script replacement or
retirement claim, and the repository gate rejects it unless its v2 row cites
and digest-pins confined replay evidence whose complete fixture set agrees.

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

Before step 6 or 8 is claimed, record the script's own selftest-fixture
citation and every fixture's two exit codes in a reviewed
`achta.replacement-replay.v1` artifact, commit that artifact inside Achta, and
pin its exact SHA-256 plus the same complete rows in
`replacement-claims.json`; `make replacement-check` must pass.

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

Schemas added for v0.3.0:

- `achta.reachability.v1`
- `achta.toolchain-parity.v1`
- `achta.witness-earned.v1`

Schemas added after v0.4.1:

- `achta.recipe-check.v1`
- `achta.ledger-census.v1`

Schemas added after v0.4.2:

- `achta.floor-census.v1`

Schemas added after v0.4.4:

- `achta.mirror-check.v1`

Schemas added for v0.5.0:

- `achta.parts-operation.v1`
- `achta.parts-verify.v1`
- `achta.ledger-rows.v1`

Schemas added for v0.5.1:

- `achta.count-check.v1`
- `achta.declared-route-check.v1`

Schemas added for v0.5.3:

- `achta.replacement-check.v1`

Canonical artifact schemas added for v0.5.0:

- `achta.parts-lock.v1`

Canonical artifact schemas added for v0.5.3:

- `achta.replacement-claims.v1`

Canonical artifact schemas added for v0.5.5:

- `achta.replacement-claims.v2`
- `achta.replacement-replay.v1`

Do not publish a schema until the corresponding implementation and conformance
tests are complete.

## 15. Testing requirements

### 15.1 Unit tests

Add focused tests for:

- workspace discovery, explicit-root precedence, and explicit wiki-directory
  selection;
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
- capabilities independence from the current directory and workspace files;
- exact golden encoding for every advertised stable machine interface;
- sibling-pin singleton parsing, dirty/current/stale/no-reach transitions, and
  CI provenance ancestry checks;
- fail-closed reachability, catch-all rejection, exact skip evidence, record-only
  witness-cycle refusal, and toolchain pin/digest parity.
- exact master/mirror byte equality, one-byte and whitespace differences,
  absent mirrors, absent masters, a workspace-root master selected through
  `--wiki-dir`, matching and tampered recorded digests, one-byte master drift,
  absent digest files, exact-basename and explicit exact-name selection, and
  ambiguous digest records.
- canonical part-lock parsing/rendering, initial and bumped versions, exact
  digest agreement, edited and absent locked parts, unlocked neighboring parts,
  malformed locks, unsafe files, and concurrent-change refusal.
- caller-shaped ledger rows where each structural violation fires, clean rows
  remain silent, unconfigured semantic synonyms remain outside evaluation, and
  closure quotes must repeat exact bytes.
- caller-shaped count documents where an incorrect breakdown and insufficient
  distinct citations each fire, a correct breakdown and cited claim stay
  clean, repeated claim patterns remain additive, last matching section scope
  and caller exemptions are honored, a nearby proof mismatch fires, a counted
  target plus an assertion in the same paragraph uses the smallest positive
  target count, and incomplete vocabulary cannot evaluate.
- declared-route workflow sets where default exact-one and opt-in per-file-any
  cardinality are distinct, every route-free file is still ban-scanned, each
  route-using file requires its scoped key, direct directories are bytewise
  ordered, and job-level keys cannot satisfy workflow-level scope.
- replacement manifests where candidate statuses remain non-claims, claimed
  named scripts require a selftest citation plus a confined digest-pinned replay
  artifact, exact closed-set rows pass, contradictory, missing, extra, or
  digest-mismatched rows fail, legacy v1 claims remain attestations, and unknown
  or malformed fields cannot evaluate.

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
7. Part lock creation, explicit version bump, matching verification, edited
   part drift, and unlocked neighboring part drift.
8. Ledger-row clean, open-completion, absent closure-quote, and paraphrased
   closure-quote fixtures through the public 0/1/2 command contract.
9. Count-check clean, bad breakdown, citation undercount, missing input, and
   direct Markdown-directory selection.
10. The frozen v0.5.1 count, declared-route, and mirror replacement claims all
    fail because their v1 citations are uncheckable attestations.
11. The frozen v0.5.4 count and declared-route claim rows fail in all six places
    where their pinned vendored replay records different exits.

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
make replacement-check
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
- create annotated semantic-version tags and make the release workflow refuse
  a lightweight tag before validation or publication;
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

A release is complete only after its source commit is pushed, its tag and
private release are published, the matching Homebrew cask is committed and
pushed to the tap, and that cask is installed through Homebrew. Completion is
proved by the installed binary reporting the release version and advertising
the newly shipped capability; a local commit or green build is not a release.

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
- A repository-owned wiki can pass freshness and slice-log checks without a
  self-referential commit SHA.
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
- Vulnerability advisory and published-fix evaluation.
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

## 24. v0.3.0 decisions

The v0.3.0 release records these choices explicitly:

1. Achta owns one parsed `xrepo:` grammar and rejects duplicate sibling names;
   the three byte-pinned shell implementations remain compatibility callers
   until a later measured shadow-parity slice explicitly retires them.
2. Witness staleness has three outcomes: `current`, `proven_no_reach`, and
   stale. No-reach is an explicit path declaration, rejects catch-all patterns,
   records exact excluded paths, and fails closed on every unmatched path.
3. CI provenance is locally validated evidence, not a remote CI query. Checks
   require green, complete, clean, commit-tied records on local HEAD ancestry
   and never fetch.
4. Earned-cycle checking is a separate read-only gate. Achta still has no
   `witness run` and never executes the recorded targets.
5. Toolchain parity uses a bounded declarative JSON manifest and fixed version
   and digest comparisons; manifests cannot contain executable commands.
6. Published-fix vulnerability policy remains outside Achta because advisory,
   ecosystem, and fix-availability semantics belong to a dedicated analyzer.

## 25. v0.4.0 decisions

1. Repository-owned wikis use `co_versioned: true` and omit
   `verified_against:`. The containing commit is the version boundary.
2. `wiki/log.md` takes precedence over a repository-root `log.md` during slice
   checks so locally owned history is mechanically enforced.
3. `--wiki-dir PATH` directly selects a repository-owned wiki while preserving
   the workspace root needed for sibling repository resolution. The selected
   path must be the canonical direct `wiki` child, and it is mutually exclusive
   with `--workspace`.
4. Repository-specific facts have exactly one wiki owner; migration removes the
   former central page, decision, and log records instead of retaining mirrors.

## 26. v0.4.5 decisions

1. A missing master is always `cannot_evaluate`, exit 2. Without reference bytes
   equality has no truth value; adding an allowance flag would preserve the
   silent fail-open this command is intended to remove.
2. A missing mirror is an evaluated mismatch, exit 1: once the master exists,
   absence conclusively means the named mirror is not an identical copy.
3. Mirror equality is SHA-256 over exact file bytes. Achta refuses newline,
   whitespace, encoding, or semantic normalization.

## 27. v0.5.0 decisions

1. Achta owns `achta.parts-lock.v1` because `parts lock` creates the artifact;
   the format is not shapeable through flags. This differs from P-063 census
   input: the census already belongs to the caller, while the lock is an Achta
   output and shared contract.
2. “Each part names a live section of a canonical document” remains outside
   Achta. The caller owns the canonical document, heading vocabulary, and
   reference semantics; Achta establishes exact bytes and closed-set membership
   only.
3. The conversion batch ships together on the v0.5.0 source line. A minor
   version marks the caller-facing command expansion and gives migrations one
   released baseline rather than another patch-level slice.
4. Ledger-row status and prose vocabulary remains caller-owned: cell indexes,
   open/closed ID prefixes, identity terminator, completion, exemption,
   condition, and quote markers are explicit flags with no defaults.
5. Achta accepts the structural half of completion claims and closure quotes.
   It refuses natural-language completion inference and condition-satisfaction
   truth; a configured literal is either present or absent, and a quoted string
   is either repeated byte-for-byte or it is not.

## 28. v0.5.1 decisions

1. A digest record is evidence about the selected master only when its filename
   equals the master's basename byte-for-byte. A single differently named
   record is `cannot_evaluate`, not a digest mismatch, because it claims a
   different artifact.
2. A multi-record file is accepted only when every line is canonical and
   exactly one record names the master basename. Selecting that unique exact
   match is deterministic; zero or duplicate matches are ambiguous and exit 2.
3. Digest text is canonical lowercase hexadecimal, two ASCII spaces, then a
   non-empty filename. Achta refuses case folding, whitespace repair, line
   ending conversion, path normalization, and all master-content
   normalization. A valid recorded digest unequal to the computed digest is an
   evaluated failure, exit 1.
4. Count checking owns only mechanics: unsigned decimal arithmetic, exact
   aliases, line-scoped breakdown sums, blank-line paragraph scope, and
   distinct exact citation matches. The caller supplies every total, part,
   claim, and citation pattern with no default.
5. The citation half is accepted only after the caller's claim expression has
   made the prose classification explicit. Achta refuses to infer English fix
   claims or decide that a matched citation proves a disposition; those are
   meaning judgements, not count mechanics.
6. Document selection is explicit. Files retain caller order; caller-named
   directories contribute direct Markdown files in deterministic filename
   order. No conventional log path is auto-detected.

## 29. v0.5.2 decisions

1. Claim/proof parity remains structural only after caller classification.
   Repeatable proof-claim and proof patterns, an explicit line distance, a
   caller-selected last-section pattern, and caller-patterned exemptions have
   no defaults. Achta compares counts and locations but still refuses to decide
   that prose makes a claim or that proof text is true or relevant.
2. Route cardinality is a per-workflow decision. The compatibility default
   stays exactly one route per file; explicit `per-file-any` permits zero or
   many route uses in each file. A required key is still exactly once in every
   file using the designated route, and bans are evaluated in every selected
   file regardless of route count. Direct directory selection is deterministic
   and does not auto-detect workflow layout.
3. A digest record name is opaque caller vocabulary, not a filesystem path.
   Basename selection remains the default for compatibility; `--digest-name`
   selects one exact record including path-shaped names without path, case,
   whitespace, or content normalization. Zero or duplicate exact records
   cannot evaluate.
4. v0.5.2 is a patch because it adds compatibility flags and closes parity
   gaps in three existing stable verbs without removing commands, changing
   defaults, or changing their 0/1/2 and schema contracts.

## 30. v0.5.3 decisions

1. Achta refuses to infer from prose that a command replaces or retires a
   script. The classification remains caller-owned as repeatable exact
   `--claim-status` values with no default, consistent with P-063.
2. `achta.replacement-claims.v1` is the sole authoritative named-script claim
   surface. Claim statuses require both citation fields plus one or more
   per-fixture script/verb exit pairs; `make verify` runs the check so an
   uncited canonical claim cannot pass the repository gate.
3. Citation contents remain attestations, not independently verified facts.
   Achta owns presence, bounded structure, and exit-code equality while
   refusing to execute either implementation or judge the replay's truth.
4. The v0.5.1-shaped manifest naming `count check`, `declared-route check`, and
   `mirror check --digest` as replacements with no script-side replay is a
   permanent red fixture.
5. v0.5.3 is a patch: it adds one read-only command, one artifact schema, and
   one repository gate without changing any existing command or schema.

## 31. v0.5.4 parity decisions

1. A comment-only workflow is evaluated as an empty mapping, not refused.
   Comments are expressly outside route and ban matching, so zero live scalars
   are complete evidence rather than unavailable evidence. `per-file-any` can
   therefore pass it; exact-one remains an evaluated route-count failure.
2. A designated-route workflow must contain its caller-named required key
   exactly once across the document and exactly once in the caller-selected
   scope. YAML scope remains structural, but ignoring an identical key in a
   second mapping would preserve THE-SECOND-INSTALL defect.
3. Missing digest files remain `cannot_evaluate`, exit 2: without the recorded
   fact there is no comparison to evaluate. A syntactically valid recorded
   digest that disagrees remains exit 1. Path-form record names remain opaque
   caller vocabulary and require exact `--digest-name`; basename fallback is
   not path normalization. Because two vendored-copy cases were not replayed,
   `mirror check --digest` remains a candidate regardless of the two explicit
   exit-class decisions.
4. Counted-target nouns and fix-assertion verbs remain caller vocabulary.
   Achta accepts the generic structural relation that one pattern from each
   family occurs in the same paragraph, then conservatively requires citations
   for the smallest positive target count. This closes fixture g without a
   paragraph-spanning expression fitted to that fixture and without claiming
   to understand the prose.
5. v0.5.4 is a patch release: it closes measured behavioral gaps in two
   existing verbs and adds only caller-opted pattern vocabulary. No command,
   default, stable schema, or 0/1/2 exit contract is removed.

## 32. v0.5.5 decisions

1. The claim manifest advances to `achta.replacement-claims.v2`. A replacement
   or retirement status must cite a workspace-confined
   `achta.replacement-replay.v1` artifact, pin its exact SHA-256, and match the
   artifact's verb/script identity, selftest citation, and complete fixture set
   row-for-row. Legacy v1 claim rows are attestations and cannot establish
   replacement.
2. The committed replay document in a sibling wiki is provenance, not an input
   Achta can verify within its own workspace. Its measured rows are vendored
   into Achta with source commit and digest metadata; the claim pins the
   vendored bytes. Achta still does not execute the script or verb, so author
   honesty remains outside the gate.
3. Both `count check` and `declared-route check` remain candidates. The vendored
   replay proves the v0.5.4 manifest contradicted six cited rows; no replay is
   rerun or parity claim restored in this slice.
4. Future release tags are annotated. The release workflow rejects a
   lightweight tag object before building or publishing, making tag type part
   of the checked release contract rather than operator convention.
5. v0.5.5 is a patch: it corrects a false-positive governance gate and release
   metadata discipline without removing a command or changing an existing
   machine result schema.

## 33. Success measure

Achta succeeds when agents and humans stop writing one-off scripts for the same
workspace bookkeeping, while every claim remains explicit and every canonical
artifact remains reviewable in Git.

The measure is not the number of scripts deleted. It is fewer malformed edits,
fewer stale witnesses and baselines, fewer working-directory mistakes, and no
loss of the fail-closed safeguards those scripts currently provide.
