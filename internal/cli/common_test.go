package cli

import "testing"

func TestRejectSecretLikeInput(t *testing.T) {
	for _, path := range []string{".env", "dev.env", "client.key", "release-token.txt", "credentials.json", "recovery-url.md"} {
		if err := rejectSecretLikeInput(path); err == nil {
			t.Fatalf("secret-like path %q passed", path)
		}
	}
	for _, path := range []string{"decision-body.md", "reason.txt", "monkey-notes.md", "keynote.md"} {
		if err := rejectSecretLikeInput(path); err != nil {
			t.Fatalf("safe path %q rejected: %v", path, err)
		}
	}
}
