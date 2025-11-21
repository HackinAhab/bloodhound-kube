package relationships

import (
	"fmt"
	"sort"
)

// Local type definitions to avoid import cycle
type BloodHoundNode struct {
	ID         string         `json:"id"`
	Kinds      []string       `json:"kinds"`
	Properties map[string]any `json:"properties,omitempty"`
}

type BloodHoundEdgeRef struct {
	MatchBy string `json:"match_by,omitempty"`
	Value   string `json:"value"`
	Kind    string `json:"kind,omitempty"`
}

type BloodHoundEdge struct {
	Start      BloodHoundEdgeRef `json:"start"`
	End        BloodHoundEdgeRef `json:"end"`
	Kind       string            `json:"kind"`
	Properties map[string]any    `json:"properties,omitempty"`
}

// Engine is the core relationship processing engine that applies rules to create edges
type Engine struct {
	rules    []Rule
	resolver *Resolver
	cache    map[string][]BloodHoundEdge
}

// NewEngine creates a new relationship engine
func NewEngine() *Engine {
	return &Engine{
		rules:    make([]Rule, 0),
		resolver: NewResolver(),
		cache:    make(map[string][]BloodHoundEdge),
	}
}

// LoadRules loads relationship rules from a file
func (e *Engine) LoadRules(filepath string) error {
	rules, err := LoadRulesFromFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	e.rules = rules
	return nil
}

// LoadRulesFromDirectory loads all YAML rule files from a directory
func (e *Engine) LoadRulesFromDirectory(dirPath string) error {
	rules, err := LoadRulesFromDirectory(dirPath)
	if err != nil {
		return fmt.Errorf("failed to load rules from directory: %w", err)
	}

	e.rules = rules
	return nil
}

// LoadRulesWithFallback loads rules from primary directory with fallback to embedded rules only
func (e *Engine) LoadRulesWithFallback(primaryDir string, embeddedRules string) error {
	// Try primary directory first
	if err := e.LoadRulesFromDirectory(primaryDir); err == nil {
		return nil
	}

	// Use embedded rules as fallback
	if embeddedRules != "" {
		return e.LoadRulesFromString(embeddedRules)
	}

	return fmt.Errorf("no valid rules found in %s and no embedded rules available", primaryDir)
}

// LoadAdditionalRulesFile loads rules from an additional file and appends them to existing rules
func (e *Engine) LoadAdditionalRulesFile(filepath string) error {
	rules, err := LoadRulesFromFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to load additional rules from %s: %w", filepath, err)
	}

	// Check for duplicate rule names
	existingNames := make(map[string]bool)
	for _, rule := range e.rules {
		existingNames[rule.Name] = true
	}

	for _, rule := range rules {
		if existingNames[rule.Name] {
			return fmt.Errorf("duplicate rule name '%s' found in additional rules file", rule.Name)
		}
	}

	// Append new rules to existing ones
	e.rules = append(e.rules, rules...)
	return nil
}

// LoadRulesFromString loads relationship rules from a YAML string
func (e *Engine) LoadRulesFromString(yamlContent string) error {
	rules, err := LoadRulesFromString(yamlContent)
	if err != nil {
		return fmt.Errorf("failed to load rules from string: %w", err)
	}

	e.rules = rules
	return nil
}

// SetRules sets the rules directly
func (e *Engine) SetRules(rules []Rule) {
	e.rules = rules
}

// GetRules returns the current rules
func (e *Engine) GetRules() []Rule {
	return e.rules
}

// ApplyRules applies all enabled rules to create edges between the given nodes
func (e *Engine) ApplyRules(nodes []BloodHoundNode) []BloodHoundEdge {
	if len(e.rules) == 0 || len(nodes) == 0 {
		return []BloodHoundEdge{}
	}

	// Index nodes by resource type for efficient lookup
	nodeIndex := e.indexNodesByType(nodes)

	var allEdges []BloodHoundEdge

	// Get enabled rules sorted by priority
	enabledRules := GetEnabledRules(e.rules)
	sortedRules := SortRulesByPriority(enabledRules)

	// Apply each rule
	for _, rule := range sortedRules {
		edges := e.applyRule(rule, nodeIndex)
		allEdges = append(allEdges, edges...)
	}

	return allEdges
}

// applyRule applies a single rule to create edges
func (e *Engine) applyRule(rule Rule, nodeIndex map[string][]BloodHoundNode) []BloodHoundEdge {
	var edges []BloodHoundEdge

	// Check if this is a direct or via-based rule
	if len(rule.ViaType) > 0 {
		edges = e.applyViaRule(rule, nodeIndex)
	} else {
		edges = e.applyDirectRule(rule, nodeIndex)
	}

	return edges
}

// applyDirectRule applies a rule that creates direct edges between source and target
func (e *Engine) applyDirectRule(rule Rule, nodeIndex map[string][]BloodHoundNode) []BloodHoundEdge {
	var edges []BloodHoundEdge

	// Get source nodes
	var sourceNodes []BloodHoundNode
	for _, sourceType := range rule.SourceType {
		if nodes, exists := nodeIndex[sourceType]; exists {
			sourceNodes = append(sourceNodes, nodes...)
		}
	}

	// Get target nodes
	var targetNodes []BloodHoundNode
	for _, targetType := range rule.TargetType {
		if nodes, exists := nodeIndex[targetType]; exists {
			targetNodes = append(targetNodes, nodes...)
		}
	}

	// Try to create edges between each source and target pair
	for _, source := range sourceNodes {
		for _, target := range targetNodes {
			// Evaluate rule conditions
			match, err := e.resolver.EvaluateConditions(rule, &source, &target, nil)
			if err != nil {
				// Log error but continue processing
				continue
			}

			if match {
				edge := e.createEdge(rule, &source, &target)
				edges = append(edges, *edge)
			}
		}
	}

	return edges
}

// applyViaRule applies a rule that creates edges via intermediate nodes
func (e *Engine) applyViaRule(rule Rule, nodeIndex map[string][]BloodHoundNode) []BloodHoundEdge {
	var edges []BloodHoundEdge

	// Get source nodes
	var sourceNodes []BloodHoundNode
	for _, sourceType := range rule.SourceType {
		if nodes, exists := nodeIndex[sourceType]; exists {
			sourceNodes = append(sourceNodes, nodes...)
		}
	}

	// Get target nodes
	var targetNodes []BloodHoundNode
	for _, targetType := range rule.TargetType {
		if nodes, exists := nodeIndex[targetType]; exists {
			targetNodes = append(targetNodes, nodes...)
		}
	}

	// Get via nodes
	var viaNodes []BloodHoundNode
	for _, viaType := range rule.ViaType {
		if nodes, exists := nodeIndex[viaType]; exists {
			viaNodes = append(viaNodes, nodes...)
		}
	}

	// Try to create edges between each source, via, and target combination
	for _, source := range sourceNodes {
		for _, via := range viaNodes {
			for _, target := range targetNodes {
				// Evaluate rule conditions with via node
				match, err := e.resolver.EvaluateConditions(rule, &source, &target, &via)
				if err != nil {
					// Log error but continue processing
					continue
				}

				if match {
					edge := e.createEdge(rule, &source, &target)
					// Add via node information to edge properties
					edge.Properties["via_node_id"] = via.ID
					edge.Properties["via_node_type"] = via.Properties["resource_type"]
					if viaName, exists := via.Properties["name"]; exists {
						edge.Properties["via_node_name"] = viaName
					}
					edges = append(edges, *edge)
				}
			}
		}
	}

	return edges
}

// createEdge creates a BloodHound edge from a rule and source/target nodes
func (e *Engine) createEdge(rule Rule, source, target *BloodHoundNode) *BloodHoundEdge {
	// Create edge without using pools for now to avoid import cycle
	edge := &BloodHoundEdge{
		Properties: make(map[string]any),
	}

	// Set edge endpoints
	edge.Start = BloodHoundEdgeRef{
		MatchBy: "id",
		Value:   source.ID,
	}
	edge.End = BloodHoundEdgeRef{
		MatchBy: "id",
		Value:   target.ID,
	}

	// Set edge kind (sanitized) - simple Pascal case conversion
	edge.Kind = sanitizePascalCase(rule.EdgeType)

	// Set edge properties
	edge.Properties["rule_name"] = rule.Name
	edge.Properties["rule_priority"] = rule.Priority
	edge.Properties["source_type"] = source.Properties["resource_type"]
	edge.Properties["target_type"] = target.Properties["resource_type"]

	if sourceName, exists := source.Properties["name"]; exists {
		edge.Properties["source_name"] = sourceName
	}
	if targetName, exists := target.Properties["name"]; exists {
		edge.Properties["target_name"] = targetName
	}

	if sourceNamespace, exists := source.Properties["namespace"]; exists {
		edge.Properties["source_namespace"] = sourceNamespace
	}
	if targetNamespace, exists := target.Properties["namespace"]; exists {
		edge.Properties["target_namespace"] = targetNamespace
	}

	return edge
}

// sanitizePascalCase converts a string to PascalCase for edge types
func sanitizePascalCase(s string) string {
	if s == "" {
		return s
	}

	// Simple Pascal case conversion
	result := ""
	capitalizeNext := true

	for _, char := range s {
		if char == '_' || char == '-' || char == ' ' {
			capitalizeNext = true
			continue
		}

		if capitalizeNext {
			result += string(char - 32) // Convert to uppercase
			capitalizeNext = false
		} else {
			result += string(char)
		}
	}

	return result
}

// indexNodesByType creates an index of nodes by their resource type for efficient lookup
func (e *Engine) indexNodesByType(nodes []BloodHoundNode) map[string][]BloodHoundNode {
	index := make(map[string][]BloodHoundNode)

	for _, node := range nodes {
		if resourceType, exists := node.Properties["resource_type"]; exists {
			if typeStr, ok := resourceType.(string); ok {
				index[typeStr] = append(index[typeStr], node)
			}
		}
	}

	return index
}

// GetStats returns statistics about the relationship engine
func (e *Engine) GetStats() EngineStats {
	enabledRules := GetEnabledRules(e.rules)

	return EngineStats{
		TotalRules:    len(e.rules),
		EnabledRules:  len(enabledRules),
		DisabledRules: len(e.rules) - len(enabledRules),
		CachedResults: len(e.cache),
	}
}

// EngineStats provides statistics about the relationship engine
type EngineStats struct {
	TotalRules    int
	EnabledRules  int
	DisabledRules int
	CachedResults int
}

// ClearCache clears the engine's cache
func (e *Engine) ClearCache() {
	for k := range e.cache {
		delete(e.cache, k)
	}
	e.resolver.ClearCache()
}

// ValidateRules validates all loaded rules
func (e *Engine) ValidateRules() []error {
	var errors []error

	for _, rule := range e.rules {
		if err := ValidateRule(rule); err != nil {
			errors = append(errors, fmt.Errorf("rule '%s': %w", rule.Name, err))
		}
	}

	return errors
}

// FilterNodesByType returns nodes of the specified resource type
func FilterNodesByType(nodes []BloodHoundNode, resourceType string) []BloodHoundNode {
	var filtered []BloodHoundNode

	for _, node := range nodes {
		if nodeType, exists := node.Properties["resource_type"]; exists {
			if typeStr, ok := nodeType.(string); ok && typeStr == resourceType {
				filtered = append(filtered, node)
			}
		}
	}

	return filtered
}

// SortEdgesByKind sorts edges by their kind for consistent output
func SortEdgesByKind(edges []BloodHoundEdge) {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Kind < edges[j].Kind
	})
}

// DeduplicateEdges removes duplicate edges based on source, target, and kind
func DeduplicateEdges(edges []BloodHoundEdge) []BloodHoundEdge {
	seen := make(map[string]bool)
	var unique []BloodHoundEdge

	for _, edge := range edges {
		key := fmt.Sprintf("%s->%s:%s", edge.Start.Value, edge.End.Value, edge.Kind)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, edge)
		}
	}

	return unique
}
