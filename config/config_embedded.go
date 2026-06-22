//go:build embedded

package config

import _ "embed"

//go:embed custom_queries.json
var embeddedQueriesJSON []byte

//go:embed schema.json
var embeddedSchemaJSON []byte

// GetEmbeddedQueries returns the embedded queries JSON data.
// Returns nil if no embedded data is available.
func GetEmbeddedQueries() ([]byte, error) {
	if len(embeddedQueriesJSON) == 0 {
		return nil, nil
	}
	return embeddedQueriesJSON, nil
}

// GetEmbeddedSchema returns the embedded schema JSON data.
// Returns nil if no embedded data is available.
func GetEmbeddedSchema() ([]byte, error) {
	if len(embeddedSchemaJSON) == 0 {
		return nil, nil
	}
	return embeddedSchemaJSON, nil
}

// HasEmbeddedQueries returns true if embedded queries are available.
func HasEmbeddedQueries() bool {
	return len(embeddedQueriesJSON) > 0
}

// HasEmbeddedSchema returns true if embedded schema is available.
func HasEmbeddedSchema() bool {
	return len(embeddedSchemaJSON) > 0
}
