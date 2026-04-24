package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacReadSecretsEdgesRule struct{}

func (r rbacReadSecretsEdgesRule) Name() string { return "rbac_read_secrets" }

var edgePropertiesRBACReadSecrets = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to read secrets.",
}

func (r rbacReadSecretsEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, saReadSecretNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, saReadSecretCluster(ctx)...)
	return edges
}

func saReadSecretNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"secrets"}
	verbs := []string{"get", "list", "watch"}

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
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			for i := range space.Secrets {
				secret := &space.Secrets[i]
				if all {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, secret, "SAReadSecret", edgePropertiesRBACReadSecrets))
					continue
				}
				if _, ok := names[secret.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, secret, "SAReadSecret", edgePropertiesRBACReadSecrets))
				}
			}
		}
	}
	return edges
}

func saReadSecretCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"secrets"}
	verbs := []string{"get", "list", "watch"}

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
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllSecrets) > 0 {
					agg := &ctx.Core.Cluster.AllSecrets[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "SAReadSecret", edgePropertiesRBACReadSecrets))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Secrets {
					secret := &space.Secrets[i]
					if _, ok := names[secret.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, secret, "SAReadSecret", edgePropertiesRBACReadSecrets))
					}
				}
			}
		}
	}
	return edges
}

type rbacReadConfigMapsEdgesRule struct{}

func (r rbacReadConfigMapsEdgesRule) Name() string { return "rbac_read_configmaps" }

var edgePropertiesRBACReadConfigMaps = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to read configmaps.",
}

func (r rbacReadConfigMapsEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, saReadConfigMapNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, saReadConfigMapCluster(ctx)...)
	return edges
}

func saReadConfigMapNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"configmaps"}
	verbs := []string{"get", "list", "watch"}

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
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			for i := range space.ConfigMaps {
				configMap := &space.ConfigMaps[i]
				if all {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, configMap, "ReadConfigMap", edgePropertiesRBACReadConfigMaps))
					continue
				}
				if _, ok := names[configMap.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, configMap, "ReadConfigMap", edgePropertiesRBACReadConfigMaps))
				}
			}
		}
	}
	return edges
}

func saReadConfigMapCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"configmaps"}
	verbs := []string{"get", "list", "watch"}

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
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllConfigMaps) > 0 {
					agg := &ctx.Core.Cluster.AllConfigMaps[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ReadConfigMap", edgePropertiesRBACReadConfigMaps))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.ConfigMaps {
					configMap := &space.ConfigMaps[i]
					if _, ok := names[configMap.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, configMap, "ReadConfigMap", edgePropertiesRBACReadConfigMaps))
					}
				}
			}
		}
	}
	return edges
}
