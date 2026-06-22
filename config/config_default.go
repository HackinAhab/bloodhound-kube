//go:build !embedded

package config

// GetEmbeddedQueries returns nil when not built with embedded tag.
func GetEmbeddedQueries() ([]byte, error) {
	return nil, nil
}

// GetEmbeddedSchema returns nil when not built with embedded tag.
func GetEmbeddedSchema() ([]byte, error) {
	return nil, nil
}

// HasEmbeddedQueries returns false when not built with embedded tag.
func HasEmbeddedQueries() bool {
	return false
}

// HasEmbeddedSchema returns false when not built with embedded tag.
func HasEmbeddedSchema() bool {
	return false
}
