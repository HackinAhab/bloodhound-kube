package edges

import (
	"maps"
	"sort"
	"strings"

	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

func CreateEdge(start, end nodes.EdgeNode, kind string) model.BloodHoundEdge {
	return CreateEdgeWithProperties(start, end, kind, nil)
}

func CreateEdgeWithProperties(start, end nodes.EdgeNode, kind string, props map[string]any) model.BloodHoundEdge {
	properties := map[string]any{}
	maps.Copy(properties, props)
	if len(properties) == 0 {
		properties = nil
	}
	return model.BloodHoundEdge{
		Start: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   start.EdgeID(),
			Kind:    start.EdgeKind(),
		},
		End: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   end.EdgeID(),
			Kind:    end.EdgeKind(),
		},
		Kind:       kind,
		Properties: properties,
	}
}

func NormalizeCapability(capability string) string {
	if strings.HasPrefix(capability, "CAP_") {
		return capability
	}
	if capability == "" {
		return ""
	}
	return "CAP_" + capability
}

func HasCapability(pod nodes.Pod, capability string) bool {
	if capability == "" {
		return false
	}
	for _, capAdd := range pod.CapabilitiesAdd {
		if NormalizeCapability(capAdd) == capability {
			return true
		}
	}
	return false
}

func labelsMatchOnly(labels map[string]any, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	if labels == nil {
		return false
	}
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// hostPathMatchesAny checks if the given hostPath matches any of the paths in checkPaths, either as an exact match or as a parent directory
func hostPathMatchesAny(hostPath string, checkPath []string) bool {
	if hostPath == "" {
		return false
	}
	for _, path := range checkPath {
		if hostPath == path {
			return true
		}
		if strings.HasPrefix(hostPath, path+"/") {
			return true
		}
	}
	return false
}

// DeduplicateEdges removes duplicate edges based on start→end:kind key
func DeduplicateEdges(edges []model.BloodHoundEdge) []model.BloodHoundEdge {
	type edgeKey struct {
		start string
		end   string
		kind  string
	}

	seen := make(map[edgeKey]struct{})
	var unique []model.BloodHoundEdge

	for _, edge := range edges {
		key := edgeKey{start: edge.Start.Value, end: edge.End.Value, kind: edge.Kind}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, edge)
	}

	return unique
}

// SortEdgesByKind sorts edges alphabetically by kind
func SortEdgesByKind(edges []model.BloodHoundEdge) {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Kind < edges[j].Kind
	})
}

// GetEdgeStats returns statistics about edges
func GetEdgeStats(edges []model.BloodHoundEdge) map[string]int {
	stats := make(map[string]int)

	for _, edge := range edges {
		stats[edge.Kind]++
	}

	return stats
}
