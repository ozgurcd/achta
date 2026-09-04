package witness

import "testing"

func FuzzParse(f *testing.F) {
	f.Add([]byte("schema: gate-run.v1\nplan: verify\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = Parse(data)
	})
}
