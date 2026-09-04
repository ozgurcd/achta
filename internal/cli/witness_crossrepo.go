package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ozgurcd/achta/internal/gitstate"
	"github.com/ozgurcd/achta/internal/reachability"
	"github.com/ozgurcd/achta/internal/witness"
	"github.com/ozgurcd/achta/internal/workspace"
)

type ciProvenanceCheck struct {
	Status     string `json:"status"`
	RunURL     string `json:"run_url"`
	Attempt    int    `json:"attempt"`
	SHA        string `json:"sha"`
	Limitation string `json:"limitation"`
}

type siblingWitnessCheck struct {
	Name           string               `json:"name"`
	Status         string               `json:"status"`
	RecordedHead   string               `json:"recorded_head,omitempty"`
	RepositoryHead string               `json:"repository_head,omitempty"`
	Reachability   *reachability.Result `json:"reachability,omitempty"`
}

func collectSiblingPins(ws workspace.Workspace, primary string, values []string) ([]witness.SiblingPin, error) {
	mappings, err := resolveSiblingMappings(ws, primary, values)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(mappings))
	for name := range mappings {
		names = append(names, name)
	}
	sort.Strings(names)
	pins := make([]witness.SiblingPin, 0, len(names))
	for _, name := range names {
		repo := mappings[name]
		clean, err := gitstate.IsClean(repo)
		if err != nil {
			return nil, invalid("sibling %s status: %v", name, err)
		}
		if !clean {
			return nil, mismatch("sibling %s is dirty", name)
		}
		head, err := gitstate.Head(repo)
		if err != nil {
			return nil, invalid("sibling %s HEAD: %v", name, err)
		}
		digest, err := gitstate.TreeDigestAll(repo)
		if err != nil {
			return nil, invalid("sibling %s digest: %v", name, err)
		}
		pins = append(pins, witness.SiblingPin{Name: name, Head: head, TreeValue: digest})
	}
	return pins, nil
}

func evaluateWitnessCheck(ws workspace.Workspace, repo, relative string, record witness.Record, summary witness.Summary, siblingValues, noReachValues, siblingNoReachValues []string) (witnessCheckDocument, error) {
	primaryExclusions, err := parseNoReach(noReachValues)
	if err != nil {
		return witnessCheckDocument{}, invalid("%v", err)
	}
	if _, err := reachability.Classify(nil, primaryExclusions); err != nil {
		return witnessCheckDocument{}, invalid("primary no-reach declarations: %v", err)
	}
	mappings, err := resolveSiblingMappings(ws, repo, siblingValues)
	if err != nil {
		return witnessCheckDocument{}, err
	}
	scoped, err := parseSiblingNoReach(siblingNoReachValues)
	if err != nil {
		return witnessCheckDocument{}, invalid("%v", err)
	}
	recordedNames := make(map[string]bool, len(record.Siblings))
	for _, pin := range record.Siblings {
		recordedNames[pin.Name] = true
		if _, ok := mappings[pin.Name]; !ok {
			return witnessCheckDocument{}, invalid("recorded sibling %q requires --sibling %s=REPOSITORY", pin.Name, pin.Name)
		}
	}
	for name := range mappings {
		if !recordedNames[name] {
			return witnessCheckDocument{}, invalid("--sibling %q is not pinned by the record", name)
		}
	}
	for name, exclusions := range scoped {
		if !recordedNames[name] {
			return witnessCheckDocument{}, invalid("--sibling-no-reach names unpinned sibling %q", name)
		}
		if _, err := reachability.Classify(nil, exclusions); err != nil {
			return witnessCheckDocument{}, invalid("sibling %s no-reach declarations: %v", name, err)
		}
	}

	document := witnessCheckDocument{SchemaVersion: witnessCheckSchema, Status: "pass", Summary: summary}
	if record.RepoDirty || record.TreeDirty || summary.Status != "green" || summary.Completeness != "complete" {
		document.Status = "fail"
	}
	if record.CI != nil {
		ciCheck, err := evaluateCIProvenance(repo, record)
		if err != nil {
			return witnessCheckDocument{}, err
		}
		document.CIProvenance = &ciCheck
		if ciCheck.Status == "accepted" {
			document.Summary.Freshness = "accepted_ci_ancestor"
		} else {
			document.Status = "fail"
		}
	} else if document.Summary.Freshness != "current" {
		proof, proven, err := classifyAdvance(repo, relative, record.RepoHead, primaryExclusions)
		if err != nil {
			return witnessCheckDocument{}, invalid("primary reachability: %v", err)
		}
		document.PrimaryReachability = proof
		if proven {
			document.Summary.Freshness = "proven_no_reach"
		} else {
			document.Status = "fail"
		}
	}

	for _, pin := range record.Siblings {
		check, err := evaluateSiblingPin(mappings[pin.Name], pin, scoped[pin.Name])
		if err != nil {
			return witnessCheckDocument{}, invalid("check sibling %s: %v", pin.Name, err)
		}
		document.Siblings = append(document.Siblings, check)
		if check.Status != "current" && check.Status != "proven_no_reach" {
			document.Status = "fail"
		}
	}
	return document, nil
}

func evaluateCIProvenance(repo string, record witness.Record) (ciProvenanceCheck, error) {
	check := ciProvenanceCheck{
		Status:     "refused",
		RunURL:     record.CI.RunURL,
		Attempt:    record.CI.Attempt,
		SHA:        record.CI.SHA,
		Limitation: "uses only local Git ancestry and performs no fetch or network request",
	}
	if record.TreeKind != "commit" || record.TreeValue != record.CI.SHA || record.RepoDirty || record.TreeDirty || record.Result != "green" {
		return check, nil
	}
	recordedHead, err := gitstate.ResolveCommit(repo, record.RepoHead)
	if err != nil || recordedHead != record.CI.SHA {
		return check, nil
	}
	tiedCommit, err := gitstate.ResolveCommit(repo, record.TreeValue)
	if err != nil || tiedCommit != record.CI.SHA {
		return check, nil
	}
	ancestor, err := gitstate.IsAncestor(repo, record.CI.SHA)
	if err != nil {
		return ciProvenanceCheck{}, err
	}
	if ancestor {
		check.Status = "accepted"
	}
	return check, nil
}

func evaluateSiblingPin(repo string, pin witness.SiblingPin, exclusions []reachability.Exclusion) (siblingWitnessCheck, error) {
	check := siblingWitnessCheck{Name: pin.Name, Status: "stale"}
	recordedHead, err := gitstate.ResolveCommit(repo, pin.Head)
	if err != nil {
		return check, nil
	}
	check.RecordedHead = recordedHead
	currentHead, err := gitstate.Head(repo)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	check.RepositoryHead = currentHead
	clean, err := gitstate.IsClean(repo)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	if pin.Dirty {
		check.Status = "dirty_at_finalize"
		return check, nil
	}
	if !clean {
		check.Status = "dirty_now"
		return check, nil
	}
	digest, err := gitstate.TreeDigestAll(repo)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	if recordedHead == currentHead {
		if digest == pin.TreeValue {
			check.Status = "current"
		}
		return check, nil
	}
	ancestor, err := gitstate.IsAncestor(repo, recordedHead)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	if !ancestor {
		return check, nil
	}
	changed, err := gitstate.ChangedPaths(repo, recordedHead)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	classified, err := reachability.Classify(changed, exclusions)
	if err != nil {
		return siblingWitnessCheck{}, err
	}
	check.Reachability = &classified
	if classified.Decision == "SKIPPABLE" {
		check.Status = "proven_no_reach"
	}
	return check, nil
}

func classifyAdvance(repo, recordRelative, recordedHead string, exclusions []reachability.Exclusion) (*reachability.Result, bool, error) {
	base, err := gitstate.ResolveCommit(repo, recordedHead)
	if err != nil {
		return nil, false, nil
	}
	currentHead, err := gitstate.Head(repo)
	if err != nil {
		return nil, false, err
	}
	if base == currentHead {
		return nil, false, nil
	}
	ancestor, err := gitstate.IsAncestor(repo, base)
	if err != nil {
		return nil, false, err
	}
	if !ancestor {
		return nil, false, nil
	}
	clean, err := gitstate.IsCleanExcept(repo, recordRelative)
	if err != nil {
		return nil, false, err
	}
	if !clean {
		return nil, false, nil
	}
	changed, err := gitstate.ChangedPaths(repo, base)
	if err != nil {
		return nil, false, err
	}
	classified, err := reachability.Classify(changed, exclusions)
	if err != nil {
		return nil, false, err
	}
	return &classified, classified.Decision == "SKIPPABLE", nil
}

func resolveSiblingMappings(ws workspace.Workspace, primary string, values []string) (map[string]string, error) {
	mappings := make(map[string]string, len(values))
	for _, value := range values {
		name, repoValue, ok := strings.Cut(value, "=")
		if !ok || name == "" || repoValue == "" || strings.ContainsAny(name, "\x00\r\n") {
			return nil, invalid("--sibling values must be NAME=REPOSITORY")
		}
		if _, exists := mappings[name]; exists {
			return nil, invalid("duplicate --sibling name %q", name)
		}
		if err := rejectSecretLikeInput(repoValue); err != nil {
			return nil, invalid("sibling %s: %v", name, err)
		}
		repo, err := confinedPath(ws, repoValue)
		if err != nil {
			return nil, err
		}
		if filepath.Clean(repo) == filepath.Clean(primary) {
			return nil, invalid("sibling %q resolves to the primary repository", name)
		}
		mappings[name] = repo
	}
	return mappings, nil
}

func parseSiblingNoReach(values []string) (map[string][]reachability.Exclusion, error) {
	result := make(map[string][]reachability.Exclusion)
	for _, value := range values {
		name, declaration, ok := strings.Cut(value, ":")
		if !ok || name == "" || declaration == "" {
			return nil, fmt.Errorf("--sibling-no-reach values must be NAME:PATTERN=WHY")
		}
		exclusions, err := parseNoReach([]string{declaration})
		if err != nil {
			return nil, err
		}
		result[name] = append(result[name], exclusions[0])
	}
	return result, nil
}
