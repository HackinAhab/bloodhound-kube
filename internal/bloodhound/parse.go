package bloodhound

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

func GenerateObjectID(resourceType, namespace, name string) string {
	var identifier string
	if namespace != "" {
		identifier = fmt.Sprintf("%s/%s/%s", resourceType, namespace, name)
	} else {
		identifier = fmt.Sprintf("%s/%s", resourceType, name)
	}

	hash := sha256.Sum256([]byte(identifier))
	return fmt.Sprintf("%x", hash)[:16]
}

func GenerateNodeID(label, resourceType, namespace, name string) string {
	baseID := GenerateObjectID(resourceType, namespace, name)
	return fmt.Sprintf("%s:%s", label, baseID)
}

func SanitizeLabel(label string) string {
	sanitized := strings.ToUpper(label)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return sanitized
}

func CreateNodeFromResource(kinds []string, resourceType, namespace, name string, properties map[string]any) BloodHoundNode {
	nodeID := GenerateNodeID(kinds[0], resourceType, namespace, name)

	if properties == nil {
		properties = make(map[string]any)
	}

	properties["name"] = name
	properties["resource_type"] = resourceType
	properties["objectid"] = nodeID

	if namespace != "" {
		properties["namespace"] = namespace
	}

	return BloodHoundNode{
		ID:         nodeID,
		Kinds:      kinds,
		Properties: properties,
	}
}

func CreateNodeWithConfig(config ResourceConfig, resourceType, namespace, name string, resource any) (BloodHoundNode, error) {
	kinds := []string{config.PrimaryKind}
	kinds = append(kinds, config.SecondaryKinds...)

	properties, err := config.PropertyMapper.MapProperties(resource)
	if err != nil {
		return BloodHoundNode{}, fmt.Errorf("failed to map properties: %w", err)
	}

	return CreateNodeFromResource(kinds, resourceType, namespace, name, properties), nil
}

func CreateEdge(sourceID, targetID, label string, properties map[string]any) BloodHoundEdge {
	if properties == nil {
		properties = make(map[string]any)
	}

	return BloodHoundEdge{
		Source:     sourceID,
		Target:     targetID,
		Label:      SanitizeLabel(label),
		Properties: properties,
	}
}

func ParseFromNDJSON(ndjsonData []byte) ([]ResourceData, error) {
	lines := strings.Split(string(ndjsonData), "\n")
	var resources []ResourceData

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resource ResourceData
		if err := json.Unmarshal([]byte(line), &resource); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", i+1, err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func ConvertToBloodHound(ndjsonData []byte) (*ParsedResult, error) {
	resources, err := ParseFromNDJSON(ndjsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NDJSON: %w", err)
	}

	return DefaultRegistry.ParseBatch(resources)
}

func ConvertToBloodHoundJSON(ndjsonData []byte) ([]byte, error) {
	result, err := ConvertToBloodHound(ndjsonData)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}
