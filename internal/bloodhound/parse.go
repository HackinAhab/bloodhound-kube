package bloodhound

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound/relationships"
)

// Main parsing function - OPA-based streaming approach
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string) (*BloodHoundResult, error) {
	// Parse raw JSONL into resources
	resources, err := parseJSONLToResources(jsonlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSONL: %w", err)
	}

	// Create nodes using OPA policies with streaming
	nodes, err := createNodesWithOPA(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to create nodes: %w", err)
	}

	// Create relationships using OPA policies
	edges, err := createRelationshipsWithOPA(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationships: %w", err)
	}

	for i := range nodes {
		nodes[i].Properties = SanitizeProperties(nodes[i].Properties)
	}

	for i := range edges {
		edges[i].Properties = SanitizeProperties(edges[i].Properties)
	}

	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{SourceKind: "kubernetes"},
		Graph:    BloodHoundGraph{Nodes: nodes, Edges: edges},
	}, nil
}

// parseJSONLToResources converts JSONL bytes to raw resource maps
// Extracts the .resource field from the JSONL wrapper structure
func parseJSONLToResources(jsonlData []byte) ([]map[string]any, error) {
	lines := strings.Split(string(jsonlData), "\n")
	var resources []map[string]any

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var wrapper map[string]any
		if err := json.Unmarshal([]byte(line), &wrapper); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", i+1, err)
		}

		// Extract the actual K8s resource from the wrapper
		// JSONL format: {"type": "secret", "timestamp": "...", "resource": {...}}
		// OPA policies expect: {"kind": "Secret", "metadata": {...}, ...}
		if resource, ok := wrapper["resource"].(map[string]any); ok {
			resources = append(resources, resource)
		} else {
			// Skip lines without a valid resource field
			fmt.Printf("Warning: Line %d missing 'resource' field, skipping\n", i+1)
		}
	}

	return resources, nil
}

// createNodesWithOPA uses OPA policies to create nodes from resources
// Processes in chunks of 10K for memory efficiency
func createNodesWithOPA(resources []map[string]any) ([]BloodHoundNode, error) {
	// Create OPA engine and load node policies
	engine, err := relationships.NewOPAEngine("config/policies")
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA engine: %w", err)
	}

	// Load node creation policies
	if err := engine.SetNodePolicyDir("config/policies/nodes"); err != nil {
		return nil, fmt.Errorf("failed to load node policies: %w", err)
	}

	// Process in chunks for memory efficiency
	const chunkSize = 10000
	var allNodes []BloodHoundNode

	for i := 0; i < len(resources); i += chunkSize {
		end := min(i+chunkSize, len(resources))

		chunk := resources[i:end]

		// Query OPA for nodes
		sharedNodes, err := engine.QueryNodes(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to query nodes for chunk %d-%d: %w", i, end, err)
		}

		// Convert shared types to main package types
		for _, sharedNode := range sharedNodes {
			allNodes = append(allNodes, BloodHoundNode{
				ID:         sharedNode.ID,
				Kinds:      sharedNode.Kinds,
				Properties: sharedNode.Properties,
			})
		}
	}

	return allNodes, nil
}

// createRelationshipsWithOPA uses OPA policies to create edges from nodes
func createRelationshipsWithOPA(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	return createRelationships(nodes)
}

func ConvertToBloodHound(jsonlData []byte) (*ParsedResult, error) {
	result, err := ConvertToBloodHoundResult(jsonlData, "")
	if err != nil {
		return nil, err
	}

	return &ParsedResult{
		Nodes: result.Graph.Nodes,
		Edges: result.Graph.Edges,
	}, nil
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

// createRelationships applies OPA/Rego policies to create edges between nodes
func createRelationships(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	// Create OPA engine with policy directory
	engine, err := relationships.NewOPAEngine("config/policies")
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA engine: %w", err)
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

	// Apply Rego policies to create relationships
	sharedEdges, err := engine.ApplyRules(sharedNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to apply OPA policies: %w", err)
	}

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
