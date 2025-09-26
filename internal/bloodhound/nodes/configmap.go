package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
)

type ConfigMapPropertyMapper struct{}

func (m *ConfigMapPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	configMapData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configmap resource: %w", err)
	}

	var configMap map[string]any
	if err := json.Unmarshal(configMapData, &configMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configmap: %w", err)
	}

	properties := map[string]any{}

	if name, ok := configMap["name"].(string); ok {
		properties["name"] = name
	}

	if data, ok := configMap["data"].(map[string]any); ok {
		properties["data_keys_count"] = len(data)
		properties["has_data"] = len(data) > 0

		var suspiciousKeys []string
		var potentialCredentials []string
		hasSensitivePatterns := false

		for key, value := range data {
			lowerKey := strings.ToLower(key)

			sensitivePatterns := []string{
				"password", "passwd", "pass", "pwd",
				"token", "key", "secret", "credential", "cred",
				"auth", "authorization", "bearer",
				"api_key", "apikey", "access_key",
				"private", "cert", "certificate", "pem",
			}

			for _, pattern := range sensitivePatterns {
				if strings.Contains(lowerKey, pattern) {
					suspiciousKeys = append(suspiciousKeys, key)
					hasSensitivePatterns = true
					break
				}
			}

			if valueStr, ok := value.(string); ok {
				if len(valueStr) > 20 && strings.Contains(valueStr, "=") {
					potentialCredentials = append(potentialCredentials, key)
				}

				if strings.Contains(valueStr, "://") && (strings.Contains(valueStr, "@") || strings.Contains(valueStr, "token=")) {
					potentialCredentials = append(potentialCredentials, key)
					hasSensitivePatterns = true
				}

				if strings.Count(valueStr, ".") >= 2 && len(valueStr) > 50 {
					potentialCredentials = append(potentialCredentials, key)
					hasSensitivePatterns = true
				}
			}
		}

		properties["data"] = data
		properties["has_suspicious_keys"] = len(suspiciousKeys) > 0
		properties["has_sensitive_patterns"] = hasSensitivePatterns

		properties["suspicious_keys_count"] = len(suspiciousKeys)
		properties["potential_credentials_count"] = len(potentialCredentials)
	} else {
		properties["has_data"] = false
		properties["data_keys_count"] = 0
	}

	if binaryData, ok := configMap["binaryData"].(map[string]any); ok {
		properties["binary_data_keys_count"] = len(binaryData)
		properties["has_binary_data"] = len(binaryData) > 0

		// Count binary keys but don't store the array
	} else {
		properties["has_binary_data"] = false
		properties["binary_data_keys_count"] = 0
	}

	if metadata, ok := configMap["metadata"].(map[string]any); ok {
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			properties["annotations_count"] = len(annotations)

			if _, exists := annotations["kubectl.kubernetes.io/last-applied-configuration"]; exists {
				properties["managed_by_kubectl"] = true
			}
		}
	}

	return properties, nil
}

type ConfigMapParser struct {
	config bloodhound.ResourceConfig
}

func NewConfigMapParser() *ConfigMapParser {
	return &ConfigMapParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "configmap",
			PrimaryKind:    "ConfigMap",
			SecondaryKinds: []string{},
			PropertyMapper: &ConfigMapPropertyMapper{},
		},
	}
}

func (p *ConfigMapParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *ConfigMapParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *ConfigMapParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *ConfigMapParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	configMapData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configmap resource: %w", err)
	}

	var configMap map[string]any
	if err := json.Unmarshal(configMapData, &configMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configmap: %w", err)
	}

	name := configMap["name"].(string)

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create configmap node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
