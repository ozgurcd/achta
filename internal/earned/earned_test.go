package earned

import "testing"

// RULE: WITNESS-EARNED-1
func TestClassifyRefusesRecordOnlyCycle(t *testing.T) {
	result := Classify([]string{"GATE-RUN.txt", "state/MINT-STATE.json", "policy/ledger-amendments.json"})
	if result.Decision != "REFUSE" || len(result.Substantive) != 0 || len(result.RecordOnly) != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClassifyAcceptsSubstantiveCycle(t *testing.T) {
	result := Classify([]string{"GATE-RUN.txt", "internal/gate.go"})
	if result.Decision != "EARNED" || len(result.Substantive) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
