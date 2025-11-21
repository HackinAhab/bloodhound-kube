package relationships

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Resolver evaluates rule conditions against BloodHound nodes
type Resolver struct {
	cache map[string]bool // Cache for expensive evaluations
}

// NewResolver creates a new condition resolver
func NewResolver() *Resolver {
	return &Resolver{
		cache: make(map[string]bool),
	}
}

// EvaluateConditions evaluates all conditions for a rule against given nodes
func (r *Resolver) EvaluateConditions(rule Rule, source, target *BloodHoundNode, via *BloodHoundNode) (bool, error) {
	for _, condition := range rule.Conditions {
		result, err := r.EvaluateCondition(condition, source, target, via)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
		}
		if !result {
			return false, nil // All conditions must be true
		}
	}
	return true, nil
}

// EvaluateCondition evaluates a single condition string
func (r *Resolver) EvaluateCondition(condition string, source, target *BloodHoundNode, via *BloodHoundNode) (bool, error) {
	// Parse condition into components
	expr, err := parseExpression(condition)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition: %w", err)
	}

	return r.evaluateExpression(expr, source, target, via)
}

// Expression represents a parsed condition expression
type Expression struct {
	Left     string
	Operator string
	Right    string
}

// parseExpression parses a condition string into an Expression
func parseExpression(condition string) (*Expression, error) {
	condition = strings.TrimSpace(condition)

	// Support common operators
	operators := []string{"==", "!=", "in", "subset_of", "contains"}

	for _, op := range operators {
		if idx := strings.Index(condition, " "+op+" "); idx != -1 {
			left := strings.TrimSpace(condition[:idx])
			right := strings.TrimSpace(condition[idx+len(op)+2:])
			return &Expression{
				Left:     left,
				Operator: op,
				Right:    right,
			}, nil
		}
	}

	return nil, fmt.Errorf("unsupported condition format: %s", condition)
}

// evaluateExpression evaluates a parsed expression
func (r *Resolver) evaluateExpression(expr *Expression, source, target *BloodHoundNode, via *BloodHoundNode) (bool, error) {
	leftValue, err := r.resolveValue(expr.Left, source, target, via)
	if err != nil {
		return false, fmt.Errorf("failed to resolve left value '%s': %w", expr.Left, err)
	}

	rightValue, err := r.resolveValue(expr.Right, source, target, via)
	if err != nil {
		return false, fmt.Errorf("failed to resolve right value '%s': %w", expr.Right, err)
	}

	switch expr.Operator {
	case "==":
		return r.equals(leftValue, rightValue), nil
	case "!=":
		return !r.equals(leftValue, rightValue), nil
	case "in":
		return r.contains(rightValue, leftValue), nil
	case "contains":
		return r.contains(leftValue, rightValue), nil
	case "subset_of":
		return r.subsetOf(leftValue, rightValue), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}

// resolveValue resolves a value reference to its actual value
func (r *Resolver) resolveValue(valueRef string, source, target *BloodHoundNode, via *BloodHoundNode) (interface{}, error) {
	// Handle string literals (quoted strings)
	if strings.HasPrefix(valueRef, "'") && strings.HasSuffix(valueRef, "'") {
		return valueRef[1 : len(valueRef)-1], nil
	}
	if strings.HasPrefix(valueRef, "\"") && strings.HasSuffix(valueRef, "\"") {
		return valueRef[1 : len(valueRef)-1], nil
	}

	// Handle numeric literals
	if num, err := strconv.ParseFloat(valueRef, 64); err == nil {
		return num, nil
	}

	// Handle boolean literals
	if valueRef == "true" {
		return true, nil
	}
	if valueRef == "false" {
		return false, nil
	}

	// Handle node property references
	if strings.HasPrefix(valueRef, "source.") {
		if source == nil {
			return nil, fmt.Errorf("source node is nil")
		}
		return r.getNodeProperty(source, valueRef[7:])
	}

	if strings.HasPrefix(valueRef, "target.") {
		if target == nil {
			return nil, fmt.Errorf("target node is nil")
		}
		return r.getNodeProperty(target, valueRef[7:])
	}

	if strings.HasPrefix(valueRef, "via.") {
		if via == nil {
			return nil, fmt.Errorf("via node is nil")
		}
		return r.getNodeProperty(via, valueRef[4:])
	}

	return nil, fmt.Errorf("unknown value reference: %s", valueRef)
}

// getNodeProperty extracts a property value from a node using dot notation
func (r *Resolver) getNodeProperty(node *BloodHoundNode, propertyPath string) (interface{}, error) {
	// Handle special node properties
	switch propertyPath {
	case "id":
		return node.ID, nil
	case "kinds":
		return node.Kinds, nil
	case "name":
		if name, exists := node.Properties["name"]; exists {
			return name, nil
		}
		return nil, fmt.Errorf("node does not have a name property")
	case "namespace":
		if namespace, exists := node.Properties["namespace"]; exists {
			return namespace, nil
		}
		return "", nil // Namespace can be empty for cluster-scoped resources
	case "resource_type":
		if resourceType, exists := node.Properties["resource_type"]; exists {
			return resourceType, nil
		}
		return nil, fmt.Errorf("node does not have a resource_type property")
	}

	// Handle nested property access
	return r.getNestedProperty(node.Properties, propertyPath)
}

// getNestedProperty retrieves a nested property using dot notation
func (r *Resolver) getNestedProperty(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		// Handle array indexing like "subjects[*].name"
		if strings.Contains(part, "[") {
			return r.handleArrayAccess(current, part, strings.Join(parts[i+1:], "."))
		}

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return nil, nil // Property doesn't exist
			}
		default:
			return nil, fmt.Errorf("cannot access property '%s' on non-object type", part)
		}
	}

	return current, nil
}

// handleArrayAccess handles array access patterns like "subjects[*].name"
func (r *Resolver) handleArrayAccess(data interface{}, arrayPart string, remainingPath string) (interface{}, error) {
	// Parse array part (e.g., "subjects[*]" -> "subjects", "*")
	bracketStart := strings.Index(arrayPart, "[")
	if bracketStart == -1 {
		return nil, fmt.Errorf("invalid array access syntax: %s", arrayPart)
	}

	arrayName := arrayPart[:bracketStart]
	indexPart := arrayPart[bracketStart+1 : len(arrayPart)-1] // Remove [ and ]

	// Get the array
	var arrayData interface{}
	switch v := data.(type) {
	case map[string]interface{}:
		arrayData = v[arrayName]
	default:
		return nil, fmt.Errorf("cannot access array '%s' on non-object type", arrayName)
	}

	if arrayData == nil {
		return nil, nil
	}

	// Convert to slice
	arrayValue := reflect.ValueOf(arrayData)
	if arrayValue.Kind() != reflect.Slice && arrayValue.Kind() != reflect.Array {
		return nil, fmt.Errorf("property '%s' is not an array", arrayName)
	}

	// Handle different index types
	if indexPart == "*" {
		// Wildcard - collect all values
		var results []interface{}
		for i := 0; i < arrayValue.Len(); i++ {
			item := arrayValue.Index(i).Interface()
			if remainingPath == "" {
				results = append(results, item)
			} else {
				if val, err := r.getNestedProperty(item, remainingPath); err == nil && val != nil {
					results = append(results, val)
				}
			}
		}
		return results, nil
	} else {
		// Specific index
		index, err := strconv.Atoi(indexPart)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %s", indexPart)
		}
		if index < 0 || index >= arrayValue.Len() {
			return nil, nil // Index out of bounds
		}

		item := arrayValue.Index(index).Interface()
		if remainingPath == "" {
			return item, nil
		}
		return r.getNestedProperty(item, remainingPath)
	}
}

// equals compares two values for equality
func (r *Resolver) equals(left, right interface{}) bool {
	if left == nil || right == nil {
		return left == right
	}

	// Convert to strings for comparison if types differ
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return leftStr == rightStr
}

// contains checks if a collection contains a value
func (r *Resolver) contains(collection, value interface{}) bool {
	if collection == nil {
		return false
	}

	// Handle string arrays
	if arr, ok := collection.([]interface{}); ok {
		valueStr := fmt.Sprintf("%v", value)
		for _, item := range arr {
			if fmt.Sprintf("%v", item) == valueStr {
				return true
			}
		}
		return false
	}

	// Handle string slices
	if arr, ok := collection.([]string); ok {
		valueStr := fmt.Sprintf("%v", value)
		for _, item := range arr {
			if item == valueStr {
				return true
			}
		}
		return false
	}

	// Handle reflection-based arrays
	collectionValue := reflect.ValueOf(collection)
	if collectionValue.Kind() == reflect.Slice || collectionValue.Kind() == reflect.Array {
		valueStr := fmt.Sprintf("%v", value)
		for i := 0; i < collectionValue.Len(); i++ {
			item := collectionValue.Index(i).Interface()
			if fmt.Sprintf("%v", item) == valueStr {
				return true
			}
		}
	}

	return false
}

// subsetOf checks if left is a subset of right (for label matching)
func (r *Resolver) subsetOf(left, right interface{}) bool {
	// Handle map subset checking (common for label selectors)
	leftMap, leftIsMap := left.(map[string]interface{})
	rightMap, rightIsMap := right.(map[string]interface{})

	if leftIsMap && rightIsMap {
		for key, leftValue := range leftMap {
			rightValue, exists := rightMap[key]
			if !exists || !r.equals(leftValue, rightValue) {
				return false
			}
		}
		return true
	}

	// Handle string map versions
	leftStrMap, leftIsStrMap := left.(map[string]string)
	rightStrMap, rightIsStrMap := right.(map[string]string)

	if leftIsStrMap && rightIsStrMap {
		for key, leftValue := range leftStrMap {
			rightValue, exists := rightStrMap[key]
			if !exists || leftValue != rightValue {
				return false
			}
		}
		return true
	}

	// Handle mixed map types
	if leftIsStrMap && rightIsMap {
		for key, leftValue := range leftStrMap {
			rightValue, exists := rightMap[key]
			if !exists || !r.equals(leftValue, rightValue) {
				return false
			}
		}
		return true
	}

	return false
}

// ClearCache clears the resolver's cache
func (r *Resolver) ClearCache() {
	for k := range r.cache {
		delete(r.cache, k)
	}
}
