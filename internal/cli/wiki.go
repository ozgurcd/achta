package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/safefile"
	achtawiki "github.com/ozgurcd/achta/internal/wiki"
)

type wikiPinDocument struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Repository    string `json:"repository"`
	SHA           string `json:"sha"`
	Verified      string `json:"verified"`
	Changed       bool   `json:"changed"`
}

func runWiki(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "pin")
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", err)
	}
	if len(rest) == 0 {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("wiki pin requires a repository name"))
	}
	repository := rest[0]
	set := flagSet("wiki pin")
	sha := set.String("sha", "", "full repository HEAD SHA")
	verified := set.String("verified", "", "review date in YYYY-MM-DD")
	attest := set.Bool("attest-reviewed", false, "attest that review occurred")
	check := set.Bool("check", false, "validate and report without writing")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest[1:]); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", err)
	}
	opts.json = *jsonMode
	if !*attest {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", mismatch("explicit --attest-reviewed is required"))
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", err)
	}
	repoPath, err := confinedPath(ws, repository)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", err)
	}
	head, err := gitstate.Head(repoPath)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("repository HEAD: %v", err))
	}
	if *sha != head {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", mismatch("--sha must equal repository HEAD"))
	}
	branch, err := gitstate.Branch(repoPath)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("repository branch: %v", err))
	}
	pagePath := filepath.Join(ws.Root, "wiki", "repos", repository+".md")
	snapshot, err := safefile.Read(ws.Root, pagePath, achtawiki.MaxPageSize)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("read wiki page: %v", err))
	}
	result, err := achtawiki.RenderPin(snapshot.Data, repository, *sha, *verified, branch)
	if err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("render wiki pin: %v", err))
	}
	status := "updated"
	code := 0
	if *check {
		if result.Changed {
			status, code = "would_change", 1
		} else {
			status = "unchanged"
		}
	} else if !result.Changed {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", mismatch("wiki pin is already current"))
	} else if err := snapshot.Replace(result.Data, achtawiki.MaxPageSize); err != nil {
		return renderError(stdout, stderr, opts.json, "achta.wiki-pin.v1", invalid("replace wiki page: %v", err))
	}
	doc := wikiPinDocument{SchemaVersion: "achta.wiki-pin.v1", Status: status, Repository: repository, SHA: *sha, Verified: *verified, Changed: result.Changed}
	if opts.json {
		if writeJSON(stdout, stderr, doc) != 0 {
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "wiki pin %s: %s at %s (%s)\n", status, repository, (*sha)[:7], *verified)
	}
	return code
}
