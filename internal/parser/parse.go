package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"bloodhound-kube/internal/edges"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
	"bloodhound-kube/internal/utils"

	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
)

// Main parsing function - Go-based edge rules
func ConvertToBloodHoundResult(jsonlData []byte, clusterName string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	return ConvertToBloodHoundResultFromReader(bytes.NewReader(jsonlData), clusterName, parseUndefinedNodes)
}

// ConvertToBloodHoundResultFromReader parses JSONL data from a reader.
func ConvertToBloodHoundResultFromReader(reader io.Reader, clusterName string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
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
	edges, err := createRelationships(coreFacts)
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
		var resource model.ResourceData
		if err := json.Unmarshal(raw, &resource); err != nil {
			log.Error("Parse JSONL line failed", "line", line, "error", err)
			return errors.New("parse JSONL failed")
		}

		// Extract the actual K8s resource from the wrapper
		// JSONL format: {"type": "secret", "timestamp": "...", "resource": {...}}
		// Edge rules expect: {"kind": "Secret", "metadata": {...}, ...}
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
func ParseFromJSONL(jsonlData []byte) ([]model.ResourceData, error) {
	return ParseFromJSONLReader(bytes.NewReader(jsonlData))
}

// ParseFromJSONLReader parses JSONL data from a reader.
func ParseFromJSONLReader(reader io.Reader) ([]model.ResourceData, error) {
	log := utils.DefaultLogger().Component("parser")
	var resources []model.ResourceData

	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		var resource model.ResourceData
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

// createRelationships builds edges from typed core facts
func createRelationships(coreFacts *model.CoreFacts) ([]model.BloodHoundEdge, error) {
	if coreFacts == nil {
		return nil, errors.New("create relationships failed")
	}
	edges := edges.BuildEdges(coreFacts)
	return edges, nil
}

func createNodes(resources []map[string]any, parseUndefinedNodes bool) ([]model.BloodHoundNode, *model.CoreFacts) {
	nodeList := []model.BloodHoundNode{}
	coreFacts := model.NewCoreFacts()

	for _, resource := range resources {
		result, ok := nodes.Build(resource)
		if !ok && parseUndefinedNodes {
			result, ok = nodes.BuildGenericNode(resource)
		}
		if !ok {
			continue
		}

		nodeList = append(nodeList, model.BloodHoundNode{
			ID:         result.Node.ID,
			Kinds:      result.Node.Kinds,
			Properties: result.Node.Properties,
		})

		for _, entry := range result.Core {
			coreFacts.Add(entry)
		}
	}

	external := nodes.ExternalNode()
	nodeList = append(nodeList, model.BloodHoundNode{
		ID:         external.ID,
		Kinds:      external.Kinds,
		Properties: external.Properties,
	})
	coreFacts.Add(nodes.CoreEntry{Cluster: true, Data: nodes.ExternalCoreEntry()})

	return nodeList, coreFacts
}
