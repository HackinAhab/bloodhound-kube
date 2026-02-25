package edges

import (
	"bloodhound-kube/internal/model"
)

type rbacEdgesRule struct{}

func (r rbacEdgesRule) Name() string {
	return "rbac_base"
}

func (r rbacEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, roleToServiceAccountFromRoleBinding(ctx, ns)...)
		edges = append(edges, clusterRoleToServiceAccountFromRoleBinding(ctx, ns)...)
		edges = append(edges, serviceAccountToSecret(ctx, ns, space)...)
		// edges = append(edges, podDebugNamespaced(ctx, ns, space)...)
		// edges = append(edges, podExecNamespaced(ctx, ns, space)...)
		// edges = append(edges, workloadCreateNamespaced(ctx, ns)...)
		// edges = append(edges, workloadPatchNamespaced(ctx, ns, space)...)
		// edges = append(edges, rbacCreateNamespaced(ctx, ns, space)...)
		// edges = append(edges, saImpersonateNamespaced(ctx, ns, space)...)
		// edges = append(edges, saReadSecretNamespaced(ctx, ns, space)...)
		// edges = append(edges, rbacNodeProxyToPodNamespaced(ctx, ns)...)
	}
	edges = append(edges, clusterRoleToServiceAccountFromClusterRoleBinding(ctx)...)
	// edges = append(edges, podDebugCluster(ctx)...)
	// edges = append(edges, podExecCluster(ctx)...)
	// edges = append(edges, workloadCreateCluster(ctx)...)
	// edges = append(edges, workloadPatchCluster(ctx)...)
	// edges = append(edges, rbacCreateCluster(ctx)...)
	// edges = append(edges, saImpersonateCluster(ctx)...)
	// edges = append(edges, saReadSecretCluster(ctx)...)
	// edges = append(edges, rbacNodeProxyToPodCluster(ctx)...)
	return edges
}

func clusterRoleToServiceAccountFromRoleBinding(ctx *EdgeContext, namespace string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
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
			if saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]; saIndex != nil {
				if sa := saIndex[subject.Name]; sa != nil {
					edges = append(edges, CreateEdge(clusterRole, sa, "PermissionsFromRole"))
				}
			}
		}
	}
	return edges
}

func clusterRoleToServiceAccountFromClusterRoleBinding(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
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
			if subject.Namespace == "" {
				continue
			}
			if saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]; saIndex != nil {
				if sa := saIndex[subject.Name]; sa != nil {
					edges = append(edges, CreateEdge(clusterRole, sa, "PermissionsFromRole"))
				}
			}
		}
	}
	return edges
}

func roleToServiceAccountFromRoleBinding(ctx *EdgeContext, namespace string) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	roleIndex := ctx.Index.RolesByNamespace[namespace]
	bindingIndex := ctx.Index.RoleBindingsByNamespace[namespace]
	if len(bindingIndex) == 0 || len(roleIndex) == 0 || len(serviceAccounts) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
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
			if sa := serviceAccounts[subject.Name]; sa != nil {
				edges = append(edges, CreateEdge(role, sa, "PermissionsFromRole"))
			}
		}
	}
	return edges
}

func serviceAccountToSecret(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	if len(serviceAccounts) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range space.Secrets {
		secret := &space.Secrets[i]
		if secret.SecretType != "kubernetes.io/service-account-token" {
			continue
		}
		saName := ""
		if secret.AnnotationsMap != nil {
			if name, ok := secret.AnnotationsMap["kubernetes.io/service-account.name"].(string); ok {
				saName = name
			}
		}
		if saName == "" {
			continue
		}
		if sa := serviceAccounts[saName]; sa != nil {
			edges = append(edges, CreateEdge(sa, secret, "SAToken"))
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(rbacEdgesRule{})
}
