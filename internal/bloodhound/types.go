package bloodhound

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BloodHoundObject represents a unified object structure for both nodes and edges
type BloodHoundObject struct {
	ObjectID   string                 `json:"ObjectID"`
	Properties map[string]interface{} `json:"Properties"`
	Labels     []string               `json:"Labels"`
	Type       string                 `json:"Type"` // "Node" or "Edge"
}

// ObjectBuilder interface for creating BloodHound objects from Kubernetes resources
type ObjectBuilder interface {
	Build(ctx context.Context, obj *unstructured.Unstructured) ([]*BloodHoundObject, error)
	GetSupportedKinds() []string
}

// BuilderRegistry manages object builders for different Kubernetes resource types
type BuilderRegistry struct {
	builders map[string]ObjectBuilder
}

func NewBuilderRegistry() *BuilderRegistry {
	return &BuilderRegistry{
		builders: make(map[string]ObjectBuilder),
	}
}

func (r *BuilderRegistry) RegisterBuilder(kind string, builder ObjectBuilder) {
	r.builders[kind] = builder
}

func (r *BuilderRegistry) GetBuilder(kind string) (ObjectBuilder, error) {
	builder, exists := r.builders[kind]
	if !exists {
		return nil, fmt.Errorf("no builder registered for kind: %s", kind)
	}
	return builder, nil
}

func (r *BuilderRegistry) GetAllBuilders() map[string]ObjectBuilder {
	return r.builders
}

// Thread-safe result accumulator for concurrent processing
type ConcurrentResult struct {
	mu      sync.RWMutex
	objects []*BloodHoundObject
}

func NewConcurrentResult() *ConcurrentResult {
	return &ConcurrentResult{
		objects: make([]*BloodHoundObject, 0),
	}
}

func (cr *ConcurrentResult) AddObjects(objects ...*BloodHoundObject) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.objects = append(cr.objects, objects...)
}

func (cr *ConcurrentResult) GetObjects() []*BloodHoundObject {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return append([]*BloodHoundObject(nil), cr.objects...)
}

// Legacy types for backwards compatibility
type BloodHoundNode struct {
	ID         string         `json:"id"`
	Kinds      []string       `json:"kinds"`
	Properties map[string]any `json:"properties,omitempty"`
}

type BloodHoundEdgeRef struct {
	MatchBy string `json:"match_by,omitempty"`
	Value   string `json:"value"`
	Kind    string `json:"kind,omitempty"`
}

type BloodHoundEdge struct {
	Start      BloodHoundEdgeRef `json:"start"`
	End        BloodHoundEdgeRef `json:"end"`
	Kind       string            `json:"kind"`
	Properties map[string]any    `json:"properties,omitempty"`
}

type BloodHoundMetadata struct {
	SourceKind string `json:"source_kind,omitempty"`
}

type BloodHoundGraph struct {
	Nodes []BloodHoundNode `json:"nodes"`
	Edges []BloodHoundEdge `json:"edges"`
}

type BloodHoundResult struct {
	Metadata *BloodHoundMetadata `json:"metadata,omitempty"`
	Graph    BloodHoundGraph     `json:"graph"`
}

type ParsedResult struct {
	Nodes []BloodHoundNode `json:"nodes"`
	Edges []BloodHoundEdge `json:"edges"`
}

func (pr *ParsedResult) ToBloodHoundResult(clusterName string) *BloodHoundResult {
	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind: "Kubernetes",
		},
		Graph: BloodHoundGraph{
			Nodes: pr.Nodes,
			Edges: pr.Edges,
		},
	}
}

type ResourceData struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace,omitempty"`
	Resource  any    `json:"resource"`
	Timestamp string `json:"timestamp"`
}

type PropertyMapper interface {
	MapProperties(resource any) (map[string]any, error)
}

type ResourceConfig struct {
	ResourceType   string
	PrimaryKind    string
	SecondaryKinds []string
	PropertyMapper PropertyMapper
}

type Parser interface {
	GetResourceType() string

	Parse(resource ResourceData) (*ParsedResult, error)

	GetSupportedKinds() []string

	GetConfig() ResourceConfig
}

// MultiTypeParser interface for parsers that handle multiple resource types
type MultiTypeParser interface {
	Parser
	GetSupportedResourceTypes() []string
}

type ParseRegistry struct {
	parsers map[string]Parser
}

func NewParseRegistry() *ParseRegistry {
	registry := &ParseRegistry{
		parsers: make(map[string]Parser),
	}
	return registry
}

func (r *ParseRegistry) Register(parser Parser) {
	// Register parser for its primary resource type
	r.parsers[parser.GetResourceType()] = parser

	// If it's a MultiTypeParser, also register for all supported resource types
	if multiParser, ok := parser.(MultiTypeParser); ok {
		for _, resourceType := range multiParser.GetSupportedResourceTypes() {
			r.parsers[resourceType] = parser
		}
	}
}

func (r *ParseRegistry) GetParser(resourceType string) (Parser, bool) {
	parser, exists := r.parsers[resourceType]
	return parser, exists
}

func (r *ParseRegistry) GetAllParsers() map[string]Parser {
	return r.parsers
}

func (r *ParseRegistry) ParseResource(resource ResourceData) (*ParsedResult, error) {
	parser, exists := r.GetParser(resource.Type)
	if !exists {
		return &ParsedResult{
			Nodes: []BloodHoundNode{},
			Edges: []BloodHoundEdge{},
		}, nil
	}

	return parser.Parse(resource)
}

func (r *ParseRegistry) ParseBatch(resources []ResourceData) (*ParsedResult, error) {
	combined := &ParsedResult{
		Nodes: []BloodHoundNode{},
		Edges: []BloodHoundEdge{},
	}

	for _, resource := range resources {
		result, err := r.ParseResource(resource)
		if err != nil {
			return nil, err
		}

		combined.Nodes = append(combined.Nodes, result.Nodes...)
		combined.Edges = append(combined.Edges, result.Edges...)
	}

	return combined, nil
}

var DefaultRegistry = NewParseRegistry()
