package edges

import (
	"fmt"

	"bloodhound-kube/internal/model"
)

type hostPortsEdgesRule struct{}

func (r hostPortsEdgesRule) Name() string {
	return "host_ports"
}

func (r hostPortsEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.NodeName == "" {
				continue
			}
			node := ctx.Index.NodesByName[pod.NodeName]
			if node == nil {
				continue
			}
			for _, container := range pod.Containers {
				for _, hostPortEntry := range MapSlice(container, "hostPorts") {
					portMap, ok := hostPortEntry.(map[string]any)
					if !ok {
						continue
					}
					hostPort := MapNumber(portMap, "hostPort")
					if hostPort == 0 {
						continue
					}
					props := map[string]any{
						"HostPort": hostPort,
					}
					edges = append(edges, CreateEdgeWithProperties(node, pod, "HostPort", props))
					if ctx.Index.External != nil {
						desc := fmt.Sprintf("External access to node %s via host port %d", node.Name, hostPort)
						edges = append(edges, CreateEdgeWithProperties(ctx.Index.External, node, "ExternalHostPort", map[string]any{
							"Description": desc,
						}))
					}
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(hostPortsEdgesRule{})
}
