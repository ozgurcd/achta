package cli

import (
	"fmt"
	"io"

	"github.com/ozgurcd/achta/internal/decision"
	"github.com/ozgurcd/achta/internal/safefile"
)

const decisionAddSchema = "achta.decision-add.v1"

type decisionAddDocument struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Register      string `json:"register"`
	DecisionID    string `json:"decision_id"`
	Changed       bool   `json:"changed"`
}

func runDecision(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	rest, err := requireSubcommand(args, "add")
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	set := flagSet("decision add")
	title := set.String("title", "", "complete one-line decision title")
	bodyValue := set.String("body-file", "", "decision body file")
	registerValue := set.String("register", "wiki/platform/decisions.md", "decision register path")
	prefix := set.String("prefix", "P", "decision ID prefix")
	check := set.Bool("check", false, "validate and report without writing")
	jsonMode := set.Bool("json", opts.json, "emit JSON")
	if err := parseFlags(set, rest); err != nil {
		opts.json = *jsonMode
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	opts.json = *jsonMode
	if *title == "" || *bodyValue == "" {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, invalid("--title and --body-file are required"))
	}
	if err := rejectSecretLikeInput(*bodyValue); err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	ws, err := resolveWorkspace(opts)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	registerPath, err := confinedPath(ws, *registerValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	bodyPath, err := confinedPath(ws, *bodyValue)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, err)
	}
	register, err := safefile.Read(ws.Root, registerPath, decision.MaxRegister)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, invalid("read decision register: %v", err))
	}
	body, err := safefile.Read(ws.Root, bodyPath, decision.MaxBody)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, invalid("read decision body: %v", err))
	}
	result, err := decision.Add(register.Data, *prefix, *title, body.Data)
	if err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, invalid("add decision: %v", err))
	}
	status, code := "updated", 0
	if *check {
		status, code = "would_change", 1
	} else if !result.Changed {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, mismatch("decision register is unchanged"))
	} else if err := register.Replace(result.Data, decision.MaxRegister); err != nil {
		return renderError(stdout, stderr, opts.json, decisionAddSchema, invalid("replace decision register: %v", err))
	}
	doc := decisionAddDocument{SchemaVersion: decisionAddSchema, Status: status, Register: *registerValue, DecisionID: result.ID, Changed: result.Changed}
	if opts.json {
		if writeJSON(stdout, stderr, doc) != 0 {
			return 2
		}
		return code
	}
	fmt.Fprintf(stdout, "decision add %s: %s in %s\n", status, result.ID, *registerValue)
	return code
}
