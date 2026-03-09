package parser

import (
	"bytes"
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
func ConvertToBloodHoundResult(jsonlData []byte, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	return ConvertToBloodHoundResultFromReader(bytes.NewReader(jsonlData), parseUndefinedNodes)
}

// ConvertToBloodHoundResultFromReader parses JSONL data from a reader.
func ConvertToBloodHoundResultFromReader(reader io.Reader, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
	log := utils.DefaultLogger().Component("parser")

	processStart := time.Now()
	nodes, coreFacts, parsedResources, err := createNodesAndCoreFactsFromReader(reader, parseUndefinedNodes)
	if err != nil {
		return nil, err
	}
	log.Debug("Parsed resources and built core facts", "resources", parsedResources, "nodes", len(nodes), "duration", time.Since(processStart))

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

// createRelationships builds edges from typed core facts
func createRelationships(coreFacts *model.CoreFacts) ([]model.BloodHoundEdge, error) {
	if coreFacts == nil {
		return nil, errors.New("create relationships failed")
	}
	edges := edges.BuildEdges(coreFacts)
	return edges, nil
}

func createNodesAndCoreFactsFromReader(reader io.Reader, parseUndefinedNodes bool) ([]model.BloodHoundNode, *model.CoreFacts, int, error) {
	if reader == nil {
		return nil, nil, 0, errors.New("parse JSONL failed")
	}

	nodeList := []model.BloodHoundNode{}
	coreFacts := model.NewCoreFacts()
	parsedResources := 0

	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		decoded, err := utils.DecodeJSON(raw)
		if err != nil {
			return fmt.Errorf("parse JSONL resource %d: %w", line, err)
		}

		result, ok, err := buildNodeFromDecoded(decoded, parseUndefinedNodes)
		if err != nil {
			return fmt.Errorf("build resource %d: %w", line, err)
		}
		if !ok {
			return nil
		}

		parsedResources++

		nodeList = append(nodeList, model.BloodHoundNode{
			ID:         result.Node.ID,
			Kinds:      result.Node.Kinds,
			Properties: result.Node.Properties,
		})

		for _, entry := range result.Core {
			coreFacts.Add(entry)
		}

		return nil
	})
	if err != nil {
		return nil, nil, 0, err
	}

	addAggregateNodes(&nodeList, coreFacts)

	external := nodes.ExternalNode()
	nodeList = append(nodeList, model.BloodHoundNode{
		ID:         external.ID,
		Kinds:      external.Kinds,
		Properties: external.Properties,
	})
	coreFacts.Add(nodes.CoreEntry{Cluster: true, Data: nodes.ExternalCoreEntry()})

	return nodeList, coreFacts, parsedResources, nil
}

func addAggregateNodes(nodeList *[]model.BloodHoundNode, coreFacts *model.CoreFacts) {
	if nodeList == nil || coreFacts == nil {
		return
	}
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllPods())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllSecrets())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllServiceAccounts())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllNodes())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllDeployments())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllDaemonSets())
	appendBuildResult(nodeList, coreFacts, nodes.BuildAllStatefulSets())
}

func appendBuildResult(nodeList *[]model.BloodHoundNode, coreFacts *model.CoreFacts, result nodes.BuildResult) {
	*nodeList = append(*nodeList, model.BloodHoundNode{
		ID:         result.Node.ID,
		Kinds:      result.Node.Kinds,
		Properties: result.Node.Properties,
	})
	for _, entry := range result.Core {
		coreFacts.Add(entry)
	}
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
