//go:build !no_addons && !no_calico

package calico

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodecalico "bloodhound-kube/internal/nodes/addons/calico"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

// base/newCore/ensureNamespace/hasEdge are duplicated in each addon edge-rule
// test package (see also edges/addons/cilium) — unexported test helpers can't
// be shared across packages, and a shared test-support package would be
// overkill for four tiny functions.

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

func TestGlobalNetworkPolicyEdgesRule_MatchesAllUsesClusterAggregate(t *testing.T) {
	core := newCore()
	core.Cluster.GlobalNetworkPolicies = append(core.Cluster.GlobalNetworkPolicies, nodecalico.GlobalNetworkPolicy{
		GraphNodeBase:      base("BHK_GlobalNetworkPolicy", "", "deny-all"),
		MatchesAll:         true,
		SelectorRecognized: true,
	})
	allPodsResult := platform.BuildAllPods()
	core.Cluster.AllPods = append(core.Cluster.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("BHK_Pod", "ns1", "my-pod")})

	core.Cluster.Nodes = append(core.Cluster.Nodes, platform.Node{GraphNodeBase: base("BHK_Node", "", "worker1")})
	core.Cluster.HostEndpoints = append(core.Cluster.HostEndpoints, nodecalico.HostEndpoint{
		Name:      "eth0-worker1",
		NodeName:  "worker1",
		LabelsMap: map[string]any{"role": "worker_public"},
	})

	ctx := framework.NewContext(core)
	edges := globalNetworkPolicyEdgesRule{}.Apply(ctx)

	aggID := nodefw.BuildID("BHK_AllPods", "", "BHK_AllPods")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "my-pod")
	nodeID := nodefw.BuildID("BHK_Node", "", "worker1")
	policyID := nodefw.BuildID("BHK_GlobalNetworkPolicy", "", "deny-all")

	if !hasEdge(edges, policyID, aggID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge from policy to cluster AllPods aggregate, got %v", edges)
	}
	if hasEdge(edges, policyID, podID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to individual pod when MatchesAll")
	}
	if !hasEdge(edges, policyID, nodeID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to node with a HostEndpoint when MatchesAll, got %v", edges)
	}
}

func TestGlobalNetworkPolicyEdgesRule_LabelSelectorMatchesPodsAcrossNamespaces(t *testing.T) {
	core := newCore()
	core.Cluster.GlobalNetworkPolicies = append(core.Cluster.GlobalNetworkPolicies, nodecalico.GlobalNetworkPolicy{
		GraphNodeBase:      base("BHK_GlobalNetworkPolicy", "", "web-policy"),
		SelectorLabels:     map[string]string{"role": "frontend"},
		SelectorRecognized: true,
	})

	ns1 := ensureNamespace(core, "ns1")
	ns1.Pods = append(ns1.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns1", "web-pod", map[string]any{"role": "frontend"}, nil),
	})
	ns2 := ensureNamespace(core, "ns2")
	ns2.Pods = append(ns2.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase("BHK_Pod", "ns2", "other-web-pod", map[string]any{"role": "frontend"}, nil),
	})
	ns2.Pods = append(ns2.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns2", "backend-pod"),
	})

	ctx := framework.NewContext(core)
	edges := globalNetworkPolicyEdgesRule{}.Apply(ctx)

	policyID := nodefw.BuildID("BHK_GlobalNetworkPolicy", "", "web-policy")
	ns1PodID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")
	ns2PodID := nodefw.BuildID("BHK_Pod", "ns2", "other-web-pod")
	backendPodID := nodefw.BuildID("BHK_Pod", "ns2", "backend-pod")

	if !hasEdge(edges, policyID, ns1PodID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to matching pod in ns1")
	}
	if !hasEdge(edges, policyID, ns2PodID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to matching pod in ns2 (cluster-scoped policy)")
	}
	if hasEdge(edges, policyID, backendPodID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to non-matching pod")
	}
}

func TestGlobalNetworkPolicyEdgesRule_HostEndpointResolvesToNode(t *testing.T) {
	core := newCore()
	core.Cluster.GlobalNetworkPolicies = append(core.Cluster.GlobalNetworkPolicies, nodecalico.GlobalNetworkPolicy{
		GraphNodeBase:      base("BHK_GlobalNetworkPolicy", "", "allow-bigfix-port"),
		SelectorLabels:     map[string]string{"role": "worker_public"},
		SelectorRecognized: true,
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: base("BHK_Node", "", "worker1")},
		platform.Node{GraphNodeBase: base("BHK_Node", "", "worker2")},
	)
	core.Cluster.HostEndpoints = append(core.Cluster.HostEndpoints, nodecalico.HostEndpoint{
		Name:      "eth0-worker1",
		NodeName:  "worker1",
		LabelsMap: map[string]any{"role": "worker_public"},
	})

	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("BHK_Pod", "ns1", "unrelated-pod")})

	ctx := framework.NewContext(core)
	edges := globalNetworkPolicyEdgesRule{}.Apply(ctx)

	policyID := nodefw.BuildID("BHK_GlobalNetworkPolicy", "", "allow-bigfix-port")
	worker1ID := nodefw.BuildID("BHK_Node", "", "worker1")
	worker2ID := nodefw.BuildID("BHK_Node", "", "worker2")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "unrelated-pod")

	if !hasEdge(edges, policyID, worker1ID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to the node resolved from the matching HostEndpoint, got %v", edges)
	}
	if hasEdge(edges, policyID, worker2ID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to a node with no matching HostEndpoint")
	}
	if hasEdge(edges, policyID, podID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to a pod that doesn't match the selector")
	}
}

func TestGlobalNetworkPolicyEdgesRule_HostEndpointUnknownNodeProducesNoEdge(t *testing.T) {
	core := newCore()
	core.Cluster.GlobalNetworkPolicies = append(core.Cluster.GlobalNetworkPolicies, nodecalico.GlobalNetworkPolicy{
		GraphNodeBase:      base("BHK_GlobalNetworkPolicy", "", "allow-bigfix-port"),
		SelectorLabels:     map[string]string{"role": "worker_public"},
		SelectorRecognized: true,
	})
	core.Cluster.HostEndpoints = append(core.Cluster.HostEndpoints, nodecalico.HostEndpoint{
		Name:      "eth0-ghost",
		NodeName:  "not-a-collected-node",
		LabelsMap: map[string]any{"role": "worker_public"},
	})

	ctx := framework.NewContext(core)
	edges := globalNetworkPolicyEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges when the HostEndpoint's node wasn't collected, got %v", edges)
	}
}

func TestGlobalNetworkPolicyEdgesRule_UnrecognizedSelectorProducesNoEdges(t *testing.T) {
	core := newCore()
	core.Cluster.GlobalNetworkPolicies = append(core.Cluster.GlobalNetworkPolicies, nodecalico.GlobalNetworkPolicy{
		GraphNodeBase:      base("BHK_GlobalNetworkPolicy", "", "has-policy"),
		SelectorRecognized: false,
	})
	allPodsResult := platform.BuildAllPods()
	core.Cluster.AllPods = append(core.Cluster.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ctx := framework.NewContext(core)
	edges := globalNetworkPolicyEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for unrecognized selector, got %v", edges)
	}
}
