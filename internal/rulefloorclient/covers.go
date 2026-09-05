package rulefloorclient

import (
	"errors"
	"fmt"
)

// Covers is rulefloor.covers.v1: every rule ID mapped to the qualified
// file:Symbol entries its red-proof mutation covers (an unmapped rule lists
// none). Achta consumes it the way it consumes ledger-diff.v1 — Rulefloor's
// stable machine output, never RULE-FLOOR.md's columns.
type Covers struct {
	SchemaVersion string              `json:"schema_version"`
	Rules         map[string][]string `json:"rules"`
}

// Covers runs `rulefloor covers --json --repo <repo>` as an argument vector
// and validates the document.
func (c Client) Covers(repo string) (Covers, error) {
	var covers Covers
	if err := c.invoke(&covers, 0, "covers", "--json", "--repo", repo); err != nil {
		return Covers{}, err
	}
	if covers.SchemaVersion != "rulefloor.covers.v1" {
		return Covers{}, fmt.Errorf("unsupported covers schema %q", covers.SchemaVersion)
	}
	if covers.Rules == nil {
		return Covers{}, errors.New("rulefloor covers document has no rules object")
	}
	return covers, nil
}
