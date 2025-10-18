package collector

// This file contains type definitions for Kubernetes storage resources:
// ConfigMaps, Secrets, and related structures.

type ConfigMap struct {
	CommonResourceMeta
	DataKeys       []string          `json:"data_keys"`
	Data           map[string]string `json:"data,omitempty"`
	BinaryDataKeys []string          `json:"binary_data_keys,omitempty"`
	BinaryData     map[string][]byte `json:"binary_data,omitempty"`
}

type Secret struct {
	CommonResourceMeta
	Type     string            `json:"type"`
	DataKeys []string          `json:"data_keys"`
	Data     map[string]string `json:"data,omitempty"` // Made optional since data might be redacted
}
