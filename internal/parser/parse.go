package parser

import (
	"encoding/json"
	"errors"
	"strings"

	"bloodhound-kube/internal/utils"
	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
)

// Main parsing function - OPA-based streaming approach
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string, policyDirs []string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	_ = clusterName
	log := utils.DefaultLogger().Component("parser")
	// Parse raw JSONL into resources
	resources, err := parseJSONLToResources(jsonlData)
	if err != nil {
		return nil, err
	}

	// Create nodes using OPA policies with streaming
	nodes, err := createNodesWithOPA(resources, policyDirs, parseUndefinedNodes)
	if err != nil {
		return nil, err
	}

	// Create relationships using OPA policies
	edges, err := createRelationshipsWithOPA(nodes, policyDirs)
	if err != nil {
		return nil, err
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
			log.Error("Create node failed", "id", n.ID, "error", err)
			return nil, errors.New("parse failed")
		}

		graph.AddNode(openNode)
	}

	for _, e := range edges {
		props := properties.NewPropertiesFromMap(e.Properties)
		openEdge, err := edge.NewEdge(e.Start.Value, e.End.Value, e.Kind, props)
		if err != nil {
			log.Error("Create edge failed", "kind", e.Kind, "error", err)
			return nil, errors.New("parse failed")
		}

		graph.AddEdge(openEdge)
	}

	return graph, nil
}

// parseJSONLToResources converts JSONL bytes to raw resource maps
// Extracts the .resource field from the JSONL wrapper structure
func parseJSONLToResources(jsonlData []byte) ([]map[string]any, error) {
	log := utils.DefaultLogger().Component("parser")
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
		log.Warn("Missing resource field; skipping line", "line", i+1)
	}

	return extracted, nil
}

// createNodesWithOPA uses OPA policies to create nodes from resources
// Processes in chunks of 10K for memory efficiency
func createNodesWithOPA(resources []map[string]any, policyDirs []string, parseUndefinedNodes bool) ([]BloodHoundNode, error) {
	log := utils.DefaultLogger().Component("parser")
	// Create OPA engine and load node policies
	engine := NewOPAEngineForNodes(policyDirs)

	// Load node creation policies
	if err := engine.SetNodePolicyDir("rego/nodes"); err != nil {
		return nil, err
	}

	// Process in chunks for memory efficiency
	const chunkSize = 10000
	var allNodes []BloodHoundNode

	for i := 0; i < len(resources); i += chunkSize {
		end := min(i+chunkSize, len(resources))

		chunk := resources[i:end]

		// Query OPA for nodes
		nodes, err := engine.QueryNodes(chunk, parseUndefinedNodes)
		if err != nil {
			log.Error("Query nodes failed", "chunk_start", i, "chunk_end", end, "error", err)
			return nil, errors.New("create nodes failed")
		}
		allNodes = append(allNodes, nodes...)
	}

	return allNodes, nil
}

// createRelationshipsWithOPA uses OPA policies to create edges from nodes
func createRelationshipsWithOPA(nodes []BloodHoundNode, policyDirs []string) ([]BloodHoundEdge, error) {
	return createRelationships(nodes, policyDirs)
}

// JSONL parsing utility
func ParseFromJSONL(jsonlData []byte) ([]ResourceData, error) {
	log := utils.DefaultLogger().Component("parser")
	lines := strings.Split(string(jsonlData), "\n")
	var resources []ResourceData

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resource ResourceData
		if err := json.Unmarshal([]byte(line), &resource); err != nil {
			log.Error("Parse JSONL line failed", "line", i+1, "error", err)
			return nil, errors.New("parse JSONL failed")
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// createRelationships applies OPA/Rego policies to create edges between nodes
func createRelationships(nodes []BloodHoundNode, policyDirs []string) ([]BloodHoundEdge, error) {
	log := utils.DefaultLogger().Component("parser")
	// Create OPA engine with policy directory
	engine, err := NewOPAEngine("rego/edges", policyDirs)
	if err != nil {
		return nil, err
	}

	// Apply Rego policies to create relationships
	edges, err := engine.ApplyRules(nodes)
	if err != nil {
		log.Error("Apply OPA policies failed", "error", err)
		return nil, errors.New("create relationships failed")
	}

	return edges, nil
}
