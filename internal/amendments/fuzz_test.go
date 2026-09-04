package amendments

import "testing"

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"schema_version":"ledger-amendments.v1","base_commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","changes":[]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = Parse(data)
	})
}
