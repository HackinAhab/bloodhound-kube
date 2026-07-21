//go:build all_addons

package cilium

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodecilium "bloodhound-kube/internal/nodes/addons/cilium"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

// base/newCore/ensureNamespace/hasEdge are duplicated in each addon edge-rule
// test package (see also edges/addons/calico) — unexported test helpers can't
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

func TestCiliumNetworkPolicyEdgesRule_EmptySelectorUsesAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.CiliumNetworkPolicies = append(ns.CiliumNetworkPolicies, nodecilium.CiliumNetworkPolicy{
		GraphNodeBase:     base("BHK_CiliumNetworkPolicy", "ns1", "deny-all"),
		PodSelectorLabels: nil,
	})
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns1", "my-pod"),
	})
	allPodsResult := platform.BuildAllPodsNS("ns1")
	ns.AllPods = append(ns.AllPods, allPodsResult.Core[0].Data.(platform.AllPods))

	ctx := framework.NewContext(core)
	edges := ciliumNetworkPolicyEdgesRule{}.Apply(ctx)

	aggID := nodefw.BuildID("BHK_AllPods", "ns1", "BHK_AllPods")
	podID := nodefw.BuildID("BHK_Pod", "ns1", "my-pod")
	policyID := nodefw.BuildID("BHK_CiliumNetworkPolicy", "ns1", "deny-all")

	if !hasEdge(edges, policyID, aggID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge from policy to AllPods aggregate, got %v", edges)
	}
	if hasEdge(edges, policyID, podID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to individual pod when aggregate is present")
	}
}

func TestCiliumNetworkPolicyEdgesRule_LabelSelectorMatchesSpecificPods(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.CiliumNetworkPolicies = append(ns.CiliumNetworkPolicies, nodecilium.CiliumNetworkPolicy{
		GraphNodeBase:     base("BHK_CiliumNetworkPolicy", "ns1", "web-policy"),
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

	ctx := framework.NewContext(core)
	edges := ciliumNetworkPolicyEdgesRule{}.Apply(ctx)

	policyID := nodefw.BuildID("BHK_CiliumNetworkPolicy", "ns1", "web-policy")
	webPodID := nodefw.BuildID("BHK_Pod", "ns1", "web-pod")
	otherPodID := nodefw.BuildID("BHK_Pod", "ns1", "other-pod")

	if !hasEdge(edges, policyID, webPodID, "BHK_AppliesTo") {
		t.Fatalf("expected AppliesTo edge to matching pod")
	}
	if hasEdge(edges, policyID, otherPodID, "BHK_AppliesTo") {
		t.Fatalf("unexpected AppliesTo edge to non-matching pod")
	}
}
