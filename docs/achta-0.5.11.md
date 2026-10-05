# ACHTA-0.5.11 evidence

Baseline: de3395b5133883899952d803957196ceb57e7a61, v0.5.10, clean and equal to origin/main.
The source-built baseline selected D-035 from a copy of the consumer register.
`--body-file -` exited 2 with "read decision body: lstat …/-: no such file or directory".
An outside-workspace path exited 2 with "escapes workspace".

## Implementation and proof

`runWithInput` already receives stdin; dispatch now passes it to decision add.
The stdin reader stops at MaxBody + 1; unchanged decision.Add enforces the 65536
byte maximum and body grammar. File inputs retain confinement and secret-like
filename refusal. There was no secret-content classifier in the baseline;
stdin has no filename. No content classifier is invented by this patch.

New test file: internal/cli/decision_stdin_test.go. Three top-level tests and
15 subtests ran red against the baseline and green with the fix. The first
run included "exit = 2, want 0", "help missing \"--body-file -\"", and
"decision add capabilities omit stdin form". The green run reported PASS for
TestDecisionStdin, TestDecisionStdinBoundAndConfinement and
TestDecisionStdinHelpAndCapabilities. The fixture covers D and P, check mode,
empty and whitespace-only input, NUL, CR, headings, the exact maximum, oversize,
bounded reading, read errors, outside paths, secret-like file paths, and JSON.
Disabling only the stdin branch made identity, platform, check and maximum red
again: exit 2 instead of 0 or 1, with "is not a regular file". Restoring it
returned all three tests to green. DECISION-STDIN-1 records that mutation.

Owner RESUME 2 authorizes the testVersion constant and four human/JSON
version/capability fixtures to advance to v0.5.11. The decision-help case
now checks code 0, empty stderr, the global help prefix and both required flags.
Other existing test cases remain unchanged. Capability fixtures already carried
the additive input forms from the initial implementation. Source Version
advances to v0.5.11 following the repository's source-version release mechanism;
there is no bump target in its Makefile. Release checks judge the actual binary.

## Invocation corrections

One apply_patch failed before writing because help spacing did not match.
A gograph-first hook rejected non-Go file enumeration as a usage query; work
stopped until the owner said continue. Rulefloor arm initially refused two
identical rule tags; the extra annotation was removed without changing tests,
and the unique binding armed successfully. None of these is a test red proof.

## Consumer references left open

Read-only census of tracked regular UTF-8 files in the four requested repositories,
excluding symlinks, environment files, key containers and binary files. The scan
used the literal 0.5.10 and explicit Achta minimum-version expressions. Each
matching physical line appears once below; snippets are shortened around the match.
Historical log references and generated provenance are records, not pins to rewrite.
Live pins/minima remain owner work. No consumer file was changed.

Consumer heads when measured: OSS 3059ce9 (dirty), UI 979869c (clean),
parent wiki 5e6dafe (dirty), lictor b7c1c62 (dirty).

```text
identuum-idp-oss/Makefile:138: ## THIS GATE NEEDS achta >= v0.4.1, AND SAYS SO BY NAME. achta is installed
identuum-idp-oss/Makefile:199: >/dev/null 2>&1 || { echo "wiki-fresh: achta is not installed — this gate needs achta >= v0.4.1 (wiki check --only freshness)" >&2; exit 2; }; \
identuum-idp-oss/Makefile:202: not defined: -only"*) echo "wiki-fresh: achta >= v0.4.1 required (wiki check --only freshness); installed: $$(achta version)" >&2; exit 2;; esac; \
identuum-ui/Makefile:51: >/dev/null 2>&1 || { echo "wiki-fresh: achta is not installed — this gate needs achta >= v0.4.1 (wiki check --only freshness)" >&2; exit 2; }; \
identuum-ui/Makefile:54: not defined: -only"*) echo "wiki-fresh: achta >= v0.4.1 required (wiki check --only freshness); installed: $$(achta version)" >&2; exit 2;; esac; \
wiki/Makefile:262: ES (2026-09-17, owner decision). achta v0.5.10's
wiki/Makefile:781: # achta v0.5.10 carries `--repo NAME` and `--exclude NAME=REASON` and names
wiki/agent-rules.md:324: `--help`, an MCP `tools/list`): `achta v0.5.10 (module v0.5.10, agreement
wiki/agent-rules.md:970: v0.5.10 carries `--repo NAME` and `--exclude NAME=REASON` and names every
wiki/agent-rules.md:975: (MEASURED 2026-09-17: under v0.5.10 the same three pins unchanged, and the
wiki/log.md:8230: The gate depends on achta >= v0.4.1 and nothing pinned achta anywhere — not
wiki/log/0089-2026-09-09-the-queue.md:55: v0.5.9, not v0.5.10 (G-4's request stands); OSS bin/identuum-idp is
wiki/log/0133-2026-09-13-the-overtaken-lines.md:64: v0.5.10; installed is v0.5.9)". Installed is still v0.5.9. The subject
wiki/log/0159-2026-09-16-three-more-closes.md:10: v0.5.10; installed is v0.5.9 per `achta version --json`)." — closed in
wiki/log/0159-2026-09-16-three-more-closes.md:17: the seed's v0.5.10.
wiki/log/0162-2026-09-18-the-tools-the-agent-uses.md:8: ONS, each quoted from the tool: `achta v0.5.10 (module v0.5.10,
wiki/log/0162-2026-09-18-the-tools-the-agent-uses.md:28: v0.5.10 that is a PREVIEW that exits 1 when a page is stale — measured on
wiki/log/0162-2026-09-18-the-tools-the-agent-uses.md:31: le on ONE line each: the header, which v0.5.10 now writes as
wiki/log/0162-2026-09-18-the-tools-the-agent-uses.md:32: "Generated by achta v0.5.10 (`achta wiki derive --write`)" where the old
wiki/log/0162-2026-09-18-the-tools-the-agent-uses.md:49: HEAD under v0.5.10 it was RED — "check FAILED: wiki-check derive —
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:10: on adapter sentences); achta v0.5.10 (per-repository `wiki check --repo/--exclude`,
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:44: queue/GENERAL.md:31 — achta v0.5.10 (2026-09-17): `wiki pin` advances only verified_against, verified and updated; DERIVED and lead bytes proven identical (THE
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:47: queue/GENERAL.md:33 — achta v0.5.10: `wiki freshness` enforces by default, `--report-only` labels non-enforcement, `wiki check` accepts `--repo`/`--exclude NAM
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:56: queue/GENERAL.md:39 — achta v0.5.10: `recipe check --expect-file PATH --expect-order` judges order; the wiki requires it on every recipe-pinning check entry si
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:59: queue/GENERAL.md:41 — achta v0.5.10: `wiki pin` bumps `verified:` and `updated:` together; the three pages' dates correct themselves at their next pin.
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:62: queue/GENERAL.md:42 — achta v0.5.10: `wiki pin` prints "A new front lead is the PM's to write; lead text and DERIVED content are unchanged."
wiki/log/0163-2026-09-18-the-closes-the-tools-earned.md:65: queue/GENERAL.md:44 — achta v0.5.10: the header reads "Generated by achta v0.5.10 (`achta wiki derive --write`)"; regenerated on the three ag pages in 846a485;
wiki/log/0170-2026-09-19-the-six-that-were-owed.md:31: header now "Generated by achta v0.5.10 (`achta wiki derive --write`)" and
wiki/log/0277-2026-10-02-wiki-tooling-2.md:27: eck --repo --commits [--ahead]`; achta v0.5.10's
wiki/log/0301-2026-10-05-wiki-lictor-pins.md:28: - achta: `toolchain check` (v0.5.10) compares one repository's own JSON
wiki/queue/GENERAL.md:39: OJECT_DESC.md §3, AMENDMENT 1) — achta v0.5.10; Codex agent under LICTOR-4 (STOP), 2026-09-21
wiki/queue/GENERAL.md:55: - achta wiki derive (v0.5.10) renders deployment/docker-compose.yml as "publishes **no host port**" since 25a0787 changed the ports to `"${IDENTUUM_IDP_
wiki/repos/identuum-ag-ce.md:31: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
wiki/repos/identuum-ag-oss.md:35: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
wiki/repos/identuum-ag.md:82: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
wiki/repos/identuum-idp-ce.md:25: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
wiki/repos/identuum-idp-oss.md:27: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
wiki/repos/identuum-idp-oss.md:340: catches exactly that answer and prints "achta >= v0.4.1 required (wiki check --only freshness); installed: <achta version>", exit 2; a missing achta is named the same way
wiki/repos/identuum-ui.md:61: Generated by achta v0.5.10 (`achta wiki derive --write`) — **do not edit inside this block by hand**; your edit will be overwritten on the next run. E
lictor/.github/workflows/verify.yml:16: ACHTA_VERSION: v0.5.10
lictor/Makefile:21: a version --json | jq -e '.version == "v0.5.10" and .version_agreement == "pass"'
lictor/PROJECT_SPEC.md:144: eck v1.7.0, rulefloor v0.9.1 and Achta v0.5.10.
lictor/README.md:141: ccheck v0.8.1, rulefloor v0.9.1, Achta v0.5.10 and jq must be on PATH.
lictor/docs/lictor-4.md:55: Installed Achta v0.5.10, source de3395b. Exact patterns are in PROJECT_SPEC.md.
lictor/internal/route/route_test.go:33: version":"achta.version.v1","version":"v0.5.10","version_agreement":"pass"}`, output: `{"schema_version":"achta.declared-route-check.v1","status":"pass"}`}
lictor/internal/route/route_test.go:125: v := "v0.5.10"
lictor/internal/route/route_test.go:127: | stderr.String() != "achta: installed v0.5.10; declared v0.5.10\n" {
lictor/wiki/log.md:155: be required-key-only. Installed Achta v0.5.10 has no such mode. Its source at
lictor/wiki/log.md:234: Route delegates to Achta v0.5.10 without YAML judgement. OSS selects Rulefloor
```

## Close verification

After the owner-authorized release and help expectations, make verify exited 0.
Its final line was `TIMINGS shown compile_groups=10/22 slow_rows=10/50`.
The Gograph session achta0511_20261005_201848 ended with 50 commands, plan/review
true and grade C (78.4%); achta0511resume2_20261005_214347 ended with 4 commands,
plan/review true and grade A (100%). Both ended before verification.

The actual scoped-help precedent in TestWikiBehaviorFailureAndHelp checks
`code != 0 || !strings.Contains(stdout.String(), pair.flag)`, where wiki check
uses `--exclude`. The corrected decision-help test follows flag presence and
also requires global help as a prefix and empty stderr. Its other cases keep
the existing exact assertions. Version-only updates: testVersion in
internal/cli/run_test.go and testdata/human/{version,capabilities}.txt plus
testdata/machine/{version,capabilities}.json. The capability input-form additions
preceded these authorized version-only updates.

A later hook denied read-only `git config user.name` in a multi-command call
without an explicit cd: `achta hook cd: DENY — statement 4 (git config) may write
without an absolute repository selector`. Owner RESUME 3 authorized the exact
compliant identity read, which returned Ozgur Demir and ozgurcd@gmail.com.

The parent wiki's postcheck requires a parent log entry and parent repository
page even for an independently owned wiki. Applying that recipe here conflicts
with Achta's own AGENTS.md prohibition on a parent Achta page; the repository's
own wiki-check remains authoritative. No parent record is created as a workaround.
