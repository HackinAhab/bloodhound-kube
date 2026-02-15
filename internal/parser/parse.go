package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
)

// Main parsing function - OPA-based streaming approach
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string) (*gopengraph.OpenGraph, error) {
	_ = clusterName
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

	graph := gopengraph.NewOpenGraph("Kubernetes")

	for _, n := range nodes {
		props := properties.NewPropertiesFromMap(n.Properties)
		openNode, err := node.NewNode(n.ID, n.Kinds, props)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", n.ID, err)
		}

		graph.AddNode(openNode)
	}

	for _, e := range edges {
		props := properties.NewPropertiesFromMap(e.Properties)
		openEdge, err := edge.NewEdge(e.Start.Value, e.End.Value, e.Kind, props)
		if err != nil {
			return nil, fmt.Errorf("failed to create edge %s: %w", e.Kind, err)
		}

		graph.AddEdge(openEdge)
	}

	return graph, nil
}

// parseJSONLToResources converts JSONL bytes to raw resource maps
// Extracts the .resource field from the JSONL wrapper structure
func parseJSONLToResources(jsonlData []byte) ([]map[string]any, error) {
	resources, err := ParseFromJSONL(jsonlData)
	if err != nil {
		return nil, err
	}

	var extracted []map[string]any
	for i, resource := range resources {
		// Extract the actual K8s resource from the wrapper
		// JSONL format: {"type": "secret", "timestamp": "...", "resource": {...}}
		// OPA policies expect: {"kind": "Secret", "metadata": {...}, ...}
		if payload, ok := resource.Resource.(map[string]any); ok {
			extracted = append(extracted, payload)
			continue
		}
		fmt.Printf("Warning: Line %d missing 'resource' field, skipping\n", i+1)
	}

	return extracted, nil
}

// createNodesWithOPA uses OPA policies to create nodes from resources
// Processes in chunks of 10K for memory efficiency
func createNodesWithOPA(resources []map[string]any) ([]BloodHoundNode, error) {
	// Create OPA engine and load node policies
	engine := NewOPAEngineForNodes()

	// Load node creation policies
	if err := engine.SetNodePolicyDir("rego/nodes"); err != nil {
		return nil, fmt.Errorf("failed to load node policies: %w", err)
	}

	// Process in chunks for memory efficiency
	const chunkSize = 10000
	var allNodes []BloodHoundNode

	for i := 0; i < len(resources); i += chunkSize {
		end := min(i+chunkSize, len(resources))

		chunk := resources[i:end]

		// Query OPA for nodes
		nodes, err := engine.QueryNodes(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to query nodes for chunk %d-%d: %w", i, end, err)
		}
		allNodes = append(allNodes, nodes...)
	}

	return allNodes, nil
}

// createRelationshipsWithOPA uses OPA policies to create edges from nodes
func createRelationshipsWithOPA(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	return createRelationships(nodes)
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
	engine, err := NewOPAEngine("rego/edges")
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA engine: %w", err)
	}

	// Apply Rego policies to create relationships
	edges, err := engine.ApplyRules(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to apply OPA policies: %w", err)
	}

	return edges, nil
}
