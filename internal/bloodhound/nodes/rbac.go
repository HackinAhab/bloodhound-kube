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

	properties := map[string]any{}

	kind, _ := rbacResource["kind"].(string)

	switch kind {
	case "Role", "ClusterRole":
		return m.extractRoleProperties(rbacResource)
	case "RoleBinding", "ClusterRoleBinding":
		return m.extractBindingProperties(rbacResource)
	case "ServiceAccount":
		return m.extractServiceAccountProperties(rbacResource)
	}

	return properties, nil
}

func (m *RBACPropertyMapper) extractRoleProperties(role map[string]any) (map[string]any, error) {
	properties := map[string]any{
		"rbac_type": "role",
	}

	kind, _ := role["kind"].(string)
	properties["is_cluster_role"] = kind == "ClusterRole"

	if rules, ok := role["rules"].([]any); ok {
		highRiskPerms := []string{}
		hasWildcardVerbs := false
		hasWildcardResources := false
		canEscalatePrivileges := false
		canImpersonate := false

		for _, rule := range rules {
			if ruleMap, ok := rule.(map[string]any); ok {
				if verbs, ok := ruleMap["verbs"].([]any); ok {
					for _, verb := range verbs {
						if verbStr, ok := verb.(string); ok {
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

				if resources, ok := ruleMap["resources"].([]any); ok {
					for _, resource := range resources {
						if resourceStr, ok := resource.(string); ok {
							if resourceStr == "*" {
								hasWildcardResources = true
								highRiskPerms = append(highRiskPerms, "wildcard-resources")
							} else if strings.Contains(resourceStr, "secrets") {
								highRiskPerms = append(highRiskPerms, "secrets-access")
							}
						}
					}
				}
			}
		}

		properties["has_wildcard_verbs"] = hasWildcardVerbs
		properties["has_wildcard_resources"] = hasWildcardResources
		properties["can_escalate_privileges"] = canEscalatePrivileges
		properties["can_impersonate"] = canImpersonate
		properties["rules_count"] = len(rules)

		if len(highRiskPerms) > 0 {
			properties["high_risk_permissions"] = highRiskPerms
		}
	}

	return properties, nil
}

func (m *RBACPropertyMapper) extractBindingProperties(binding map[string]any) (map[string]any, error) {
	properties := map[string]any{
		"rbac_type": "binding",
	}

	kind, _ := binding["kind"].(string)
	properties["is_cluster_binding"] = kind == "ClusterRoleBinding"

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
		properties["subject_types"] = subjectTypes
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

	var name string
	if metadata, ok := rbacResource["metadata"].(map[string]any); ok && metadata != nil {
		name, _ = metadata["name"].(string)
	}
	kind, _ := rbacResource["kind"].(string)

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

	switch kind {
	case "ClusterRole":
		bhNode.Kinds = []string{"ClusterRole"}
	case "RoleBinding":
		bhNode.Kinds = []string{"RoleBinding"}
	case "ClusterRoleBinding":
		bhNode.Kinds = []string{"ClusterRoleBinding"}
	case "ServiceAccount":
		bhNode.Kinds = []string{"ServiceAccount"}
	default:
		bhNode.Kinds = []string{"Role"}
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
