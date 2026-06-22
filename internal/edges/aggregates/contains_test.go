package aggregates

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

func base(kind, namespace, name string) nodefw.GraphNodeBase {
	return nodefw.NewGraphNodeBase(kind, namespace, name, nil, nil)
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

func countEdges(edges []model.BloodHoundEdge, startID, kind string) int {
	count := 0
	for _, edge := range edges {
		if edge.Kind == kind && edge.Start.Value == startID {
			count++
		}
	}
	return count
}

// TestAggregateContainsNamespacedSecrets locks in the primary invariant:
// the per-namespace AllSecrets aggregate emits a Contains edge to every
// individual Secret in that namespace.
func TestAggregateContainsNamespacedSecrets(t *testing.T) {
	core := model.NewCoreFacts()
	ns1 := ensureNamespace(core, "ns1")

	ns1.Secrets = []workload.Secret{
		{GraphNodeBase: base("BHK_Secret", "ns1", "alpha")},
		{GraphNodeBase: base("BHK_Secret", "ns1", "beta")},
		{GraphNodeBase: base("BHK_Secret", "ns1", "gamma")},
	}
	ns1.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "ns1", "BHK_AllSecrets")},
	}

	ctx := framework.NewContext(core)
	edges := aggregateContainsRule{}.Apply(ctx)

	aggID := nodefw.BuildID("BHK_AllSecrets", "ns1", "BHK_AllSecrets")
	if got := countEdges(edges, aggID, "BHK_Contains"); got != 3 {
		t.Fatalf("expected 3 Contains edges from ns1 AllSecrets, got %d", got)
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		secretID := nodefw.BuildID("BHK_Secret", "ns1", name)
		if !hasEdge(edges, aggID, secretID, "BHK_Contains") {
			t.Errorf("missing Contains edge %s -> %s", aggID, secretID)
		}
	}
}

// TestAggregateContainsClusterToNamespaceAggregate verifies the
// cluster-aggregate -> namespace-aggregate edges that preserve transitive
// reachability when a query starts from the cluster aggregate.
func TestAggregateContainsClusterToNamespaceAggregate(t *testing.T) {
	core := model.NewCoreFacts()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")

	core.Cluster.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "", "BHK_AllSecrets")},
	}
	ns1.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "ns1", "BHK_AllSecrets")},
	}
	ns2.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "ns2", "BHK_AllSecrets")},
	}

	ctx := framework.NewContext(core)
	edges := aggregateContainsRule{}.Apply(ctx)

	clusterID := nodefw.BuildID("BHK_AllSecrets", "", "BHK_AllSecrets")
	if !hasEdge(edges, clusterID, nodefw.BuildID("BHK_AllSecrets", "ns1", "BHK_AllSecrets"), "BHK_Contains") {
		t.Errorf("missing Contains edge: cluster AllSecrets -> ns1 AllSecrets")
	}
	if !hasEdge(edges, clusterID, nodefw.BuildID("BHK_AllSecrets", "ns2", "BHK_AllSecrets"), "BHK_Contains") {
		t.Errorf("missing Contains edge: cluster AllSecrets -> ns2 AllSecrets")
	}
}

// TestAggregateContainsEmptyNamespace locks in the "always emit ns-aggregate
// even when empty" decision: a namespace with no secrets still gets the
// cluster -> ns Contains edge, but no individual Contains edges.
func TestAggregateContainsEmptyNamespace(t *testing.T) {
	core := model.NewCoreFacts()
	empty := ensureNamespace(core, "empty-ns")

	core.Cluster.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "", "BHK_AllSecrets")},
	}
	empty.AllSecrets = []platform.AllSecrets{
		{GraphNodeBase: base("BHK_AllSecrets", "empty-ns", "BHK_AllSecrets")},
	}
	// Note: empty.Secrets is empty.

	ctx := framework.NewContext(core)
	edges := aggregateContainsRule{}.Apply(ctx)

	clusterID := nodefw.BuildID("BHK_AllSecrets", "", "BHK_AllSecrets")
	nsAggID := nodefw.BuildID("BHK_AllSecrets", "empty-ns", "BHK_AllSecrets")

	if !hasEdge(edges, clusterID, nsAggID, "BHK_Contains") {
		t.Errorf("missing cluster -> ns Contains edge for empty namespace")
	}
	if got := countEdges(edges, nsAggID, "BHK_Contains"); got != 0 {
		t.Errorf("expected 0 Contains edges from empty-ns AllSecrets, got %d", got)
	}
}

// TestAggregateContainsAllKinds is a smoke test that every aggregate kind
// emits Contains edges when both the aggregate and individual resources are
// present. It documents the full set of kinds the rule covers.
func TestAggregateContainsAllKinds(t *testing.T) {
	core := model.NewCoreFacts()
	ns := ensureNamespace(core, "ns1")

	ns.Pods = []workload.Pod{{GraphNodeBase: base("BHK_Pod", "ns1", "p1")}}
	ns.Secrets = []workload.Secret{{GraphNodeBase: base("BHK_Secret", "ns1", "s1")}}
	ns.ConfigMaps = []workload.ConfigMap{{GraphNodeBase: base("BHK_ConfigMap", "ns1", "c1")}}
	// (rbac.ServiceAccount is exercised in nsAggregate via space.AllServiceAccounts)
	ns.Deployments = []workload.Deployment{{GraphNodeBase: base("BHK_Deployment", "ns1", "d1")}}
	ns.DaemonSets = []workload.DaemonSetCore{{GraphNodeBase: base("BHK_DaemonSet", "ns1", "ds1")}}
	ns.StatefulSets = []workload.StatefulSetCore{{GraphNodeBase: base("BHK_StatefulSet", "ns1", "ss1")}}
	ns.Jobs = []workload.Job{{GraphNodeBase: base("BHK_Job", "ns1", "j1")}}
	ns.CronJobs = []workload.CronJob{{GraphNodeBase: base("BHK_CronJob", "ns1", "cj1")}}

	ns.AllPods = []platform.AllPods{{GraphNodeBase: base("BHK_AllPods", "ns1", "BHK_AllPods")}}
	ns.AllSecrets = []platform.AllSecrets{{GraphNodeBase: base("BHK_AllSecrets", "ns1", "BHK_AllSecrets")}}
	ns.AllConfigMaps = []platform.AllConfigMaps{{GraphNodeBase: base("BHK_AllConfigMaps", "ns1", "BHK_AllConfigMaps")}}
	ns.AllDeployments = []platform.AllDeployments{{GraphNodeBase: base("BHK_AllDeployments", "ns1", "BHK_AllDeployments")}}
	ns.AllDaemonSets = []platform.AllDaemonSets{{GraphNodeBase: base("BHK_AllDaemonSets", "ns1", "BHK_AllDaemonSets")}}
	ns.AllStatefulSets = []platform.AllStatefulSets{{GraphNodeBase: base("BHK_AllStatefulSets", "ns1", "BHK_AllStatefulSets")}}
	ns.AllJobs = []platform.AllJobs{{GraphNodeBase: base("BHK_AllJobs", "ns1", "BHK_AllJobs")}}
	ns.AllCronJobs = []platform.AllCronJobs{{GraphNodeBase: base("BHK_AllCronJobs", "ns1", "BHK_AllCronJobs")}}

	ctx := framework.NewContext(core)
	edges := aggregateContainsRule{}.Apply(ctx)

	cases := []struct {
		aggKind  string
		resource string
	}{
		{"BHK_AllPods", "BHK_Pod"},
		{"BHK_AllSecrets", "BHK_Secret"},
		{"BHK_AllConfigMaps", "BHK_ConfigMap"},
		{"BHK_AllDeployments", "BHK_Deployment"},
		{"BHK_AllDaemonSets", "BHK_DaemonSet"},
		{"BHK_AllStatefulSets", "BHK_StatefulSet"},
		{"BHK_AllJobs", "BHK_Job"},
		{"BHK_AllCronJobs", "BHK_CronJob"},
	}
	for _, tc := range cases {
		aggID := nodefw.BuildID(tc.aggKind, "ns1", tc.aggKind)
		if got := countEdges(edges, aggID, "BHK_Contains"); got != 1 {
			t.Errorf("%s: expected 1 Contains edge, got %d", tc.aggKind, got)
		}
	}
}

// TestAggregateContainsNoEdgeWithoutAggregate ensures we don't emit
// individual Contains edges when the per-namespace aggregate is absent.
// This guards against accidental fan-out if a future change forgets to
// register or emit the aggregate node.
func TestAggregateContainsNoEdgeWithoutAggregate(t *testing.T) {
	core := model.NewCoreFacts()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = []workload.Secret{
		{GraphNodeBase: base("BHK_Secret", "ns1", "alpha")},
	}
	// No ns.AllSecrets entry.

	ctx := framework.NewContext(core)
	edges := aggregateContainsRule{}.Apply(ctx)

	for _, e := range edges {
		if e.Kind == "BHK_Contains" && e.End.Value == nodefw.BuildID("BHK_Secret", "ns1", "alpha") {
			t.Fatalf("unexpected Contains edge to Secret when ns aggregate is absent: %+v", e)
		}
	}
}
