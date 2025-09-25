package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
)

type IngressPropertyMapper struct{}

func (m *IngressPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	ingressData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ingress resource: %w", err)
	}

	var ingress map[string]any
	if err := json.Unmarshal(ingressData, &ingress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ingress: %w", err)
	}

	properties := map[string]any{}

	spec, _ := ingress["spec"].(map[string]any)
	if spec == nil {
		return properties, nil
	}

	// TLS configuration analysis
	hasTLS := false
	tlsSecrets := []string{}
	tlsHosts := []string{}

	if tls, ok := spec["tls"].([]any); ok {
		hasTLS = len(tls) > 0
		properties["tls_config_count"] = len(tls)

		for _, tlsConfig := range tls {
			if tlsMap, ok := tlsConfig.(map[string]any); ok {
				if secretName, ok := tlsMap["secretName"].(string); ok {
					tlsSecrets = append(tlsSecrets, secretName)
				}
				if hosts, ok := tlsMap["hosts"].([]any); ok {
					for _, host := range hosts {
						if hostStr, ok := host.(string); ok {
							tlsHosts = append(tlsHosts, hostStr)
						}
					}
				}
			}
		}
	}

	properties["has_tls"] = hasTLS
	if len(tlsSecrets) > 0 {
		properties["tls_secrets"] = tlsSecrets
	}
	if len(tlsHosts) > 0 {
		properties["tls_hosts"] = tlsHosts
	}

	if ingressClass, ok := spec["ingressClassName"].(string); ok {
		properties["ingress_class"] = ingressClass
	}

	if defaultBackend, ok := spec["defaultBackend"].(map[string]any); ok {
		properties["has_default_backend"] = true

		if service, ok := defaultBackend["service"].(map[string]any); ok {
			if serviceName, ok := service["name"].(string); ok {
				properties["default_backend_service"] = serviceName
			}
		}
	}

	if rules, ok := spec["rules"].([]any); ok {
		properties["rules_count"] = len(rules)

		var exposedHosts []string
		var exposedPaths []string
		hasWildcardHost := false
		hasWildcardPath := false
		pathTypesUsed := []string{}

		for _, rule := range rules {
			if ruleMap, ok := rule.(map[string]any); ok {
				if host, ok := ruleMap["host"].(string); ok {
					exposedHosts = append(exposedHosts, host)
					if strings.Contains(host, "*") {
						hasWildcardHost = true
					}
				}

				if http, ok := ruleMap["http"].(map[string]any); ok {
					if paths, ok := http["paths"].([]any); ok {
						for _, path := range paths {
							if pathMap, ok := path.(map[string]any); ok {
								if pathValue, ok := pathMap["path"].(string); ok {
									exposedPaths = append(exposedPaths, pathValue)
									if strings.Contains(pathValue, "*") || pathValue == "/" {
										hasWildcardPath = true
									}
								}

								if pathType, ok := pathMap["pathType"].(string); ok {
									pathTypesUsed = append(pathTypesUsed, pathType)
								}
							}
						}
					}
				}
			}
		}

		properties["exposed_hosts"] = exposedHosts
		properties["exposed_paths"] = exposedPaths
		properties["has_wildcard_host"] = hasWildcardHost
		properties["has_wildcard_path"] = hasWildcardPath
		properties["path_types_used"] = pathTypesUsed
		properties["unique_hosts_count"] = len(exposedHosts)
		properties["unique_paths_count"] = len(exposedPaths)
	}

	if metadata, ok := ingress["metadata"].(map[string]any); ok {
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			properties["annotations_count"] = len(annotations)

			securityAnnotations := map[string]any{}

			// SSL redirect
			if sslRedirect, exists := annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; exists {
				securityAnnotations["ssl_redirect"] = sslRedirect
				properties["forces_ssl_redirect"] = sslRedirect == "true"
			}

			if authType, exists := annotations["nginx.ingress.kubernetes.io/auth-type"]; exists {
				securityAnnotations["auth_type"] = authType
				properties["has_authentication"] = true
			}

			if rateLimit, exists := annotations["nginx.ingress.kubernetes.io/rate-limit"]; exists {
				securityAnnotations["rate_limit"] = rateLimit
				properties["has_rate_limiting"] = true
			}

			if whitelist, exists := annotations["nginx.ingress.kubernetes.io/whitelist-source-range"]; exists {
				securityAnnotations["whitelist_source_range"] = whitelist
				properties["has_ip_whitelist"] = true
			}

			if len(securityAnnotations) > 0 {
				properties["security_annotations"] = securityAnnotations
			}
		}
	}

	return properties, nil
}

type IngressParser struct {
	config bloodhound.ResourceConfig
}

func NewIngressParser() *IngressParser {
	return &IngressParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "ingress",
			PrimaryKind:    "Ingress",
			SecondaryKinds: []string{},
			PropertyMapper: &IngressPropertyMapper{},
		},
	}
}

func (p *IngressParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *IngressParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *IngressParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *IngressParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	ingressData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ingress resource: %w", err)
	}

	var ingress map[string]any
	if err := json.Unmarshal(ingressData, &ingress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ingress: %w", err)
	}

	var name string
	if metadata, ok := ingress["metadata"].(map[string]any); ok && metadata != nil {
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
		return nil, fmt.Errorf("failed to create ingress node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
