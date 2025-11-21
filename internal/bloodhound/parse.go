package bloodhound

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound/relationships"
)

// Simplified direct conversion functions (keeping for backward compatibility)
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

func CreateEdge(sourceID, targetID, kind string, properties map[string]any) BloodHoundEdge {
	if properties == nil {
		properties = make(map[string]any)
	}

	return BloodHoundEdge{
		Start: BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   sourceID,
		},
		End: BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   targetID,
		},
		Kind:       SanitizePascalCase(kind),
		Properties: properties,
	}
}

// Main parsing functions - simplified to use converter directly
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string) (*BloodHoundResult, error) {
	// Convert JSONL to nodes using the new converter
	converter := NewResourceConverter()
	nodes, err := converter.ConvertJSONLData(jsonlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert JSONL: %w", err)
	}

	// Create relationships using the rules engine
	edges, err := createRelationships(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationships: %w", err)
	}

	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind: "kubernetes",
		},
		Graph: BloodHoundGraph{
			Nodes: nodes,
			Edges: edges,
		},
	}, nil
}

func ConvertToBloodHound(jsonlData []byte) (*ParsedResult, error) {
	result, err := ConvertToBloodHoundResult(jsonlData, "")
	if err != nil {
		return nil, err
	}

	// Convert to legacy format for backward compatibility
	return &ParsedResult{
		Nodes: result.Graph.Nodes,
		Edges: result.Graph.Edges,
	}, nil
}

func ConvertToBloodHoundJSON(jsonlData []byte) ([]byte, error) {
	result, err := ConvertToBloodHound(jsonlData)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}

// Package-level variable to store additional rules file path
var additionalRulesFilePath string

// SetAdditionalRulesFile sets the path to an additional rules file
func SetAdditionalRulesFile(filepath string) {
	additionalRulesFilePath = filepath
}

// JSONL parsing utility
func ParseFromJSONL(jsonlData []byte) ([]ResourceData, error) {
	lines := strings.Split(string(jsonlData), "\n")
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

// createRelationships applies rules engine to create edges between nodes
func createRelationships(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	// Create rules engine
	engine := relationships.NewEngine()

	// Load rules from config/rules directory with fallback to embedded rules
	if err := engine.LoadRulesWithFallback(
		"config/rules",    // Primary: config/rules/*.yaml
		getBuiltinRules(), // Fallback: embedded rules
	); err != nil {
		return nil, fmt.Errorf("failed to load relationship rules: %w", err)
	}

	// Load additional rules file if specified
	if additionalRulesFilePath != "" {
		if err := engine.LoadAdditionalRulesFile(additionalRulesFilePath); err != nil {
			return nil, fmt.Errorf("failed to load additional rules file: %w", err)
		}
	}

	// Convert to shared types for the relationship engine
	sharedNodes := make([]relationships.BloodHoundNode, len(nodes))
	for i, node := range nodes {
		sharedNodes[i] = relationships.BloodHoundNode{
			ID:         node.ID,
			Kinds:      node.Kinds,
			Properties: node.Properties,
		}
	}

	// Apply rules to create relationships
	sharedEdges := engine.ApplyRules(sharedNodes)

	// Deduplicate and sort edges
	sharedEdges = relationships.DeduplicateEdges(sharedEdges)
	relationships.SortEdgesByKind(sharedEdges)

	// Convert back to main package types
	edges := make([]BloodHoundEdge, len(sharedEdges))
	for i, edge := range sharedEdges {
		edges[i] = BloodHoundEdge{
			Start: BloodHoundEdgeRef{
				MatchBy: edge.Start.MatchBy,
				Value:   edge.Start.Value,
				Kind:    edge.Start.Kind,
			},
			End: BloodHoundEdgeRef{
				MatchBy: edge.End.MatchBy,
				Value:   edge.End.Value,
				Kind:    edge.End.Kind,
			},
			Kind:       edge.Kind,
			Properties: edge.Properties,
		}
	}

	return edges, nil
}

// getBuiltinRules returns basic embedded rules as final fallback
func getBuiltinRules() string {
	return `
version: "1.0"
relationships:
  - name: "deployment_owns_pods"
    description: "Deployment owns Pods via ReplicaSet"
    source_type: ["deployment"]
    target_type: ["pod"]
    edge_type: "Owns"
    conditions:
      - "source.selector in target.labels"
    priority: 5
    enabled: true

  - name: "service_exposes_pods"
    description: "Service exposes Pods via selector"
    source_type: ["service"]
    target_type: ["pod"]
    edge_type: "Exposes"
    conditions:
      - "source.selector subset_of target.labels"
    priority: 7
    enabled: true

  - name: "secret_mounted_by_pod"
    description: "Secret is mounted by Pod"
    source_type: ["secret"]
    target_type: ["pod"]
    edge_type: "MountedBy"
    conditions:
      - "source.name in target.volumes[*].secret.secretName"
    priority: 8
    enabled: true
`
}

// Legacy concurrent processing (simplified)
func ConcurrentParseProcessor(resources []ResourceData, workerCount int) (*BloodHoundResult, error) {
	// For simplicity, just process directly without complex worker pools
	var allNodes []BloodHoundNode

	for _, resource := range resources {
		parser, exists := DefaultRegistry.GetParser(resource.Type)
		if !exists {
			continue
		}

		result, err := parser.Parse(resource)
		if err != nil {
			continue
		}

		allNodes = append(allNodes, result.Nodes...)
	}

	// Create relationships
	edges, err := createRelationships(allNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationships: %w", err)
	}

	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind: "kubernetes",
		},
		Graph: BloodHoundGraph{
			Nodes: allNodes,
			Edges: edges,
		},
	}, nil
}
