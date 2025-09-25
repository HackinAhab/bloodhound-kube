package bloodhound

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"
)

func GenerateObjectID(resourceType, namespace, name string) string {
	var identifier string
	if namespace != "" {
		identifier = fmt.Sprintf("%s/%s/%s", resourceType, namespace, name)
	} else {
		identifier = fmt.Sprintf("%s/%s", resourceType, name)
	}

	hash := sha256.Sum256([]byte(identifier))
	return fmt.Sprintf("%x", hash)[:16]
}

func GenerateNodeID(label, resourceType, namespace, name string) string {
	baseID := GenerateObjectID(resourceType, namespace, name)
	return fmt.Sprintf("%s:%s", label, baseID)
}

func SanitizeLabel(label string) string {
	sanitized := strings.ToUpper(label)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return sanitized
}

func CreateNodeFromResource(kinds []string, resourceType, namespace, name string, properties map[string]any) BloodHoundNode {
	nodeID := GenerateNodeID(kinds[0], resourceType, namespace, name)

	if properties == nil {
		properties = make(map[string]any)
	}

	properties["name"] = name
	properties["resource_type"] = resourceType
	properties["objectid"] = nodeID

	if namespace != "" {
		properties["namespace"] = namespace
	}

	return BloodHoundNode{
		ID:         nodeID,
		Kinds:      kinds,
		Properties: properties,
	}
}

func CreateNodeWithConfig(config ResourceConfig, resourceType, namespace, name string, resource any) (BloodHoundNode, error) {
	kinds := []string{config.PrimaryKind}
	kinds = append(kinds, config.SecondaryKinds...)

	properties, err := config.PropertyMapper.MapProperties(resource)
	if err != nil {
		return BloodHoundNode{}, fmt.Errorf("failed to map properties: %w", err)
	}

	// Flatten nested objects to ensure BloodHound schema compliance
	properties = FlattenProperties(properties, "")

	return CreateNodeFromResource(kinds, resourceType, namespace, name, properties), nil
}

func CreateEdge(sourceID, targetID, kind string, properties map[string]any) BloodHoundEdge {
	if properties == nil {
		properties = make(map[string]any)
	}

	return BloodHoundEdge{
		Start: BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   sourceID,
		},
		End: BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   targetID,
		},
		Kind:       SanitizePascalCase(kind),
		Properties: properties,
	}
}

// SanitizePascalCase converts edge kinds to PascalCase without dashes
func SanitizePascalCase(input string) string {
	words := strings.FieldsFunc(input, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]) + strings.ToLower(word[1:]))
		}
	}
	return result.String()
}

// FlattenProperties converts nested objects to primitive key-value pairs
func FlattenProperties(input map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	for key, value := range input {
		flatKey := key
		if prefix != "" {
			flatKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]any:
			// Recursively flatten nested objects
			nested := FlattenProperties(v, flatKey)
			maps.Copy(result, nested)
		case []any:
			// Convert object arrays to primitive arrays or flatten if needed
			if len(v) > 0 {
				if isObjectArray(v) {
					result[flatKey] = convertObjectArrayToPrimitives(v)
				} else {
					result[flatKey] = v
				}
			}
		default:
			result[flatKey] = value
		}
	}

	return result
}

// Helper function to check if array contains objects
func isObjectArray(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	_, isObj := arr[0].(map[string]any)
	return isObj
}

// Helper function to convert object arrays to string arrays
func convertObjectArrayToPrimitives(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if obj, ok := item.(map[string]any); ok {
			// Extract a meaningful string representation
			if name, exists := obj["name"]; exists {
				if nameStr, ok := name.(string); ok {
					result = append(result, nameStr)
				}
			} else if id, exists := obj["id"]; exists {
				if idStr, ok := id.(string); ok {
					result = append(result, idStr)
				}
			}
		}
	}
	return result
}

// ConvertToBloodHoundResult creates a BloodHound-compliant result with metadata
func ConvertToBloodHoundResult(ndjsonData []byte, clusterName string) (*BloodHoundResult, error) {
	resources, err := ParseFromNDJSON(ndjsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NDJSON: %w", err)
	}

	parsed, err := DefaultRegistry.ParseBatch(resources)
	if err != nil {
		return nil, err
	}

	// Process RBAC edges after all resources are parsed
	rbacEdges := ProcessGlobalRBACEdges(resources)
	parsed.Edges = append(parsed.Edges, rbacEdges...)

	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind: "Kubernetes",
		},
		Graph: BloodHoundGraph{
			Nodes: parsed.Nodes,
			Edges: parsed.Edges,
		},
	}, nil
}

// ConcurrentParseProcessor handles large-scale parsing with concurrency
func ConcurrentParseProcessor(resources []ResourceData, workerCount int) (*BloodHoundResult, error) {
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	resourceChan := make(chan ResourceData, len(resources))
	resultChan := make(chan *ParsedResult, len(resources))
	errorChan := make(chan error, len(resources))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for resource := range resourceChan {
				result, err := DefaultRegistry.ParseResource(resource)
				if err != nil {
					errorChan <- err
					return
				}
				resultChan <- result
			}
		}()
	}

	// Send resources to workers
	go func() {
		defer close(resourceChan)
		for _, resource := range resources {
			resourceChan <- resource
		}
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	concurrentResult := NewConcurrentResult()
	for result := range resultChan {
		concurrentResult.AddNodes(result.Nodes...)
		concurrentResult.AddEdges(result.Edges...)
	}

	// Check for errors
	select {
	case err := <-errorChan:
		return nil, err
	default:
	}

	finalGraph := concurrentResult.GetResult()
	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind: "Kubernetes",
		},
		Graph: finalGraph,
	}, nil
}

func ParseFromNDJSON(ndjsonData []byte) ([]ResourceData, error) {
	lines := strings.Split(string(ndjsonData), "\n")
	var resources []ResourceData

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resource ResourceData
		if err := json.Unmarshal([]byte(line), &resource); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", i+1, err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func ConvertToBloodHound(ndjsonData []byte) (*ParsedResult, error) {
	resources, err := ParseFromNDJSON(ndjsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NDJSON: %w", err)
	}

	return DefaultRegistry.ParseBatch(resources)
}

func ConvertToBloodHoundJSON(ndjsonData []byte) ([]byte, error) {
	result, err := ConvertToBloodHound(ndjsonData)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}

// ProcessGlobalRBACEdges creates direct edges for RBAC resources following the pattern:
// ClusterRole -> ServiceAccount/Group/User (via ClusterRoleBinding)
// Role -> ServiceAccount/Group/User (via RoleBinding)
func ProcessGlobalRBACEdges(resources []ResourceData) []BloodHoundEdge {
	var edges []BloodHoundEdge

	// Separate resources by type for easier processing
	var roleBindings []ResourceData
	var clusterRoleBindings []ResourceData

	for _, resource := range resources {
		switch resource.Type {
		case "role_binding":
			roleBindings = append(roleBindings, resource)
		case "cluster_role_binding":
			clusterRoleBindings = append(clusterRoleBindings, resource)
		}
	}

	// Create direct Role -> Subject edges via RoleBindings
	for _, binding := range roleBindings {
		bindingData, err := extractRBACResource(binding.Resource)
		if err != nil {
			continue
		}

		roleRef, exists := bindingData["role_ref"].(map[string]any)
		if !exists {
			if roleRef, exists = bindingData["roleRef"].(map[string]any); !exists {
				continue
			}
		}

		roleKind, _ := roleRef["kind"].(string)
		roleName, _ := roleRef["name"].(string)

		if roleKind == "" || roleName == "" {
			continue
		}

		// Create source node ID based on role type
		var sourceNodeID string

		switch roleKind {
		case "ClusterRole":
			sourceNodeID = GenerateNodeID("ClusterRole", "cluster_role", "", roleName)
		case "Role":
			sourceNodeID = GenerateNodeID("Role", "role", binding.Namespace, roleName)
		default:
			continue
		}

		// Create direct edges from Role/ClusterRole to subjects
		if subjects, exists := bindingData["subjects"].([]any); exists {
			bindingName := ""
			if name, ok := bindingData["name"].(string); ok {
				bindingName = name
			}

			for _, subject := range subjects {
				if subjectMap, ok := subject.(map[string]any); ok {
					subjectEdge := createRoleToSubjectEdge(sourceNodeID, subjectMap, binding.Namespace, roleKind, roleName, bindingName)
					if subjectEdge != nil {
						edges = append(edges, *subjectEdge)
					}
				}
			}
		}
	}

	// Create direct ClusterRole -> Subject edges via ClusterRoleBindings
	for _, binding := range clusterRoleBindings {
		bindingData, err := extractRBACResource(binding.Resource)
		if err != nil {
			continue
		}

		roleRef, exists := bindingData["role_ref"].(map[string]any)
		if !exists {
			if roleRef, exists = bindingData["roleRef"].(map[string]any); !exists {
				continue
			}
		}

		roleKind, _ := roleRef["kind"].(string)
		roleName, _ := roleRef["name"].(string)

		if roleKind == "" || roleName == "" {
			continue
		}

		// Create source node ID - should be ClusterRole
		var sourceNodeID string

		if roleKind == "ClusterRole" {
			sourceNodeID = GenerateNodeID("ClusterRole", "cluster_role", "", roleName)
		} else {
			continue
		}

		// Create direct edges from ClusterRole to subjects
		if subjects, exists := bindingData["subjects"].([]any); exists {
			bindingName := ""
			if name, ok := bindingData["name"].(string); ok {
				bindingName = name
			}

			for _, subject := range subjects {
				if subjectMap, ok := subject.(map[string]any); ok {
					subjectEdge := createRoleToSubjectEdge(sourceNodeID, subjectMap, "", roleKind, roleName, bindingName)
					if subjectEdge != nil {
						edges = append(edges, *subjectEdge)
					}
				}
			}
		}
	}

	return edges
}

// Helper function to extract RBAC resource data
func extractRBACResource(resource any) (map[string]any, error) {
	resourceData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RBAC resource: %w", err)
	}

	var rbacResource map[string]any
	if err := json.Unmarshal(resourceData, &rbacResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RBAC resource: %w", err)
	}

	return rbacResource, nil
}

// Helper function to create role to subject edges
func createRoleToSubjectEdge(roleNodeID string, subject map[string]any, defaultNamespace, roleKind, roleName, bindingName string) *BloodHoundEdge {
	subjectKind, _ := subject["kind"].(string)
	subjectName, _ := subject["name"].(string)

	if subjectKind == "" || subjectName == "" {
		return nil
	}

	var targetNodeID string

	switch subjectKind {
	case "ServiceAccount":
		subjectNamespace, _ := subject["namespace"].(string)
		if subjectNamespace == "" {
			subjectNamespace = defaultNamespace
		}
		if subjectNamespace == "" {
			return nil // ServiceAccount must have a namespace
		}
		targetNodeID = GenerateNodeID("ServiceAccount", "service_account", subjectNamespace, subjectName)
	case "User":
		targetNodeID = GenerateNodeID("User", "user", "", subjectName)
	case "Group":
		targetNodeID = GenerateNodeID("Group", "group", "", subjectName)
	default:
		return nil
	}

	properties := map[string]any{
		"role_kind":    roleKind,
		"role_name":    roleName,
		"binding_name": bindingName,
		"subject_kind": subjectKind,
		"subject_name": subjectName,
	}

	if subjectNamespace, exists := subject["namespace"].(string); exists && subjectNamespace != "" {
		properties["subject_namespace"] = subjectNamespace
	}

	edge := CreateEdge(roleNodeID, targetNodeID, "HasRole", properties)
	return &edge
}

// // Helper function to create binding to subject edges
// func createBindingToSubjectEdge(bindingNodeID string, subject map[string]any, defaultNamespace, bindingType string) *BloodHoundEdge {
// 	subjectKind, _ := subject["kind"].(string)
// 	subjectName, _ := subject["name"].(string)

// 	if subjectKind == "" || subjectName == "" {
// 		return nil
// 	}

// 	var targetNodeID string
// 	var edgeType string

// 	switch subjectKind {
// 	case "ServiceAccount":
// 		subjectNamespace, _ := subject["namespace"].(string)
// 		if subjectNamespace == "" {
// 			subjectNamespace = defaultNamespace
// 		}
// 		if subjectNamespace == "" {
// 			return nil // ServiceAccount must have a namespace
// 		}
// 		targetNodeID = GenerateNodeID("ServiceAccount", "service_account", subjectNamespace, subjectName)
// 		edgeType = bindingType + "ToServiceAccount"
// 	case "User":
// 		targetNodeID = GenerateNodeID("User", "user", "", subjectName)
// 		edgeType = bindingType + "ToUser"
// 	case "Group":
// 		targetNodeID = GenerateNodeID("Group", "group", "", subjectName)
// 		edgeType = bindingType + "ToGroup"
// 	default:
// 		return nil
// 	}

// 	properties := map[string]any{
// 		"subject_kind": subjectKind,
// 		"subject_name": subjectName,
// 	}

// 	if subjectNamespace, exists := subject["namespace"].(string); exists && subjectNamespace != "" {
// 		properties["subject_namespace"] = subjectNamespace
// 	}

// 	edge := CreateEdge(bindingNodeID, targetNodeID, edgeType, properties)
// 	return &edge
// }
