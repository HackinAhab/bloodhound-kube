//go:build !embedded

package config

// GetEmbeddedQueries returns nil when not built with embedded tag.
func GetEmbeddedQueries() ([]byte, error) {
	return nil, nil
}

// GetEmbeddedTypes returns nil when not built with embedded tag.
func GetEmbeddedTypes() ([]byte, error) {
	return nil, nil
}

// HasEmbeddedQueries returns false when not built with embedded tag.
func HasEmbeddedQueries() bool {
	return false
}

// HasEmbeddedTypes returns false when not built with embedded tag.
func HasEmbeddedTypes() bool {
	return false
}
