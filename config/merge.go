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

// SchemaInfo represents the schema metadata block.
type SchemaInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Version     string `json:"version"`
	Namespace   string `json:"namespace"`
}

// NodeKind represents a node kind definition in the schema.
type NodeKind struct {
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Description   string `json:"description"`
	IsDisplayKind bool   `json:"is_display_kind"`
	Icon          string `json:"icon"`
	Color         string `json:"color"`
}

// RelationshipKind represents a relationship kind definition in the schema.
type RelationshipKind struct {
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Description   string `json:"description"`
	IsTraversable bool   `json:"is_traversable"`
}

// CustomTypesConfig represents the custom types configuration file structure.
type CustomTypesConfig struct {
	Schema            *SchemaInfo        `json:"schema,omitempty"`
	NodeKinds         []NodeKind         `json:"node_kinds"`
	RelationshipKinds []RelationshipKind `json:"relationship_kinds"`
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

// MergeSchema merges embedded and user-provided custom types.
// User types take precedence over embedded types with the same name.
// Returns merged JSON with all unique types.
func MergeSchema(embedded, userProvided []byte) ([]byte, error) {
	if len(embedded) == 0 && len(userProvided) == 0 {
		return nil, fmt.Errorf("no custom types data to merge")
	}

	var embeddedConfig CustomTypesConfig
	if len(embedded) > 0 {
		if err := json.Unmarshal(embedded, &embeddedConfig); err != nil {
			return nil, fmt.Errorf("failed to parse embedded custom types: %w", err)
		}
	}

	var userConfig CustomTypesConfig
	if len(userProvided) > 0 {
		if err := json.Unmarshal(userProvided, &userConfig); err != nil {
			return nil, fmt.Errorf("failed to parse user custom types: %w", err)
		}
	}

	merged := CustomTypesConfig{
		Schema: embeddedConfig.Schema,
	}
	if userConfig.Schema != nil {
		merged.Schema = userConfig.Schema
	}

	merged.NodeKinds = mergeByName(embeddedConfig.NodeKinds, userConfig.NodeKinds)
	merged.RelationshipKinds = mergeRelByName(embeddedConfig.RelationshipKinds, userConfig.RelationshipKinds)

	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged custom types: %w", err)
	}

	return result, nil
}

func mergeByName(base, override []NodeKind) []NodeKind {
	seen := make(map[string]int, len(base)+len(override))
	merged := make([]NodeKind, 0, len(base)+len(override))

	for _, nk := range base {
		seen[nk.Name] = len(merged)
		merged = append(merged, nk)
	}
	for _, nk := range override {
		if idx, ok := seen[nk.Name]; ok {
			merged[idx] = nk
		} else {
			seen[nk.Name] = len(merged)
			merged = append(merged, nk)
		}
	}
	return merged
}

func mergeRelByName(base, override []RelationshipKind) []RelationshipKind {
	seen := make(map[string]int, len(base)+len(override))
	merged := make([]RelationshipKind, 0, len(base)+len(override))

	for _, rk := range base {
		seen[rk.Name] = len(merged)
		merged = append(merged, rk)
	}
	for _, rk := range override {
		if idx, ok := seen[rk.Name]; ok {
			merged[idx] = rk
		} else {
			seen[rk.Name] = len(merged)
			merged = append(merged, rk)
		}
	}
	return merged
}
