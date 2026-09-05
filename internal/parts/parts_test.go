package parts

import (
	"strings"
	"testing"
)

func TestUpdateParseVerifyAndBump(t *testing.T) {
	input := []Part{{Name: "beta.txt", Data: []byte("beta\n")}, {Name: "alpha.txt", Data: []byte("alpha\n")}}
	lock, encoded, err := Update(nil, false, input)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Version != "v1" || !strings.HasPrefix(string(encoded), "# achta.parts-lock.v1\nVERSION v1\n\nalpha.txt ") {
		t.Fatalf("unexpected lock:\n%s", encoded)
	}
	parsed, err := Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Verify("parts", "parts.lock", parsed, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || result.Version != "v1" || len(result.Checks) != 2 {
		t.Fatalf("result = %#v", result)
	}
	bumped, _, err := Update(&parsed, true, input)
	if err != nil {
		t.Fatal(err)
	}
	if bumped.Version != "v2" {
		t.Fatalf("bumped version = %q", bumped.Version)
	}
}

func TestVerifyNamesEditedAbsentAndUnlockedParts(t *testing.T) {
	input := []Part{{Name: "alpha.txt", Data: []byte("alpha\n")}, {Name: "beta.txt", Data: []byte("beta\n")}}
	lock, _, err := Update(nil, false, input)
	if err != nil {
		t.Fatal(err)
	}
	changed := []Part{{Name: "alpha.txt", Data: []byte("alphA\n")}, {Name: "extra.txt", Data: []byte("extra\n")}}
	result, err := Verify("parts", "parts.lock", lock, changed)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || len(result.Checks) != 3 {
		t.Fatalf("result = %#v", result)
	}
	want := map[string]string{"alpha.txt": "differs", "beta.txt": "absent", "extra.txt": "unlocked"}
	for _, check := range result.Checks {
		if want[check.Name] != check.Status {
			t.Fatalf("check = %#v, want status %q", check, want[check.Name])
		}
	}
}

func TestParseRefusesNonCanonicalLock(t *testing.T) {
	cases := []string{
		"VERSION v1\n\na.txt " + strings.Repeat("a", 64) + "\n",
		"# achta.parts-lock.v1\nVERSION v01\n\na.txt " + strings.Repeat("a", 64) + "\n",
		"# achta.parts-lock.v1\nVERSION v1\n\nb.txt " + strings.Repeat("b", 64) + "\na.txt " + strings.Repeat("a", 64) + "\n",
		"# achta.parts-lock.v1\r\nVERSION v1\r\n\r\na.txt " + strings.Repeat("a", 64) + "\r\n",
	}
	for _, fixture := range cases {
		if _, err := Parse([]byte(fixture)); err == nil {
			t.Fatalf("Parse accepted %q", fixture)
		}
	}
}
