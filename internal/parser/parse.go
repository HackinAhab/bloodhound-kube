package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"bloodhound-kube/internal/edges"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/utils"

	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
)

const defaultClusterName = "default"

func ConvertToBloodHoundResultFromReader(reader io.Reader, clusterName string, parseUndefinedNodes bool) (*gopengraph.OpenGraph, error) {
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

	applyClusterToGraph(nodes, edges, clusterName)

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

func applyClusterToGraph(nodes []model.BloodHoundNode, edges []model.BloodHoundEdge, clusterName string) {
	normalizedCluster := normalizeClusterName(clusterName)
	idMap := make(map[string]string, len(nodes))

	for i := range nodes {
		originalID := nodes[i].ID
		nodes[i].ID = clusterScopedID(normalizedCluster, originalID)
		idMap[originalID] = nodes[i].ID
		if nodes[i].Properties == nil {
			nodes[i].Properties = map[string]any{}
		}
		nodes[i].Properties["cluster"] = normalizedCluster
	}

	for i := range edges {
		if startID, ok := idMap[edges[i].Start.Value]; ok {
			edges[i].Start.Value = startID
		} else {
			edges[i].Start.Value = clusterScopedID(normalizedCluster, edges[i].Start.Value)
		}
		if endID, ok := idMap[edges[i].End.Value]; ok {
			edges[i].End.Value = endID
		} else {
			edges[i].End.Value = clusterScopedID(normalizedCluster, edges[i].End.Value)
		}
	}
}

func normalizeClusterName(clusterName string) string {
	trimmed := strings.TrimSpace(clusterName)
	if trimmed == "" {
		return defaultClusterName
	}
	return trimmed
}

func clusterScopedID(clusterName, id string) string {
	return clusterName + ":" + id
}

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

	external := platform.ExternalNode()
	nodeList = append(nodeList, model.BloodHoundNode{
		ID:         external.ID,
		Kinds:      external.Kinds,
		Properties: external.Properties,
	})
	coreFacts.Add(nodes.CoreEntry{Cluster: true, Data: platform.ExternalCoreEntry()})

	return nodeList, coreFacts, parsedResources, nil
}

func addAggregateNodes(nodeList *[]model.BloodHoundNode, coreFacts *model.CoreFacts) {
	if nodeList == nil || coreFacts == nil {
		return
	}
	appendBuildResult(nodeList, coreFacts, platform.BuildAllPods())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllSecrets())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllConfigMaps())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllServiceAccounts())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllNodes())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllDeployments())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllDaemonSets())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllStatefulSets())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllJobs())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllCronJobs())
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
		result, ok = platform.BuildGenericNode(resource)
	}
	return result, ok, nil
}
