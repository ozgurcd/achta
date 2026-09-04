package earned

import (
	"path"
	"path/filepath"
	"sort"
)

const Schema = "achta.witness-earned.v1"

type Result struct {
	SchemaVersion string   `json:"schema_version"`
	Decision      string   `json:"decision"`
	Changed       []string `json:"changed"`
	RecordOnly    []string `json:"record_only"`
	Substantive   []string `json:"substantive"`
}

func Classify(changed []string) Result {
	result := Result{
		SchemaVersion: Schema,
		Decision:      "REFUSE",
		Changed:       append([]string(nil), changed...),
		RecordOnly:    []string{},
		Substantive:   []string{},
	}
	sort.Strings(result.Changed)
	for _, changedPath := range result.Changed {
		if isRecordPath(changedPath) {
			result.RecordOnly = append(result.RecordOnly, changedPath)
		} else {
			result.Substantive = append(result.Substantive, changedPath)
		}
	}
	if len(result.Substantive) > 0 {
		result.Decision = "EARNED"
	}
	return result
}

func isRecordPath(value string) bool {
	base := path.Base(filepath.ToSlash(value))
	if base == "ledger-amendments.json" || base == "MINT-STATE.json" {
		return true
	}
	matched, _ := path.Match("GATE-RUN*.txt", base)
	return matched
}
