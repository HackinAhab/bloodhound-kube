//go:build all_addons || calico

package calico

import (
	. "bloodhound-kube/internal/nodes/framework"
)

// BuildHostEndpointMapNode parses a Calico HostEndpoint CRD into a CoreEntry
// only — it deliberately returns a zero-value Node (empty ID) since
// HostEndpoint has no independent security-relevant identity beyond "these
// labels, on this node". Callers must skip adding a graph node when
// result.Node.ID == "".
func BuildHostEndpointMapNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}

	spec := GetMap(resource, "spec")
	nodeName := GetString(spec, "node")
	labelsMap := GetMap(metadata, "labels")

	core := CoreEntry{
		Cluster: true,
		Data: HostEndpoint{
			Name:      name,
			NodeName:  nodeName,
			LabelsMap: labelsMap,
		},
	}

	return BuildResult{Core: []CoreEntry{core}}, true
}
