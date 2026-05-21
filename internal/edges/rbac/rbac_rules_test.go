package rbac

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
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

func TestAccessForResourceAndParsePerms(t *testing.T) {
	perms := []string{
		"secrets: get, list",
		"secrets/my-secret: watch",
		"bad-entry",
	}

	all, names := accessForResource(perms, []string{"secrets"}, []string{"get", "watch"})
	if !all {
		t.Fatalf("expected full secrets access")
	}
	if len(names) != 0 {
		t.Fatalf("expected no named resources when wildcard resource exists")
	}

	all, names = accessForResource([]string{"secrets/my-secret: watch"}, []string{"secrets"}, []string{"get", "watch"})
	if all {
		t.Fatalf("expected named-only access")
	}
	if !hasName(names, "my-secret") {
		t.Fatalf("expected named access for my-secret, got %#v", names)
	}

	parsed := parseRBACPerms(perms)
	if len(parsed) != 2 {
		t.Fatalf("expected 2 parsed entries, got %d", len(parsed))
	}
}

func TestRBACBaseBindingsRule(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "team-a")

	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "team-a", "sa-1")})
	ns.Roles = append(ns.Roles, rbac.Role{GraphNodeBase: base("Role", "team-a", "read-role")})
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "team-a", "rb-role"),
			RoleKind:      "Role",
			RoleName:      "read-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1"}},
		},
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "team-a", "rb-cluster-role"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-admin-lite",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1"}},
		},
	)

	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "cluster-admin-lite")},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-1"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-admin-lite",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1", Namespace: "team-a"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEdgesRule{}.Apply(ctx)
	if len(edges) != 3 {
		t.Fatalf("expected 3 RoleBound edges, got %d", len(edges))
	}

	saID := nodefw.BuildID("ServiceAccount", "team-a", "sa-1")
	if !hasEdge(edges, nodefw.BuildID("Role", "team-a", "read-role"), saID, "RoleBound") {
		t.Fatalf("missing role->serviceaccount edge")
	}
	if !hasEdge(edges, nodefw.BuildID("ClusterRole", "", "cluster-admin-lite"), saID, "RoleBound") {
		t.Fatalf("missing clusterrole->serviceaccount edge")
	}
}

func TestRBACReadSecretsRule(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "reader")})
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "target-secret")},
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "other-secret")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "secret-role"),
			PermsDisplay:  []string{"secrets/target-secret: get"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-secret"),
			RoleKind:      "Role",
			RoleName:      "secret-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadSecretsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 SAReadSecret edge, got %d", len(edges))
	}

	if !hasEdge(edges,
		nodefw.BuildID("ServiceAccount", "ns1", "reader"),
		nodefw.BuildID("Secret", "ns1", "target-secret"),
		"SAReadSecret",
	) {
		t.Fatalf("missing reader->target-secret edge")
	}
}

func TestRBACReadSecretsClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "cluster-reader")})
	core.Cluster.AllSecrets = append(core.Cluster.AllSecrets,
		platform.AllSecrets{GraphNodeBase: base("AllSecrets", "", "AllSecrets")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "secret-cluster-role"), PermsDisplay: []string{"secrets: get"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-secret"),
			RoleKind:      "ClusterRole",
			RoleName:      "secret-cluster-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "cluster-reader", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadSecretsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 aggregate SAReadSecret edge, got %d", len(edges))
	}
	if !hasEdge(edges,
		nodefw.BuildID("ServiceAccount", "ns1", "cluster-reader"),
		nodefw.BuildID("AllSecrets", "", "AllSecrets"),
		"SAReadSecret",
	) {
		t.Fatalf("missing cluster-reader->AllSecrets edge")
	}
}
func TestRBACReadSecretsNamespacedAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "ns-reader")},
	)
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "alpha")},
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "beta")},
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "gamma")},
	)
	ns.AllSecrets = append(ns.AllSecrets,
		platform.AllSecrets{GraphNodeBase: base("AllSecrets", "ns1", "AllSecrets")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "secret-wildcard"),
			PermsDisplay:  []string{"secrets: get, list, watch"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-wildcard"),
			RoleKind:      "Role",
			RoleName:      "secret-wildcard",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "ns-reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadSecretsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 namespaced aggregate SAReadSecret edge, got %d", len(edges))
	}

	saID := nodefw.BuildID("ServiceAccount", "ns1", "ns-reader")
	aggID := nodefw.BuildID("AllSecrets", "ns1", "AllSecrets")
	if !hasEdge(edges, saID, aggID, "SAReadSecret") {
		t.Fatalf("missing ns-reader -> AllSecrets:ns1 SAReadSecret edge")
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		secretID := nodefw.BuildID("Secret", "ns1", name)
		if hasEdge(edges, saID, secretID, "SAReadSecret") {
			t.Fatalf("unexpected per-secret SAReadSecret edge to %s when aggregate should be used", secretID)
		}
	}
}

func TestRBACReadConfigMapsNamespacedAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "cm-reader")},
	)
	ns.ConfigMaps = append(ns.ConfigMaps,
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "alpha")},
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "beta")},
	)
	ns.AllConfigMaps = append(ns.AllConfigMaps,
		platform.AllConfigMaps{GraphNodeBase: base("AllConfigMaps", "ns1", "AllConfigMaps")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "cm-wildcard"),
			PermsDisplay:  []string{"configmaps: get, list, watch"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-cm-wildcard"),
			RoleKind:      "Role",
			RoleName:      "cm-wildcard",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "cm-reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadConfigMapsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 namespaced aggregate ReadConfigMap edge, got %d", len(edges))
	}

	saID := nodefw.BuildID("ServiceAccount", "ns1", "cm-reader")
	aggID := nodefw.BuildID("AllConfigMaps", "ns1", "AllConfigMaps")
	if !hasEdge(edges, saID, aggID, "ReadConfigMap") {
		t.Fatalf("missing cm-reader -> AllConfigMaps:ns1 ReadConfigMap edge")
	}
	for _, name := range []string{"alpha", "beta"} {
		cmID := nodefw.BuildID("ConfigMap", "ns1", name)
		if hasEdge(edges, saID, cmID, "ReadConfigMap") {
			t.Fatalf("unexpected per-configmap ReadConfigMap edge to %s when aggregate should be used", cmID)
		}
	}
}

// TestRBACReadConfigMapsNamespacedNamed is the named-perms regression for
// the ConfigMap path: granting access to `configmaps/foo` must emit an edge
// to the named ConfigMap, not to the aggregate.
func TestRBACReadConfigMapsNamespacedNamed(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "scoped-reader")},
	)
	ns.ConfigMaps = append(ns.ConfigMaps,
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "target")},
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "other")},
	)
	ns.AllConfigMaps = append(ns.AllConfigMaps,
		platform.AllConfigMaps{GraphNodeBase: base("AllConfigMaps", "ns1", "AllConfigMaps")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "scoped-cm"),
			PermsDisplay:  []string{"configmaps/target: get"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-scoped-cm"),
			RoleKind:      "Role",
			RoleName:      "scoped-cm",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "scoped-reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadConfigMapsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 named ReadConfigMap edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "scoped-reader")
	if !hasEdge(edges, saID, nodefw.BuildID("ConfigMap", "ns1", "target"), "ReadConfigMap") {
		t.Fatalf("missing scoped-reader -> target ReadConfigMap edge")
	}
	if hasEdge(edges, saID, nodefw.BuildID("AllConfigMaps", "ns1", "AllConfigMaps"), "ReadConfigMap") {
		t.Fatalf("unexpected ReadConfigMap edge to aggregate when only named perm is granted")
	}
	if hasEdge(edges, saID, nodefw.BuildID("ConfigMap", "ns1", "other"), "ReadConfigMap") {
		t.Fatalf("unexpected ReadConfigMap edge to non-targeted ConfigMap")
	}
}

func TestRBACNodeProxyRule(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "proxy-sa")})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-on-node1"), NodeName: "node1"},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-on-node2"), NodeName: "node2"},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "node-proxy-role"), PermsDisplay: []string{"nodes/proxy/node1: get"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-node-proxy"),
			RoleKind:      "Role",
			RoleName:      "node-proxy-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "proxy-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacNodeProxyToPodNamespaced(ctx, "ns1")
	if len(edges) != 1 {
		t.Fatalf("expected 1 NodeProxy edge, got %d", len(edges))
	}
	if !hasEdge(edges,
		nodefw.BuildID("ServiceAccount", "ns1", "proxy-sa"),
		nodefw.BuildID("Pod", "ns1", "pod-on-node1"),
		"NodeProxy",
	) {
		t.Fatalf("missing proxy-sa->pod-on-node1 NodeProxy edge")
	}
}

func TestRBACWorkloadPatchClusterNamedPods(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")

	ns1.ServiceAccounts = append(ns1.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "patcher")})
	ns1.Pods = append(ns1.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "shared-pod")})
	ns2.Pods = append(ns2.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns2", "shared-pod")})

	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{
			GraphNodeBase: base("ClusterRole", "", "patch-pods"),
			PermsDisplay:  []string{"pods/shared-pod: patch"},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-patch"),
			RoleKind:      "ClusterRole",
			RoleName:      "patch-pods",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "patcher", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := workloadPatchCluster(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 WorkloadPatch edges across namespaces, got %d", len(edges))
	}

	start := nodefw.BuildID("ServiceAccount", "ns1", "patcher")
	if !hasEdge(edges, start, nodefw.BuildID("Pod", "ns1", "shared-pod"), "WorkloadPatch") {
		t.Fatalf("missing edge to ns1/shared-pod")
	}
	if !hasEdge(edges, start, nodefw.BuildID("Pod", "ns2", "shared-pod"), "WorkloadPatch") {
		t.Fatalf("missing edge to ns2/shared-pod")
	}
}

func TestRBACCreateAndWorkloadCreateRules(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts, rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "creator")})
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "role-a")},
		rbac.Role{GraphNodeBase: base("Role", "ns1", "role-b")},
		rbac.Role{GraphNodeBase: base("Role", "ns1", "rbac-create-role"), PermsDisplay: []string{"rolebindings: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-rbac-create"),
			RoleKind:      "Role",
			RoleName:      "rbac-create-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "creator"}},
		},
	)

	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "cluster-target")},
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "workload-create-cluster-role"), PermsDisplay: []string{"pods: create"}},
	)
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: base("Node", "", "node-1")},
		platform.Node{GraphNodeBase: base("Node", "", "node-2")},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-workload-create"),
			RoleKind:      "ClusterRole",
			RoleName:      "workload-create-cluster-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "creator", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	rbacCreateEdges := rbacCreateEdgesRule{}.Apply(ctx)
	if len(rbacCreateEdges) != 5 {
		t.Fatalf("expected 5 RBACCreate edges, got %d", len(rbacCreateEdges))
	}

	start := nodefw.BuildID("ServiceAccount", "ns1", "creator")
	if !hasEdge(rbacCreateEdges, start, nodefw.BuildID("Role", "ns1", "role-a"), "RBACCreate") {
		t.Fatalf("missing RBACCreate edge to role-a")
	}
	if !hasEdge(rbacCreateEdges, start, nodefw.BuildID("Role", "ns1", "role-b"), "RBACCreate") {
		t.Fatalf("missing RBACCreate edge to role-b")
	}
	if !hasEdge(rbacCreateEdges, start, nodefw.BuildID("Role", "ns1", "rbac-create-role"), "RBACCreate") {
		t.Fatalf("missing RBACCreate edge to rbac-create-role")
	}
	if !hasEdge(rbacCreateEdges, start, nodefw.BuildID("ClusterRole", "", "cluster-target"), "RBACCreate") {
		t.Fatalf("missing RBACCreate edge to cluster-target")
	}
	if !hasEdge(rbacCreateEdges, start, nodefw.BuildID("ClusterRole", "", "workload-create-cluster-role"), "RBACCreate") {
		t.Fatalf("missing RBACCreate edge to workload-create-cluster-role")
	}

	workloadCreateEdges := rbacCreateWorkloadEdgesRule{}.Apply(ctx)
	if len(workloadCreateEdges) != 2 {
		t.Fatalf("expected 2 WorkloadCreate edges, got %d", len(workloadCreateEdges))
	}
	if !hasEdge(workloadCreateEdges, start, nodefw.BuildID("Node", "", "node-1"), "WorkloadCreate") {
		t.Fatalf("missing WorkloadCreate edge to node-1")
	}
	if !hasEdge(workloadCreateEdges, start, nodefw.BuildID("Node", "", "node-2"), "WorkloadCreate") {
		t.Fatalf("missing WorkloadCreate edge to node-2")
	}
}

// --- ReadLogs ----------------------------------------------------------------

func TestRBACReadLogsNamespacedWildcard(t *testing.T) {
	core := newCore()
	nsA := ensureNamespace(core, "ns-a")
	nsB := ensureNamespace(core, "ns-b")

	nsA.ServiceAccounts = append(nsA.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns-a", "log-reader")},
	)
	nsA.Pods = append(nsA.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns-a", "pod-1")},
		workload.Pod{GraphNodeBase: base("Pod", "ns-a", "pod-2")},
	)
	nsB.Pods = append(nsB.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns-b", "pod-other")},
	)
	nsA.Roles = append(nsA.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns-a", "logs-role"),
			PermsDisplay:  []string{"pods/log: get, list, watch"},
		},
	)
	nsA.RoleBindings = append(nsA.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns-a", "rb-logs"),
			RoleKind:      "Role",
			RoleName:      "logs-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "log-reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 ReadLogs edges (one per pod in ns-a), got %d", len(edges))
	}

	saID := nodefw.BuildID("ServiceAccount", "ns-a", "log-reader")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns-a", "pod-1"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to ns-a/pod-1")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns-a", "pod-2"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to ns-a/pod-2")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns-b", "pod-other"), "ReadLogs") {
		t.Fatalf("unexpected cross-namespace ReadLogs edge to ns-b/pod-other")
	}
}

func TestRBACReadLogsNamespacedNamedPod(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "scoped-reader")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "specific-pod")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "other-pod")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "scoped-logs-role"),
			PermsDisplay:  []string{"pods/log/specific-pod: get"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-scoped-logs"),
			RoleKind:      "Role",
			RoleName:      "scoped-logs-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "scoped-reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 named-pod ReadLogs edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "scoped-reader")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "specific-pod"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to ns1/specific-pod")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "other-pod"), "ReadLogs") {
		t.Fatalf("unexpected ReadLogs edge to ns1/other-pod")
	}
}

func TestRBACReadLogsClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "cluster-log-reader")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-1")},
	)
	core.Cluster.AllPods = append(core.Cluster.AllPods,
		platform.AllPods{GraphNodeBase: base("AllPods", "", "AllPods")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{
			GraphNodeBase: base("ClusterRole", "", "cluster-logs"),
			PermsDisplay:  []string{"pods/log: get"},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-logs"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-logs",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "cluster-log-reader", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 aggregate ReadLogs edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "cluster-log-reader")
	if !hasEdge(edges, saID, nodefw.BuildID("AllPods", "", "AllPods"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to AllPods aggregate")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-1"), "ReadLogs") {
		t.Fatalf("unexpected per-pod ReadLogs edge when aggregate should be used")
	}
}

func TestRBACReadLogsClusterNamedPodsAcrossNamespaces(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")

	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "named-reader")},
	)
	ns1.Pods = append(ns1.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "log-pod")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "other-pod")},
	)
	ns2.Pods = append(ns2.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns2", "log-pod")},
	)

	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{
			GraphNodeBase: base("ClusterRole", "", "named-logs"),
			PermsDisplay:  []string{"pods/log/log-pod: get"},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-named-logs"),
			RoleKind:      "ClusterRole",
			RoleName:      "named-logs",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "named-reader", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 named-pod cluster ReadLogs edges, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "named-reader")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "log-pod"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to ns1/log-pod")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns2", "log-pod"), "ReadLogs") {
		t.Fatalf("missing ReadLogs edge to ns2/log-pod")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "other-pod"), "ReadLogs") {
		t.Fatalf("unexpected ReadLogs edge to ns1/other-pod")
	}
}

func TestRBACReadLogsRejectsWrongVerb(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "writer")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-1")})
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "wrong-verb"),
			PermsDisplay:  []string{"pods/log: create"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-wrong-verb"),
			RoleKind:      "Role",
			RoleName:      "wrong-verb",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "writer"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected 0 ReadLogs edges for wrong verb, got %d", len(edges))
	}
}

func TestRBACReadLogsRejectsPodsResourceWithoutLogSubresource(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "pod-getter")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-1")})
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "pod-get-role"),
			// `get pods` does NOT grant access to `pods/log` in Kubernetes.
			PermsDisplay: []string{"pods: get, list, watch"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-pod-get"),
			RoleKind:      "Role",
			RoleName:      "pod-get-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "pod-getter"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacReadLogsEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected 0 ReadLogs edges when only `pods` (without /log subresource) is granted, got %d", len(edges))
	}
}
