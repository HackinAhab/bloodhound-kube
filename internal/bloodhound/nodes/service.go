package nodes

import (
	"encoding/json"
	"fmt"
	"strconv"

	"bloodhound-kube/internal/bloodhound"
)

type ServicePropertyMapper struct{}

func (m *ServicePropertyMapper) MapProperties(resource any) (map[string]any, error) {
	serviceData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service resource: %w", err)
	}

	var service map[string]any
	if err := json.Unmarshal(serviceData, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service: %w", err)
	}

	properties := map[string]any{}

	spec, _ := service["spec"].(map[string]any)
	if spec == nil {
		return properties, nil
	}

	serviceType, _ := spec["type"].(string)
	properties["service_type"] = serviceType
	properties["is_load_balancer"] = serviceType == "LoadBalancer"
	properties["is_node_port"] = serviceType == "NodePort"
	properties["is_external"] = serviceType == "LoadBalancer" || serviceType == "NodePort"

	if externalTrafficPolicy, ok := spec["externalTrafficPolicy"].(string); ok {
		properties["external_traffic_policy"] = externalTrafficPolicy
		properties["preserves_source_ip"] = externalTrafficPolicy == "Local"
	}

	if sessionAffinity, ok := spec["sessionAffinity"].(string); ok {
		properties["session_affinity"] = sessionAffinity
		properties["has_session_affinity"] = sessionAffinity != "None"
	}

	if ports, ok := spec["ports"].([]any); ok {
		properties["port_count"] = len(ports)

		hasNodePort := false
		hasPrivilegedPorts := false
		protocolTCP := false
		protocolUDP := false

		for _, port := range ports {
			if portMap, ok := port.(map[string]any); ok {
				if portNum, ok := portMap["port"]; ok {
					if portInt, err := strconv.Atoi(fmt.Sprintf("%.0f", portNum)); err == nil {
						if portInt < 1024 {
							hasPrivilegedPorts = true
						}
					}
				}

				if protocol, ok := portMap["protocol"].(string); ok {
					if protocol == "TCP" {
						protocolTCP = true
					} else if protocol == "UDP" {
						protocolUDP = true
					}
				}

				if _, ok := portMap["nodePort"]; ok {
					hasNodePort = true
				}
			}
		}

		properties["has_node_port"] = hasNodePort
		properties["has_privileged_ports"] = hasPrivilegedPorts
		properties["uses_tcp"] = protocolTCP
		properties["uses_udp"] = protocolUDP
	}

	if selector, ok := spec["selector"].(map[string]any); ok {
		properties["has_selector"] = len(selector) > 0
		properties["selector_count"] = len(selector)

		if app, exists := selector["app"]; exists {
			properties["selects_by_app"] = true
			properties["app_selector"] = app
		}
	} else {
		properties["has_selector"] = false
		properties["is_headless"] = true
	}

	if externalName, ok := spec["externalName"].(string); ok {
		properties["external_name"] = externalName
		properties["is_external_name"] = true
	}

	if sourceRanges, ok := spec["loadBalancerSourceRanges"].([]any); ok {
		properties["load_balancer_source_ranges_count"] = len(sourceRanges)
		properties["has_source_restrictions"] = len(sourceRanges) > 0
	}

	return properties, nil
}

type ServiceParser struct {
	config bloodhound.ResourceConfig
}

func NewServiceParser() *ServiceParser {
	return &ServiceParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "service",
			PrimaryKind:    "Service",
			SecondaryKinds: []string{},
			PropertyMapper: &ServicePropertyMapper{},
		},
	}
}

func (p *ServiceParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *ServiceParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *ServiceParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *ServiceParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	serviceData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service resource: %w", err)
	}

	var service map[string]any
	if err := json.Unmarshal(serviceData, &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service: %w", err)
	}

	var name string
	if metadata, ok := service["metadata"].(map[string]any); ok && metadata != nil {
		name, _ = metadata["name"].(string)
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
