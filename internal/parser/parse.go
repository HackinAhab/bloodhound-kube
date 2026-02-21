package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"bloodhound-kube/internal/parser/nodes"
	"bloodhound-kube/internal/utils"

	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
)

// Main parsing function - OPA-based streaming approach
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string, policyDirs []string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	return ConvertToBloodHoundResultFromReader(bytes.NewReader(jsonlData), clusterName, policyDirs, parseUndefinedNodes)
}

// ConvertToBloodHoundResultFromReader parses JSONL data from a reader.
func ConvertToBloodHoundResultFromReader(reader io.Reader, clusterName string, policyDirs []string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	_ = clusterName
	log := utils.DefaultLogger().Component("parser")

	parseStart := time.Now()
	resources, err := parseJSONLToResourcesReader(reader)
	if err != nil {
		return nil, err
	}
	log.Debug("Parsed JSONL resources", "count", len(resources), "duration", time.Since(parseStart))

	buildStart := time.Now()
	nodes, coreFacts := createNodes(resources, parseUndefinedNodes)
	log.Debug("Built nodes and core facts", "nodes", len(nodes), "duration", time.Since(buildStart))

	edgeStart := time.Now()
	edges, err := createRelationships(nodes, coreFacts, policyDirs)
	if err != nil {
		return nil, err
	}
	log.Debug("Created edges", "edges", len(edges), "duration", time.Since(edgeStart))

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
	return parseJSONLToResourcesReader(bytes.NewReader(jsonlData))
}

func parseJSONLToResourcesReader(reader io.Reader) ([]map[string]any, error) {
	log := utils.DefaultLogger().Component("parser")

	var extracted []map[string]any
	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		var resource ResourceData
		if err := json.Unmarshal(raw, &resource); err != nil {
			log.Error("Parse JSONL line failed", "line", line, "error", err)
			return errors.New("parse JSONL failed")
		}

		// Extract the actual K8s resource from the wrapper
		// JSONL format: {"type": "secret", "timestamp": "...", "resource": {...}}
		// OPA policies expect: {"kind": "Secret", "metadata": {...}, ...}
		if payload, ok := resource.Resource.(map[string]any); ok {
			extracted = append(extracted, payload)
			return nil
		}
		log.Warn("Missing resource field; skipping line", "line", line)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return extracted, nil
}

// JSONL parsing utility
func ParseFromJSONL(jsonlData []byte) ([]ResourceData, error) {
	return ParseFromJSONLReader(bytes.NewReader(jsonlData))
}

// ParseFromJSONLReader parses JSONL data from a reader.
func ParseFromJSONLReader(reader io.Reader) ([]ResourceData, error) {
	log := utils.DefaultLogger().Component("parser")
	var resources []ResourceData

	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		var resource ResourceData
		if err := json.Unmarshal(raw, &resource); err != nil {
			log.Error("Parse JSONL line failed", "line", line, "error", err)
			return errors.New("parse JSONL failed")
		}
		resources = append(resources, resource)
		return nil
	})
	if err != nil {
		return nil, err
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
