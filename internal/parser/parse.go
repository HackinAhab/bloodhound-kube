package parser

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"bloodhound-kube/internal/edges"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/utils"

	"github.com/TheManticoreProject/gopengraph"
	"github.com/TheManticoreProject/gopengraph/edge"
	"github.com/TheManticoreProject/gopengraph/node"
	"github.com/TheManticoreProject/gopengraph/properties"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	edges := edges.BuildEdges(coreFacts)
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
			return nil, fmt.Errorf("create node %s: %w", n.ID, err)
		}

		graph.AddNode(openNode)
	}

	for _, e := range edges {
		props := properties.NewPropertiesFromMap(e.Properties)
		openEdge, err := edge.NewEdge(e.Start.Value, e.End.Value, e.Kind, props)
		if err != nil {
			return nil, fmt.Errorf("create edge %s: %w", e.Kind, err)
		}

		graph.AddEdge(openEdge)
	}

	return graph, nil
}

func applyClusterToGraph(nodes []model.BloodHoundNode, edges []model.BloodHoundEdge, clusterName string) {
	normalizedCluster := strings.TrimSpace(clusterName)
	if normalizedCluster == "" {
		normalizedCluster = defaultClusterName
	}
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

func clusterScopedID(clusterName, id string) string {
	return clusterName + ":" + id
}

func createNodesAndCoreFactsFromReader(reader io.Reader, parseUndefinedNodes bool) ([]model.BloodHoundNode, *model.CoreFacts, int, error) {
	if reader == nil {
		return nil, nil, 0, errors.New("parse JSONL failed")
	}

	log := utils.DefaultLogger().Component("parser")

	nodeList := []model.BloodHoundNode{}
	coreFacts := model.NewCoreFacts()
	parsedResources := 0

	// Typed-wins dedup: a resource served under multiple API groups (e.g.
	// Calico's crd.projectcalico.org/v1 and projectcalico.org/v3) resolves to
	// the same canonical node ID. Track the winning node per ID and its
	// index in nodeList so a later typed result can replace an earlier generic
	// one. GenericNode CoreEntries are inert (no edge rule consumes them), so a
	// superseded generic's already-added Core fact is harmless (see plan A).
	type seenNode struct {
		index   int
		generic bool
	}
	seen := map[string]seenNode{}

	err := utils.ReadJSONL(reader, func(line int, raw []byte) error {
		decoded, err := utils.DecodeJSON(raw)
		if err != nil {
			return fmt.Errorf("parse JSONL resource %d: %w", line, err)
		}

		result, ok, generic, err := buildNodeFromDecoded(decoded, parseUndefinedNodes)
		if err != nil {
			return fmt.Errorf("build resource %d: %w", line, err)
		}
		if !ok {
			return nil
		}

		id := result.Node.ID
		if id != "" {
			if prev, dup := seen[id]; dup {
				kind := ""
				if len(result.Node.Kinds) > 0 {
					kind = result.Node.Kinds[0]
				}
				if prev.generic && !generic {
					// Typed result supersedes an earlier generic node: replace
					// the node entry in place and add the typed Core facts.
					nodeList[prev.index] = model.BloodHoundNode{
						ID:         result.Node.ID,
						Kinds:      result.Node.Kinds,
						Properties: result.Node.Properties,
					}
					seen[id] = seenNode{index: prev.index, generic: false}
					for _, entry := range result.Core {
						coreFacts.Add(entry)
					}
					parsedResources++
					log.Debug("Replaced generic node with typed builder result", "id", id, "kind", kind)
					return nil
				}
				reason := "duplicate-generic"
				if !prev.generic {
					reason = "already-typed"
				}
				log.Debug("Skipped duplicate node", "id", id, "kind", kind, "reason", reason)
				return nil
			}

			seen[id] = seenNode{index: len(nodeList), generic: generic}
			nodeList = append(nodeList, model.BloodHoundNode{
				ID:         result.Node.ID,
				Kinds:      result.Node.Kinds,
				Properties: result.Node.Properties,
			})
		}

		parsedResources++

		for _, entry := range result.Core {
			coreFacts.Add(entry)
		}

		return nil
	})
	if err != nil {
		return nil, nil, 0, err
	}

	enrichPodNodesWithControllerEnv(nodeList, coreFacts)

	addAggregateNodes(&nodeList, coreFacts)
	synthesizeUsersAndGroups(&nodeList, coreFacts)

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
	appendBuildResult(nodeList, coreFacts, platform.BuildAllClusterRoles())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllUsers())
	appendBuildResult(nodeList, coreFacts, platform.BuildAllGroups())

	// Per-namespace aggregates: emitted only when the namespace has at least one
	// resource of that kind. AllNodes and AllClusterRoles are excluded —
	// Nodes and ClusterRoles are cluster-scoped.
	//
	// Snapshot the namespace keys before iterating because appendBuildResult
	// calls coreFacts.Add, which writes into the namespace map. Adding the
	// namespaced aggregate to its own namespace doesn't introduce a new key
	// (the namespace already exists), but snapshotting keeps iteration order
	// deterministic.
	namespaces := make([]string, 0, len(coreFacts.Namespaces))
	for ns := range coreFacts.Namespaces {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	for _, ns := range namespaces {
		space := coreFacts.Namespaces[ns]
		if len(space.Pods) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllPodsNS(ns))
		}
		if len(space.Secrets) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllSecretsNS(ns))
		}
		if len(space.ConfigMaps) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllConfigMapsNS(ns))
		}
		if len(space.ServiceAccounts) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllServiceAccountsNS(ns))
		}
		if len(space.Deployments) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllDeploymentsNS(ns))
		}
		if len(space.DaemonSets) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllDaemonSetsNS(ns))
		}
		if len(space.StatefulSets) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllStatefulSetsNS(ns))
		}
		if len(space.Jobs) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllJobsNS(ns))
		}
		if len(space.CronJobs) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllCronJobsNS(ns))
		}
		if len(space.Roles) > 0 {
			appendBuildResult(nodeList, coreFacts, platform.BuildAllRolesNS(ns))
		}
	}
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

func buildNodeFromDecoded(decoded utils.DecodedResource, parseUndefinedNodes bool) (nodes.BuildResult, bool, bool, error) {
	if decoded.Object != nil {
		if result, ok := nodes.BuildTyped(decoded.GVK, decoded.Object); ok {
			return result, true, false, nil
		}
		resource, err := utils.ToMap(decoded.Object)
		if err != nil {
			return nodes.BuildResult{}, false, false, err
		}
		return buildNodeFromMap(resource, parseUndefinedNodes)
	}

	if decoded.Raw != nil {
		return buildNodeFromMap(decoded.Raw, parseUndefinedNodes)
	}

	return nodes.BuildResult{}, false, false, nil
}

// buildNodeFromMap returns (result, ok, generic, err). generic is true only
// when the result came from BuildGenericNode (i.e. no typed builder matched),
// so the caller can apply typed-wins dedup.
func buildNodeFromMap(resource map[string]any, parseUndefinedNodes bool) (nodes.BuildResult, bool, bool, error) {
	if resource == nil {
		return nodes.BuildResult{}, false, false, nil
	}
	result, ok := nodes.Build(resource)
	if !ok {
		if gvk, valid := gvkFromMap(resource); valid {
			result, ok = nodes.BuildTypedFromMap(gvk, resource)
		}
	}
	if ok {
		return result, true, false, nil
	}
	if parseUndefinedNodes {
		result, ok = platform.BuildGenericNode(resource)
		return result, ok, ok, nil
	}
	return nodes.BuildResult{}, false, false, nil
}

func gvkFromMap(resource map[string]any) (schema.GroupVersionKind, bool) {
	apiVersion, _ := resource["apiVersion"].(string)
	kind, _ := resource["kind"].(string)
	if apiVersion == "" || kind == "" {
		return schema.GroupVersionKind{}, false
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, false
	}
	return gv.WithKind(kind), true
}

func synthesizeUsersAndGroups(nodeList *[]model.BloodHoundNode, coreFacts *model.CoreFacts) {
	if nodeList == nil || coreFacts == nil {
		return
	}
	seen := map[string]struct{}{}

	for _, space := range coreFacts.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.RoleBindings {
			for _, subject := range space.RoleBindings[i].Subjects {
				synthesizeSubject(nodeList, coreFacts, seen, subject.Kind, subject.Name)
			}
		}
	}

	for i := range coreFacts.Cluster.ClusterRoleBindings {
		for _, subject := range coreFacts.Cluster.ClusterRoleBindings[i].Subjects {
			synthesizeSubject(nodeList, coreFacts, seen, subject.Kind, subject.Name)
		}
	}
}

func synthesizeSubject(nodeList *[]model.BloodHoundNode, coreFacts *model.CoreFacts, seen map[string]struct{}, kind, name string) {
	if name == "" {
		return
	}

	var result nodes.BuildResult
	var ok bool
	switch kind {
	case "User":
		result, ok = rbac.BuildUserNode(name)
	case "Group":
		result, ok = rbac.BuildGroupNode(name)
	default:
		return
	}
	if !ok {
		return
	}
	if _, exists := seen[result.Node.ID]; exists {
		return
	}
	seen[result.Node.ID] = struct{}{}
	appendBuildResult(nodeList, coreFacts, result)
}
