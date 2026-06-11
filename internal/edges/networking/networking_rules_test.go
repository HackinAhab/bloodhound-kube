package networking

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	netnodes "bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
)

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
		GraphNodeBase: base("Service", "ns1", "my-nodeport"),
		ServiceType:   "NodePort",
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("External", "", "external")
	svcID := nodefw.BuildID("Service", "ns1", "my-nodeport")
	if !hasEdge(edges, externalID, svcID, "ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to NodePort service")
	}
}

func TestServiceEdgesRuleLoadBalancer(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("Service", "ns1", "my-lb"),
		ServiceType:   "LoadBalancer",
		ExternalIPs:   []string{"10.0.0.1"},
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := serviceEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("External", "", "external")
	svcID := nodefw.BuildID("Service", "ns1", "my-lb")
	if !hasEdge(edges, externalID, svcID, "ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to LoadBalancer service")
	}
}

func TestServiceEdgesRuleClusterIPSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("Service", "ns1", "my-clusterip"),
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
		GraphNodeBase: base("Service", "ns1", "my-nodeport"),
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
		GraphNodeBase: base("Ingress", "ns1", "my-ingress"),
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("External", "", "external")
	ingressID := nodefw.BuildID("Ingress", "ns1", "my-ingress")
	if !hasEdge(edges, externalID, ingressID, "ExternalRoutesTo") {
		t.Fatalf("missing ExternalRoutesTo edge from External to Ingress")
	}
}

func TestIngressEdgesRuleRoutesToService(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Services = append(ns.Services, netnodes.Service{
		GraphNodeBase: base("Service", "ns1", "backend-svc"),
	})
	ns.Ingresses = append(ns.Ingresses, netnodes.Ingress{
		GraphNodeBase: base("Ingress", "ns1", "my-ingress"),
		BackendRefs: []netnodes.HTTPRouteBackendRef{
			{Namespace: "ns1", Name: "backend-svc"},
		},
	})
	core.Cluster.External = append(core.Cluster.External, platform.ExternalCoreEntry())

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	ingressID := nodefw.BuildID("Ingress", "ns1", "my-ingress")
	svcID := nodefw.BuildID("Service", "ns1", "backend-svc")
	if !hasEdge(edges, ingressID, svcID, "RoutesTo") {
		t.Fatalf("missing RoutesTo edge from Ingress to backend-svc")
	}
}

func TestIngressEdgesRuleNoExternalNoEdge(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Ingresses = append(ns.Ingresses, netnodes.Ingress{
		GraphNodeBase: base("Ingress", "ns1", "my-ingress"),
	})
	// No external node.

	ctx := framework.NewContext(core)
	edges := ingressEdgesRule{}.Apply(ctx)
	for _, e := range edges {
		if e.Kind == "ExternalRoutesTo" {
			t.Fatalf("unexpected ExternalRoutesTo edge when no External node is present: %+v", e)
		}
	}
}
