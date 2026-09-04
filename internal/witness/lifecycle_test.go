package witness

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// RULE: WITNESS-LIFECYCLE-1
func TestLifecycleGreenWithMillisecondTiming(t *testing.T) {
	started := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	data, err := Init("make verify", strings.Repeat("a", 40), []string{"build", "test"}, started)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := int64(1250)
	data, err = Step(data, "build", 0, &elapsed, []string{"build OK"})
	if err != nil {
		t.Fatal(err)
	}
	data, err = Step(data, "test", 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err = Finalize(data, "sha256", strings.Repeat("b", 64), started.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	record, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if record.Result != "green" || record.ElapsedMS["build"] != 1250 {
		t.Fatalf("record = %+v", record)
	}
}

func TestLifecycleRefusesUnplannedDuplicateAndPrematureGreen(t *testing.T) {
	started := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	data, err := Init("gate", strings.Repeat("a", 40), []string{"test"}, started)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Step(data, "other", 0, nil, nil); !errors.Is(err, ErrOperationRefused) {
		t.Fatalf("unplanned target err = %v", err)
	}
	red, err := Finalize(data, "sha256", strings.Repeat("b", 64), started.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	record, err := Parse(red)
	if err != nil || record.Result != "red" || record.TreeKind != "" {
		t.Fatalf("premature finalize = %+v, err=%v", record, err)
	}
}
