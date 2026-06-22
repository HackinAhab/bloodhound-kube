package networking

import (
	"fmt"
	"sort"
	"strings"

	. "bloodhound-kube/internal/nodes/framework"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NetworkPolicy struct {
	GraphNodeBase
	PodSelectorLabels map[string]string
}

func BuildNetworkPolicyNode(obj runtime.Object) (BuildResult, bool) {
	policy, ok := obj.(*networkingv1.NetworkPolicy)
	if !ok || policy == nil {
		return BuildResult{}, false
	}
	name := policy.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := policy.Namespace
	labelsMap := StringMapToAnyMap(policy.Labels)
	annotationsMap := StringMapToAnyMap(policy.Annotations)
	selectorLabels := policy.Spec.PodSelector.MatchLabels

	policyTypes := summarizeNetpolPolicyTypes(policy.Spec.PolicyTypes)
	podSelector := summarizeNetpolSelector(policy.Spec.PodSelector)
	ingress := summarizeNetpolIngressRules(policy.Spec.Ingress, netpolHasType(policy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress))
	egress := summarizeNetpolEgressRules(policy.Spec.Egress, netpolHasType(policy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress))

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"policyTypes": policyTypes,
		"podSelector": podSelector,
		"ingress":     ingress,
		"egress":      egress,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: NetworkPolicy{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("BHK_NetworkPolicy", namespace, name),
				Kinds:          []string{"BHK_NetworkPolicy"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			PodSelectorLabels: selectorLabels,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("BHK_NetworkPolicy", namespace, name),
			Kinds:      []string{"BHK_NetworkPolicy"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func netpolHasType(types []networkingv1.PolicyType, t networkingv1.PolicyType) bool {
	for _, pt := range types {
		if pt == t {
			return true
		}
	}
	return false
}

func summarizeNetpolPolicyTypes(types []networkingv1.PolicyType) []string {
	result := make([]string, 0, len(types))
	for _, t := range types {
		result = append(result, string(t))
	}
	return result
}

func summarizeNetpolSelector(sel metav1.LabelSelector) []string {
	if len(sel.MatchLabels) == 0 && len(sel.MatchExpressions) == 0 {
		return []string{"<all pods>"}
	}
	var parts []string
	for k, v := range sel.MatchLabels {
		parts = append(parts, k+"="+v)
	}
	for _, expr := range sel.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			parts = append(parts, fmt.Sprintf("%s In [%s]", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpNotIn:
			parts = append(parts, fmt.Sprintf("%s NotIn [%s]", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpExists:
			parts = append(parts, expr.Key+" Exists")
		case metav1.LabelSelectorOpDoesNotExist:
			parts = append(parts, expr.Key+" DoesNotExist")
		}
	}
	sort.Strings(parts)
	return parts
}

func summarizeNetpolPort(port networkingv1.NetworkPolicyPort) string {
	proto := "TCP"
	if port.Protocol != nil {
		proto = string(*port.Protocol)
	}
	portStr := "*"
	if port.Port != nil {
		portStr = port.Port.String()
	}
	if port.EndPort != nil {
		return fmt.Sprintf("port %s-%d/%s", portStr, *port.EndPort, proto)
	}
	return fmt.Sprintf("port %s/%s", portStr, proto)
}

func summarizeNetpolPeer(peer networkingv1.NetworkPolicyPeer) string {
	if peer.IPBlock != nil {
		cidr := peer.IPBlock.CIDR
		if len(peer.IPBlock.Except) > 0 {
			return fmt.Sprintf("cidr %s except [%s]", cidr, strings.Join(peer.IPBlock.Except, ", "))
		}
		return "cidr " + cidr
	}
	hasPod := peer.PodSelector != nil
	hasNS := peer.NamespaceSelector != nil
	switch {
	case hasPod && hasNS:
		podParts := summarizeNetpolSelector(*peer.PodSelector)
		nsParts := summarizeNetpolSelector(*peer.NamespaceSelector)
		return fmt.Sprintf("pods matching %s in namespaces %s", strings.Join(podParts, ","), strings.Join(nsParts, ","))
	case hasPod:
		parts := summarizeNetpolSelector(*peer.PodSelector)
		return "pods matching " + strings.Join(parts, ",")
	case hasNS:
		parts := summarizeNetpolSelector(*peer.NamespaceSelector)
		return "namespaces matching " + strings.Join(parts, ",")
	}
	return "unknown peer"
}

func summarizeNetpolIngressRules(rules []networkingv1.NetworkPolicyIngressRule, typePresent bool) []string {
	if typePresent && len(rules) == 0 {
		return []string{"<deny all>"}
	}
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		result = append(result, summarizeNetpolIngressRule(rule))
	}
	return result
}

func summarizeNetpolEgressRules(rules []networkingv1.NetworkPolicyEgressRule, typePresent bool) []string {
	if typePresent && len(rules) == 0 {
		return []string{"<deny all>"}
	}
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		result = append(result, summarizeNetpolEgressRule(rule))
	}
	return result
}

func summarizeNetpolIngressRule(rule networkingv1.NetworkPolicyIngressRule) string {
	if len(rule.Ports) == 0 && len(rule.From) == 0 {
		return "allow all"
	}
	var parts []string
	if len(rule.Ports) > 0 {
		portStrs := make([]string, 0, len(rule.Ports))
		for _, p := range rule.Ports {
			portStrs = append(portStrs, summarizeNetpolPort(p))
		}
		parts = append(parts, "ports ["+strings.Join(portStrs, ", ")+"]")
	}
	if len(rule.From) > 0 {
		peerStrs := make([]string, 0, len(rule.From))
		for _, peer := range rule.From {
			peerStrs = append(peerStrs, summarizeNetpolPeer(peer))
		}
		parts = append(parts, "from ["+strings.Join(peerStrs, ", ")+"]")
	}
	return strings.Join(parts, " ")
}

func summarizeNetpolEgressRule(rule networkingv1.NetworkPolicyEgressRule) string {
	if len(rule.Ports) == 0 && len(rule.To) == 0 {
		return "allow all"
	}
	var parts []string
	if len(rule.Ports) > 0 {
		portStrs := make([]string, 0, len(rule.Ports))
		for _, p := range rule.Ports {
			portStrs = append(portStrs, summarizeNetpolPort(p))
		}
		parts = append(parts, "ports ["+strings.Join(portStrs, ", ")+"]")
	}
	if len(rule.To) > 0 {
		peerStrs := make([]string, 0, len(rule.To))
		for _, peer := range rule.To {
			peerStrs = append(peerStrs, summarizeNetpolPeer(peer))
		}
		parts = append(parts, "to ["+strings.Join(peerStrs, ", ")+"]")
	}
	return strings.Join(parts, " ")
}
