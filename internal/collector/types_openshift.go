package collector

import (
	"time"
)

// This file contains type definitions for OpenShift-specific resources:
// Routes, Projects, Images, and related structures.

type Route struct {
	CommonResourceMeta
	Host    string    `json:"host"`
	Path    string    `json:"path,omitempty"`
	Service string    `json:"service"`
	Port    string    `json:"port,omitempty"`
	TLS     *RouteTLS `json:"tls,omitempty"`
}

type RouteTLS struct {
	Termination string `json:"termination"`
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`
}

type Project struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
}

type Image struct {
	Name                 string            `json:"name"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	DockerImageReference string            `json:"docker_image_reference"`
	DockerImageManifest  string            `json:"docker_image_manifest,omitempty"`
	DockerImageMetadata  string            `json:"docker_image_metadata,omitempty"`
}
