//go:build embedded

package config

import _ "embed"

//go:embed custom_queries.json
var embeddedQueriesJSON []byte

//go:embed custom_types.json
var embeddedTypesJSON []byte

// GetEmbeddedQueries returns the embedded queries JSON data.
// Returns nil if no embedded data is available.
func GetEmbeddedQueries() ([]byte, error) {
	if len(embeddedQueriesJSON) == 0 {
		return nil, nil
	}
	return embeddedQueriesJSON, nil
}

// GetEmbeddedTypes returns the embedded custom types JSON data.
// Returns nil if no embedded data is available.
func GetEmbeddedTypes() ([]byte, error) {
	if len(embeddedTypesJSON) == 0 {
		return nil, nil
	}
	return embeddedTypesJSON, nil
}

// HasEmbeddedQueries returns true if embedded queries are available.
func HasEmbeddedQueries() bool {
	return len(embeddedQueriesJSON) > 0
}

// HasEmbeddedTypes returns true if embedded custom types are available.
func HasEmbeddedTypes() bool {
	return len(embeddedTypesJSON) > 0
}
