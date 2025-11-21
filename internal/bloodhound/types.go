package bloodhound

// Core BloodHound graph structures
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

// ResourceData represents a Kubernetes resource for parsing
type ResourceData struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace,omitempty"`
	Resource  any    `json:"resource"`
	Timestamp string `json:"timestamp"`
}

// ResourceConfig defines configuration for resource parsing
type ResourceConfig struct {
	ResourceType   string
	PrimaryKind    string
	SecondaryKinds []string
	PropertyMapper PropertyMapper
}

// Global registry instance
var DefaultRegistry = NewParseRegistry()
