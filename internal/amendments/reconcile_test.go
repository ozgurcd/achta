package amendments

import (
	"strings"
	"testing"

	"github.com/ozgurcd/achta/internal/rulefloorclient"
)

// RULE: AMENDMENT-RECONCILE-1
func TestReconcileBothDirectionsAndDigest(t *testing.T) {
	manifest := testManifest()
	manifest.Changes = []Change{{RuleID: "A-1", ChangeClass: "sentence_changed", AfterSentenceSHA256: strings.Repeat("b", 64), Reason: "Changed wording."}}
	diff := rulefloorclient.LedgerDiff{SchemaVersion: "rulefloor.ledger-diff.v1", Status: "different", BaseCommit: manifest.BaseCommit, Rules: []rulefloorclient.RuleChange{{RuleID: "A-1", Changes: []string{"sentence_changed"}, AfterSentenceSHA256: strings.Repeat("b", 64)}}, TotalRuleChanges: 1}
	result := Reconcile(manifest, diff, []byte("diff"))
	if result.Status != "pass" || result.ActualChanges != 1 {
		t.Fatalf("result = %+v", result)
	}
	manifest.Changes[0].AfterSentenceSHA256 = strings.Repeat("c", 64)
	result = Reconcile(manifest, diff, []byte("diff"))
	if result.Status != "fail" || len(result.Problems) != 1 {
		t.Fatalf("result = %+v", result)
	}
	diff.HeadersChanged = true
	diff.HeaderChanges = []string{"FLOOR changed"}
	manifest.Changes[0].AfterSentenceSHA256 = strings.Repeat("b", 64)
	result = Reconcile(manifest, diff, []byte("diff"))
	if result.Status != "fail" || len(result.Problems) != 1 || result.HeaderChanges[0] != "FLOOR changed" {
		t.Fatalf("header reconciliation = %+v", result)
	}
}

func TestReconcileFindsMissingAndExtra(t *testing.T) {
	manifest := testManifest()
	manifest.Changes = []Change{{RuleID: "A-1", ChangeClass: "rule_added", Reason: "Adds A."}}
	diff := rulefloorclient.LedgerDiff{BaseCommit: manifest.BaseCommit, Rules: []rulefloorclient.RuleChange{{RuleID: "B-1", Changes: []string{"rule_added"}}}, TotalRuleChanges: 1}
	result := Reconcile(manifest, diff, []byte("diff"))
	if result.Status != "fail" || len(result.Problems) != 2 {
		t.Fatalf("result = %+v", result)
	}
}
