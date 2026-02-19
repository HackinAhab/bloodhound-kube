package parser

import (
	"encoding/json"
	"errors"
	"strings"

	"bloodhound-kube/internal/parser/nodes"
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

	// Create nodes in Go and build core facts
	nodes, coreFacts := createNodes(resources, parseUndefinedNodes)

	// Create relationships using OPA policies
	edges, err := createRelationships(nodes, coreFacts, policyDirs)
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
func createRelationships(nodes []BloodHoundNode, coreFacts *CoreFacts, policyDirs []string) ([]BloodHoundEdge, error) {
	log := utils.DefaultLogger().Component("parser")
	// Create OPA engine with policy directory
	engine, err := NewOPAEngine("rego/edges", policyDirs)
	if err != nil {
		return nil, err
	}

	// Apply Rego policies to create relationships
	edges, err := engine.ApplyRulesWithCore(nodes, coreFacts)
	if err != nil {
		log.Error("Apply OPA policies failed", "error", err)
		return nil, errors.New("create relationships failed")
	}

	return edges, nil
}

func createNodes(resources []map[string]any, parseUndefinedNodes bool) ([]BloodHoundNode, *CoreFacts) {
	nodeList := []BloodHoundNode{}
	coreFacts := NewCoreFacts()

	for _, resource := range resources {
		result, ok := nodes.Build(resource)
		if !ok && parseUndefinedNodes {
			result, ok = nodes.BuildGenericNode(resource)
		}
		if !ok {
			continue
		}

		nodeList = append(nodeList, BloodHoundNode{
			ID:         result.Node.ID,
			Kinds:      result.Node.Kinds,
			Properties: result.Node.Properties,
		})

		for _, entry := range result.Core {
			coreFacts.Add(entry)
		}
	}

	external := nodes.ExternalNode()
	nodeList = append(nodeList, BloodHoundNode{
		ID:         external.ID,
		Kinds:      external.Kinds,
		Properties: external.Properties,
	})

	return nodeList, coreFacts
}
