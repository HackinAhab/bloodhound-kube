package collector

import (
	"context"
)

type ResourceHandler interface {
	GetName() string
	IsClusterScoped() bool
	Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error)
}

type BaseHandler struct {
	name          string
	clusterScoped bool
}

func (h *BaseHandler) GetName() string {
	return h.name
}

func (h *BaseHandler) IsClusterScoped() bool {
	return h.clusterScoped
}
