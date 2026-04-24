package config

import (
	"encoding/json"
	"fmt"
)

// Query represents a single saved query.
type Query struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
}

// QueriesConfig represents the queries configuration file structure.
type QueriesConfig struct {
	Queries []Query `json:"queries"`
}

// IconConfig represents the icon configuration for a custom node type.
type IconConfig struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// NodeTypeConfig represents the configuration for a custom node type.
type NodeTypeConfig struct {
	Icon IconConfig `json:"icon"`
}

// CustomTypesConfig represents the custom types configuration file structure.
type CustomTypesConfig struct {
	CustomTypes map[string]NodeTypeConfig `json:"custom_types"`
}

// MergeQueries merges embedded and user-provided queries.
// User queries take precedence over embedded queries with the same name.
// Returns merged JSON with user queries first, then non-duplicate embedded queries.
func MergeQueries(embedded, userProvided []byte) ([]byte, error) {
	if len(embedded) == 0 && len(userProvided) == 0 {
		return nil, fmt.Errorf("no queries data to merge")
	}

	// Parse embedded queries
	var embeddedConfig QueriesConfig
	if len(embedded) > 0 {
		if err := json.Unmarshal(embedded, &embeddedConfig); err != nil {
			return nil, fmt.Errorf("failed to parse embedded queries: %w", err)
		}
	}

	// Parse user queries
	var userConfig QueriesConfig
	if len(userProvided) > 0 {
		if err := json.Unmarshal(userProvided, &userConfig); err != nil {
			return nil, fmt.Errorf("failed to parse user queries: %w", err)
		}
	}

	// Create map to track query names (for deduplication)
	seen := make(map[string]bool)
	merged := QueriesConfig{
		Queries: make([]Query, 0, len(userConfig.Queries)+len(embeddedConfig.Queries)),
	}

	// Add user queries first (they take precedence)
	for _, query := range userConfig.Queries {
		merged.Queries = append(merged.Queries, query)
		seen[query.Name] = true
	}

	// Add embedded queries that don't conflict with user queries
	for _, query := range embeddedConfig.Queries {
		if !seen[query.Name] {
			merged.Queries = append(merged.Queries, query)
			seen[query.Name] = true
		}
	}

	// Marshal back to JSON
	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged queries: %w", err)
	}

	return result, nil
}

// MergeCustomTypes merges embedded and user-provided custom types.
// User types take precedence over embedded types with the same name.
// Returns merged JSON with all unique types.
func MergeCustomTypes(embedded, userProvided []byte) ([]byte, error) {
	if len(embedded) == 0 && len(userProvided) == 0 {
		return nil, fmt.Errorf("no custom types data to merge")
	}

	// Parse embedded types
	var embeddedConfig CustomTypesConfig
	if len(embedded) > 0 {
		if err := json.Unmarshal(embedded, &embeddedConfig); err != nil {
			return nil, fmt.Errorf("failed to parse embedded custom types: %w", err)
		}
	}
	if embeddedConfig.CustomTypes == nil {
		embeddedConfig.CustomTypes = make(map[string]NodeTypeConfig)
	}

	// Parse user types
	var userConfig CustomTypesConfig
	if len(userProvided) > 0 {
		if err := json.Unmarshal(userProvided, &userConfig); err != nil {
			return nil, fmt.Errorf("failed to parse user custom types: %w", err)
		}
	}
	if userConfig.CustomTypes == nil {
		userConfig.CustomTypes = make(map[string]NodeTypeConfig)
	}

	// Create merged map (start with embedded, then override with user)
	merged := CustomTypesConfig{
		CustomTypes: make(map[string]NodeTypeConfig),
	}

	// Add all embedded types
	for name, config := range embeddedConfig.CustomTypes {
		merged.CustomTypes[name] = config
	}

	// Override with user types (user takes precedence)
	for name, config := range userConfig.CustomTypes {
		merged.CustomTypes[name] = config
	}

	// Marshal back to JSON
	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged custom types: %w", err)
	}

	return result, nil
}
