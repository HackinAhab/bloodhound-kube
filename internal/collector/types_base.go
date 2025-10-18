package collector

import (
	"time"
)

// This file contains base types and common structures used across all collectors.

// CommonResourceMeta contains fields that appear across multiple resource types
type CommonResourceMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"` // Some resources like Nodes don't have namespace
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}
