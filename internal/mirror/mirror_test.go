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
