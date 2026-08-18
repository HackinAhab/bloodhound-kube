//go:build !no_addons && !no_istio

package istio

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

func Register(reg *framework.Registry) {
	reg.Register(istioEdgesRule{})
}

type istioEdgesRule struct{}

func (r istioEdgesRule) Name() string { return "istio" }

func (r istioEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}

		for i := range space.IstioGateways {
			gw := &space.IstioGateways[i]
			if ctx.Index.External != nil {
				edges = append(edges, framework.CreateEdge(ctx.Index.External, gw, "BHK_ExternalRoutesTo"))
			}
			for _, ref := range gw.SecretRefs {
				if secrets := ctx.Index.SecretsByNamespace[ref.Namespace]; secrets != nil {
					if secret := secrets[ref.Name]; secret != nil {
						edges = append(edges, framework.CreateEdge(gw, secret, "BHK_ManagedBy"))
					}
				}
			}
		}

		for i := range space.VirtualServices {
			vs := &space.VirtualServices[i]
			for _, ref := range vs.GatewayRefs {
				if gateways := ctx.Index.IstioGatewaysByNamespace[ref.Namespace]; gateways != nil {
					if gw := gateways[ref.Name]; gw != nil {
						edges = append(edges, framework.CreateEdge(gw, vs, "BHK_RoutesTo"))
					}
				}
			}
			for _, host := range vs.DestinationHosts {
				svcNS, svcName := resolveServiceHost(host, ns)
				if services := ctx.Index.ServicesByNamespace[svcNS]; services != nil {
					if svc := services[svcName]; svc != nil {
						edges = append(edges, framework.CreateEdge(vs, svc, "BHK_RoutesTo"))
					}
				}
			}
		}

		for i := range space.PeerAuthentications {
			pa := &space.PeerAuthentications[i]
			props := map[string]any{"mtlsMode": pa.MTLSMode}
			if len(pa.SelectorLabels) == 0 {
				if agg := framework.FirstEdgeNode(space.AllPods); agg != nil {
					edges = append(edges, framework.CreateEdgeWithProperties(pa, agg, "BHK_AppliesTo", props))
				}
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, pa.SelectorLabels) {
					edges = append(edges, framework.CreateEdgeWithProperties(pa, pod, "BHK_AppliesTo", props))
				}
			}
		}

		for i := range space.AuthorizationPolicies {
			ap := &space.AuthorizationPolicies[i]
			props := map[string]any{"action": ap.Action}
			if len(ap.SelectorLabels) == 0 {
				if agg := framework.FirstEdgeNode(space.AllPods); agg != nil {
					edges = append(edges, framework.CreateEdgeWithProperties(ap, agg, "BHK_AppliesTo", props))
				}
			} else {
				for j := range space.Pods {
					pod := &space.Pods[j]
					if framework.LabelsMatchOnly(pod.LabelsMap, ap.SelectorLabels) {
						edges = append(edges, framework.CreateEdgeWithProperties(ap, pod, "BHK_AppliesTo", props))
					}
				}
			}

			for _, principal := range ap.Principals {
				if sas := ctx.Index.ServiceAccountsByNamespace[principal.Namespace]; sas != nil {
					if sa := sas[principal.Name]; sa != nil {
						edges = append(edges, framework.CreateEdge(ap, sa, "BHK_AppliesTo"))
					}
				}
			}
		}
	}
	return edges
}

// resolveServiceHost parses a VirtualService destination.host value into a
// namespace/name pair. Accepts a bare service name (same namespace as the
// VirtualService), "name.namespace", or the fully-qualified
// "name.namespace.svc.cluster.local" form.
func resolveServiceHost(host, defaultNamespace string) (namespace, name string) {
	labels := strings.Split(host, ".")
	switch {
	case len(labels) >= 2:
		return labels[1], labels[0]
	default:
		return defaultNamespace, host
	}
}
