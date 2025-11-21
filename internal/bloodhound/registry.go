package bloodhound

import "fmt"

// ParseRegistry manages parsers for different Kubernetes resource types
type ParseRegistry struct {
	parsers map[string]Parser
}

// NewParseRegistry creates a new parse registry
func NewParseRegistry() *ParseRegistry {
	registry := &ParseRegistry{
		parsers: make(map[string]Parser),
	}
	return registry
}

// Register registers a parser for its resource type(s)
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

// GetParser retrieves a parser for the given resource type
func (r *ParseRegistry) GetParser(resourceType string) (Parser, bool) {
	parser, exists := r.parsers[resourceType]
	return parser, exists
}

// GetAllParsers returns all registered parsers
func (r *ParseRegistry) GetAllParsers() map[string]Parser {
	return r.parsers
}

// ParseResource parses a single resource using the appropriate parser
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

// ParseBatch parses multiple resources and combines the results
func (r *ParseRegistry) ParseBatch(resources []ResourceData) (*ParsedResult, error) {
	combined := &ParsedResult{
		Nodes: []BloodHoundNode{},
		Edges: []BloodHoundEdge{},
	}

	for _, resource := range resources {
		result, err := r.ParseResource(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource %s/%s: %w", resource.Type, resource.Namespace, err)
		}

		combined.Nodes = append(combined.Nodes, result.Nodes...)
		combined.Edges = append(combined.Edges, result.Edges...)
	}

	return combined, nil
}
