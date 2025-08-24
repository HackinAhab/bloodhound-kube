package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RBACResources struct {
	Roles               []Role               `json:"roles"`
	RoleBindings        []RoleBinding        `json:"role_bindings"`
	ClusterRoles        []ClusterRole        `json:"cluster_roles"`
	ClusterRoleBindings []ClusterRoleBinding `json:"cluster_role_bindings"`
	ServiceAccounts     []ServiceAccount     `json:"service_accounts"`
}

type Role struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Rules       []PolicyRule      `json:"rules"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type RoleBinding struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	RoleRef     RoleRef           `json:"role_ref"`
	Subjects    []Subject         `json:"subjects"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type ClusterRole struct {
	Name        string            `json:"name"`
	Rules       []PolicyRule      `json:"rules"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type ClusterRoleBinding struct {
	Name        string            `json:"name"`
	RoleRef     RoleRef           `json:"role_ref"`
	Subjects    []Subject         `json:"subjects"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type ServiceAccount struct {
	Name                     string            `json:"name"`
	Namespace                string            `json:"namespace"`
	Secrets                  []string          `json:"secrets,omitempty"`
	ImagePullSecrets         []string          `json:"image_pull_secrets,omitempty"`
	AutomountServiceAccount  bool              `json:"automount_service_account"`
	Labels                   map[string]string `json:"labels,omitempty"`
	Annotations              map[string]string `json:"annotations,omitempty"`
	CreatedAt                string            `json:"created_at"`
}

type PolicyRule struct {
	APIGroups       []string `json:"api_groups,omitempty"`
	Resources       []string `json:"resources,omitempty"`
	ResourceNames   []string `json:"resource_names,omitempty"`
	Verbs           []string `json:"verbs"`
	NonResourceURLs []string `json:"non_resource_urls,omitempty"`
}

type RoleRef struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	APIGroup string `json:"api_group"`
}

type Subject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	APIGroup  string `json:"api_group,omitempty"`
}

func (c *Collector) CollectRBAC(ctx context.Context, namespace string) (*RBACResources, error) {
	c.logger.Info("Collecting RBAC resources", "namespace", namespace)

	rbac := &RBACResources{}

	roleList, err := c.client.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	for _, role := range roleList.Items {
		var rules []PolicyRule
		for _, rule := range role.Rules {
			rules = append(rules, PolicyRule{
				APIGroups:       rule.APIGroups,
				Resources:       rule.Resources,
				ResourceNames:   rule.ResourceNames,
				Verbs:           rule.Verbs,
				NonResourceURLs: rule.NonResourceURLs,
			})
		}

		rbac.Roles = append(rbac.Roles, Role{
			Name:        role.Name,
			Namespace:   role.Namespace,
			Rules:       rules,
			Labels:      role.Labels,
			Annotations: role.Annotations,
			CreatedAt:   role.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	roleBindingList, err := c.client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	for _, rb := range roleBindingList.Items {
		var subjects []Subject
		for _, subj := range rb.Subjects {
			subjects = append(subjects, Subject{
				Kind:      subj.Kind,
				Name:      subj.Name,
				Namespace: subj.Namespace,
				APIGroup:  subj.APIGroup,
			})
		}

		rbac.RoleBindings = append(rbac.RoleBindings, RoleBinding{
			Name:      rb.Name,
			Namespace: rb.Namespace,
			RoleRef: RoleRef{
				Kind:     rb.RoleRef.Kind,
				Name:     rb.RoleRef.Name,
				APIGroup: rb.RoleRef.APIGroup,
			},
			Subjects:    subjects,
			Labels:      rb.Labels,
			Annotations: rb.Annotations,
			CreatedAt:   rb.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	clusterRoleList, err := c.client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles: %w", err)
	}

	for _, cr := range clusterRoleList.Items {
		var rules []PolicyRule
		for _, rule := range cr.Rules {
			rules = append(rules, PolicyRule{
				APIGroups:       rule.APIGroups,
				Resources:       rule.Resources,
				ResourceNames:   rule.ResourceNames,
				Verbs:           rule.Verbs,
				NonResourceURLs: rule.NonResourceURLs,
			})
		}

		rbac.ClusterRoles = append(rbac.ClusterRoles, ClusterRole{
			Name:        cr.Name,
			Rules:       rules,
			Labels:      cr.Labels,
			Annotations: cr.Annotations,
			CreatedAt:   cr.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	clusterRoleBindingList, err := c.client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings: %w", err)
	}

	for _, crb := range clusterRoleBindingList.Items {
		var subjects []Subject
		for _, subj := range crb.Subjects {
			subjects = append(subjects, Subject{
				Kind:      subj.Kind,
				Name:      subj.Name,
				Namespace: subj.Namespace,
				APIGroup:  subj.APIGroup,
			})
		}

		rbac.ClusterRoleBindings = append(rbac.ClusterRoleBindings, ClusterRoleBinding{
			Name: crb.Name,
			RoleRef: RoleRef{
				Kind:     crb.RoleRef.Kind,
				Name:     crb.RoleRef.Name,
				APIGroup: crb.RoleRef.APIGroup,
			},
			Subjects:    subjects,
			Labels:      crb.Labels,
			Annotations: crb.Annotations,
			CreatedAt:   crb.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	serviceAccountList, err := c.client.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	for _, sa := range serviceAccountList.Items {
		var secrets []string
		for _, secret := range sa.Secrets {
			secrets = append(secrets, secret.Name)
		}

		var imagePullSecrets []string
		for _, ips := range sa.ImagePullSecrets {
			imagePullSecrets = append(imagePullSecrets, ips.Name)
		}

		automount := true
		if sa.AutomountServiceAccountToken != nil {
			automount = *sa.AutomountServiceAccountToken
		}

		rbac.ServiceAccounts = append(rbac.ServiceAccounts, ServiceAccount{
			Name:                    sa.Name,
			Namespace:               sa.Namespace,
			Secrets:                 secrets,
			ImagePullSecrets:        imagePullSecrets,
			AutomountServiceAccount: automount,
			Labels:                  sa.Labels,
			Annotations:             sa.Annotations,
			CreatedAt:               sa.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.logger.Info("Successfully collected RBAC resources", 
		"roles", len(rbac.Roles), 
		"role_bindings", len(rbac.RoleBindings), 
		"cluster_roles", len(rbac.ClusterRoles), 
		"cluster_role_bindings", len(rbac.ClusterRoleBindings), 
		"service_accounts", len(rbac.ServiceAccounts))
	
	return rbac, nil
}
