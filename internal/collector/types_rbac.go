package collector

// This file contains type definitions for Kubernetes RBAC resources:
// Roles, ClusterRoles, RoleBindings, ClusterRoleBindings, and related structures.

type RBACResource struct {
	CommonResourceMeta
	Kind     string        `json:"kind"`
	Rules    []PolicyRule  `json:"rules,omitempty"`
	Subjects []RBACSubject `json:"subjects,omitempty"`
	RoleRef  *RoleRef      `json:"role_ref,omitempty"`
}

type PolicyRule struct {
	APIGroups     []string `json:"api_groups,omitempty"`
	Resources     []string `json:"resources,omitempty"`
	Verbs         []string `json:"verbs,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
}

type RBACSubject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type RoleRef struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	APIGroup string `json:"api_group"`
}
