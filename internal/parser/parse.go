package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	rawResources, err := parseJSONLToRawResourcesReader(reader)
	if err != nil {
		return nil, err
	}
	log.Debug("Parsed JSONL resources", "count", len(rawResources), "duration", time.Since(parseStart))

	buildStart := time.Now()
	nodes, coreFacts, err := createNodesFromRawResources(rawResources, parseUndefinedNodes)
	if err != nil {
		return nil, err
	}
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
func parseJSONLToResources(jsonlData []byte) ([]map[string]any, error) {
	return parseJSONLToResourcesReader(bytes.NewReader(jsonlData))
}

func parseJSONLToResourcesReader(reader io.Reader) ([]map[string]any, error) {
	log := utils.DefaultLogger().Component("parser")

	var extracted []map[string]any
	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		var resource map[string]any
		if err := json.Unmarshal(raw, &resource); err != nil {
			log.Error("Parse JSONL line failed", "line", line, "error", err)
			return errors.New("parse JSONL failed")
		}
		if resource == nil {
			log.Warn("Missing resource payload; skipping line", "line", line)
			return nil
		}
		extracted = append(extracted, resource)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return extracted, nil
}

func parseJSONLToRawResourcesReader(reader io.Reader) ([]json.RawMessage, error) {
	log := utils.DefaultLogger().Component("parser")

	var extracted []json.RawMessage
	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		if !json.Valid(raw) {
			log.Error("Parse JSONL line failed", "line", line)
			return errors.New("parse JSONL failed")
		}
		copied := make([]byte, len(raw))
		copy(copied, raw)
		extracted = append(extracted, json.RawMessage(copied))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return extracted, nil
}

// JSONL parsing utility
func ParseFromJSONL(jsonlData []byte) ([]map[string]any, error) {
	return ParseFromJSONLReader(bytes.NewReader(jsonlData))
}

// ParseFromJSONLReader parses JSONL data from a reader.
func ParseFromJSONLReader(reader io.Reader) ([]map[string]any, error) {
	log := utils.DefaultLogger().Component("parser")
	var resources []map[string]any

	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		var resource map[string]any
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

func createNodesFromRawResources(resources []json.RawMessage, parseUndefinedNodes bool) ([]model.BloodHoundNode, *model.CoreFacts, error) {
	nodeList := []model.BloodHoundNode{}
	coreFacts := model.NewCoreFacts()

	for i, raw := range resources {
		decoded, err := utils.DecodeJSON(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("parse JSONL resource %d: %w", i+1, err)
		}

		result, ok, err := buildNodeFromDecoded(decoded, parseUndefinedNodes)
		if err != nil {
			return nil, nil, fmt.Errorf("build resource %d: %w", i+1, err)
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

	return nodeList, coreFacts, nil
}

func buildNodeFromDecoded(decoded utils.DecodedResource, parseUndefinedNodes bool) (nodes.BuildResult, bool, error) {
	if decoded.Object != nil {
		if result, ok := nodes.BuildTyped(decoded.GVK, decoded.Object); ok {
			return result, true, nil
		}
		resource, err := utils.ToMap(decoded.Object)
		if err != nil {
			return nodes.BuildResult{}, false, err
		}
		return buildNodeFromMap(resource, parseUndefinedNodes)
	}

	if decoded.Raw != nil {
		return buildNodeFromMap(decoded.Raw, parseUndefinedNodes)
	}

	return nodes.BuildResult{}, false, nil
}

func buildNodeFromMap(resource map[string]any, parseUndefinedNodes bool) (nodes.BuildResult, bool, error) {
	if resource == nil {
		return nodes.BuildResult{}, false, nil
	}
	result, ok := nodes.Build(resource)
	if !ok && parseUndefinedNodes {
		result, ok = nodes.BuildGenericNode(resource)
	}
	return result, ok, nil
}
