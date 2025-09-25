package edges

import (
	"bloodhound-kube/internal/bloodhound"
	"encoding/json"
	"fmt"
)

// RBACEdgeBuilder handles creation of edges between RBAC resources
type RBACEdgeBuilder struct {
	*EdgeBuilder
}

// NewRBACEdgeBuilder creates a new RBAC-specific edge builder
func NewRBACEdgeBuilder() *RBACEdgeBuilder {
	return &RBACEdgeBuilder{
		EdgeBuilder: NewEdgeBuilder(),
	}
}

// CreateRBACEdges processes RBAC binding resources and creates edges between roles and subjects
func (reb *RBACEdgeBuilder) CreateRBACEdges(resource bloodhound.ResourceData) ([]bloodhound.BloodHoundEdge, error) {
	var edges []bloodhound.BloodHoundEdge

	// Only process role bindings and cluster role bindings
	if resource.Type != "role_binding" && resource.Type != "cluster_role_binding" {
		return edges, nil
	}

	// Parse the resource data
	resourceData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RBAC binding resource: %w", err)
	}

	var binding map[string]any
	if err := json.Unmarshal(resourceData, &binding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC binding resource: %w", err)
	}

	// Extract roleRef to determine target role (try both camelCase and snake_case)
	var roleRef map[string]any
	var ok bool
	if roleRef, ok = binding["roleRef"].(map[string]any); !ok {
		if roleRef, ok = binding["role_ref"].(map[string]any); !ok {
			return edges, nil // No roleRef, skip
		}
	}

	roleName, ok := roleRef["name"].(string)
	if !ok {
		return edges, nil // No role name, skip
	}

	roleKind, ok := roleRef["kind"].(string)
	if !ok {
		return edges, nil // No role kind, skip
	}

	// Generate role node ID based on role type
	var roleResourceType string
	var roleNodeKind string
	if roleKind == "ClusterRole" {
		roleResourceType = "cluster_role"
		roleNodeKind = "ClusterRole"
	} else {
		roleResourceType = "role"
		roleNodeKind = "Role"
	}

	// For roles, we need to handle namespace
	var roleNamespace string
	if roleKind == "Role" {
		roleNamespace = resource.Namespace
	}

	roleNodeID := bloodhound.GenerateNodeID(roleNodeKind, roleResourceType, roleNamespace, roleName)

	// Extract subjects to determine source entities
	subjects, ok := binding["subjects"].([]any)
	if !ok {
		return edges, nil // No subjects, skip
	}

	bindingName, _ := binding["name"].(string)

	// Create edges for each subject
	for _, subject := range subjects {
		subjectMap, ok := subject.(map[string]any)
		if !ok {
			continue
		}

		subjectKind, ok := subjectMap["kind"].(string)
		if !ok {
			continue
		}

		subjectName, ok := subjectMap["name"].(string)
		if !ok {
			continue
		}

		// Get subject namespace
		subjectNamespace := ""
		if ns, exists := subjectMap["namespace"].(string); exists {
			subjectNamespace = ns
		} else if resource.Type == "role_binding" {
			// For role bindings, subjects in same namespace if not specified
			subjectNamespace = resource.Namespace
		}

		// Generate subject node ID based on subject type
		var subjectNodeID string
		switch subjectKind {
		case "ServiceAccount":
			subjectNodeID = bloodhound.GenerateNodeID("ServiceAccount", "service_account", subjectNamespace, subjectName)
		case "User":
			subjectNodeID = bloodhound.GenerateNodeID("User", "user", "", subjectName)
		case "Group":
			subjectNodeID = bloodhound.GenerateNodeID("Group", "group", "", subjectName)
		default:
			continue // Unsupported subject kind
		}

		// Create edge properties
		edgeProperties := map[string]any{
			"binding_name": bindingName,
			"binding_type": resource.Type,
			"subject_kind": subjectKind,
			"subject_name": subjectName,
			"role_name":    roleName,
			"role_kind":    roleKind,
		}

		if subjectNamespace != "" {
			edgeProperties["subject_namespace"] = subjectNamespace
		}

		if roleNamespace != "" {
			edgeProperties["role_namespace"] = roleNamespace
		}

		// Create edge from subject to role (subject "has" role permissions)
		edge := reb.CreateEdge(subjectNodeID, roleNodeID, "HasRole", edgeProperties)
		edges = append(edges, edge)
	}

	return edges, nil
}

// CreateRoleToPermissionEdges creates edges representing specific permissions granted by roles
// This creates more granular edges for each permission rather than just role assignments
func (reb *RBACEdgeBuilder) CreateRoleToPermissionEdges(resource bloodhound.ResourceData) ([]bloodhound.BloodHoundEdge, error) {
	var edges []bloodhound.BloodHoundEdge

	// Only process roles and cluster roles
	if resource.Type != "role" && resource.Type != "cluster_role" {
		return edges, nil
	}

	// Parse the resource data
	resourceData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RBAC role resource: %w", err)
	}

	var role map[string]any
	if err := json.Unmarshal(resourceData, &role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC role resource: %w", err)
	}

	roleName, ok := role["name"].(string)
	if !ok {
		return edges, nil
	}

	// Generate role node ID
	var roleNodeKind string
	if resource.Type == "cluster_role" {
		roleNodeKind = "ClusterRole"
	} else {
		roleNodeKind = "Role"
	}

	roleNodeID := bloodhound.GenerateNodeID(roleNodeKind, resource.Type, resource.Namespace, roleName)

	// Extract rules to create permission edges
	rules, ok := role["rules"].([]any)
	if !ok {
		return edges, nil
	}

	// Create virtual permission nodes for each significant permission combination
	permissionIndex := 0
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]any)
		if !ok {
			continue
		}

		// Extract API groups, resources, and verbs
		apiGroups := []string{}
		if apiGroupsData, ok := ruleMap["api_groups"].([]any); ok {
			for _, group := range apiGroupsData {
				if groupStr, ok := group.(string); ok {
					apiGroups = append(apiGroups, groupStr)
				}
			}
		}

		resources := []string{}
		if resourcesData, ok := ruleMap["resources"].([]any); ok {
			for _, res := range resourcesData {
				if resStr, ok := res.(string); ok {
					resources = append(resources, resStr)
				}
			}
		}

		verbs := []string{}
		if verbsData, ok := ruleMap["verbs"].([]any); ok {
			for _, verb := range verbsData {
				if verbStr, ok := verb.(string); ok {
					verbs = append(verbs, verbStr)
				}
			}
		}

		resourceNames := []string{}
		if resourceNamesData, ok := ruleMap["resource_names"].([]any); ok {
			for _, resourceName := range resourceNamesData {
				if resourceNameStr, ok := resourceName.(string); ok {
					resourceNames = append(resourceNames, resourceNameStr)
				}
			}
		}

		// Create edges for each permission combination
		for _, apiGroup := range apiGroups {
			for _, resourceType := range resources {
				for _, verb := range verbs {
					permissionIndex++

					// Create a virtual permission identifier
					permissionID := fmt.Sprintf("Permission:%s:%s:%s:%s:%d",
						bloodhound.GenerateObjectID(resource.Type, resource.Namespace, roleName),
						apiGroup, resourceType, verb, permissionIndex)

					// Edge properties for the permission
					permissionProps := map[string]any{
						"api_group":     apiGroup,
						"resource_type": resourceType,
						"verb":          verb,
						"role_name":     roleName,
						"role_type":     resource.Type,
					}

					if len(resourceNames) > 0 {
						permissionProps["resource_names"] = resourceNames
					}

					if resource.Namespace != "" {
						permissionProps["namespace"] = resource.Namespace
					}

					// Create edge from role to permission
					edge := reb.CreateEdge(roleNodeID, permissionID, "GrantsPermission", permissionProps)
					edges = append(edges, edge)
				}
			}
		}
	}

	return edges, nil
}
