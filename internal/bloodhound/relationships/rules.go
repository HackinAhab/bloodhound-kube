package relationships

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule represents a relationship rule that defines how edges are created between nodes
type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	SourceType  []string `yaml:"source_type"`
	TargetType  []string `yaml:"target_type"`
	ViaType     []string `yaml:"via_type,omitempty"`
	EdgeType    string   `yaml:"edge_type"`
	Conditions  []string `yaml:"conditions"`
	Priority    int      `yaml:"priority"`
	Enabled     bool     `yaml:"enabled"`
}

// RuleSet contains a collection of relationship rules
type RuleSet struct {
	Version       string `yaml:"version"`
	Relationships []Rule `yaml:"relationships"`
}

// LoadRulesFromFile loads relationship rules from a YAML file
func LoadRulesFromFile(filepath string) ([]Rule, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", filepath, err)
	}

	var ruleSet RuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return nil, fmt.Errorf("failed to parse rules file %s: %w", filepath, err)
	}

	// Validate all rules
	var validRules []Rule
	for _, rule := range ruleSet.Relationships {
		if err := ValidateRule(rule); err != nil {
			return nil, fmt.Errorf("invalid rule %s: %w", rule.Name, err)
		}
		validRules = append(validRules, rule)
	}

	return validRules, nil
}

// LoadRulesFromString loads relationship rules from a YAML string
func LoadRulesFromString(yamlContent string) ([]Rule, error) {
	var ruleSet RuleSet
	if err := yaml.Unmarshal([]byte(yamlContent), &ruleSet); err != nil {
		return nil, fmt.Errorf("failed to parse rules YAML: %w", err)
	}

	// Validate all rules
	var validRules []Rule
	for _, rule := range ruleSet.Relationships {
		if err := ValidateRule(rule); err != nil {
			return nil, fmt.Errorf("invalid rule %s: %w", rule.Name, err)
		}
		validRules = append(validRules, rule)
	}

	return validRules, nil
}

// ValidateRule validates a relationship rule for correctness
func ValidateRule(rule Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if len(rule.SourceType) == 0 {
		return fmt.Errorf("rule must have at least one source type")
	}

	if len(rule.TargetType) == 0 {
		return fmt.Errorf("rule must have at least one target type")
	}

	if rule.EdgeType == "" {
		return fmt.Errorf("rule must have an edge type")
	}

	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}

	// Validate that via_type is only used when there are via conditions
	if len(rule.ViaType) > 0 {
		hasViaCondition := false
		for _, condition := range rule.Conditions {
			if containsViaReference(condition) {
				hasViaCondition = true
				break
			}
		}
		if !hasViaCondition {
			return fmt.Errorf("rule specifies via_type but no conditions reference via nodes")
		}
	}

	return nil
}

// containsViaReference checks if a condition string references via nodes
func containsViaReference(condition string) bool {
	// Simple check for "via." in condition
	return len(condition) > 4 && condition[:4] == "via."
}

// FilterRulesByType returns rules that match the given source and target types
func FilterRulesByType(rules []Rule, sourceType, targetType string) []Rule {
	var matchingRules []Rule

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Check if source type matches
		sourceMatches := false
		for _, st := range rule.SourceType {
			if st == sourceType {
				sourceMatches = true
				break
			}
		}

		// Check if target type matches
		targetMatches := false
		for _, tt := range rule.TargetType {
			if tt == targetType {
				targetMatches = true
				break
			}
		}

		if sourceMatches && targetMatches {
			matchingRules = append(matchingRules, rule)
		}
	}

	return matchingRules
}

// SortRulesByPriority sorts rules by priority (higher priority first)
func SortRulesByPriority(rules []Rule) []Rule {
	// Create a copy to avoid modifying the original slice
	sorted := make([]Rule, len(rules))
	copy(sorted, rules)

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority < sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// GetRuleByName finds a rule by its name
func GetRuleByName(rules []Rule, name string) (*Rule, bool) {
	for _, rule := range rules {
		if rule.Name == name {
			return &rule, true
		}
	}
	return nil, false
}

// GetEnabledRules filters rules to return only enabled ones
func GetEnabledRules(rules []Rule) []Rule {
	var enabledRules []Rule
	for _, rule := range rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}
	return enabledRules
}

// RuleStats provides statistics about a rule set
type RuleStats struct {
	TotalRules    int
	EnabledRules  int
	DisabledRules int
	RulesByType   map[string]int
}

// GetRuleStats returns statistics about the given rules
func GetRuleStats(rules []Rule) RuleStats {
	stats := RuleStats{
		TotalRules:  len(rules),
		RulesByType: make(map[string]int),
	}

	for _, rule := range rules {
		if rule.Enabled {
			stats.EnabledRules++
		} else {
			stats.DisabledRules++
		}

		// Count by source types
		for _, sourceType := range rule.SourceType {
			stats.RulesByType[sourceType]++
		}
	}

	return stats
}

// LoadRulesFromDirectory loads all .yaml files from a directory and combines their rules
func LoadRulesFromDirectory(dirPath string) ([]Rule, error) {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("rules directory does not exist: %s", dirPath)
	}

	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory %s: %w", dirPath, err)
	}

	var allRules []Rule
	var loadedFiles []string

	// Process each .yaml or .yml file
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		fileName := entry.Name()
		if !strings.HasSuffix(strings.ToLower(fileName), ".yaml") && !strings.HasSuffix(strings.ToLower(fileName), ".yml") {
			continue // Skip non-YAML files
		}

		filePath := filepath.Join(dirPath, fileName)
		rules, err := LoadRulesFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load rules from %s: %w", filePath, err)
		}

		allRules = append(allRules, rules...)
		loadedFiles = append(loadedFiles, fileName)
	}

	if len(loadedFiles) == 0 {
		return nil, fmt.Errorf("no YAML rule files found in directory: %s", dirPath)
	}

	// Check for duplicate rule names across files
	ruleNames := make(map[string]string) // rule name -> file name
	for _, rule := range allRules {
		if existingFile, exists := ruleNames[rule.Name]; exists {
			return nil, fmt.Errorf("duplicate rule name '%s' found in multiple files: %s and %s",
				rule.Name, existingFile, getFileForRule(rule, loadedFiles, allRules))
		}
		ruleNames[rule.Name] = getFileForRule(rule, loadedFiles, allRules)
	}

	return allRules, nil
}

// getFileForRule is a helper function to determine which file a rule came from (for error reporting)
func getFileForRule(targetRule Rule, loadedFiles []string, allRules []Rule) string {
	// This is a simple implementation - in practice you might want to track this more precisely
	// For now, we'll return the first file name as a fallback
	if len(loadedFiles) > 0 {
		return loadedFiles[0]
	}
	return "unknown"
}

// LoadRulesFromDirectoryWithFallback tries to load rules from a directory with fallback options
func LoadRulesFromDirectoryWithFallback(primaryDir, fallbackDir string, embeddedRules string) ([]Rule, error) {
	// Try primary directory first
	if rules, err := LoadRulesFromDirectory(primaryDir); err == nil {
		return rules, nil
	}

	// Try fallback directory
	if fallbackDir != "" {
		if rules, err := LoadRulesFromDirectory(fallbackDir); err == nil {
			return rules, nil
		}
	}

	// Use embedded rules as final fallback
	if embeddedRules != "" {
		return LoadRulesFromString(embeddedRules)
	}

	return nil, fmt.Errorf("failed to load rules from %s, %s, or embedded rules", primaryDir, fallbackDir)
}
