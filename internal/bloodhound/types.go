package bloodhound

import (
	"sync"
	"time"
)

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
	SourceKind          string `json:"source_kind,omitempty"`
	ClusterName         string `json:"cluster_name,omitempty"`
	CollectionTimestamp string `json:"collection_timestamp,omitempty"`
	Version             string `json:"version,omitempty"`
}

type BloodHoundGraph struct {
	Nodes []BloodHoundNode `json:"nodes"`
	Edges []BloodHoundEdge `json:"edges"`
}

type BloodHoundResult struct {
	Metadata *BloodHoundMetadata `json:"metadata,omitempty"`
	Graph    BloodHoundGraph     `json:"graph"`
}

// Thread-safe result accumulator for concurrent processing
type ConcurrentResult struct {
	mu    sync.RWMutex
	nodes []BloodHoundNode
	edges []BloodHoundEdge
}

func NewConcurrentResult() *ConcurrentResult {
	return &ConcurrentResult{
		nodes: make([]BloodHoundNode, 0),
		edges: make([]BloodHoundEdge, 0),
	}
}

func (cr *ConcurrentResult) AddNodes(nodes ...BloodHoundNode) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.nodes = append(cr.nodes, nodes...)
}

func (cr *ConcurrentResult) AddEdges(edges ...BloodHoundEdge) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.edges = append(cr.edges, edges...)
}

func (cr *ConcurrentResult) GetResult() BloodHoundGraph {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return BloodHoundGraph{
		Nodes: append([]BloodHoundNode(nil), cr.nodes...),
		Edges: append([]BloodHoundEdge(nil), cr.edges...),
	}
}

// Legacy type for backwards compatibility
type ParsedResult struct {
	Nodes []BloodHoundNode `json:"nodes"`
	Edges []BloodHoundEdge `json:"edges"`
}

// Convert legacy ParsedResult to new structure
func (pr *ParsedResult) ToBloodHoundResult(clusterName string) *BloodHoundResult {
	return &BloodHoundResult{
		Metadata: &BloodHoundMetadata{
			SourceKind:          "KubeBase",
			ClusterName:         clusterName,
			CollectionTimestamp: time.Now().UTC().Format(time.RFC3339),
			Version:             "1.0",
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
	r.parsers[parser.GetResourceType()] = parser
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
