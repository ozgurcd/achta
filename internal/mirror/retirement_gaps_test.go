package mirror

import (
	"strings"
	"testing"
)

// RULE: MIRROR-DIGEST-NAMED-RECORD-1
func TestCheckWithDigestSelectsAnExplicitExactRecordName(t *testing.T) {
	master := []byte("master\n")
	masterSHA := digest(master)
	data := []byte(strings.Repeat("0", 64) + "  rulefloor-install-gate.sh\n" + masterSHA + "  tools/rulefloor-install-gate.sh\n")
	result, err := CheckWithDigest("wiki/tools/rulefloor-install-gate.sh", master, nil, DigestInput{
		Path: "wiki/contracts/rulefloor-install-gate.master.sha256",
		Name: "tools/rulefloor-install-gate.sh",
		Data: data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || result.Digest == nil || result.Digest.Name != "tools/rulefloor-install-gate.sh" || result.Digest.RecordedSHA256 != masterSHA {
		t.Fatalf("explicit record result = %+v", result)
	}
	if _, err := CheckWithDigest("wiki/tools/rulefloor-install-gate.sh", master, nil, DigestInput{Path: "digest", Name: "missing", Data: data}); err == nil {
		t.Fatal("absent explicit record name was accepted")
	}
}
