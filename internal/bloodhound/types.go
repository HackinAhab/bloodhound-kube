package bloodhound

type BloodHoundNode struct {
	ID         string         `json:"id"`
	Kinds      []string       `json:"kinds"`
	Properties map[string]any `json:"properties,omitempty"`
}

type BloodHoundEdge struct {
	Source     string         `json:"Source"`
	Target     string         `json:"Target"`
	Label      string         `json:"Label"`
	Properties map[string]any `json:"Properties,omitempty"`
}

type ParsedResult struct {
	Nodes []BloodHoundNode `json:"nodes"`
	Edges []BloodHoundEdge `json:"edges"`
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
