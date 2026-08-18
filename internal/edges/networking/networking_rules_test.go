package networking

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	netnodes "bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

// ---------------------------------------------------------------------------
// gatewayEdgesRule
// ---------------------------------------------------------------------------

func TestGatewayEdgesRule_HTTPRoute(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns.HTTPRoutes = append(ns.HTTPRoutes, netnodes.HTTPRoute{
		GraphNodeBase:     base("BHK_HTTPRoute", "ns1", "my-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_HTTPRoute", "ns1", "my-route")
	if !hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Gateway to HTTPRoute, got %v", edges)
	}
}

func TestGatewayEdgesRule_GRPCRoute(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns.GRPCRoutes = append(ns.GRPCRoutes, netnodes.GRPCRoute{
		GraphNodeBase:     base("BHK_GRPCRoute", "ns1", "my-grpc-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_GRPCRoute", "ns1", "my-grpc-route")
	if !hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Gateway to GRPCRoute, got %v", edges)
	}
}

func TestGatewayEdgesRule_TCPRoute(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns.TCPRoutes = append(ns.TCPRoutes, netnodes.TCPRoute{
		GraphNodeBase:     base("BHK_TCPRoute", "ns1", "my-tcp-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_TCPRoute", "ns1", "my-tcp-route")
	if !hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Gateway to TCPRoute, got %v", edges)
	}
}

func TestGatewayEdgesRule_TLSRoute(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns.TLSRoutes = append(ns.TLSRoutes, netnodes.TLSRoute{
		GraphNodeBase:     base("BHK_TLSRoute", "ns1", "my-tls-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_TLSRoute", "ns1", "my-tls-route")
	if !hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Gateway to TLSRoute, got %v", edges)
	}
}

func TestGatewayEdgesRule_CrossNamespace(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns1.Gateways = append(ns1.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns2 := ensureNamespace(core, "ns2")
	ns2.HTTPRoutes = append(ns2.HTTPRoutes, netnodes.HTTPRoute{
		GraphNodeBase:     base("BHK_HTTPRoute", "ns2", "cross-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_HTTPRoute", "ns2", "cross-route")
	if !hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Gateway (ns1) to HTTPRoute (ns2), got %v", edges)
	}
}

func TestGatewayEdgesRule_NoMatchingParentRef(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	ns.HTTPRoutes = append(ns.HTTPRoutes, netnodes.HTTPRoute{
		GraphNodeBase:     base("BHK_HTTPRoute", "ns1", "my-route"),
		ParentGatewayRefs: []nodefw.ParentGatewayRef{{Namespace: "ns1", Name: "other-gw"}},
	})

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	routeID := nodefw.BuildID("BHK_HTTPRoute", "ns1", "my-route")
	if hasEdge(edges, gwID, routeID, "BHK_RoutesTo") {
		t.Fatalf("unexpected RoutesTo edge when parentRef names a different gateway")
	}
}

func TestGatewayEdgesRule_ExternalToGateway(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Gateways = append(ns.Gateways, netnodes.Gateway{
		GraphNodeBase: base("BHK_Gateway", "ns1", "gw"),
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := gatewayEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("BHK_External", "", "external")
	gwID := nodefw.BuildID("BHK_Gateway", "ns1", "gw")
	if !hasEdge(edges, externalID, gwID, "BHK_ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to Gateway, got %v", edges)
	}
}

func base(kind, namespace, name string) nodefw.GraphNodeBase {
	return nodefw.NewGraphNodeBase(kind, namespace, name, nil, nil)
}

func newCore() *model.CoreFacts {
	return &model.CoreFacts{
		Namespaces: map[string]*model.Namespace{},
		Cluster:    &model.Cluster{},
	}
}

func ensureNamespace(core *model.CoreFacts, namespace string) *model.Namespace {
	if core.Namespaces[namespace] == nil {
		core.Namespaces[namespace] = &model.Namespace{}
	}
	return core.Namespaces[namespace]
}

func hasEdge(edges []model.BloodHoundEdge, startID, endID, kind string) bool {
	for _, edge := range edges {
		if edge.Kind == kind && edge.Start.Value == startID && edge.End.Value == endID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// serviceEdgesRule
// ---------------------------------------------------------------------------

func TestServiceEdgesRuleNodePortWithExternal(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "my-nodeport"),
		ServiceType:   "NodePort",
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("BHK_External", "", "external")
	svcID := nodefw.BuildID("BHK_Service", "ns1", "my-nodeport")
	if !hasEdge(edges, externalID, svcID, "BHK_ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to NodePort service")
	}
}

func TestServiceEdgesRuleLoadBalancer(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "my-lb"),
		ServiceType:   "LoadBalancer",
		ExternalIPs:   []string{"10.0.0.1"},
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("BHK_External", "", "external")
	svcID := nodefw.BuildID("BHK_Service", "ns1", "my-lb")
	if !hasEdge(edges, externalID, svcID, "BHK_ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to LoadBalancer service")
	}
}

func TestServiceEdgesRuleClusterIPSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "my-clusterip"),
		ServiceType:   "ClusterIP",
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for ClusterIP service, got %d", len(edges))
	}
}

func TestServiceEdgesRuleNilExternalSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "my-nodeport"),
		ServiceType:   "NodePort",
	})
	// No external node added.

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges when External is nil, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// ingressEdgesRule
// ---------------------------------------------------------------------------

func TestIngressEdgesRuleExternalToIngress(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Ingresses = append(ns.Ingresses, netnodes.Ingress{
		GraphNodeBase: base("BHK_Ingress", "ns1", "my-ingress"),
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("BHK_External", "", "external")
	ingressID := nodefw.BuildID("BHK_Ingress", "ns1", "my-ingress")
	if !hasEdge(edges, externalID, ingressID, "BHK_ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to Ingress")
	}
}

func TestIngressEdgesRuleRoutesToService(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "backend-svc"),
	})
	ns.Ingresses = append(ns.Ingresses, netnodes.Ingress{
		GraphNodeBase: base("BHK_Ingress", "ns1", "my-ingress"),
		BackendRefs: []netnodes.HTTPRouteBackendRef{
			{Namespace: "ns1", Name: "backend-svc"},
		},
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	ingressID := nodefw.BuildID("BHK_Ingress", "ns1", "my-ingress")
	svcID := nodefw.BuildID("BHK_Service", "ns1", "backend-svc")
	if !hasEdge(edges, ingressID, svcID, "BHK_RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Ingress to backend-svc")
	}
}

func TestIngressEdgesRuleNoExternalNoEdge(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Ingresses = append(ns.Ingresses, netnodes.Ingress{
		GraphNodeBase: base("BHK_Ingress", "ns1", "my-ingress"),
	})
	// No external node.

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	for _, e := range edges {
		if e.Kind == "BHK_ExternalRoutesTo" {
			t.Fatalf("unexpected ExternalRoutesTo edge when no External node is present: %+v", e)
		}
	}
}

// ---------------------------------------------------------------------------
// networkPolicyEdgesRule
// ---------------------------------------------------------------------------

func TestNetworkPolicyEdgesRule_EmptySelectorUsesAggregate(t *testing.T) {
	// Empty podSelector → AppliesTo must point at the AllPods aggregate, not individual pods.
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.NetworkPolicies = append(ns.NetworkPolicies, netnodes.NetworkPolicy{
		GraphNodeBase:     base("BHK_NetworkPolicy", "ns1", "deny-all"),
		PodSelectorLabels: nil,
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns1", "my-pod"),
	})
	allPodsResult := platform.BuildAllPodsNS("ns1")
	ns.AllPods = append(ns.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ctx := framework.NewContext(core)
	edges := networkPolicyEdgesRule{}.Apply(ctx)

	aggID := nodefw.BuildID("BHK_AllPods", "ns1", "BHK_AllPods")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "my-pod")
	netpolID := nodefw.BuildID("BHK_NetworkPolicy", "ns1", "deny-all")

	if !hasEdge(edges, netpolID, aggID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge from netpol to AllPods aggregate, got %v", edges)
	}
	if hasEdge(edges, netpolID, podID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to individual pod when aggregate is present")
	}
}

func TestNetworkPolicyEdgesRule_EmptySelectorNoAggregateSkipped(t *testing.T) {
	// Empty podSelector but no AllPods aggregate present → no edges emitted.
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.NetworkPolicies = append(ns.NetworkPolicies, netnodes.NetworkPolicy{
		GraphNodeBase:     base("BHK_NetworkPolicy", "ns1", "deny-all"),
		PodSelectorLabels: nil,
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns1", "my-pod"),
	})
	// AllPods deliberately not populated.

	ctx := framework.NewContext(core)
	edges := networkPolicyEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges when aggregate absent, got %d", len(edges))
	}
}

func TestNetworkPolicyEdgesRule_LabelSelectorMatchesSpecificPods(t *testing.T) {
	// Non-empty selector must still fan out to matching individual pods.
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.NetworkPolicies = append(ns.NetworkPolicies, netnodes.NetworkPolicy{
		GraphNodeBase:     base("BHK_NetworkPolicy", "ns1", "web-policy"),
		PodSelectorLabels: map[string]string{"app": "web"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{
			GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"app": "web"}, nil),
		},
		workload.Pod{
			GraphNodeBase: base("BHK_Pod", "ns1", "other-pod"),
		},
	)
	allPodsResult := platform.BuildAllPodsNS("ns1")
	ns.AllPods = append(ns.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ctx := framework.NewContext(core)
	edges := networkPolicyEdgesRule{}.Apply(ctx)

	netpolID := nodefw.BuildID("BHK_NetworkPolicy", "ns1", "web-policy")
	webPodID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")
	otherPodID := nodefw.BuildID("BHK_Pod", "ns1", "other-pod")
	aggID := nodefw.BuildID("BHK_AllPods", "ns1", "BHK_AllPods")

	if !hasEdge(edges, netpolID, webPodID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to matching pod")
	}
	if hasEdge(edges, netpolID, otherPodID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to non-matching pod")
	}
	if hasEdge(edges, netpolID, aggID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to aggregate when selector is non-empty")
	}
}

// ---------------------------------------------------------------------------
// serviceRoutesToRule - networkPolicyRestricted property
// ---------------------------------------------------------------------------

func TestServiceRoutesToPods_NetworkPolicyRestricted(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "web-svc"),
		SelectorMap:   map[string]string{"app": "web"},
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"app": "web"}, nil),
	})
	ns.NetworkPolicies = append(ns.NetworkPolicies, netnodes.NetworkPolicy{
		GraphNodeBase:     base("BHK_NetworkPolicy", "ns1", "deny-all"),
		PodSelectorLabels: map[string]string{"app": "web"},
	})

	ctx := framework.NewContext(core)
	edges := serviceRoutesToRule{}.Apply(ctx)

	svcID := nodefw.BuildID("BHK_Service", "ns1", "web-svc")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")

	if !hasEdge(edges, svcID, podID, "BHK_RoutesTo") {
		t.Fatalf("expected RoutesTo edge from service to pod, got %v", edges)
	}
	for _, e := range edges {
		if e.Kind == "BHK_RoutesTo" && e.Start.Value == svcID && e.End.Value == podID {
			if e.Properties["networkPolicyRestricted"] != true {
				t.Fatalf("expected networkPolicyRestricted=true, got %v", e.Properties)
			}
		}
	}
}

func TestServiceRoutesToPods_NoNetworkPolicy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "web-svc"),
		SelectorMap:   map[string]string{"app": "web"},
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"app": "web"}, nil),
	})

	ctx := framework.NewContext(core)
	edges := serviceRoutesToRule{}.Apply(ctx)

	svcID := nodefw.BuildID("BHK_Service", "ns1", "web-svc")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")

	if !hasEdge(edges, svcID, podID, "BHK_RoutesTo") {
		t.Fatalf("expected RoutesTo edge from service to pod, got %v", edges)
	}
	for _, e := range edges {
		if e.Kind == "BHK_RoutesTo" && e.Start.Value == svcID && e.End.Value == podID {
			if e.Properties != nil {
				t.Fatalf("expected nil properties when no NetworkPolicy selects pod, got %v", e.Properties)
			}
		}
	}
}

func TestServiceRoutesToPods_EmptyNetworkPolicySelectorMatchesAll(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("BHK_Service", "ns1", "web-svc"),
		SelectorMap:   map[string]string{"app": "web"},
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"app": "web"}, nil),
	})
	ns.NetworkPolicies = append(ns.NetworkPolicies, netnodes.NetworkPolicy{
		GraphNodeBase:     base("BHK_NetworkPolicy", "ns1", "blanket-deny"),
		PodSelectorLabels: nil,
	})

	ctx := framework.NewContext(core)
	edges := serviceRoutesToRule{}.Apply(ctx)

	svcID := nodefw.BuildID("BHK_Service", "ns1", "web-svc")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")

	for _, e := range edges {
		if e.Kind == "BHK_RoutesTo" && e.Start.Value == svcID && e.End.Value == podID {
			if e.Properties["networkPolicyRestricted"] != true {
				t.Fatalf("expected networkPolicyRestricted=true for empty selector (matches all), got %v", e.Properties)
			}
		}
	}
}
