package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading and validating configuration files
type Loader struct {
	configDir string
}

// NewLoader creates a new configuration loader
func NewLoader(configDir string) *Loader {
	if configDir == "" {
		configDir = "config" // Default config directory
	}
	return &Loader{
		configDir: configDir,
	}
}

// LoadCollections loads and validates a collections configuration file
func (l *Loader) LoadCollections(filename string) (*CollectionsConfig, error) {
	// Build full path
	path := filepath.Join(l.configDir, filename)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read collections config %s: %w", path, err)
	}

	// Parse YAML
	var config CollectionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse collections config %s: %w", path, err)
	}

	// Set defaults
	config.SetDefaults()

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("collections config validation failed for %s: %w", path, err)
	}

	return &config, nil
}

// LoadParsers loads and validates a parsers configuration file
func (l *Loader) LoadParsers(filename string) (*ParsersConfig, error) {
	// Build full path
	path := filepath.Join(l.configDir, filename)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read parsers config %s: %w", path, err)
	}

	// Parse YAML
	var config ParsersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse parsers config %s: %w", path, err)
	}

	// Set defaults
	config.SetDefaults()

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("parsers config validation failed for %s: %w", path, err)
	}

	return &config, nil
}

// LoadCollectionsFromBytes loads collections config from byte slice
func LoadCollectionsFromBytes(data []byte) (*CollectionsConfig, error) {
	var config CollectionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse collections config: %w", err)
	}

	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("collections config validation failed: %w", err)
	}

	return &config, nil
}

// LoadParsersFromBytes loads parsers config from byte slice
func LoadParsersFromBytes(data []byte) (*ParsersConfig, error) {
	var config ParsersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse parsers config: %w", err)
	}

	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("parsers config validation failed: %w", err)
	}

	return &config, nil
}

// ConfigExists checks if a configuration file exists
func (l *Loader) ConfigExists(filename string) bool {
	path := filepath.Join(l.configDir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// GetConfigPath returns the full path to a config file
func (l *Loader) GetConfigPath(filename string) string {
	return filepath.Join(l.configDir, filename)
}

// ValidateCollectionsFile validates a collections config file without loading it fully
func ValidateCollectionsFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var config CollectionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ValidateParsersFile validates a parsers config file without loading it fully
func ValidateParsersFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var config ParsersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// LoadCollectionsWithFallback tries to load from primary path, falls back to secondary
func LoadCollectionsWithFallback(primaryPath, fallbackPath string) (*CollectionsConfig, error) {
	// Try primary path first
	if data, err := os.ReadFile(primaryPath); err == nil {
		config, err := LoadCollectionsFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load primary config %s: %w", primaryPath, err)
		}
		return config, nil
	}

	// Try fallback path
	if data, err := os.ReadFile(fallbackPath); err == nil {
		config, err := LoadCollectionsFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load fallback config %s: %w", fallbackPath, err)
		}
		return config, nil
	}

	return nil, fmt.Errorf("no valid collections config found (tried: %s, %s)", primaryPath, fallbackPath)
}

// LoadParsersWithFallback tries to load from primary path, falls back to secondary
func LoadParsersWithFallback(primaryPath, fallbackPath string) (*ParsersConfig, error) {
	// Try primary path first
	if data, err := os.ReadFile(primaryPath); err == nil {
		config, err := LoadParsersFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load primary config %s: %w", primaryPath, err)
		}
		return config, nil
	}

	// Try fallback path
	if data, err := os.ReadFile(fallbackPath); err == nil {
		config, err := LoadParsersFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load fallback config %s: %w", fallbackPath, err)
		}
		return config, nil
	}

	return nil, fmt.Errorf("no valid parsers config found (tried: %s, %s)", primaryPath, fallbackPath)
}
