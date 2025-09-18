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
		wg.Go(func() {
			for resource := range resourceChan {
				result, err := DefaultRegistry.ParseResource(resource)
				if err != nil {
					errorChan <- err
					return
				}
				resultChan <- result
			}
		})
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
