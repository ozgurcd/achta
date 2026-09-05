package cli

import (
	"bytes"
	"testing"

	"github.com/ozgurcd/achta/internal/amendments"
	"github.com/ozgurcd/achta/internal/census"
	"github.com/ozgurcd/achta/internal/earned"
	"github.com/ozgurcd/achta/internal/floorcensus"
	"github.com/ozgurcd/achta/internal/ledgerrows"
	"github.com/ozgurcd/achta/internal/mirror"
	"github.com/ozgurcd/achta/internal/reachability"
	"github.com/ozgurcd/achta/internal/recipe"
	"github.com/ozgurcd/achta/internal/slicecheck"
	"github.com/ozgurcd/achta/internal/toolchain"
	achtawiki "github.com/ozgurcd/achta/internal/wiki"
	"github.com/ozgurcd/achta/internal/witness"
)

func TestStableMachineContracts(t *testing.T) {
	assertStableMachineContracts(t)
}

func assertStableMachineContracts(t *testing.T) {
	t.Helper()
	exitCode := 0
	wallMS := int64(20)
	targetMS := int64(12)
	fixtureMS := int64(1000)
	documents := []struct {
		name  string
		value any
	}{
		{
			name: "wiki-pin.json",
			value: wikiPinDocument{
				SchemaVersion: "achta.wiki-pin.v1",
				Status:        "updated",
				Repository:    "sample",
				SHA:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Verified:      "2026-09-04",
				Changed:       true,
			},
		},
		{
			name: "witness-summary.json",
			value: witness.Summary{
				SchemaVersion:           "achta.witness-summary.v1",
				RecordSchemaVersion:     "gate-run.v1",
				Status:                  "green",
				Completeness:            "complete",
				Freshness:               "current",
				RepositoryHead:          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				RecordedHead:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PlannedTargets:          1,
				RecordedTargets:         1,
				PassedTargets:           1,
				StartedAt:               "2026-09-04T10:00:00Z",
				FinishedAt:              "2026-09-04T10:00:01Z",
				WallElapsedMS:           &fixtureMS,
				RecordedTargetElapsedMS: &fixtureMS,
				SlowTargets:             []witness.SlowTarget{{Name: "verify", ElapsedMS: 1000}},
			},
		},
		{
			name: "amendments-operation.json",
			value: amendmentOperation{
				SchemaVersion: "achta.amendments-operation.v1",
				Operation:     "declare",
				Status:        "updated",
				Manifest:      "ledger-amendments.json",
				BaseCommit:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				RuleID:        "A-1",
				ChangeClass:   "rule_added",
				Changed:       true,
			},
		},
		{
			name: "amendments-reconciliation.json",
			value: amendments.Reconciliation{
				SchemaVersion:       "achta.amendments-reconciliation.v1",
				Status:              "pass",
				BaseCommit:          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				RulefloorExecutable: "/usr/local/bin/rulefloor",
				DiffSHA256:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				HeaderChanges:       []string{},
				Problems:            []string{},
			},
		},
		{
			name: "wiki-freshness.json",
			value: achtawiki.FreshnessResult{
				SchemaVersion: achtawiki.FreshnessSchema,
				Status:        "pass",
				Fresh:         1,
				Pages: []achtawiki.FreshnessPage{{
					Repository:  "sample",
					Page:        "wiki/repos/sample.md",
					Status:      "fresh",
					Verified:    "2026-09-04",
					DeclaredSHA: "0123456789abcdef0123456789abcdef01234567",
					HeadSHA:     "0123456789abcdef0123456789abcdef01234567",
				}},
			},
		},
		{
			name: "wiki-derive.json",
			value: achtawiki.DeriveResult{
				SchemaVersion: achtawiki.DeriveSchema,
				Status:        "unchanged",
				Scanned:       1,
				Pages: []achtawiki.DerivePage{{
					Repository: "sample",
					Page:       "wiki/repos/sample.md",
					Status:     "unchanged",
				}},
			},
		},
		{
			name: "wiki-unpushed.json",
			value: achtawiki.UnpushedResult{
				SchemaVersion: achtawiki.UnpushedSchema,
				Status:        "published",
				Repository:    "sample",
				Branch:        "main",
				Upstream:      "origin/main",
				Limitation:    "uses the local upstream tracking reference and performs no fetch",
			},
		},
		{
			name: "wiki-check.json",
			value: wikiCheckDocument{
				SchemaVersion: wikiCheckSchema,
				Status:        "pass",
				WikiDir:       "/workspace/wiki",
				Checks: []wikiCheckItem{
					{Name: "freshness", Status: "pass", Detail: "all repository pages are current"},
					{Name: "derive", Status: "pass", Detail: "all generated blocks are current"},
				},
			},
		},
		{
			name: "witness-operation.json",
			value: witnessOperationDocument{
				SchemaVersion: witnessOperationSchema,
				Operation:     "step",
				Status:        "recorded",
				Record:        "sample/GATE-RUN.txt",
				Target:        "test",
				ExitCode:      &exitCode,
				Changed:       true,
			},
		},
		{
			name: "witness-check.json",
			value: witnessCheckDocument{
				SchemaVersion: witnessCheckSchema,
				Status:        "pass",
				Summary: witness.Summary{
					SchemaVersion:           "achta.witness-summary.v1",
					RecordSchemaVersion:     "gate-run.v1",
					Status:                  "green",
					Completeness:            "complete",
					Freshness:               "current",
					RepositoryHead:          "0123456789abcdef0123456789abcdef01234567",
					RecordedHead:            "0123456",
					PlannedTargets:          1,
					RecordedTargets:         1,
					PassedTargets:           1,
					StartedAt:               "2026-09-04T10:00:00Z",
					FinishedAt:              "2026-09-04T10:00:00.020Z",
					WallElapsedMS:           &wallMS,
					RecordedTargetElapsedMS: &targetMS,
					SlowTargets:             []witness.SlowTarget{{Name: "test", ElapsedMS: 12}},
				},
			},
		},
		{
			name: "reachability.json",
			value: reachability.Result{
				SchemaVersion: reachability.Schema,
				Decision:      "SKIPPABLE",
				Changed:       []string{"docs/guide.md"},
				Excluded:      []reachability.ExcludedPath{{Path: "docs/guide.md", Pattern: "docs/**", Why: "documentation only"}},
				Reaching:      []string{},
				Unknown:       []string{},
			},
		},
		{
			name: "witness-earned.json",
			value: earned.Result{
				SchemaVersion: earned.Schema,
				Decision:      "REFUSE",
				Changed:       []string{"GATE-RUN.txt"},
				RecordOnly:    []string{"GATE-RUN.txt"},
				Substantive:   []string{},
			},
		},
		{
			name: "toolchain-parity.json",
			value: toolchain.Result{
				SchemaVersion: toolchain.ResultSchema,
				Status:        "pass",
				Pins:          []toolchain.PinCheck{{Name: "rulefloor", Kind: "version", Env: "RULEFLOOR_VERSION", Workspace: "v0.9.0", CI: "v0.9.0", Status: "pass"}},
			},
		},
		{
			name: "slice-check.json",
			value: slicecheck.Result{
				SchemaVersion: slicecheck.Schema,
				Status:        "pass",
				Repository:    "sample",
				Commits:       1,
				Passed:        1,
				Checks: []slicecheck.Check{{
					Name:   "clean-tree",
					Status: "pass",
					Detail: "working tree is clean",
				}},
			},
		},
		{
			name: "recipe-check.json",
			value: recipe.Result{
				SchemaVersion:   recipe.Schema,
				Status:          "pass",
				Target:          "check",
				Lines:           3,
				Violations:      []recipe.Violation{},
				MissingExpected: []string{},
			},
		},
		{
			name: "parts-operation.json",
			value: partsOperationDocument{
				SchemaVersion: "achta.parts-operation.v1",
				Operation:     "lock",
				Status:        "updated",
				Directory:     "wiki/prompt/parts",
				LockFile:      "wiki/prompt/parts.lock",
				Version:       "v9",
				Parts:         2,
				Changed:       true,
				Bumped:        true,
			},
		},
		{
			name: "parts-verify.json",
			value: struct {
				SchemaVersion string `json:"schema_version"`
				Status        string `json:"status"`
				Directory     string `json:"dir"`
				LockFile      string `json:"lock"`
				Version       string `json:"version"`
				Locked        int    `json:"locked"`
				OnDisk        int    `json:"on_disk"`
				Checks        []struct {
					Name         string `json:"name"`
					Status       string `json:"status"`
					LockedSHA256 string `json:"locked_sha256,omitempty"`
					ActualSHA256 string `json:"actual_sha256,omitempty"`
				} `json:"checks"`
			}{
				SchemaVersion: "achta.parts-verify.v1",
				Status:        "fail",
				Directory:     "wiki/prompt/parts",
				LockFile:      "wiki/prompt/parts.lock",
				Version:       "v9",
				Locked:        2,
				OnDisk:        2,
				Checks: []struct {
					Name         string `json:"name"`
					Status       string `json:"status"`
					LockedSHA256 string `json:"locked_sha256,omitempty"`
					ActualSHA256 string `json:"actual_sha256,omitempty"`
				}{
					{Name: "commit.txt", Status: "match", LockedSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ActualSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
					{Name: "extra.txt", Status: "unlocked", ActualSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
				},
			},
		},
		{
			name: "mirror-check.json",
			value: mirror.Result{
				SchemaVersion: mirror.Schema,
				Status:        "fail",
				Master:        "AGENTS.md",
				MasterSHA256:  "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0",
				Mirrors: []mirror.MirrorResult{
					{Path: "wiki/contracts/AGENTS.md", Status: "match", MasterSHA256: "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0", MirrorSHA256: "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0"},
					{Path: "wiki/tools/gate.sh", Status: "differs", MasterSHA256: "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0", MirrorSHA256: "23af1b2f80f3c9377f3a86e3a14f4fe146d6c2640aff50d8380fc7345dcf9b93"},
					{Path: "wiki/tools/absent.sh", Status: "absent", MasterSHA256: "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0"},
				},
				Digest: &mirror.DigestResult{
					File:           "contracts/AGENTS.master.sha256",
					Line:           1,
					Name:           "AGENTS.md",
					Status:         "match",
					RecordedSHA256: "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0",
					MasterSHA256:   "9b3162498c21d7f960877099174ecea13410bd21d12440b2ea8868117fc08ae0",
				},
			},
		},
		{
			name: "floor-census.json",
			value: floorcensus.Result{
				SchemaVersion: floorcensus.Schema,
				Status:        "pass",
				Rows:          2,
				Buckets:       map[string]int{"COVERED": 1, "OOS": 1},
				Plain:         1,
				Covers:        floorcensus.CoversStats{Rules: 1, Mapped: 1},
				Violations:    []floorcensus.Violation{},
				Refused:       []string{floorcensus.RefusedArmed},
			},
		},
		{
			name: "ledger-census.json",
			value: census.Result{
				SchemaVersion: census.Schema,
				Status:        "pass",
				Rows:          2,
				Sources:       []census.SourceSummary{{Path: "wiki/tools", Kind: "files", Rows: 2, OnDisk: 2, Marks: map[string]int{"KEEP": 2}}},
				Violations:    []census.Violation{},
			},
		},
		{
			name: "ledger-rows.json",
			value: ledgerrows.Result{
				SchemaVersion: ledgerrows.Schema,
				Status:        "fail",
				File:          "wiki/contracts/to-do-queue.MD",
				Rows:          2,
				OpenRows:      1,
				ClosedRows:    1,
				Violations: []ledgerrows.Violation{
					{Line: 7, Rule: ledgerrows.RuleOpenCompletion, Identity: "CE-SEC-5b", Text: "open ID cell conflicts with an explicit completion marker in the prose cell"},
					{Line: 12, Rule: ledgerrows.RuleConditionQuote, Identity: "DOC-2", Text: "quoted string does not appear verbatim elsewhere in the prose cell"},
				},
				Refused: []string{ledgerrows.RefusedClaimMeaning, ledgerrows.RefusedConditionTruth},
			},
		},
	}

	for _, document := range documents {
		t.Run(document.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := writeJSON(&stdout, &stderr, document.value); code != 0 {
				t.Fatalf("writeJSON code = %d, stderr = %q", code, stderr.String())
			}
			assertGolden(t, document.name, stdout.String())
		})
	}
}
