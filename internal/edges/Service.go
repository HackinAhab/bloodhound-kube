package edges

import (
	"fmt"
	"strings"

	"bloodhound-kube/internal/model"
)

type serviceEdgesRule struct{}

func (r serviceEdgesRule) Name() string {
	return "services"
}

func init() {
	RegisterEdgeRule(serviceEdgesRule{})
}

func (r serviceEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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
				edges = append(edges, CreateEdgeWithProperties(ctx.Index.External, svc, "ExternalRoutesTo", map[string]any{
					"Description": description,
				}))
				continue
			}
			edges = append(edges, CreateEdge(ctx.Index.External, svc, "ExternalRoutesTo"))
		}
	}

	return edges
}
