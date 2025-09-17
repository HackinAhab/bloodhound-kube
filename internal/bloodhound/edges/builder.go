package edges

import (
	"bloodhound-kube/internal/bloodhound"
)

// EdgeBuilder provides utilities for creating BloodHound edges
type EdgeBuilder struct{}

// NewEdgeBuilder creates a new EdgeBuilder instance
func NewEdgeBuilder() *EdgeBuilder {
	return &EdgeBuilder{}
}

// CreateEdge creates a new BloodHound-compliant edge using the updated structure
func (eb *EdgeBuilder) CreateEdge(sourceID, targetID, kind string, properties map[string]any) bloodhound.BloodHoundEdge {
	return bloodhound.CreateEdge(sourceID, targetID, kind, properties)
}

// CreateSimpleEdge creates a simple edge without properties
func (eb *EdgeBuilder) CreateSimpleEdge(sourceID, targetID, kind string) bloodhound.BloodHoundEdge {
	return bloodhound.CreateEdge(sourceID, targetID, kind, nil)
}

// CreateEdgeWithFlattenedProperties creates an edge and ensures properties are flattened
func (eb *EdgeBuilder) CreateEdgeWithFlattenedProperties(sourceID, targetID, kind string, properties map[string]any) bloodhound.BloodHoundEdge {
	if properties == nil {
		properties = make(map[string]any)
	}

	// Properties will be flattened by CreateEdge function
	return bloodhound.CreateEdge(sourceID, targetID, kind, properties)
}
