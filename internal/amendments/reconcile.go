package amendments

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/ozgurcd/achta/internal/rulefloorclient"
)

type Reconciliation struct {
	SchemaVersion       string   `json:"schema_version"`
	Status              string   `json:"status"`
	BaseCommit          string   `json:"base_commit"`
	RulefloorExecutable string   `json:"rulefloor_executable"`
	DiffSHA256          string   `json:"ledger_diff_sha256"`
	HeaderChanges       []string `json:"header_changes"`
	ActualChanges       int      `json:"actual_changes"`
	DeclaredChanges     int      `json:"declared_changes"`
	Problems            []string `json:"problems"`
}

func Reconcile(manifest Manifest, diff rulefloorclient.LedgerDiff, rawDiff []byte) Reconciliation {
	result := Reconciliation{SchemaVersion: "achta.amendments-reconciliation.v1", Status: "pass", BaseCommit: manifest.BaseCommit, HeaderChanges: append([]string{}, diff.HeaderChanges...), DeclaredChanges: len(manifest.Changes), Problems: []string{}}
	digest := sha256.Sum256(rawDiff)
	result.DiffSHA256 = hex.EncodeToString(digest[:])
	if diff.BaseCommit != manifest.BaseCommit {
		result.Problems = append(result.Problems, fmt.Sprintf("manifest base %s differs from resolved diff base %s", manifest.BaseCommit, diff.BaseCommit))
	}
	for _, change := range diff.HeaderChanges {
		result.Problems = append(result.Problems, "undeclared ledger header change: "+change)
	}
	actual := make(map[string]rulefloorclient.RuleChange)
	for _, rule := range diff.Rules {
		for _, class := range rule.Changes {
			key := rule.RuleID + "\x00" + class
			actual[key] = rule
			result.ActualChanges++
		}
	}
	declared := make(map[string]Change)
	for _, change := range manifest.Changes {
		key := change.RuleID + "\x00" + change.ChangeClass
		declared[key] = change
		measured, ok := actual[key]
		if !ok {
			result.Problems = append(result.Problems, fmt.Sprintf("declared change absent: %s/%s", change.RuleID, change.ChangeClass))
			continue
		}
		if change.ChangeClass == "sentence_changed" && measured.AfterSentenceSHA256 != change.AfterSentenceSHA256 {
			result.Problems = append(result.Problems, fmt.Sprintf("sentence digest mismatch: %s", change.RuleID))
		}
	}
	for key, rule := range actual {
		if _, ok := declared[key]; ok {
			continue
		}
		for _, class := range rule.Changes {
			if key == rule.RuleID+"\x00"+class {
				result.Problems = append(result.Problems, fmt.Sprintf("undeclared change: %s/%s", rule.RuleID, class))
				break
			}
		}
	}
	sort.Strings(result.HeaderChanges)
	sort.Strings(result.Problems)
	if len(result.Problems) > 0 {
		result.Status = "fail"
	}
	return result
}
