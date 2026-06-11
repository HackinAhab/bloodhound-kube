package rbac

import (
	"sort"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
)

type rbacEdgesRule struct{}

func (r rbacEdgesRule) Name() string { return "rbac_base" }

// roleBoundEdgeKey identifies a unique RoleBound edge by its endpoints. Two
// bindings that connect the same role and ServiceAccount collapse into a
// single edge to keep graph cardinality bounded.
type roleBoundEdgeKey struct {
	startID string
	endID   string
}

// bindingProvenance captures the identity of a single RoleBinding or
// ClusterRoleBinding that contributes to a RoleBound edge.
type bindingProvenance struct {
	BindingKind      string // "RoleBinding" | "ClusterRoleBinding"
	BindingName      string
	BindingNamespace string // "" for ClusterRoleBinding
	RoleKind         string // "Role" | "ClusterRole"
	RoleName         string
}

// roleBoundAccumulator gathers every binding that produces the same
// (role, ServiceAccount) link so we can emit one merged edge per pair.
type roleBoundAccumulator struct {
	start    nodefw.EdgeNode
	end      nodefw.EdgeNode
	bindings []bindingProvenance
}

func (r rbacEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	// We accumulate locally rather than relying on framework.DeduplicateEdges
	// because that helper only keeps the first edge for a given
	// (start, end, kind) tuple and would discard properties contributed by
	// later bindings. Merging here preserves full provenance.
	acc := map[roleBoundEdgeKey]*roleBoundAccumulator{}
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		accumulateRoleToSAFromRoleBinding(ctx, ns, acc)
		accumulateClusterRoleToSAFromRoleBinding(ctx, ns, acc)
	}
	accumulateClusterRoleToSAFromClusterRoleBinding(ctx, acc)
	return makeRoleBoundEdges(acc)
}

// recordBinding inserts/updates an accumulator entry for the given endpoints.
func recordBinding(acc map[roleBoundEdgeKey]*roleBoundAccumulator, start, end nodefw.EdgeNode, prov bindingProvenance) {
	key := roleBoundEdgeKey{startID: start.EdgeID(), endID: end.EdgeID()}
	entry, ok := acc[key]
	if !ok {
		entry = &roleBoundAccumulator{start: start, end: end}
		acc[key] = entry
	}
	entry.bindings = append(entry.bindings, prov)
}

func accumulateRoleToSAFromRoleBinding(ctx *framework.Context, namespace string, acc map[roleBoundEdgeKey]*roleBoundAccumulator) {
	if ctx == nil {
		return
	}
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	roleIndex := ctx.Index.RolesByNamespace[namespace]
	bindingIndex := ctx.Index.RoleBindingsByNamespace[namespace]
	if len(bindingIndex) == 0 || len(roleIndex) == 0 || len(serviceAccounts) == 0 {
		return
	}
	for _, binding := range bindingIndex {
		if binding.RoleKind != "Role" {
			continue
		}
		role := roleIndex[binding.RoleName]
		if role == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" {
				continue
			}
			subjectNS := subject.Namespace
			if subjectNS == "" {
				subjectNS = binding.Namespace
			}
			if subjectNS != namespace {
				continue
			}
			sa := serviceAccounts[subject.Name]
			if sa == nil {
				continue
			}
			recordBinding(acc, role, sa, bindingProvenance{
				BindingKind:      "RoleBinding",
				BindingName:      binding.Name,
				BindingNamespace: binding.Namespace,
				RoleKind:         "Role",
				RoleName:         binding.RoleName,
			})
		}
	}
}

func accumulateClusterRoleToSAFromRoleBinding(ctx *framework.Context, namespace string, acc map[roleBoundEdgeKey]*roleBoundAccumulator) {
	if ctx == nil || ctx.Core == nil {
		return
	}
	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	for _, binding := range roleBindings {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" {
				continue
			}
			subjectNS := subject.Namespace
			if subjectNS == "" {
				subjectNS = binding.Namespace
			}
			saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}
			recordBinding(acc, clusterRole, sa, bindingProvenance{
				BindingKind:      "RoleBinding",
				BindingName:      binding.Name,
				BindingNamespace: binding.Namespace,
				RoleKind:         "ClusterRole",
				RoleName:         binding.RoleName,
			})
		}
	}
}

func accumulateClusterRoleToSAFromClusterRoleBinding(ctx *framework.Context, acc map[roleBoundEdgeKey]*roleBoundAccumulator) {
	if ctx == nil {
		return
	}
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" || subject.Namespace == "" {
				continue
			}
			saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}
			recordBinding(acc, clusterRole, sa, bindingProvenance{
				BindingKind:      "ClusterRoleBinding",
				BindingName:      binding.Name,
				BindingNamespace: "",
				RoleKind:         "ClusterRole",
				RoleName:         binding.RoleName,
			})
		}
	}
}

// sortBindings orders contributing bindings deterministically by
// (namespace, name, kind) so output is stable across runs.
func sortBindings(bindings []bindingProvenance) {
	sort.SliceStable(bindings, func(i, j int) bool {
		a, b := bindings[i], bindings[j]
		if a.BindingNamespace != b.BindingNamespace {
			return a.BindingNamespace < b.BindingNamespace
		}
		if a.BindingName != b.BindingName {
			return a.BindingName < b.BindingName
		}
		return a.BindingKind < b.BindingKind
	})
}

func makeRoleBoundEdges(acc map[roleBoundEdgeKey]*roleBoundAccumulator) []model.BloodHoundEdge {
	if len(acc) == 0 {
		return nil
	}
	keys := make([]roleBoundEdgeKey, 0, len(acc))
	for k := range acc {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].startID != keys[j].startID {
			return keys[i].startID < keys[j].startID
		}
		return keys[i].endID < keys[j].endID
	})

	edges := make([]model.BloodHoundEdge, 0, len(keys))
	for _, k := range keys {
		entry := acc[k]
		if entry == nil || len(entry.bindings) == 0 {
			continue
		}
		sortBindings(entry.bindings)

		bindingsList := make([]map[string]any, 0, len(entry.bindings))
		for _, b := range entry.bindings {
			bindingsList = append(bindingsList, map[string]any{
				"kind":      b.BindingKind,
				"name":      b.BindingName,
				"namespace": b.BindingNamespace,
				"roleKind":  b.RoleKind,
				"roleName":  b.RoleName,
			})
		}

		first := entry.bindings[0]
		props := map[string]any{
			"bindingKind":      first.BindingKind,
			"bindingName":      first.BindingName,
			"bindingNamespace": first.BindingNamespace,
			"roleKind":         first.RoleKind,
			"roleName":         first.RoleName,
			"bindings":         bindingsList,
			"bindingCount":     len(entry.bindings),
		}
		edges = append(edges, framework.CreateEdgeWithProperties(entry.end, entry.start, "RoleBound", props))
	}
	return edges
}
