package declaredroute

import "testing"

// RULE: DECLARED-ROUTE-PER-FILE-ANY-1
func TestCheckSupportsAnyRouteCountPerWorkflowWhileScanningEveryFile(t *testing.T) {
	opts := Options{
		RoutePatterns:        []string{"derived-route"},
		BanPatterns:          []string{"banned-install"},
		RequiredRoutePattern: "derived-route",
		RequiredKey:          "RULEFLOOR_VERSION",
		RequiredScope:        "/env",
		RouteCardinality:     CardinalityPerFileAny,
	}
	ci := Document{Path: "ci.yml", Data: []byte("env:\n  RULEFLOOR_VERSION: v1\njobs:\n  verify:\n    run: |\n      derived-route\n  integration:\n    run: |\n      derived-route\n")}
	publish := Document{Path: "publish.yml", Data: []byte("jobs:\n  publish:\n    run: echo publish\n")}

	result, err := Check([]Document{ci, publish}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || result.Cardinality != CardinalityPerFileAny || len(result.Violations) != 0 || result.Documents[0].RequiredKey.Count != 1 || result.Documents[1].RequiredKey.Count != 0 {
		t.Fatalf("set result = %+v", result)
	}

	banned := publish
	banned.Data = []byte("jobs:\n  publish:\n    run: banned-install\n")
	result, err = Check([]Document{ci, banned}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || !hasRule(result.Violations, RuleBanned) {
		t.Fatalf("banned result = %+v", result)
	}

	missingKey := ci
	missingKey.Data = []byte("jobs:\n  verify:\n    run: derived-route\n  integration:\n    run: derived-route\n")
	result, err = Check([]Document{missingKey, publish}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || !hasRule(result.Violations, RuleRequiredKey) {
		t.Fatalf("missing declaration result = %+v", result)
	}

	result, err = Check([]Document{publish}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("route-free set result = %+v", result)
	}
}
