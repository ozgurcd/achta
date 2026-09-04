package wiki

import (
	"strings"
	"testing"
)

func FuzzRenderPin(f *testing.F) {
	f.Add([]byte(pageFixture))
	sha := strings.Repeat("b", 40)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = RenderPin(data, "sample", sha, "2026-09-04", "main")
	})
}
