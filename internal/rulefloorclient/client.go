package rulefloorclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const MaxOutput = 8 << 20

type Client struct {
	Path    string
	Timeout time.Duration
}

func (c Client) Resolve() (Client, string, error) {
	requested := c.Path
	if requested == "" {
		requested = "rulefloor"
	}
	selected, err := exec.LookPath(requested)
	if err != nil {
		return Client{}, "", fmt.Errorf("discover Rulefloor executable %q: %w", requested, err)
	}
	selected, err = filepath.Abs(selected)
	if err != nil {
		return Client{}, "", fmt.Errorf("canonicalize Rulefloor executable %q: %w", selected, err)
	}
	c.Path = filepath.Clean(selected)
	return c, c.Path, nil
}

type Capabilities struct {
	SchemaVersion     string   `json:"schema_version"`
	MachineInterfaces []string `json:"machine_interfaces"`
	LedgerFeatures    []string `json:"ledger_features"`
}

type LedgerDiff struct {
	SchemaVersion    string       `json:"schema_version"`
	Status           string       `json:"status"`
	BaseCommit       string       `json:"base_commit"`
	HeadersChanged   bool         `json:"headers_changed"`
	HeaderChanges    []string     `json:"header_changes"`
	Rules            []RuleChange `json:"rules"`
	TotalRuleChanges int          `json:"total_rule_changes"`
	Truncated        bool         `json:"truncated"`
}

type RuleChange struct {
	RuleID              string   `json:"rule_id"`
	Changes             []string `json:"changes"`
	AfterSentenceSHA256 string   `json:"after_sentence_sha256,omitempty"`
}

func (c Client) CheckCompatibility() error {
	var capabilities Capabilities
	if err := c.invoke(&capabilities, 0, "capabilities", "--json"); err != nil {
		return err
	}
	if capabilities.SchemaVersion != "rulefloor.capabilities.v1" {
		return fmt.Errorf("unsupported Rulefloor capabilities schema %q", capabilities.SchemaVersion)
	}
	if !contains(capabilities.MachineInterfaces, "rulefloor.ledger-diff.v1") {
		return errors.New("rulefloor does not advertise rulefloor.ledger-diff.v1")
	}
	if !contains(capabilities.LedgerFeatures, "ledger-diff-sentence-sha256") {
		return errors.New("rulefloor does not advertise sentence digests")
	}
	return nil
}

func (c Client) LedgerDiff(base, repo string) (LedgerDiff, []byte, error) {
	stdout, exitCode, err := c.run("ledger-diff", "--base", base, "--repo", repo, "--json")
	if err != nil {
		return LedgerDiff{}, nil, err
	}
	var diff LedgerDiff
	if err := decodeSingle(stdout, &diff); err != nil {
		return LedgerDiff{}, nil, err
	}
	if diff.SchemaVersion != "rulefloor.ledger-diff.v1" {
		return LedgerDiff{}, nil, fmt.Errorf("unsupported ledger diff schema %q", diff.SchemaVersion)
	}
	expectedExit := 2
	switch diff.Status {
	case "same":
		expectedExit = 0
	case "different":
		expectedExit = 1
	case "cannot_evaluate":
		return LedgerDiff{}, nil, errors.New("rulefloor could not evaluate ledger diff")
	default:
		return LedgerDiff{}, nil, fmt.Errorf("unsupported Rulefloor status %q", diff.Status)
	}
	if exitCode != expectedExit {
		return LedgerDiff{}, nil, fmt.Errorf("rulefloor exit/status disagreement: exit %d with status %s", exitCode, diff.Status)
	}
	if diff.Truncated {
		return LedgerDiff{}, nil, errors.New("rulefloor ledger diff is truncated")
	}
	if diff.HeadersChanged != (len(diff.HeaderChanges) > 0) {
		return LedgerDiff{}, nil, errors.New("rulefloor header-change fields disagree")
	}
	if len(diff.Rules) != diff.TotalRuleChanges {
		return LedgerDiff{}, nil, errors.New("rulefloor rule count disagreement")
	}
	for _, rule := range diff.Rules {
		if rule.RuleID == "" || len(rule.Changes) == 0 {
			return LedgerDiff{}, nil, errors.New("rulefloor returned an incomplete rule change")
		}
		for _, change := range rule.Changes {
			if !validRuleChange(change) {
				return LedgerDiff{}, nil, fmt.Errorf("rulefloor returned unsupported rule change %q", change)
			}
		}
	}
	return diff, stdout, nil
}

func (c Client) invoke(target any, expectedExit int, args ...string) error {
	stdout, exitCode, err := c.run(args...)
	if err != nil {
		return err
	}
	if exitCode != expectedExit {
		return fmt.Errorf("external command exited %d, want %d", exitCode, expectedExit)
	}
	return decodeSingle(stdout, target)
}

func (c Client) run(args ...string) ([]byte, int, error) {
	resolved, path, err := c.Resolve()
	if err != nil {
		return nil, 2, err
	}
	timeout := resolved.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		return nil, 2, errors.New("rulefloor command timed out")
	}
	if stdout.overflow || stderr.overflow {
		return nil, 2, errors.New("rulefloor output exceeded limit")
	}
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, 2, fmt.Errorf("start Rulefloor: %w", err)
		}
		exitCode = exitErr.ExitCode()
	}
	if stderr.buf.Len() > 0 {
		message := strings.TrimSpace(stderr.buf.String())
		if len(message) > 512 {
			message = message[:512] + "..."
		}
		return nil, 2, fmt.Errorf("rulefloor wrote stderr: %s", message)
	}
	if exitCode < 0 || exitCode > 2 {
		return nil, 2, fmt.Errorf("rulefloor returned unsupported exit %d", exitCode)
	}
	return append([]byte(nil), stdout.buf.Bytes()...), exitCode, nil
}

func decodeSingle(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode Rulefloor JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("rulefloor emitted more than one JSON document")
		}
		return fmt.Errorf("trailing Rulefloor output: %w", err)
	}
	return nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validRuleChange(value string) bool {
	switch value {
	case "rule_added", "rule_removed", "sentence_changed", "binding_changed", "proof_changed", "covered_symbols_changed", "test_fingerprint_changed":
		return true
	default:
		return false
	}
}

type limitedBuffer struct {
	buf      bytes.Buffer
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > MaxOutput {
		remaining := MaxOutput - b.buf.Len()
		if remaining > 0 {
			_, _ = b.buf.Write(p[:remaining])
		}
		b.overflow = true
		return len(p), nil
	}
	return b.buf.Write(p)
}
