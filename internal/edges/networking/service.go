package networking

import (
	"fmt"
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type serviceRoutesToRule struct{}

func (r serviceRoutesToRule) Name() string { return "service_routes_to_pods" }

func (r serviceRoutesToRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		podIndex := ctx.Index.PodsByNamespace[ns]
		for i := range space.Services {
			svc := &space.Services[i]
			if len(svc.SelectorMap) == 0 {
				continue
			}
			for _, pod := range podIndex {
				if framework.LabelsMatchOnly(pod.LabelsMap, svc.SelectorMap) {
					edges = append(edges, framework.CreateEdge(svc, pod, "BHK_RoutesTo"))
				}
			}
		}
	}
	return edges
}

type serviceEdgesRule struct{}

func (r serviceEdgesRule) Name() string { return "services" }

func (r serviceEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil || ctx.Index.External == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Services {
			svc := &space.Services[i]
			if svc.ServiceType != "NodePort" && svc.ServiceType != "LoadBalancer" {
				continue
			}
			if svc.ServiceType == "LoadBalancer" {
				description := fmt.Sprintf("External access to service %s", svc.Name)
				if len(svc.ExternalIPs) > 0 {
					description = fmt.Sprintf("External access to service %s via %s", svc.Name, strings.Join(svc.ExternalIPs, ", "))
				}
				edges = append(edges, framework.CreateEdgeWithProperties(ctx.Index.External, svc, "BHK_ExternalRoutesTo", map[string]any{
					"Description": description,
				}))
				continue
			}
			edges = append(edges, framework.CreateEdge(ctx.Index.External, svc, "BHK_ExternalRoutesTo"))
		}
	}
	return edges
}
