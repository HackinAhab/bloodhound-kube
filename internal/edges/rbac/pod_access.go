package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacPodExecEdgesRule struct{}

func (r rbacPodExecEdgesRule) Name() string { return "rbac_pod_exec" }

var edgePropertiesRBACPodExec = map[string]any{
	"Description": "Identity has RBAC permissions to exec into pods.",
	"Reference":   "https://kubehound.io/reference/attacks/POD_EXEC/",
}

func (r rbacPodExecEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podExecNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, podExecCluster(ctx)...)
	return edges
}

func podExecNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/exec"}
	verbs := []string{"create"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(space.AllPods) > 0 {
					agg := &space.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodExec", edgePropertiesRBACPodExec))
				}
				continue
			}
			for i := range space.Pods {
				pod := &space.Pods[i]
				if _, ok := names[pod.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodExec", edgePropertiesRBACPodExec))
				}
			}
		}
	}
	return edges
}

func podExecCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/exec"}
	verbs := []string{"create"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodExec", edgePropertiesRBACPodExec))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodExec", edgePropertiesRBACPodExec))
					}
				}
			}
		}
	}
	return edges
}

type rbacPodPortForwardEdgesRule struct{}

func (r rbacPodPortForwardEdgesRule) Name() string { return "rbac_pod_portforward" }

var edgePropertiesRBACPodPortForward = map[string]any{
	"Description": "Identity has RBAC permissions to port-forward to pods, allowing TCP tunneling to any port on any pod in scope.",
	"Reference":   "https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/",
}

func (r rbacPodPortForwardEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podPortForwardNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, podPortForwardCluster(ctx)...)
	return edges
}

func podPortForwardNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/portforward"}
	verbs := []string{"create"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(space.AllPods) > 0 {
					agg := &space.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodPortForward", edgePropertiesRBACPodPortForward))
				}
				continue
			}
			for i := range space.Pods {
				pod := &space.Pods[i]
				if _, ok := names[pod.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodPortForward", edgePropertiesRBACPodPortForward))
				}
			}
		}
	}
	return edges
}

func podPortForwardCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/portforward"}
	verbs := []string{"create"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodPortForward", edgePropertiesRBACPodPortForward))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodPortForward", edgePropertiesRBACPodPortForward))
					}
				}
			}
		}
	}
	return edges
}

type rbacPodAttachEdgesRule struct{}

func (r rbacPodAttachEdgesRule) Name() string { return "rbac_pod_attach" }

var edgePropertiesRBACPodAttach = map[string]any{
	"Description": "Identity has RBAC permissions to attach to pods (kubectl attach), allowing interaction with running container stdin/stdout.",
	"Reference":   "https://kubehound.io/reference/attacks/POD_ATTACH/",
}

func (r rbacPodAttachEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podAttachNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, podAttachCluster(ctx)...)
	return edges
}

func podAttachNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/attach"}
	verbs := []string{"create"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(space.AllPods) > 0 {
					agg := &space.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodAttach", edgePropertiesRBACPodAttach))
				}
				continue
			}
			for i := range space.Pods {
				pod := &space.Pods[i]
				if _, ok := names[pod.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodAttach", edgePropertiesRBACPodAttach))
				}
			}
		}
	}
	return edges
}

func podAttachCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/attach"}
	verbs := []string{"create"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodAttach", edgePropertiesRBACPodAttach))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodAttach", edgePropertiesRBACPodAttach))
					}
				}
			}
		}
	}
	return edges
}

type rbacPodDebugEdgesRule struct{}

func (r rbacPodDebugEdgesRule) Name() string { return "rbac_pod_debug" }

var edgePropertiesRBACPodDebug = map[string]any{
	"Description": "Identity has RBAC permissions to debug pods.",
	"Reference":   "https://kubehound.io/reference/attacks/POD_ATTACH/",
}

func (r rbacPodDebugEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podDebugNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, podDebugCluster(ctx)...)
	return edges
}

func podDebugNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/ephemeralcontainers"}
	verbs := []string{"update"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(space.AllPods) > 0 {
					agg := &space.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodDebug", edgePropertiesRBACPodDebug))
				}
				continue
			}
			for i := range space.Pods {
				pod := &space.Pods[i]
				if _, ok := names[pod.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodDebug", edgePropertiesRBACPodDebug))
				}
			}
		}
	}
	return edges
}

func podDebugCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/ephemeralcontainers"}
	verbs := []string{"update"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_PodDebug", edgePropertiesRBACPodDebug))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_PodDebug", edgePropertiesRBACPodDebug))
					}
				}
			}
		}
	}
	return edges
}
