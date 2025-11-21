package bloodhound

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ResourceConverter handles direct conversion from JSONL resources to BloodHound nodes
// This replaces the complex dispatcher/extractors/factory chain with simple, direct conversion
type ResourceConverter struct {
	registry *ParseRegistry
}

// NewResourceConverter creates a new converter with the default registry
func NewResourceConverter() *ResourceConverter {
	return &ResourceConverter{
		registry: DefaultRegistry,
	}
}

// ConvertJSONLLine converts a single JSONL line directly to a BloodHound node
func (c *ResourceConverter) ConvertJSONLLine(jsonLine []byte) (*BloodHoundNode, error) {
	// Parse the resource type first
	var resourceType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(jsonLine, &resourceType); err != nil {
		return nil, fmt.Errorf("failed to determine resource type: %w", err)
	}

	// Parse the full resource data
	var resourceData ResourceData
	if err := json.Unmarshal(jsonLine, &resourceData); err != nil {
		return nil, fmt.Errorf("failed to parse resource: %w", err)
	}

	// Use registry for all resource types
	return c.convertGeneric(resourceData)
}

// convertGeneric uses registered parsers for all resource types
func (c *ResourceConverter) convertGeneric(resource ResourceData) (*BloodHoundNode, error) {
	parser, exists := c.registry.GetParser(resource.Type)
	if !exists {
		return nil, fmt.Errorf("no parser found for resource type: %s", resource.Type)
	}

	result, err := parser.Parse(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource: %w", err)
	}

	if len(result.Nodes) > 0 {
		return &result.Nodes[0], nil
	}

	return nil, nil
}

// ConvertBatch converts multiple JSONL lines efficiently
func (c *ResourceConverter) ConvertBatch(jsonLines [][]byte) ([]*BloodHoundNode, error) {
	nodes := make([]*BloodHoundNode, 0, len(jsonLines))

	for i, line := range jsonLines {
		node, err := c.ConvertJSONLLine(line)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Warning: Failed to convert line %d: %v\n", i+1, err)
			continue
		}

		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// ConvertJSONLData converts JSONL data from bytes to nodes
func (c *ResourceConverter) ConvertJSONLData(jsonlData []byte) ([]BloodHoundNode, error) {
	lines := strings.Split(string(jsonlData), "\n")
	var nodes []BloodHoundNode

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		node, err := c.ConvertJSONLLine([]byte(line))
		if err != nil {
			fmt.Printf("Warning: Failed to convert line %d: %v\n", i+1, err)
			continue
		}

		if node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes, nil
}
