//go:build !no_addons && !no_calico

package calico

import (
	"fmt"
	"regexp"
	"strings"

	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterTypedFromMapWithFetchMode(schema.GroupVersionKind{Group: "crd.projectcalico.org", Version: "v1", Kind: "GlobalNetworkPolicy"}, BuildGlobalNetworkPolicyMapNode, FetchModeHintFull)
	RegisterTypedFromMapWithFetchMode(schema.GroupVersionKind{Group: "crd.projectcalico.org", Version: "v1", Kind: "HostEndpoint"}, BuildHostEndpointMapNode, FetchModeHintFull)
}

func BuildGlobalNetworkPolicyMapNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}

	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorExpr := GetString(spec, "selector")
	selectorLabels, matchesAll, parsed := parseCalicoSelector(selectorExpr)

	properties := map[string]any{
		"name":                   name,
		"namespace":              "",
		"labels":                 MapToSortedList(labelsMap),
		"annotations":            MapToSortedList(annotationsMap),
		"selector":               selectorExpr,
		"selectorParsed":         parsed,
		"namespaceSelector":      GetString(spec, "namespaceSelector"),
		"serviceAccountSelector": GetString(spec, "serviceAccountSelector"),
		"types":                  StringSliceFromAny(GetSlice(spec, "types")),
		"order":                  summarizeCalicoOrderAny(spec),
		"doNotTrack":             getBool(spec, "doNotTrack"),
		"applyOnForward":         getBool(spec, "applyOnForward"),
		"ingress":                summarizeCalicoRulesMap(GetSlice(spec, "ingress")),
		"egress":                 summarizeCalicoRulesMap(GetSlice(spec, "egress")),
	}

	base := NewGraphNodeBase("BHK_GlobalNetworkPolicy", "", name, labelsMap, annotationsMap)

	core := CoreEntry{
		Cluster: true,
		Data: GlobalNetworkPolicy{
			GraphNodeBase:      base,
			SelectorLabels:     selectorLabels,
			MatchesAll:         matchesAll,
			SelectorRecognized: parsed,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

var calicoEqualityClauseRe = regexp.MustCompile(`^([A-Za-z0-9_./-]+)\s*==\s*'([^']*)'$`)

func parseCalicoSelector(expr string) (labels map[string]string, matchesAll bool, ok bool) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" || trimmed == "all()" {
		return map[string]string{}, true, true
	}
	if strings.ContainsAny(trimmed, "!|()") || strings.Contains(trimmed, " in ") {
		return nil, false, false
	}

	clauses := strings.Split(trimmed, "&&")
	if len(clauses) == 1 {
		clauses = strings.Split(trimmed, ",")
	}

	parsed := make(map[string]string, len(clauses))
	for _, clause := range clauses {
		match := calicoEqualityClauseRe.FindStringSubmatch(strings.TrimSpace(clause))
		if match == nil {
			return nil, false, false
		}
		parsed[match[1]] = match[2]
	}
	if len(parsed) == 0 {
		return nil, false, false
	}
	return parsed, false, true
}

func getBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func summarizeCalicoOrderAny(spec map[string]any) string {
	v, ok := spec["order"]
	if !ok || v == nil {
		return ""
	}
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("%g", n)
	case int64:
		return fmt.Sprintf("%d", n)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func summarizeCalicoRulesMap(rules []any) []string {
	result := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, summarizeCalicoRuleMap(ruleMap))
	}
	return result
}

func summarizeCalicoRuleMap(rule map[string]any) string {
	parts := []string{GetString(rule, "action")}
	if proto := GetString(rule, "protocol"); proto != "" {
		parts = append(parts, "protocol "+proto)
	}
	if src := summarizeCalicoEntityRuleMap(GetMap(rule, "source")); src != "" {
		parts = append(parts, "from "+src)
	}
	if dst := summarizeCalicoEntityRuleMap(GetMap(rule, "destination")); dst != "" {
		parts = append(parts, "to "+dst)
	}
	return strings.Join(parts, " ")
}

func summarizeCalicoEntityRuleMap(entity map[string]any) string {
	if len(entity) == 0 {
		return ""
	}
	var parts []string
	if sel := GetString(entity, "selector"); sel != "" {
		parts = append(parts, "selector("+sel+")")
	}
	if nsSel := GetString(entity, "namespaceSelector"); nsSel != "" {
		parts = append(parts, "namespaceSelector("+nsSel+")")
	}
	if nets := StringSliceFromAny(GetSlice(entity, "nets")); len(nets) > 0 {
		parts = append(parts, "nets "+strings.Join(nets, ","))
	}
	if sa := GetMap(entity, "serviceAccounts"); len(sa) > 0 {
		parts = append(parts, "serviceAccounts")
	}
	if svc := GetMap(entity, "services"); len(svc) > 0 {
		parts = append(parts, fmt.Sprintf("service %s/%s", GetString(svc, "namespace"), GetString(svc, "name")))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
