package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
)

type GatewayPropertyMapper struct{}

func (m *GatewayPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	gatewayData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway resource: %w", err)
	}

	var gateway map[string]any
	if err := json.Unmarshal(gatewayData, &gateway); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway: %w", err)
	}

	properties := map[string]any{}

	spec, _ := gateway["spec"].(map[string]any)
	if spec == nil {
		return properties, nil
	}

	if gatewayClass, ok := spec["gatewayClassName"].(string); ok {
		properties["gateway_class"] = gatewayClass
	}

	if listeners, ok := spec["listeners"].([]any); ok {
		properties["listeners_count"] = len(listeners)

		var protocols []string
		var ports []int
		hasTLS := false
		hasHTTPS := false
		hasHTTP := false
		hasWildcardHostname := false
		allowedRoutes := []string{}

		for _, listener := range listeners {
			if listenerMap, ok := listener.(map[string]any); ok {
				if protocol, ok := listenerMap["protocol"].(string); ok {
					protocols = append(protocols, protocol)

					switch strings.ToLower(protocol) {
					case "https", "tls":
						hasTLS = true
						hasHTTPS = true
					case "http":
						hasHTTP = true
					}
				}

				if port, ok := listenerMap["port"].(float64); ok {
					portInt := int(port)
					ports = append(ports, portInt)
				}

				if hostname, ok := listenerMap["hostname"].(string); ok {
					if strings.Contains(hostname, "*") {
						hasWildcardHostname = true
					}
				}

				// TLS configuration
				if tls, ok := listenerMap["tls"].(map[string]any); ok {
					hasTLS = true

					if mode, ok := tls["mode"].(string); ok {
						properties["tls_mode"] = mode
						properties["tls_terminate"] = mode == "Terminate"
						properties["tls_passthrough"] = mode == "Passthrough"
					}
				}

				if allowedRoutesMap, ok := listenerMap["allowedRoutes"].(map[string]any); ok {
					if namespaces, ok := allowedRoutesMap["namespaces"].(map[string]any); ok {
						if from, ok := namespaces["from"].(string); ok {
							allowedRoutes = append(allowedRoutes, from)
						}
					}
				}
			}
		}

		properties["protocols"] = protocols
		properties["ports"] = ports
		properties["has_tls"] = hasTLS
		properties["has_https"] = hasHTTPS
		properties["has_http"] = hasHTTP
		properties["has_wildcard_hostname"] = hasWildcardHostname
		properties["mixed_protocols"] = hasHTTP && hasHTTPS
		properties["allowed_routes"] = allowedRoutes
	}

	if addresses, ok := spec["addresses"].([]any); ok {
		properties["addresses_count"] = len(addresses)

		var addressTypes []string
		for _, address := range addresses {
			if addressMap, ok := address.(map[string]any); ok {
				if addressType, ok := addressMap["type"].(string); ok {
					addressTypes = append(addressTypes, addressType)
				}
			}
		}
		properties["address_types"] = addressTypes
	}

	if status, ok := gateway["status"].(map[string]any); ok {
		if conditions, ok := status["conditions"].([]any); ok {
			properties["conditions_count"] = len(conditions)

			isReady := false
			for _, condition := range conditions {
				if condMap, ok := condition.(map[string]any); ok {
					if condType, ok := condMap["type"].(string); ok {
						if condType == "Ready" || condType == "Programmed" {
							if status, ok := condMap["status"].(string); ok {
								isReady = status == "True"
							}
						}
					}
				}
			}
			properties["is_ready"] = isReady
		}

		if listeners, ok := status["listeners"].([]any); ok {
			attachedRoutes := 0
			for _, listener := range listeners {
				if listenerMap, ok := listener.(map[string]any); ok {
					if attachedRoutesCount, ok := listenerMap["attachedRoutes"].(float64); ok {
						attachedRoutes += int(attachedRoutesCount)
					}
				}
			}
			properties["attached_routes_total"] = attachedRoutes
			properties["has_attached_routes"] = attachedRoutes > 0
		}
	}

	return properties, nil
}

type GatewayParser struct {
	config bloodhound.ResourceConfig
}

func NewGatewayParser() *GatewayParser {
	return &GatewayParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "gateway",
			PrimaryKind:    "Gateway",
			SecondaryKinds: []string{},
			PropertyMapper: &GatewayPropertyMapper{},
		},
	}
}

func (p *GatewayParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *GatewayParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *GatewayParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *GatewayParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	gatewayData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway resource: %w", err)
	}

	var gateway map[string]any
	if err := json.Unmarshal(gatewayData, &gateway); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway: %w", err)
	}

	var name string
	if metadata, ok := gateway["metadata"].(map[string]any); ok && metadata != nil {
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
		return nil, fmt.Errorf("failed to create gateway node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
