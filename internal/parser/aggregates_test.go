package parser

import (
	"testing"

	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

// seedNamespace creates a namespace entry on coreFacts so addAggregateNodes
// has something to iterate. The plan is explicit: per-namespace aggregates
// are emitted for every namespace already present in coreFacts.Namespaces.
func seedNamespace(core *model.CoreFacts, ns string) {
	if core.Namespaces[ns] == nil {
		core.Namespaces[ns] = &model.Namespace{}
	}
}

// TestAddAggregateNodes_EmitsClusterAggregates locks in the pre-existing
// cluster aggregate emission so future changes don't regress it.
func TestAddAggregateNodes_EmitsClusterAggregates(t *testing.T) {
	core := model.NewCoreFacts()
	var nodes []model.BloodHoundNode

	addAggregateNodes(&nodes, core)

	// One cluster aggregate per kind: 9 typed + AllNodes.
	wantClusterIDs := []string{
		"AllPods:AllPods",
		"AllSecrets:AllSecrets",
		"AllConfigMaps:AllConfigMaps",
		"AllServiceAccounts:AllServiceAccounts",
		"AllNodes:AllNodes",
		"AllDeployments:AllDeployments",
		"AllDaemonSets:AllDaemonSets",
		"AllStatefulSets:AllStatefulSets",
		"AllJobs:AllJobs",
		"AllCronJobs:AllCronJobs",
	}
	for _, id := range wantClusterIDs {
		if !containsNodeID(nodes, id) {
			t.Errorf("missing cluster aggregate node %q", id)
		}
	}
}

// TestAddAggregateNodes_EmitsPerNamespaceAggregates verifies that a namespace
// with resources of every kind gets all per-namespace aggregate nodes.
// AllNodes is intentionally excluded — Nodes are cluster-scoped.
func TestAddAggregateNodes_EmitsPerNamespaceAggregates(t *testing.T) {
	core := model.NewCoreFacts()
	for _, ns := range []string{"ns1", "ns2"} {
		seedNamespace(core, ns)
		core.Namespaces[ns].Pods = append(core.Namespaces[ns].Pods,
			workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", ns, "p", nil, nil)})
		core.Namespaces[ns].Secrets = append(core.Namespaces[ns].Secrets,
			workload.Secret{GraphNodeBase: nodefw.NewGraphNodeBase("Secret", ns, "s", nil, nil)})
		core.Namespaces[ns].ConfigMaps = append(core.Namespaces[ns].ConfigMaps,
			workload.ConfigMap{GraphNodeBase: nodefw.NewGraphNodeBase("ConfigMap", ns, "cm", nil, nil)})
		core.Namespaces[ns].ServiceAccounts = append(core.Namespaces[ns].ServiceAccounts,
			rbac.ServiceAccount{GraphNodeBase: nodefw.NewGraphNodeBase("ServiceAccount", ns, "sa", nil, nil)})
		core.Namespaces[ns].Deployments = append(core.Namespaces[ns].Deployments,
			workload.Deployment{GraphNodeBase: nodefw.NewGraphNodeBase("Deployment", ns, "d", nil, nil)})
		core.Namespaces[ns].DaemonSets = append(core.Namespaces[ns].DaemonSets,
			workload.DaemonSetCore{GraphNodeBase: nodefw.NewGraphNodeBase("DaemonSet", ns, "ds", nil, nil)})
		core.Namespaces[ns].StatefulSets = append(core.Namespaces[ns].StatefulSets,
			workload.StatefulSetCore{GraphNodeBase: nodefw.NewGraphNodeBase("StatefulSet", ns, "ss", nil, nil)})
		core.Namespaces[ns].Jobs = append(core.Namespaces[ns].Jobs,
			workload.Job{GraphNodeBase: nodefw.NewGraphNodeBase("Job", ns, "j", nil, nil)})
		core.Namespaces[ns].CronJobs = append(core.Namespaces[ns].CronJobs,
			workload.CronJob{GraphNodeBase: nodefw.NewGraphNodeBase("CronJob", ns, "cj", nil, nil)})
		core.Namespaces[ns].Roles = append(core.Namespaces[ns].Roles,
			rbac.Role{GraphNodeBase: nodefw.NewGraphNodeBase("Role", ns, "r", nil, nil)})
	}

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	wantPerNS := []string{
		"AllPods", "AllSecrets", "AllConfigMaps", "AllServiceAccounts",
		"AllDeployments", "AllDaemonSets", "AllStatefulSets",
		"AllJobs", "AllCronJobs", "AllRoles",
	}
	for _, ns := range []string{"ns1", "ns2"} {
		for _, kind := range wantPerNS {
			id := nodefw.BuildID(kind, ns, kind)
			if !containsNodeID(nodes, id) {
				t.Errorf("missing per-namespace aggregate node %q", id)
			}
		}
	}

	// AllNodes must NOT have a per-namespace flavor.
	for _, ns := range []string{"ns1", "ns2"} {
		bad := nodefw.BuildID("AllNodes", ns, "AllNodes")
		if containsNodeID(nodes, bad) {
			t.Errorf("unexpected per-namespace AllNodes node %q (Nodes are cluster-scoped)", bad)
		}
	}
}

// TestAddAggregateNodes_EmptyNamespaceEmitsNoAggregates verifies that a
// namespace with no resources of a given kind produces no aggregate node
// for that kind.
func TestAddAggregateNodes_EmptyNamespaceEmitsNoAggregates(t *testing.T) {
	core := model.NewCoreFacts()
	seedNamespace(core, "empty-ns") // no Pods/Secrets/etc. added

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	wantAbsent := []string{
		"AllPods", "AllSecrets", "AllConfigMaps", "AllServiceAccounts",
		"AllDeployments", "AllDaemonSets", "AllStatefulSets",
		"AllJobs", "AllCronJobs", "AllRoles",
	}
	for _, kind := range wantAbsent {
		id := nodefw.BuildID(kind, "empty-ns", kind)
		if containsNodeID(nodes, id) {
			t.Errorf("unexpected empty aggregate node %q in output", id)
		}
	}
}

// TestAddAggregateNodes_PopulatedKindEmitsAggregate verifies that only kinds
// with at least one resource in a namespace get an aggregate node.
func TestAddAggregateNodes_PopulatedKindEmitsAggregate(t *testing.T) {
	core := model.NewCoreFacts()
	seedNamespace(core, "ns1")
	core.Namespaces["ns1"].Secrets = append(
		core.Namespaces["ns1"].Secrets,
		workload.Secret{GraphNodeBase: nodefw.NewGraphNodeBase("Secret", "ns1", "s1", nil, nil)},
	)

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	if !containsNodeID(nodes, nodefw.BuildID("AllSecrets", "ns1", "AllSecrets")) {
		t.Error("expected AllSecrets aggregate for ns1 that has a secret")
	}
	if containsNodeID(nodes, nodefw.BuildID("AllPods", "ns1", "AllPods")) {
		t.Error("unexpected AllPods aggregate for ns1 with no pods")
	}
}

// TestAddAggregateNodes_RoutesIntoNamespaceCoreFacts asserts that emitting a
// per-namespace aggregate populates the namespace's typed slice (e.g.
// space.AllSecrets), so RBAC rules can read space.AllSecrets[0] later.
func TestAddAggregateNodes_RoutesIntoNamespaceCoreFacts(t *testing.T) {
	core := model.NewCoreFacts()
	seedNamespace(core, "ns1")
	// Pre-existing namespaced data should not be disturbed.
	core.Namespaces["ns1"].Secrets = append(
		core.Namespaces["ns1"].Secrets,
		workload.Secret{GraphNodeBase: nodefw.NewGraphNodeBase("Secret", "ns1", "pre-existing", nil, nil)},
	)

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	space := core.Namespaces["ns1"]
	if len(space.AllSecrets) != 1 {
		t.Fatalf("expected 1 AllSecrets entry in ns1, got %d", len(space.AllSecrets))
	}
	if space.AllSecrets[0].EdgeID() != "AllSecrets:ns1:AllSecrets" {
		t.Fatalf("AllSecrets[0] EdgeID = %q, want %q",
			space.AllSecrets[0].EdgeID(), "AllSecrets:ns1:AllSecrets")
	}
	if len(space.Secrets) != 1 || space.Secrets[0].Name != "pre-existing" {
		t.Fatalf("pre-existing namespace data was disturbed: Secrets=%v", space.Secrets)
	}

	// Cluster aggregate must also be routed into the cluster slice.
	if len(core.Cluster.AllSecrets) != 1 {
		t.Fatalf("expected 1 cluster AllSecrets entry, got %d", len(core.Cluster.AllSecrets))
	}
}

func containsNodeID(nodes []model.BloodHoundNode, id string) bool {
	for _, n := range nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}
