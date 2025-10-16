package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"time"
)

type ResourceHandler interface {
	GetName() string
	IsClusterScoped() bool
	GetDescription() string
	GetSupportedClusterTypes() []utils.ClusterType
	Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error)
}

// CollectFunc defines the signature for resource collection functions
type CollectFunc func(ctx context.Context, c *Collector, namespace string) ([]any, error)

// Handler provides the standard implementation of ResourceHandler
type Handler struct {
	name                  string
	resourceType          string
	description           string
	clusterScoped         bool
	supportedClusterTypes []utils.ClusterType
	collectFunc           CollectFunc
}

func (h *Handler) GetName() string {
	return h.name
}

func (h *Handler) IsClusterScoped() bool {
	return h.clusterScoped
}

func (h *Handler) GetDescription() string {
	return h.description
}

func (h *Handler) GetSupportedClusterTypes() []utils.ClusterType {
	return h.supportedClusterTypes
}

func (h *Handler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	resources, err := h.collectFunc(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(resources))

	for _, resource := range resources {
		batch = append(batch, Resource{
			Type:      h.resourceType,
			Namespace: namespace,
			Resource:  resource,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}

// NewHandler creates a new handler from metadata with validation
func NewHandler(meta HandlerMetadata) ResourceHandler {
	// Basic validation
	if meta.Name == "" {
		panic("handler name cannot be empty")
	}
	if meta.ResourceType == "" {
		panic("handler resource type cannot be empty")
	}
	if meta.CollectFunc == nil {
		panic("handler collect function cannot be nil")
	}
	if len(meta.SupportedClusterTypes) == 0 {
		panic("handler must support at least one cluster type")
	}

	return &Handler{
		name:                  meta.Name,
		resourceType:          meta.ResourceType,
		description:           meta.Description,
		clusterScoped:         meta.ClusterScoped,
		supportedClusterTypes: meta.SupportedClusterTypes,
		collectFunc:           meta.CollectFunc,
	}
}

// HandlerMetadata defines metadata for registering handlers
type HandlerMetadata struct {
	Name                  string
	ResourceType          string
	Description           string
	ClusterScoped         bool
	SupportedClusterTypes []utils.ClusterType
	CollectFunc           CollectFunc
}

// NewHandlerFromMetadata creates a ResourceHandler from metadata
func NewHandlerFromMetadata(meta HandlerMetadata) ResourceHandler {
	return NewHandler(meta)
}
