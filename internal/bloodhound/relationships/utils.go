package relationships

import (
	"fmt"
	"sort"
)

// DeduplicateEdges removes duplicate edges based on start→end:kind key
func DeduplicateEdges(edges []BloodHoundEdge) []BloodHoundEdge {
	seen := make(map[string]bool)
	var unique []BloodHoundEdge

	for _, edge := range edges {
		key := fmt.Sprintf("%s→%s:%s", edge.Start.Value, edge.End.Value, edge.Kind)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, edge)
		}
	}

	return unique
}

// SortEdgesByKind sorts edges alphabetically by kind
func SortEdgesByKind(edges []BloodHoundEdge) {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Kind < edges[j].Kind
	})
}

// GetEdgeStats returns statistics about edges
func GetEdgeStats(edges []BloodHoundEdge) map[string]int {
	stats := make(map[string]int)

	for _, edge := range edges {
		stats[edge.Kind]++
	}

	return stats
}
