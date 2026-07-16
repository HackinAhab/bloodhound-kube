package addons

import (
	"fmt"
	"sort"
	"strings"

	. "bloodhound-kube/internal/nodes/framework"
)

type CiliumNetworkPolicy struct {
	GraphNodeBase
	PodSelectorLabels map[string]string
}

func BuildCiliumNetworkPolicyNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	endpointSelector := GetMap(spec, "endpointSelector")
	selectorLabels := stringMapFromAny(GetMap(endpointSelector, "matchLabels"))

	properties := map[string]any{
		"name":             name,
		"namespace":        namespace,
		"labels":           MapToSortedList(labelsMap),
		"annotations":      MapToSortedList(annotationsMap),
		"endpointSelector": summarizeCiliumSelector(endpointSelector),
		"description":      GetString(spec, "description"),
		"ingress":          summarizeCiliumRules(GetSlice(spec, "ingress"), "from"),
		"egress":           summarizeCiliumRules(GetSlice(spec, "egress"), "to"),
	}

	base := NewGraphNodeBase("BHK_CiliumNetworkPolicy", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: CiliumNetworkPolicy{
			GraphNodeBase:     base,
			PodSelectorLabels: selectorLabels,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func stringMapFromAny(m map[string]any) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

func stringSliceFromAny(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// summarizeCiliumSelector mirrors summarizeNetpolSelector, but operates on the
// unstructured map form of a Cilium EndpointSelector (matchLabels/matchExpressions),
// since CiliumNetworkPolicy has no typed Go struct in this repo's dependencies.
func summarizeCiliumSelector(sel map[string]any) []string {
	matchLabels := GetMap(sel, "matchLabels")
	matchExpressions := GetSlice(sel, "matchExpressions")
	if len(matchLabels) == 0 && len(matchExpressions) == 0 {
		return []string{"<all endpoints>"}
	}
	var parts []string
	for k, v := range matchLabels {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	for _, item := range matchExpressions {
		expr, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := GetString(expr, "key")
		values := strings.Join(stringSliceFromAny(GetSlice(expr, "values")), ",")
		switch GetString(expr, "operator") {
		case "In":
			parts = append(parts, fmt.Sprintf("%s In [%s]", key, values))
		case "NotIn":
			parts = append(parts, fmt.Sprintf("%s NotIn [%s]", key, values))
		case "Exists":
			parts = append(parts, key+" Exists")
		case "DoesNotExist":
			parts = append(parts, key+" DoesNotExist")
		}
	}
	sort.Strings(parts)
	return parts
}

// summarizeCiliumRules produces display-only summaries of ingress/egress rules.
// direction is "from" for ingress rules or "to" for egress rules, matching the
// "fromEndpoints"/"toEndpoints", "fromCIDR"/"toCIDR", "fromEntities"/"toEntities"
// field-name convention shared by Cilium's IngressRule/EgressRule; "toPorts" is
// used by both directions verbatim.
func summarizeCiliumRules(rules []any, direction string) []string {
	result := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, summarizeCiliumRule(ruleMap, direction))
	}
	return result
}

func summarizeCiliumRule(rule map[string]any, direction string) string {
	var parts []string
	if endpoints := GetSlice(rule, direction+"Endpoints"); len(endpoints) > 0 {
		selStrs := make([]string, 0, len(endpoints))
		for _, e := range endpoints {
			if selMap, ok := e.(map[string]any); ok {
				selStrs = append(selStrs, strings.Join(summarizeCiliumSelector(selMap), ","))
			}
		}
		parts = append(parts, "endpoints ["+strings.Join(selStrs, "; ")+"]")
	}
	if cidrs := stringSliceFromAny(GetSlice(rule, direction+"CIDR")); len(cidrs) > 0 {
		parts = append(parts, "cidr ["+strings.Join(cidrs, ",")+"]")
	}
	if entities := stringSliceFromAny(GetSlice(rule, direction+"Entities")); len(entities) > 0 {
		parts = append(parts, "entities ["+strings.Join(entities, ",")+"]")
	}
	if ports := GetSlice(rule, "toPorts"); len(ports) > 0 {
		parts = append(parts, fmt.Sprintf("ports [%d rule(s)]", len(ports)))
	}
	if len(parts) == 0 {
		return "allow all"
	}
	return strings.Join(parts, " ")
}
