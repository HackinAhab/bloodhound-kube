package nodes

import (
	"encoding/json"
	"fmt"

	"bloodhound-kube/internal/bloodhound"
)

type NetworkPolicyPropertyMapper struct{}

func (m *NetworkPolicyPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	policyData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network policy resource: %w", err)
	}

	var policy map[string]any
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network policy: %w", err)
	}

	properties := map[string]any{}

	// The data is directly in the policy object, not under "spec"
	if podSelector, ok := policy["pod_selector"].(map[string]any); ok {
		if matchLabels, ok := podSelector["match_labels"].(map[string]any); ok {
			properties["pod_selector_labels_count"] = len(matchLabels)
			properties["has_pod_selector"] = len(matchLabels) > 0

			if len(matchLabels) == 0 {
				properties["applies_to_all_pods"] = true
			}
		} else {
			properties["applies_to_all_pods"] = true
			properties["has_pod_selector"] = false
		}
	}
	if policyTypes, ok := policy["policy_types"].([]any); ok {
		properties["policy_types_count"] = len(policyTypes)
	}

	if ingress, ok := policy["ingress"].([]any); ok {
		properties["ingress_rules_count"] = len(ingress)
		properties["has_ingress_rules"] = len(ingress) > 0

		if len(ingress) == 0 {
			if properties["controls_ingress"] == true {
				properties["denies_all_ingress"] = true
			}
		} else {
			fromRulesCount := 0
			portRulesCount := 0
			allowsFromAll := false

			for _, rule := range ingress {
				if ruleMap, ok := rule.(map[string]any); ok {
					if from, ok := ruleMap["from"].([]any); ok {
						fromRulesCount += len(from)

						if len(from) == 0 {
							allowsFromAll = true
						}
					}

					if ports, ok := ruleMap["ports"].([]any); ok {
						portRulesCount += len(ports)
					}
				}
			}

			properties["ingress_from_rules_total"] = fromRulesCount
			properties["ingress_port_rules_total"] = portRulesCount
			properties["allows_ingress_from_all"] = allowsFromAll
		}
	}

	if egress, ok := policy["egress"].([]any); ok {
		properties["egress_rules_count"] = len(egress)
		properties["has_egress_rules"] = len(egress) > 0

		if len(egress) == 0 {
			properties["denies_all_egress"] = true
		} else {
			toRulesCount := 0
			portRulesCount := 0
			allowsToAll := false

			for _, rule := range egress {
				if ruleMap, ok := rule.(map[string]any); ok {
					if to, ok := ruleMap["to"].([]any); ok {
						toRulesCount += len(to)

						if len(to) == 0 {
							allowsToAll = true
						}
					}

					if ports, ok := ruleMap["ports"].([]any); ok {
						portRulesCount += len(ports)
					}
				}
			}

			properties["egress_to_rules_total"] = toRulesCount
			properties["egress_port_rules_total"] = portRulesCount
			properties["allows_egress_to_all"] = allowsToAll
		}
	}

	deniesAllIngress := properties["denies_all_ingress"] == true
	deniesAllEgress := properties["denies_all_egress"] == true
	allowsIngressFromAll := properties["allows_ingress_from_all"] == true
	allowsEgressToAll := properties["allows_egress_to_all"] == true

	properties["is_restrictive"] = deniesAllIngress || deniesAllEgress
	properties["is_permissive"] = allowsIngressFromAll || allowsEgressToAll
	properties["is_default_deny"] = deniesAllIngress && deniesAllEgress

	return properties, nil
}

type NetworkPolicyParser struct {
	config bloodhound.ResourceConfig
}

func NewNetworkPolicyParser() *NetworkPolicyParser {
	return &NetworkPolicyParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "networkpolicy",
			PrimaryKind:    "KubeNetworkPolicy",
			SecondaryKinds: []string{},
			PropertyMapper: &NetworkPolicyPropertyMapper{},
		},
	}
}

func (p *NetworkPolicyParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *NetworkPolicyParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *NetworkPolicyParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *NetworkPolicyParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	policyData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network policy resource: %w", err)
	}

	var policy map[string]any
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network policy: %w", err)
	}

	var name string
	if nameValue, ok := policy["name"].(string); ok {
		name = nameValue
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create network policy node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
