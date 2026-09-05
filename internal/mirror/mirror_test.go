package mirror

import "testing"

func TestCheckIsByteExactAndReportsEveryMirror(t *testing.T) {
	master := []byte("master\n")
	result, err := Check("master.md", master, []Input{
		{Path: "same.md", Data: append([]byte(nil), master...)},
		{Path: "changed.md", Data: []byte("mastfr\n")},
		{Path: "spacing.md", Data: []byte("master \n")},
		{Path: "missing.md", Absent: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || len(result.Mirrors) != 4 {
		t.Fatalf("result = %+v", result)
	}
	if result.Mirrors[0].Status != "match" {
		t.Fatalf("identical mirror = %+v", result.Mirrors[0])
	}
	if result.Mirrors[1].Status != "differs" || result.Mirrors[1].MasterSHA256 == result.Mirrors[1].MirrorSHA256 {
		t.Fatalf("one-byte change = %+v", result.Mirrors[1])
	}
	if result.Mirrors[2].Status != "differs" {
		t.Fatalf("whitespace change was normalized: %+v", result.Mirrors[2])
	}
	if result.Mirrors[3].Status != "absent" || result.Mirrors[3].MirrorSHA256 != "" {
		t.Fatalf("absent mirror = %+v", result.Mirrors[3])
	}
}

func TestCheckRequiresMasterIdentityAndMirror(t *testing.T) {
	if _, err := Check("", nil, []Input{{Path: "mirror.md"}}); err == nil {
		t.Fatal("empty master path accepted")
	}
	if _, err := Check("master.md", nil, nil); err == nil {
		t.Fatal("empty mirror set accepted")
	}
}

func TestCheckWithDigestIsExactAndSelectsUniqueBasename(t *testing.T) {
	master := []byte("master\n")
	masterSHA := digest(master)
	digestFile := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  other.md\n" + masterSHA + "  master.md\n")

	result, err := CheckWithDigest("docs/master.md", master, nil, DigestInput{Path: "contracts/master.sha256", Data: digestFile})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || len(result.Mirrors) != 0 || result.Digest == nil {
		t.Fatalf("matching digest result = %+v", result)
	}
	if result.Digest.File != "contracts/master.sha256" || result.Digest.Line != 2 || result.Digest.Name != "master.md" || result.Digest.RecordedSHA256 != masterSHA || result.Digest.MasterSHA256 != masterSHA {
		t.Fatalf("selected record = %+v", result.Digest)
	}
	combined, err := CheckWithDigest("master.md", master, []Input{{Path: "copy.md", Data: append([]byte(nil), master...)}}, DigestInput{Path: "master.sha256", Data: []byte(masterSHA + "  master.md\n")})
	if err != nil {
		t.Fatal(err)
	}
	if combined.Status != "pass" || len(combined.Mirrors) != 1 || combined.Mirrors[0].Status != "match" || combined.Digest == nil || combined.Digest.Status != "match" {
		t.Fatalf("combined checks = %+v", combined)
	}

	changed, err := CheckWithDigest("master.md", []byte("mastfr\n"), nil, DigestInput{Path: "master.sha256", Data: []byte(masterSHA + "  master.md\n")})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Status != "fail" || changed.Digest == nil || changed.Digest.Status != "differs" || changed.Digest.RecordedSHA256 == changed.Digest.MasterSHA256 {
		t.Fatalf("one-byte master change = %+v", changed)
	}

	for name, data := range map[string][]byte{
		"different basename": []byte(masterSHA + "  other.md\n"),
		"duplicate basename": []byte(masterSHA + "  master.md\n" + masterSHA + "  master.md\n"),
		"spacing change":     []byte(masterSHA + " master.md\n"),
		"line ending change": []byte(masterSHA + "  master.md\r\n"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := CheckWithDigest("master.md", master, nil, DigestInput{Path: "master.sha256", Data: data}); err == nil {
				t.Fatal("non-canonical or ambiguous digest file accepted")
			}
		})
	}
}
