package bloodhound

// PropertyMapper defines the contract for extracting properties from Kubernetes resources
type PropertyMapper interface {
	MapProperties(resource any) (map[string]any, error)
}

// Parser defines the contract for parsing Kubernetes resources into BloodHound nodes
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
