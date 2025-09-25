package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
)

type RBACPropertyMapper struct{}

func (m *RBACPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	resourceData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RBAC resource: %w", err)
	}

	var rbacResource map[string]any
	if err := json.Unmarshal(resourceData, &rbacResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC resource: %w", err)
	}

	// Check for resource type in the data structure
	var resourceType string
	if rt, ok := rbacResource["resource_type"].(string); ok {
		resourceType = rt
	}

	// If we can't determine from the resource itself, we need to inspect the structure
	if resourceType == "" {
		// Check for kind field (legacy "rbac" type resources)
		if kind, ok := rbacResource["kind"].(string); ok {
			switch kind {
			case "Role":
				resourceType = "Role"
			case "ClusterRole":
				resourceType = "ClusterRole"
			case "RoleBinding":
				resourceType = "RoleBinding"
			case "ClusterRoleBinding":
				resourceType = "ClusterRoleBinding"
			case "ServiceAccount":
				resourceType = "ServiceAccount"
			}
		} else {
			// Check for fields that identify the type (new format)
			if _, hasRules := rbacResource["rules"]; hasRules {
				if _, hasNamespace := rbacResource["namespace"]; hasNamespace {
					resourceType = "role"
				} else {
					resourceType = "cluster_role"
				}
			} else if _, hasRoleRef := rbacResource["role_ref"]; hasRoleRef {
				if _, hasNamespace := rbacResource["namespace"]; hasNamespace {
					resourceType = "role_binding"
				} else {
					resourceType = "cluster_role_binding"
				}
			} else if _, hasAutomount := rbacResource["automount_service_account"]; hasAutomount {
				resourceType = "service_account"
			}
		}
	}

	switch resourceType {
	case "role", "cluster_role":
		return m.extractRoleProperties(rbacResource, resourceType)
	case "role_binding", "cluster_role_binding":
		return m.extractBindingProperties(rbacResource, resourceType)
	case "service_account":
		return m.extractServiceAccountProperties(rbacResource)
	}

	return map[string]any{}, nil
}

func (m *RBACPropertyMapper) extractRoleProperties(role map[string]any, resourceType string) (map[string]any, error) {
	properties := map[string]any{
		"rbac_type": "role",
	}

	properties["is_cluster_role"] = resourceType == "cluster_role"

	if rules, ok := role["rules"].([]any); ok {
		highRiskPerms := []string{}
		hasWildcardVerbs := false
		hasWildcardResources := false
		canEscalatePrivileges := false
		canImpersonate := false

		// Create correlated permissions list in the format:
		// ["api_group - resource - resource_names - verbs"]
		permissions := []string{}

		for _, rule := range rules {
			if ruleMap, ok := rule.(map[string]any); ok {
				// Extract API groups, resources, verbs, and resource names from this rule
				var apiGroups []string
				var resources []string
				var verbs []string
				var resourceNames []string

				if apiGroupsData, ok := ruleMap["api_groups"].([]any); ok {
					for _, group := range apiGroupsData {
						if groupStr, ok := group.(string); ok {
							apiGroups = append(apiGroups, groupStr)
						}
					}
				}

				if resourcesData, ok := ruleMap["resources"].([]any); ok {
					for _, resource := range resourcesData {
						if resourceStr, ok := resource.(string); ok {
							resources = append(resources, resourceStr)
							if resourceStr == "*" {
								hasWildcardResources = true
								highRiskPerms = append(highRiskPerms, "wildcard-resources")
							} else if strings.Contains(resourceStr, "secrets") {
								highRiskPerms = append(highRiskPerms, "secrets-access")
							}
						}
					}
				}

				if verbsData, ok := ruleMap["verbs"].([]any); ok {
					for _, verb := range verbsData {
						if verbStr, ok := verb.(string); ok {
							verbs = append(verbs, verbStr)
							switch verbStr {
							case "*":
								hasWildcardVerbs = true
								highRiskPerms = append(highRiskPerms, "wildcard-verbs")
							case "escalate", "bind":
								canEscalatePrivileges = true
								highRiskPerms = append(highRiskPerms, verbStr)
							case "impersonate":
								canImpersonate = true
								highRiskPerms = append(highRiskPerms, "impersonate")
							}
						}
					}
				}

				if resourceNamesData, ok := ruleMap["resource_names"].([]any); ok {
					for _, resourceName := range resourceNamesData {
						if resourceNameStr, ok := resourceName.(string); ok {
							resourceNames = append(resourceNames, resourceNameStr)
						}
					}
				}

				// Create correlated permission strings for this rule
				// Handle cases where arrays might be empty
				if len(apiGroups) == 0 {
					apiGroups = []string{""}
				}
				if len(resources) == 0 {
					resources = []string{"*"}
				}
				if len(verbs) == 0 {
					verbs = []string{"*"}
				}

				// Create permission entries for each combination
				for _, apiGroup := range apiGroups {
					for _, resource := range resources {
						// Format resource names
						resourceNamesStr := "[]"
						if len(resourceNames) > 0 {
							resourceNamesStr = "[" + strings.Join(resourceNames, ",") + "]"
						}

						// Format verbs
						verbsStr := strings.Join(verbs, ",")
						if len(verbs) == 1 && verbs[0] == "*" {
							verbsStr = "*"
						}

						// Create the permission string
						permission := fmt.Sprintf("%s - %s - %s - %s", apiGroup, resource, resourceNamesStr, verbsStr)
						permissions = append(permissions, permission)
					}
				}
			}
		}

		// Add permission count instead of the array
		properties["permissions_count"] = len(permissions)
		properties["has_permissions"] = len(permissions) > 0

		properties["has_wildcard_verbs"] = hasWildcardVerbs
		properties["has_wildcard_resources"] = hasWildcardResources
		properties["can_escalate_privileges"] = canEscalatePrivileges
		properties["can_impersonate"] = canImpersonate
		properties["rules_count"] = len(rules)

		properties["high_risk_permissions_count"] = len(highRiskPerms)
		properties["has_high_risk_permissions"] = len(highRiskPerms) > 0
		if len(highRiskPerms) > 0 {
			properties["has_wildcard_resources_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "wildcard-resources")
			properties["has_secrets_access_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "secrets-access")
			properties["has_wildcard_verbs_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "wildcard-verbs")
			properties["has_escalate_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "escalate")
			properties["has_bind_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "bind")
			properties["has_impersonate_perm"] = strings.Contains(strings.Join(highRiskPerms, ","), "impersonate")
		}
	}

	return properties, nil
}

func (m *RBACPropertyMapper) extractBindingProperties(binding map[string]any, resourceType string) (map[string]any, error) {
	properties := map[string]any{
		"rbac_type": "binding",
	}

	properties["is_cluster_binding"] = resourceType == "cluster_role_binding"

	if roleRef, ok := binding["roleRef"].(map[string]any); ok {
		if roleName, ok := roleRef["name"].(string); ok {
			properties["bound_role"] = roleName

			dangerousRoles := []string{"cluster-admin", "admin", "edit", "system:masters"}
			for _, dangerous := range dangerousRoles {
				if strings.Contains(strings.ToLower(roleName), dangerous) {
					properties["binds_privileged_role"] = true
					properties["privileged_role_type"] = dangerous
					break
				}
			}
		}
	}

	if subjects, ok := binding["subjects"].([]any); ok {
		properties["subject_count"] = len(subjects)

		subjectTypes := []string{}
		hasServiceAccount := false
		hasUser := false
		hasGroup := false

		for _, subject := range subjects {
			if subjectMap, ok := subject.(map[string]any); ok {
				if kind, ok := subjectMap["kind"].(string); ok {
					subjectTypes = append(subjectTypes, kind)

					switch kind {
					case "ServiceAccount":
						hasServiceAccount = true
					case "User":
						hasUser = true
					case "Group":
						hasGroup = true
					}
				}
			}
		}

		properties["has_service_account_subjects"] = hasServiceAccount
		properties["has_user_subjects"] = hasUser
		properties["has_group_subjects"] = hasGroup
		properties["subject_types_count"] = len(subjectTypes)
	}

	return properties, nil
}

func (m *RBACPropertyMapper) extractServiceAccountProperties(sa map[string]any) (map[string]any, error) {
	properties := map[string]any{
		"rbac_type": "service_account",
	}

	if automountToken, ok := sa["automountServiceAccountToken"].(bool); ok {
		properties["automount_token"] = automountToken
	}

	if secrets, ok := sa["secrets"].([]any); ok {
		properties["secrets_count"] = len(secrets)
		properties["has_secrets"] = len(secrets) > 0
	}

	if imagePullSecrets, ok := sa["imagePullSecrets"].([]any); ok {
		properties["image_pull_secrets_count"] = len(imagePullSecrets)
		properties["has_image_pull_secrets"] = len(imagePullSecrets) > 0
	}

	return properties, nil
}

type RBACParser struct {
	config bloodhound.ResourceConfig
}

func NewRBACParser() *RBACParser {
	return &RBACParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "rbac",
			PrimaryKind:    "Role",
			SecondaryKinds: []string{"ClusterRole", "RoleBinding", "ClusterRoleBinding", "ServiceAccount"},
			PropertyMapper: &RBACPropertyMapper{},
		},
	}
}

func (p *RBACParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *RBACParser) GetSupportedResourceTypes() []string {
	return []string{"rbac", "role", "role_binding", "cluster_role", "cluster_role_binding", "service_account"}
}

func (p *RBACParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *RBACParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *RBACParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	resourceData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RBAC resource: %w", err)
	}

	var rbacResource map[string]any
	if err := json.Unmarshal(resourceData, &rbacResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC resource: %w", err)
	}

	// Extract name from the resource data
	var name string
	if resourceName, ok := rbacResource["name"].(string); ok {
		name = resourceName
	}

	// Determine the kind based on resource type or existing kind field
	var kind string
	if resource.Type == "rbac" {
		// For legacy "rbac" type, use the kind field from the resource
		if resourceKind, ok := rbacResource["kind"].(string); ok {
			kind = resourceKind
		} else {
			kind = "Role" // fallback
		}
	} else {
		// For new specific types, map to the appropriate kind
		switch resource.Type {
		case "cluster_role":
			kind = "ClusterRole"
		case "role":
			kind = "Role"
		case "cluster_role_binding":
			kind = "ClusterRoleBinding"
		case "role_binding":
			kind = "RoleBinding"
		case "service_account":
			kind = "ServiceAccount"
		default:
			kind = "Role" // fallback
		}
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RBAC node: %w", err)
	}

	bhNode.Kinds = []string{kind}

	// Note: RBAC edges are now created globally in ProcessGlobalRBACEdges
	// to ensure proper Role/ClusterRole -> Binding -> Subject chain relationships
	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{}, // No edges created here anymore
	}

	return result, nil
}
