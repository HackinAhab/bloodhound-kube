package security

import (
	"fmt"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type sccEdgesRule struct{}

func (r sccEdgesRule) Name() string { return "security_context_constraints" }

func (r sccEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
			if pod.AnnotationsMap == nil {
				continue
			}
			sccName, _ := pod.AnnotationsMap["openshift.io/scc"].(string)
			if sccName == "" {
				continue
			}
			if scc := ctx.Index.SecurityContextConstraintsBy[sccName]; scc != nil {
				edges = append(edges, framework.CreateEdge(scc, pod, "BHK_EnforcedSCC"))
			}
		}
	}
	return edges
}

type hostPortsEdgesRule struct{}

func (r hostPortsEdgesRule) Name() string { return "host_ports" }

func (r hostPortsEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
				for _, hostPortEntry := range container.HostPorts {
					hostPort := hostPortEntry.HostPort
					if hostPort == 0 {
						continue
					}
					edges = append(edges, framework.CreateEdgeWithProperties(node, pod, "BHK_HostPort", map[string]any{"BHK_HostPort": hostPort}))
					if ctx.Index.External != nil {
						desc := fmt.Sprintf("External access to node %s via host port %d", node.Name, hostPort)
						edges = append(edges, framework.CreateEdgeWithProperties(ctx.Index.External, node, "BHK_ExternalHostPort", map[string]any{
							"Description": desc,
						}))
					}
				}
			}
		}
	}
	return edges
}
