package addons

import (
	"fmt"
	"regexp"
	"strings"

	. "bloodhound-kube/internal/nodes/framework"

	apiv3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/runtime"
)

type GlobalNetworkPolicy struct {
	GraphNodeBase
	PodSelectorLabels  map[string]string
	MatchesAll         bool
	SelectorRecognized bool
}

func BuildGlobalNetworkPolicyNode(obj runtime.Object) (BuildResult, bool) {
	policy, ok := obj.(*apiv3.GlobalNetworkPolicy)
	if !ok || policy == nil {
		return BuildResult{}, false
	}
	name := policy.Name
	if name == "" {
		return BuildResult{}, false
	}

	labelsMap := StringMapToAnyMap(policy.Labels)
	annotationsMap := StringMapToAnyMap(policy.Annotations)

	selectorLabels, matchesAll, parsed := parseCalicoSelector(policy.Spec.Selector)

	properties := map[string]any{
		"name":                   name,
		"namespace":              "",
		"labels":                 MapToSortedList(labelsMap),
		"annotations":            MapToSortedList(annotationsMap),
		"selector":               policy.Spec.Selector,
		"selectorParsed":         parsed,
		"namespaceSelector":      policy.Spec.NamespaceSelector,
		"serviceAccountSelector": policy.Spec.ServiceAccountSelector,
		"types":                  summarizeCalicoPolicyTypes(policy.Spec.Types),
		"order":                  summarizeCalicoOrder(policy.Spec.Order),
		"doNotTrack":             policy.Spec.DoNotTrack,
		"applyOnForward":         policy.Spec.ApplyOnForward,
		"ingress":                summarizeCalicoRules(policy.Spec.Ingress),
		"egress":                 summarizeCalicoRules(policy.Spec.Egress),
	}

	base := NewGraphNodeBase("BHK_GlobalNetworkPolicy", "", name, labelsMap, annotationsMap)

	core := CoreEntry{
		Cluster: true,
		Data: GlobalNetworkPolicy{
			GraphNodeBase:      base,
			PodSelectorLabels:  selectorLabels,
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

// parseCalicoSelector best-effort parses a Calico selector expression
// (https://docs.tigera.io/calico/latest/reference/resources/globalnetworkpolicy#selector)
// into a map of exact-match labels usable for BHK_AppliesTo edge generation,
// mirroring the label-equality-only convention used for core NetworkPolicy's
// podSelector.MatchLabels. The full selector language supports boolean
// combinators, has()/!has(), "in {}", and negation that this parser does not
// evaluate; such expressions return ok=false so the policy still gets a node
// (with the raw selector visible in its "selector" property) but produces no
// edges, rather than silently misinterpreting the expression.
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

func summarizeCalicoPolicyTypes(types []apiv3.PolicyType) []string {
	result := make([]string, 0, len(types))
	for _, t := range types {
		result = append(result, string(t))
	}
	return result
}

func summarizeCalicoOrder(order *float64) string {
	if order == nil {
		return ""
	}
	return fmt.Sprintf("%g", *order)
}

func summarizeCalicoRules(rules []apiv3.Rule) []string {
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		result = append(result, summarizeCalicoRule(rule))
	}
	return result
}

func summarizeCalicoRule(rule apiv3.Rule) string {
	parts := []string{string(rule.Action)}
	if rule.Protocol != nil {
		parts = append(parts, "protocol "+rule.Protocol.String())
	}
	if src := summarizeCalicoEntityRule(rule.Source); src != "" {
		parts = append(parts, "from "+src)
	}
	if dst := summarizeCalicoEntityRule(rule.Destination); dst != "" {
		parts = append(parts, "to "+dst)
	}
	return strings.Join(parts, " ")
}

func summarizeCalicoEntityRule(entity apiv3.EntityRule) string {
	var parts []string
	if entity.Selector != "" {
		parts = append(parts, "selector("+entity.Selector+")")
	}
	if entity.NamespaceSelector != "" {
		parts = append(parts, "namespaceSelector("+entity.NamespaceSelector+")")
	}
	if len(entity.Nets) > 0 {
		parts = append(parts, "nets "+strings.Join(entity.Nets, ","))
	}
	if entity.ServiceAccounts != nil && (len(entity.ServiceAccounts.Names) > 0 || entity.ServiceAccounts.Selector != "") {
		parts = append(parts, "serviceAccounts")
	}
	if entity.Services != nil {
		parts = append(parts, fmt.Sprintf("service %s/%s", entity.Services.Namespace, entity.Services.Name))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
