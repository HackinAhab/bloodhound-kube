package parser

import (
	"testing"

	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
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

// TestAddAggregateNodes_EmitsPerNamespaceAggregates is the core Phase 0
// invariant: every discovered namespace gets the 9 per-namespace aggregate
// nodes (AllNodes is intentionally excluded — Nodes are cluster-scoped).
func TestAddAggregateNodes_EmitsPerNamespaceAggregates(t *testing.T) {
	core := model.NewCoreFacts()
	seedNamespace(core, "ns1")
	seedNamespace(core, "ns2")

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	wantPerNS := []string{
		"AllPods", "AllSecrets", "AllConfigMaps", "AllServiceAccounts",
		"AllDeployments", "AllDaemonSets", "AllStatefulSets",
		"AllJobs", "AllCronJobs",
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

// TestAddAggregateNodes_EmptyNamespaceStillEmits locks in the design
// decision to always emit per-namespace aggregates, even when the namespace
// contains zero resources of that kind. This guarantees rule code can
// safely take space.AllSecrets[0] without nil-checking the namespace.
func TestAddAggregateNodes_EmptyNamespaceStillEmits(t *testing.T) {
	core := model.NewCoreFacts()
	seedNamespace(core, "empty-ns") // no Pods/Secrets/etc. at all

	var nodes []model.BloodHoundNode
	addAggregateNodes(&nodes, core)

	id := nodefw.BuildID("AllSecrets", "empty-ns", "AllSecrets")
	if !containsNodeID(nodes, id) {
		t.Fatalf("expected per-namespace aggregate %q for empty namespace", id)
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
