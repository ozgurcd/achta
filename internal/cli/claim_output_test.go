package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ozgurcd/achta/internal/claim"
)

// RULE: CLAIM-COMPACT-OUTPUT-1
func TestClaimOutputGoldens(t *testing.T) {
	for _, operation := range []string{"take", "status", "check", "release"} {
		for _, history := range []bool{false, true} {
			name := operation
			if history {
				name += "-history"
			}
			t.Run(name, func(t *testing.T) {
				repo := claimFixture(t)
				fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				seed := func(verb, slice string) {
					t.Helper()
					opts := claim.Options{Repo: repo, Slice: slice, Now: fixed}
					if verb == "take" {
						opts.Note = "database work"
					}
					if _, err := claim.Run(verb, opts); err != nil {
						t.Fatal(err)
					}
					fixed = fixed.Add(time.Minute)
				}
				seed("take", "old")
				seed("release", "old")
				seed("take", "recent")
				seed("release", "recent")
				if operation != "take" {
					seed("take", "current")
				}
				writeTestFile(t, filepath.Join(repo, "committed.txt"), "local\n")
				runGit(t, repo, "add", "committed.txt")
				runGit(t, repo, "commit", "-m", "local change")
				writeTestFile(t, filepath.Join(repo, "tracked.txt"), "dirty\n")
				args := []string{"claim", operation, "--repo", repo}
				if operation != "status" {
					args = append(args, "--slice", "current")
				}
				if operation == "take" {
					args = append(args, "--note", "database work")
				}
				if operation == "release" {
					args = append(args, "--slice", "owner", "--force", "--reason", "owner handoff")
				}
				if history {
					args = append(args, "--history")
				}
				var out, diagnostics bytes.Buffer
				if code := Run(args, &out, &diagnostics, testVersion); code != 0 || diagnostics.Len() != 0 {
					t.Fatalf("exit=%d diagnostics=%s", code, diagnostics.String())
				}
				got := regexp.MustCompile(`age [0-9]+s`).ReplaceAllString(out.String(), "age <AGE>s")
				got = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`).ReplaceAllString(got, "<TIME>")
				want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "human", "claim-"+name+".txt"))
				if err != nil {
					t.Fatal(err)
				}
				if got != string(want) {
					t.Fatalf("golden mismatch\ngot:\n%swant:\n%s", got, want)
				}
				// Rendering must never truncate stored history, including a new release.
				var state claim.Result
				if err := json.Unmarshal([]byte(claimRun(t, repo, 0, "status", "--json")), &state); err != nil {
					t.Fatal(err)
				}
				count := 2
				if operation == "release" {
					count++
				}
				if len(state.Releases) != count || state.Releases[0].Holder != "old" {
					t.Fatal("human output changed stored history")
				}
			})
		}
	}
}

func TestClaimOutputEmptyAndRefusals(t *testing.T) {
	repo := claimFixture(t)
	if got := claimRun(t, repo, 0, "status"); got != "claim status: pass\nholder: none\ntracked changes: 0\nunpushed commits: 0\n" {
		t.Fatalf("empty status: %s", got)
	}
	claimRun(t, repo, 0, "take", "--slice", "holder")
	for _, operation := range []string{"take", "check", "release"} {
		out := claimRun(t, repo, 1, operation, "--slice", "other")
		if !strings.Contains(out, "holder: holder; age ") || !strings.Contains(out, "held by holder; age ") || !strings.Contains(out, "tracked changes: 0\nunpushed commits: 0\n") {
			t.Fatalf("%s refusal lost current evidence: %s", operation, out)
		}
	}
	runGit(t, repo, "branch", "--unset-upstream")
	if out := claimRun(t, repo, 0, "status"); !strings.Contains(out, "unpushed commits: unknown (no local upstream)") {
		t.Fatal("missing upstream must remain unknown")
	}
}

func TestClaimHistoryJSONUnchanged(t *testing.T) {
	for _, operation := range []string{"take", "status", "check", "release"} {
		t.Run(operation, func(t *testing.T) {
			repo := claimFixture(t)
			claimRun(t, repo, 0, "take", "--slice", "older", "--json")
			claimRun(t, repo, 0, "release", "--slice", "older", "--json")
			claimRun(t, repo, 0, "take", "--slice", "recent", "--json")
			claimRun(t, repo, 0, "release", "--slice", "recent", "--json")
			claimRun(t, repo, 0, "take", "--slice", "current", "--json")
			args := []string{"--json", "--history"}
			if operation != "status" {
				args = append(args, "--slice", "current")
			}
			var result map[string]json.RawMessage
			if err := json.Unmarshal([]byte(claimRun(t, repo, 0, operation, args...)), &result); err != nil {
				t.Fatal(err)
			}
			var releases []claim.Release
			wantReleases := 2
			if operation == "release" {
				wantReleases++
			}
			if err := json.Unmarshal(result["releases"], &releases); err != nil || len(releases) != wantReleases || releases[0].Holder != "older" {
				t.Fatal("JSON must retain complete history")
			}
			wantFields := 6
			if operation == "status" || operation == "check" {
				wantFields += 5
			}
			if len(result) != wantFields || string(result["schema_version"]) != `"achta.claim.v1"` {
				t.Fatal("JSON field set or schema changed")
			}
		})
	}
}
