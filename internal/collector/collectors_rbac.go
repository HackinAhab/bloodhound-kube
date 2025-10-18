package collector

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// collectRBAC collects RBAC resources (cluster-scoped: Roles, ClusterRoles, RoleBindings, ClusterRoleBindings)
func collectRBAC(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting RBAC resources")
	c.logger.Debug("Starting RBAC collection", "namespace", namespace)

	var rbacResources []any

	if namespace != "" {
		// Collect namespaced RBAC resources for specific namespace
		rbacResources = append(rbacResources, collectRoles(ctx, c, namespace)...)
		rbacResources = append(rbacResources, collectRoleBindings(ctx, c, namespace)...)
	} else {
		// Collect cluster-scoped RBAC resources
		rbacResources = append(rbacResources, collectClusterRoles(ctx, c)...)
		rbacResources = append(rbacResources, collectClusterRoleBindings(ctx, c)...)

		// Also collect all namespaced RBAC resources when doing cluster-wide collection
		rbacResources = append(rbacResources, collectRoles(ctx, c, "")...)
		rbacResources = append(rbacResources, collectRoleBindings(ctx, c, "")...)
	}

	c.logger.Info("Successfully collected RBAC resources", "count", len(rbacResources))
	c.logger.Debug("RBAC collection completed", "processed", len(rbacResources))
	return rbacResources, nil
}

// collectRoles collects Roles from the specified namespace (or all namespaces if namespace is empty)
func collectRoles(ctx context.Context, c *Collector, namespace string) []any {
	roleList, err := c.clients.Kubernetes.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var roles []any
	for _, role := range roleList.Items {
		var rules []PolicyRule
		for _, rule := range role.Rules {
			rules = append(rules, PolicyRule{
				APIGroups:     rule.APIGroups,
				Resources:     rule.Resources,
				Verbs:         rule.Verbs,
				ResourceNames: rule.ResourceNames,
			})
		}

		roles = append(roles, RBACResource{
			CommonResourceMeta: CommonResourceMeta{
				Name:        role.Name,
				Namespace:   role.Namespace,
				Labels:      role.Labels,
				Annotations: AnnotationsCleaner(role.Annotations),
				CreatedAt:   role.CreationTimestamp.Time,
			},
			Kind:  "Role",
			Rules: rules,
		})
	}
	return roles
}

// collectRoleBindings collects RoleBindings from the specified namespace (or all namespaces if namespace is empty)
func collectRoleBindings(ctx context.Context, c *Collector, namespace string) []any {
	roleBindingList, err := c.clients.Kubernetes.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var roleBindings []any
	for _, rb := range roleBindingList.Items {
		var subjects []RBACSubject
		for _, subject := range rb.Subjects {
			subjects = append(subjects, RBACSubject{
				Kind:      subject.Kind,
				Name:      subject.Name,
				Namespace: subject.Namespace,
			})
		}

		roleBindings = append(roleBindings, RBACResource{
			CommonResourceMeta: CommonResourceMeta{
				Name:        rb.Name,
				Namespace:   rb.Namespace,
				Labels:      rb.Labels,
				Annotations: AnnotationsCleaner(rb.Annotations),
				CreatedAt:   rb.CreationTimestamp.Time,
			},
			Kind:     "RoleBinding",
			Subjects: subjects,
			RoleRef: &RoleRef{
				Kind:     rb.RoleRef.Kind,
				Name:     rb.RoleRef.Name,
				APIGroup: rb.RoleRef.APIGroup,
			},
		})
	}
	return roleBindings
}

// collectClusterRoles collects ClusterRoles (cluster-scoped)
func collectClusterRoles(ctx context.Context, c *Collector) []any {
	clusterRoleList, err := c.clients.Kubernetes.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var clusterRoles []any
	for _, cr := range clusterRoleList.Items {
		var rules []PolicyRule
		for _, rule := range cr.Rules {
			rules = append(rules, PolicyRule{
				APIGroups:     rule.APIGroups,
				Resources:     rule.Resources,
				Verbs:         rule.Verbs,
				ResourceNames: rule.ResourceNames,
			})
		}

		clusterRoles = append(clusterRoles, RBACResource{
			CommonResourceMeta: CommonResourceMeta{
				Name:        cr.Name,
				Labels:      cr.Labels,
				Annotations: AnnotationsCleaner(cr.Annotations),
				CreatedAt:   cr.CreationTimestamp.Time,
			},
			Kind:  "ClusterRole",
			Rules: rules,
		})
	}
	return clusterRoles
}

// collectClusterRoleBindings collects ClusterRoleBindings (cluster-scoped)
func collectClusterRoleBindings(ctx context.Context, c *Collector) []any {
	clusterRoleBindingList, err := c.clients.Kubernetes.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var clusterRoleBindings []any
	for _, crb := range clusterRoleBindingList.Items {
		var subjects []RBACSubject
		for _, subject := range crb.Subjects {
			subjects = append(subjects, RBACSubject{
				Kind:      subject.Kind,
				Name:      subject.Name,
				Namespace: subject.Namespace,
			})
		}

		clusterRoleBindings = append(clusterRoleBindings, RBACResource{
			CommonResourceMeta: CommonResourceMeta{
				Name:        crb.Name,
				Labels:      crb.Labels,
				Annotations: AnnotationsCleaner(crb.Annotations),
				CreatedAt:   crb.CreationTimestamp.Time,
			},
			Kind:     "ClusterRoleBinding",
			Subjects: subjects,
			RoleRef: &RoleRef{
				Kind:     crb.RoleRef.Kind,
				Name:     crb.RoleRef.Name,
				APIGroup: crb.RoleRef.APIGroup,
			},
		})
	}
	return clusterRoleBindings
}
