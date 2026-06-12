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

func findEdge(edges []model.BloodHoundEdge, startID, endID, kind string) (model.BloodHoundEdge, bool) {
	for _, edge := range edges {
		if edge.Kind == kind && edge.Start.Value == startID && edge.End.Value == endID {
			return edge, true
		}
	}
	return model.BloodHoundEdge{}, false
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
	// Two unique (start, end) pairs:
	//   1. Role/team-a/read-role -> SA/team-a/sa-1     (one binding: rb-role)
	//   2. ClusterRole/cluster-admin-lite -> SA/team-a/sa-1
	//      (two bindings merged: rb-cluster-role + crb-1)
	if len(edges) != 2 {
		t.Fatalf("expected 2 RoleBound edges (with binding merge), got %d", len(edges))
	}

	saID := nodefw.BuildID("ServiceAccount", "team-a", "sa-1")
	roleEdge, ok := findEdge(edges, saID, nodefw.BuildID("Role", "team-a", "read-role"), "RoleBound")
	if !ok {
		t.Fatalf("missing serviceaccount->role edge")
	}
	if got := roleEdge.Properties["bindingKind"]; got != "RoleBinding" {
		t.Fatalf("role edge bindingKind = %v, want RoleBinding", got)
	}
	if got := roleEdge.Properties["bindingName"]; got != "rb-role" {
		t.Fatalf("role edge bindingName = %v, want rb-role", got)
	}
	if got := roleEdge.Properties["bindingNamespace"]; got != "team-a" {
		t.Fatalf("role edge bindingNamespace = %v, want team-a", got)
	}
	if got := roleEdge.Properties["roleKind"]; got != "Role" {
		t.Fatalf("role edge roleKind = %v, want Role", got)
	}
	if got := roleEdge.Properties["roleName"]; got != "read-role" {
		t.Fatalf("role edge roleName = %v, want read-role", got)
	}
	if got := roleEdge.Properties["bindingCount"]; got != 1 {
		t.Fatalf("role edge bindingCount = %v, want 1", got)
	}
	roleBindings, ok := roleEdge.Properties["bindings"].([]map[string]any)
	if !ok || len(roleBindings) != 1 {
		t.Fatalf("role edge bindings malformed: %#v", roleEdge.Properties["bindings"])
	}
	if roleBindings[0]["name"] != "rb-role" || roleBindings[0]["kind"] != "RoleBinding" {
		t.Fatalf("role edge bindings[0] = %#v", roleBindings[0])
	}

	crEdge, ok := findEdge(edges, saID, nodefw.BuildID("ClusterRole", "", "cluster-admin-lite"), "RoleBound")
	if !ok {
		t.Fatalf("missing serviceaccount->clusterrole edge")
	}
	if got := crEdge.Properties["bindingCount"]; got != 2 {
		t.Fatalf("clusterrole edge bindingCount = %v, want 2", got)
	}
	crBindings, ok := crEdge.Properties["bindings"].([]map[string]any)
	if !ok || len(crBindings) != 2 {
		t.Fatalf("clusterrole edge bindings malformed: %#v", crEdge.Properties["bindings"])
	}
	// Sorted by (namespace, name, kind): "" < "team-a", so crb-1 comes first.
	if crBindings[0]["name"] != "crb-1" || crBindings[0]["kind"] != "ClusterRoleBinding" || crBindings[0]["namespace"] != "" {
		t.Fatalf("clusterrole edge bindings[0] = %#v", crBindings[0])
	}
	if crBindings[1]["name"] != "rb-cluster-role" || crBindings[1]["kind"] != "RoleBinding" || crBindings[1]["namespace"] != "team-a" {
		t.Fatalf("clusterrole edge bindings[1] = %#v", crBindings[1])
	}
	// The convenience scalars mirror bindings[0] after sort.
	if got := crEdge.Properties["bindingName"]; got != "crb-1" {
		t.Fatalf("clusterrole edge bindingName = %v, want crb-1 (first after sort)", got)
	}
	if got := crEdge.Properties["bindingKind"]; got != "ClusterRoleBinding" {
		t.Fatalf("clusterrole edge bindingKind = %v, want ClusterRoleBinding", got)
	}
}

func TestRBACBaseBindingsMergesMultipleClusterRoleBindings(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "team-a")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "team-a", "sa-1")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "cluster-admin")},
	)
	// Insert in reverse-alpha order to verify sort happens at materialize time.
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-b"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-admin",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1", Namespace: "team-a"}},
		},
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-a"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-admin",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1", Namespace: "team-a"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 merged RoleBound edge, got %d", len(edges))
	}
	edge := edges[0]
	if got := edge.Properties["bindingCount"]; got != 2 {
		t.Fatalf("bindingCount = %v, want 2", got)
	}
	bindings, ok := edge.Properties["bindings"].([]map[string]any)
	if !ok || len(bindings) != 2 {
		t.Fatalf("bindings malformed: %#v", edge.Properties["bindings"])
	}
	if bindings[0]["name"] != "crb-a" {
		t.Fatalf("bindings[0].name = %v, want crb-a (sorted)", bindings[0]["name"])
	}
	if bindings[1]["name"] != "crb-b" {
		t.Fatalf("bindings[1].name = %v, want crb-b (sorted)", bindings[1]["name"])
	}
	if edge.Properties["bindingName"] != "crb-a" {
		t.Fatalf("bindingName = %v, want crb-a", edge.Properties["bindingName"])
	}
}

func TestRBACBaseBindingsRoleBindingProperties(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "reader")},
	)
	ns.Roles = append(ns.Roles, rbac.Role{GraphNodeBase: base("Role", "ns1", "reader-role")})
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-reader"),
			RoleKind:      "Role",
			RoleName:      "reader-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "reader"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 RoleBound edge, got %d", len(edges))
	}
	edge := edges[0]
	bindings, ok := edge.Properties["bindings"].([]map[string]any)
	if !ok || len(bindings) != 1 {
		t.Fatalf("bindings malformed: %#v", edge.Properties["bindings"])
	}
	want := map[string]any{
		"kind":      "RoleBinding",
		"name":      "rb-reader",
		"namespace": "ns1",
		"roleKind":  "Role",
		"roleName":  "reader-role",
	}
	for k, v := range want {
		if bindings[0][k] != v {
			t.Fatalf("bindings[0][%q] = %v, want %v", k, bindings[0][k], v)
		}
	}
}

func TestRBACBaseBindingsDeterministicOrdering(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "team-a")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "team-a", "sa-1")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "cluster-role")},
	)
	// Mix RoleBinding (namespace=team-a) and ClusterRoleBinding (namespace="")
	// — sort should place the ClusterRoleBinding first since "" < "team-a".
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "team-a", "rb-zeta"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1"}},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-zeta"),
			RoleKind:      "ClusterRole",
			RoleName:      "cluster-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "sa-1", Namespace: "team-a"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 merged RoleBound edge, got %d", len(edges))
	}
	bindings, ok := edges[0].Properties["bindings"].([]map[string]any)
	if !ok || len(bindings) != 2 {
		t.Fatalf("bindings malformed: %#v", edges[0].Properties["bindings"])
	}
	if bindings[0]["namespace"] != "" || bindings[0]["name"] != "crb-zeta" {
		t.Fatalf("expected ClusterRoleBinding (namespace='') first, got %#v", bindings[0])
	}
	if bindings[1]["namespace"] != "team-a" || bindings[1]["name"] != "rb-zeta" {
		t.Fatalf("expected RoleBinding (namespace='team-a') second, got %#v", bindings[1])
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

// ---------------------------------------------------------------------------
// PodExec
// ---------------------------------------------------------------------------

func TestRBACPodExecNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "exec-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-b")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "exec-role"), PermsDisplay: []string{"pods/exec: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-exec"),
			RoleKind:      "Role",
			RoleName:      "exec-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "exec-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodExecEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 PodExec edges, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "exec-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-a"), "PodExec") {
		t.Fatalf("missing PodExec edge to pod-a")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-b"), "PodExec") {
		t.Fatalf("missing PodExec edge to pod-b")
	}
}

func TestRBACPodExecNamespacedNamedPod(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "exec-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "target-pod")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "other-pod")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "exec-role"), PermsDisplay: []string{"pods/exec/target-pod: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-exec"),
			RoleKind:      "Role",
			RoleName:      "exec-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "exec-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodExecEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 PodExec edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "exec-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "target-pod"), "PodExec") {
		t.Fatalf("missing PodExec edge to target-pod")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "other-pod"), "PodExec") {
		t.Fatalf("unexpected PodExec edge to other-pod")
	}
}

func TestRBACPodExecClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "exec-sa")},
	)
	core.Cluster.AllPods = append(core.Cluster.AllPods,
		platform.AllPods{GraphNodeBase: base("AllPods", "", "AllPods")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "exec-cr"), PermsDisplay: []string{"pods/exec: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-exec"),
			RoleKind:      "ClusterRole",
			RoleName:      "exec-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "exec-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodExecEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 aggregate PodExec edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "exec-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("AllPods", "", "AllPods"), "PodExec") {
		t.Fatalf("missing PodExec edge to AllPods aggregate")
	}
}

func TestRBACPodExecClusterNamedCrossNamespace(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")
	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "exec-sa")},
	)
	ns1.Pods = append(ns1.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "shared-pod")})
	ns2.Pods = append(ns2.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns2", "shared-pod")})
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "exec-cr"), PermsDisplay: []string{"pods/exec/shared-pod: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-exec"),
			RoleKind:      "ClusterRole",
			RoleName:      "exec-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "exec-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodExecEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 named PodExec edges across namespaces, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "exec-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "shared-pod"), "PodExec") {
		t.Fatalf("missing PodExec edge to ns1/shared-pod")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns2", "shared-pod"), "PodExec") {
		t.Fatalf("missing PodExec edge to ns2/shared-pod")
	}
}

// ---------------------------------------------------------------------------
// PodPortForward
// ---------------------------------------------------------------------------

func TestRBACPodPortForwardNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "pf-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-b")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "pf-role"), PermsDisplay: []string{"pods/portforward: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-pf"),
			RoleKind:      "Role",
			RoleName:      "pf-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "pf-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodPortForwardEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 PodPortForward edges, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "pf-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-a"), "PodPortForward") {
		t.Fatalf("missing PodPortForward edge to pod-a")
	}
}

func TestRBACPodPortForwardClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "pf-sa")},
	)
	core.Cluster.AllPods = append(core.Cluster.AllPods,
		platform.AllPods{GraphNodeBase: base("AllPods", "", "AllPods")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "pf-cr"), PermsDisplay: []string{"pods/portforward: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-pf"),
			RoleKind:      "ClusterRole",
			RoleName:      "pf-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "pf-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodPortForwardEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 aggregate PodPortForward edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "pf-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("AllPods", "", "AllPods"), "PodPortForward") {
		t.Fatalf("missing PodPortForward edge to AllPods aggregate")
	}
}

// ---------------------------------------------------------------------------
// PodAttach
// ---------------------------------------------------------------------------

func TestRBACPodAttachNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "attach-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "attach-role"), PermsDisplay: []string{"pods/attach: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-attach"),
			RoleKind:      "Role",
			RoleName:      "attach-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "attach-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodAttachEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 PodAttach edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "attach-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-a"), "PodAttach") {
		t.Fatalf("missing PodAttach edge to pod-a")
	}
}

func TestRBACPodAttachClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "attach-sa")},
	)
	core.Cluster.AllPods = append(core.Cluster.AllPods,
		platform.AllPods{GraphNodeBase: base("AllPods", "", "AllPods")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "attach-cr"), PermsDisplay: []string{"pods/attach: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-attach"),
			RoleKind:      "ClusterRole",
			RoleName:      "attach-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "attach-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodAttachEdgesRule{}.Apply(ctx)
	if len(edges) != 1 {
		t.Fatalf("expected 1 aggregate PodAttach edge, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "attach-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("AllPods", "", "AllPods"), "PodAttach") {
		t.Fatalf("missing PodAttach edge to AllPods aggregate")
	}
}

// ---------------------------------------------------------------------------
// PodDebug (pods/ephemeralcontainers: update)
// ---------------------------------------------------------------------------

func TestRBACPodDebugNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "debug-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-b")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "debug-role"), PermsDisplay: []string{"pods/ephemeralcontainers: update"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-debug"),
			RoleKind:      "Role",
			RoleName:      "debug-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "debug-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodDebugEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 PodDebug edges, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "debug-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-a"), "PodDebug") {
		t.Fatalf("missing PodDebug edge to pod-a")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-b"), "PodDebug") {
		t.Fatalf("missing PodDebug edge to pod-b")
	}
}

func TestRBACPodDebugClusterNamedCrossNamespace(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")
	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "debug-sa")},
	)
	ns1.Pods = append(ns1.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "shared-pod")})
	ns2.Pods = append(ns2.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns2", "shared-pod")})
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "debug-cr"), PermsDisplay: []string{"pods/ephemeralcontainers/shared-pod: update"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-debug"),
			RoleKind:      "ClusterRole",
			RoleName:      "debug-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "debug-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPodDebugEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 named PodDebug edges across namespaces, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "debug-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "shared-pod"), "PodDebug") {
		t.Fatalf("missing PodDebug edge to ns1/shared-pod")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns2", "shared-pod"), "PodDebug") {
		t.Fatalf("missing PodDebug edge to ns2/shared-pod")
	}
}

// ---------------------------------------------------------------------------
// Impersonate
// ---------------------------------------------------------------------------

func TestRBACImpersonateNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "actor-sa")},
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "target-sa")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "impersonate-role"), PermsDisplay: []string{"serviceaccounts: impersonate"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-imp"),
			RoleKind:      "Role",
			RoleName:      "impersonate-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "actor-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacImpersonateEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "actor-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns1", "target-sa"), "ImpersonateSA") {
		t.Fatalf("missing SAImpersonate edge to target-sa")
	}
}

func TestRBACImpersonateNamespacedImpersonateUsers(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "actor-sa")},
	)
	core.Cluster.AllServiceAccounts = append(core.Cluster.AllServiceAccounts,
		platform.AllServiceAccounts{GraphNodeBase: base("AllServiceAccounts", "", "AllServiceAccounts")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "imp-users-role"), PermsDisplay: []string{"users: impersonate"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-imp-users"),
			RoleKind:      "Role",
			RoleName:      "imp-users-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "actor-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacImpersonateEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "actor-sa")
	aggID := nodefw.BuildID("AllServiceAccounts", "", "AllServiceAccounts")
	if !hasEdge(edges, saID, aggID, "ImpersonateUsers") {
		t.Fatalf("missing ImpersonateUsers edge to AllServiceAccounts")
	}
}

func TestRBACImpersonateNamespacedImpersonateGroups(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "actor-sa")},
	)
	core.Cluster.AllServiceAccounts = append(core.Cluster.AllServiceAccounts,
		platform.AllServiceAccounts{GraphNodeBase: base("AllServiceAccounts", "", "AllServiceAccounts")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "imp-groups-role"), PermsDisplay: []string{"groups: impersonate"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-imp-groups"),
			RoleKind:      "Role",
			RoleName:      "imp-groups-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "actor-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacImpersonateEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "actor-sa")
	aggID := nodefw.BuildID("AllServiceAccounts", "", "AllServiceAccounts")
	if !hasEdge(edges, saID, aggID, "ImpersonateGroups") {
		t.Fatalf("missing ImpersonateGroups edge to AllServiceAccounts")
	}
}

func TestRBACImpersonateClusterWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "actor-sa")},
	)
	core.Cluster.AllServiceAccounts = append(core.Cluster.AllServiceAccounts,
		platform.AllServiceAccounts{GraphNodeBase: base("AllServiceAccounts", "", "AllServiceAccounts")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "imp-cr"), PermsDisplay: []string{"serviceaccounts: impersonate"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-imp"),
			RoleKind:      "ClusterRole",
			RoleName:      "imp-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "actor-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacImpersonateEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "actor-sa")
	aggID := nodefw.BuildID("AllServiceAccounts", "", "AllServiceAccounts")
	if !hasEdge(edges, saID, aggID, "ImpersonateSA") {
		t.Fatalf("missing cluster SAImpersonate edge to AllServiceAccounts")
	}
}

func TestRBACImpersonateClusterNamedSA(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")
	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "actor-sa")},
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "shared-sa")},
	)
	ns2.ServiceAccounts = append(ns2.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns2", "shared-sa")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "imp-named-cr"), PermsDisplay: []string{"serviceaccounts/shared-sa: impersonate"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-imp-named"),
			RoleKind:      "ClusterRole",
			RoleName:      "imp-named-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "actor-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacImpersonateEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "actor-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns1", "shared-sa"), "ImpersonateSA") {
		t.Fatalf("missing SAImpersonate edge to ns1/shared-sa")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns2", "shared-sa"), "ImpersonateSA") {
		t.Fatalf("missing SAImpersonate edge to ns2/shared-sa")
	}
}

// ---------------------------------------------------------------------------
// EscalateBind
// ---------------------------------------------------------------------------

func TestRBACEscalateBindNamespacedEscalateRoles(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "escalator-sa")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "victim-role")},
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "escalate-role"),
			PermsDisplay:  []string{"roles: escalate"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-escalate"),
			RoleKind:      "Role",
			RoleName:      "escalate-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "escalator-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEscalateBindEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "escalator-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Role", "ns1", "victim-role"), "RBACEscalate") {
		t.Fatalf("missing RBACEscalate edge to victim-role")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Role", "ns1", "escalate-role"), "RBACEscalate") {
		t.Fatalf("missing RBACEscalate edge to escalate-role itself")
	}
}

func TestRBACEscalateBindNamespacedBindClusterRoles(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "binder-sa")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{
			GraphNodeBase: base("Role", "ns1", "bind-role"),
			PermsDisplay:  []string{"clusterroles: bind"},
		},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-bind"),
			RoleKind:      "Role",
			RoleName:      "bind-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "binder-sa"}},
		},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "target-cr")},
	)

	ctx := framework.NewContext(core)
	edges := rbacEscalateBindEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "binder-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("ClusterRole", "", "target-cr"), "RBACBind") {
		t.Fatalf("missing RBACBind edge to target-cr")
	}
}

func TestRBACEscalateBindClusterEscalateClusterRoles(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "escalator-sa")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "victim-cr")},
		rbac.ClusterRole{
			GraphNodeBase: base("ClusterRole", "", "escalate-cr"),
			PermsDisplay:  []string{"clusterroles: escalate"},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-escalate"),
			RoleKind:      "ClusterRole",
			RoleName:      "escalate-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "escalator-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEscalateBindEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "escalator-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("ClusterRole", "", "victim-cr"), "RBACEscalate") {
		t.Fatalf("missing cluster RBACEscalate edge to victim-cr")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("ClusterRole", "", "escalate-cr"), "RBACEscalate") {
		t.Fatalf("missing cluster RBACEscalate edge to escalate-cr itself")
	}
}

func TestRBACEscalateBindClusterBindRoles(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")
	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "binder-sa")},
	)
	ns1.Roles = append(ns1.Roles, rbac.Role{GraphNodeBase: base("Role", "ns1", "role-a")})
	ns2.Roles = append(ns2.Roles, rbac.Role{GraphNodeBase: base("Role", "ns2", "role-b")})
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{
			GraphNodeBase: base("ClusterRole", "", "bind-roles-cr"),
			PermsDisplay:  []string{"roles: bind"},
		},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-bind-roles"),
			RoleKind:      "ClusterRole",
			RoleName:      "bind-roles-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "binder-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacEscalateBindEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "binder-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Role", "ns1", "role-a"), "RBACBind") {
		t.Fatalf("missing cluster RBACBind edge to ns1/role-a")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Role", "ns2", "role-b"), "RBACBind") {
		t.Fatalf("missing cluster RBACBind edge to ns2/role-b")
	}
}

// ---------------------------------------------------------------------------
// SATokenRequest
// ---------------------------------------------------------------------------

func TestRBACSATokenRequestNamespacedWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "token-minter")},
	)
	ns.AllServiceAccounts = append(ns.AllServiceAccounts,
		platform.AllServiceAccounts{GraphNodeBase: base("AllServiceAccounts", "ns1", "AllServiceAccounts")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "token-role"), PermsDisplay: []string{"serviceaccounts/token: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-token"),
			RoleKind:      "Role",
			RoleName:      "token-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "token-minter"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacSATokenRequestEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "token-minter")
	aggID := nodefw.BuildID("AllServiceAccounts", "ns1", "AllServiceAccounts")
	if !hasEdge(edges, saID, aggID, "SATokenRequest") {
		t.Fatalf("missing SATokenRequest edge to ns1 AllServiceAccounts aggregate")
	}
}

func TestRBACSATokenRequestNamespacedNamed(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "token-minter")},
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "target-sa")},
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "other-sa")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "token-role"), PermsDisplay: []string{"serviceaccounts/token/target-sa: create"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-token"),
			RoleKind:      "Role",
			RoleName:      "token-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "token-minter"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacSATokenRequestEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "token-minter")
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns1", "target-sa"), "SATokenRequest") {
		t.Fatalf("missing SATokenRequest edge to target-sa")
	}
	if hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns1", "other-sa"), "SATokenRequest") {
		t.Fatalf("unexpected SATokenRequest edge to other-sa")
	}
}

func TestRBACSATokenRequestClusterAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "token-minter")},
	)
	core.Cluster.AllServiceAccounts = append(core.Cluster.AllServiceAccounts,
		platform.AllServiceAccounts{GraphNodeBase: base("AllServiceAccounts", "", "AllServiceAccounts")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "token-cr"), PermsDisplay: []string{"serviceaccounts/token: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-token"),
			RoleKind:      "ClusterRole",
			RoleName:      "token-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "token-minter", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacSATokenRequestEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "token-minter")
	aggID := nodefw.BuildID("AllServiceAccounts", "", "AllServiceAccounts")
	if !hasEdge(edges, saID, aggID, "SATokenRequest") {
		t.Fatalf("missing cluster SATokenRequest edge to AllServiceAccounts aggregate")
	}
}

func TestRBACSATokenRequestClusterNamedCrossNamespace(t *testing.T) {
	core := newCore()
	ns1 := ensureNamespace(core, "ns1")
	ns2 := ensureNamespace(core, "ns2")
	ns1.ServiceAccounts = append(ns1.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "token-minter")},
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "shared-sa")},
	)
	ns2.ServiceAccounts = append(ns2.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns2", "shared-sa")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "token-cr"), PermsDisplay: []string{"serviceaccounts/token/shared-sa: create"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-token"),
			RoleKind:      "ClusterRole",
			RoleName:      "token-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "token-minter", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacSATokenRequestEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "token-minter")
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns1", "shared-sa"), "SATokenRequest") {
		t.Fatalf("missing SATokenRequest edge to ns1/shared-sa")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("ServiceAccount", "ns2", "shared-sa"), "SATokenRequest") {
		t.Fatalf("missing SATokenRequest edge to ns2/shared-sa")
	}
}

// ---------------------------------------------------------------------------
// NodeProxy cluster path (NodeProxyRCE)
// ---------------------------------------------------------------------------

func TestRBACNodeProxyClusterWildcard(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "proxy-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-on-node1"), NodeName: "node1"},
	)
	core.Cluster.AllNodes = append(core.Cluster.AllNodes,
		platform.AllNodes{GraphNodeBase: base("AllNodes", "", "AllNodes")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "proxy-cr"), PermsDisplay: []string{"nodes/proxy: get"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-proxy"),
			RoleKind:      "ClusterRole",
			RoleName:      "proxy-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "proxy-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacNodeProxyEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "proxy-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-on-node1"), "NodeProxyRCE") {
		t.Fatalf("missing NodeProxyRCE edge to pod-on-node1")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("AllNodes", "", "AllNodes"), "NodeProxyRCE") {
		t.Fatalf("missing NodeProxyRCE edge to AllNodes aggregate")
	}
}

func TestRBACNodeProxyClusterNamedNode(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "proxy-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-on-node1"), NodeName: "node1"},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-on-node2"), NodeName: "node2"},
	)
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: base("Node", "", "node1")},
		platform.Node{GraphNodeBase: base("Node", "", "node2")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "proxy-cr"), PermsDisplay: []string{"nodes/proxy/node1: get"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-proxy"),
			RoleKind:      "ClusterRole",
			RoleName:      "proxy-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "proxy-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacNodeProxyEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "proxy-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-on-node1"), "NodeProxyRCE") {
		t.Fatalf("missing NodeProxyRCE edge to pod-on-node1")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-on-node2"), "NodeProxyRCE") {
		t.Fatalf("unexpected NodeProxyRCE edge to pod-on-node2")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Node", "", "node1"), "NodeProxyRCE") {
		t.Fatalf("missing NodeProxyRCE edge to node1")
	}
	if hasEdge(edges, saID, nodefw.BuildID("Node", "", "node2"), "NodeProxyRCE") {
		t.Fatalf("unexpected NodeProxyRCE edge to node2")
	}
}

// ---------------------------------------------------------------------------
// WorkloadPatch
// ---------------------------------------------------------------------------

func TestRBACWorkloadPatchNamespacedPods(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "patcher-sa")},
	)
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")},
		workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-b")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "patch-role"), PermsDisplay: []string{"pods: patch"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-patch"),
			RoleKind:      "Role",
			RoleName:      "patch-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "patcher-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPatchWorkloadEdgesRule{}.Apply(ctx)
	if len(edges) != 2 {
		t.Fatalf("expected 2 WorkloadPatch edges for pods, got %d", len(edges))
	}
	saID := nodefw.BuildID("ServiceAccount", "ns1", "patcher-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-a"), "WorkloadPatch") {
		t.Fatalf("missing WorkloadPatch edge to pod-a")
	}
	if !hasEdge(edges, saID, nodefw.BuildID("Pod", "ns1", "pod-b"), "WorkloadPatch") {
		t.Fatalf("missing WorkloadPatch edge to pod-b")
	}
}

func TestRBACWorkloadPatchNamespacedDeployments(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "patcher-sa")},
	)
	ns.Deployments = append(ns.Deployments,
		workload.Deployment{GraphNodeBase: base("Deployment", "ns1", "deploy-a")},
	)
	ns.Roles = append(ns.Roles,
		rbac.Role{GraphNodeBase: base("Role", "ns1", "patch-role"), PermsDisplay: []string{"deployments: update"}},
	)
	ns.RoleBindings = append(ns.RoleBindings,
		rbac.RoleBinding{
			GraphNodeBase: base("RoleBinding", "ns1", "rb-patch"),
			RoleKind:      "Role",
			RoleName:      "patch-role",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "patcher-sa"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPatchWorkloadEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "patcher-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("Deployment", "ns1", "deploy-a"), "WorkloadPatch") {
		t.Fatalf("missing WorkloadPatch edge to deploy-a")
	}
}

func TestRBACWorkloadPatchClusterWildcardPodsAggregate(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbac.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "patcher-sa")},
	)
	core.Cluster.AllPods = append(core.Cluster.AllPods,
		platform.AllPods{GraphNodeBase: base("AllPods", "", "AllPods")},
	)
	core.Cluster.ClusterRoles = append(core.Cluster.ClusterRoles,
		rbac.ClusterRole{GraphNodeBase: base("ClusterRole", "", "patch-cr"), PermsDisplay: []string{"pods: patch"}},
	)
	core.Cluster.ClusterRoleBindings = append(core.Cluster.ClusterRoleBindings,
		rbac.ClusterRoleBinding{
			GraphNodeBase: base("ClusterRoleBinding", "", "crb-patch"),
			RoleKind:      "ClusterRole",
			RoleName:      "patch-cr",
			Subjects:      []nodefw.Subject{{Kind: "ServiceAccount", Name: "patcher-sa", Namespace: "ns1"}},
		},
	)

	ctx := framework.NewContext(core)
	edges := rbacPatchWorkloadEdgesRule{}.Apply(ctx)
	saID := nodefw.BuildID("ServiceAccount", "ns1", "patcher-sa")
	if !hasEdge(edges, saID, nodefw.BuildID("AllPods", "", "AllPods"), "WorkloadPatch") {
		t.Fatalf("missing WorkloadPatch edge to AllPods aggregate")
	}
}
