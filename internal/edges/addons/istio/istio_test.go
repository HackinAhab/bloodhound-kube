//go:build !no_addons && !no_istio

package istio

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodeistio "bloodhound-kube/internal/nodes/addons/istio"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

// base/newCore/ensureNamespace/hasEdge are duplicated in each addon edge-rule
// test package (see also edges/addons/cilium, edges/addons/calico,
// edges/addons/certmanager) — unexported test helpers can't be shared across
// packages, and a shared test-support package would be overkill for four
// tiny functions.

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

func TestIstioEdgesRule_GatewayCredentialToSecret(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.IstioGateways = append(ns.IstioGateways, nodeistio.IstioGateway{
		GraphNodeBase: base("BHK_IstioGateway", "ns1", "ingress-gw"),
		SecretRefs:    []nodeistio.SecretRef{{Namespace: "ns1", Name: "ingress-tls"}},
	})
	ns.Secrets = append(ns.Secrets, workload.Secret{GraphNodeBase: base("BHK_Secret", "ns1", "ingress-tls")})

	ctx := framework.NewContext(core)
	edges := istioEdgesRule{}.Apply(ctx)

	gwID := nodefw.BuildID("BHK_IstioGateway", "ns1", "ingress-gw")
	secretID := nodefw.BuildID("BHK_Secret", "ns1", "ingress-tls")

	if !hasEdge(edges, gwID, secretID, "BHK_ManagedBy") {
		t.Fatalf("expected IstioGateway -> Secret (TLS credential) edge, got %v", edges)
	}
}

func TestIstioEdgesRule_VirtualServiceRoutesToGatewayAndService(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.IstioGateways = append(ns.IstioGateways, nodeistio.IstioGateway{
		GraphNodeBase: base("BHK_IstioGateway", "ns1", "ingress-gw"),
	})
	ns.Services = append(ns.Services, networking.Service{GraphNodeBase: base("BHK_Service", "ns1", "web")})
	ns.VirtualServices = append(ns.VirtualServices, nodeistio.VirtualService{
		GraphNodeBase:    base("BHK_VirtualService", "ns1", "web-vs"),
		GatewayRefs:      []nodeistio.SecretRef{{Namespace: "ns1", Name: "ingress-gw"}},
		DestinationHosts: []string{"web"},
	})

	ctx := framework.NewContext(core)
	edges := istioEdgesRule{}.Apply(ctx)

	vsID := nodefw.BuildID("BHK_VirtualService", "ns1", "web-vs")
	gwID := nodefw.BuildID("BHK_IstioGateway", "ns1", "ingress-gw")
	svcID := nodefw.BuildID("BHK_Service", "ns1", "web")

	if !hasEdge(edges, vsID, gwID, "BHK_AppliesTo") {
		t.Fatalf("expected VirtualService -> IstioGateway edge, got %v", edges)
	}
	if !hasEdge(edges, vsID, svcID, "BHK_RoutesTo") {
		t.Fatalf("expected VirtualService -> Service edge, got %v", edges)
	}
}

func TestIstioEdgesRule_PeerAuthenticationEmptySelectorUsesAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.PeerAuthentications = append(ns.PeerAuthentications, nodeistio.PeerAuthentication{
		GraphNodeBase: base("BHK_PeerAuthentication", "ns1", "permissive-mtls"),
		MTLSMode:      "PERMISSIVE",
	})
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("BHK_Pod", "ns1", "my-pod")})
	allPodsResult := platform.BuildAllPodsNS("ns1")
	ns.AllPods = append(ns.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ctx := framework.NewContext(core)
	edges := istioEdgesRule{}.Apply(ctx)

	paID := nodefw.BuildID("BHK_PeerAuthentication", "ns1", "permissive-mtls")
	aggID := nodefw.BuildID("BHK_AllPods", "ns1", "BHK_AllPods")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "my-pod")

	if !hasEdge(edges, paID, aggID, "BHK_AppliesTo") {
		t.Fatalf("expected PeerAuthentication -> AllPods aggregate edge, got %v", edges)
	}
	if hasEdge(edges, paID, podID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to individual pod when aggregate is present")
	}
	for _, e := range edges {
		if e.Kind == "BHK_AppliesTo" && e.Start.Value == paID {
			if e.Properties["mtlsMode"] != "PERMISSIVE" {
				t.Fatalf("expected mtlsMode=PERMISSIVE property, got %v", e.Properties)
			}
		}
	}
}

func TestIstioEdgesRule_AuthorizationPolicySelectorAndPrincipals(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.AuthorizationPolicies = append(ns.AuthorizationPolicies, nodeistio.AuthorizationPolicy{
		GraphNodeBase:  base("BHK_AuthorizationPolicy", "ns1", "allow-frontend"),
		SelectorLabels: map[string]string{"app": "web"},
		Action:         "ALLOW",
		Principals:     []nodeistio.Principal{{Namespace: "ns1", Name: "frontend"}},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"app": "web"}, nil)},
		workload.Pod{GraphNodeBase: base("BHK_Pod", "ns1", "other-pod")},
	)
	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("BHK_ServiceAccount", "ns1", "frontend")})

	ctx := framework.NewContext(core)
	edges := istioEdgesRule{}.Apply(ctx)

	apID := nodefw.BuildID("BHK_AuthorizationPolicy", "ns1", "allow-frontend")
	webPodID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")
	otherPodID := nodefw.BuildID("BHK_Pod", "ns1", "other-pod")
	saID := nodefw.BuildID("BHK_ServiceAccount", "ns1", "frontend")

	if !hasEdge(edges, apID, webPodID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to matching pod, got %v", edges)
	}
	if hasEdge(edges, apID, otherPodID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to non-matching pod")
	}
	if !hasEdge(edges, apID, saID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge from AuthorizationPolicy to referenced ServiceAccount principal, got %v", edges)
	}
}
