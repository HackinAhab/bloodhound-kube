package bloodhound

// RegisterParser provides a way to register additional parsers at runtime
func RegisterParser(parser Parser) {
	DefaultRegistry.Register(parser)
}

// GetRegisteredParsers returns information about all registered parsers
func GetRegisteredParsers() map[string]string {
	parsers := DefaultRegistry.GetAllParsers()
	info := make(map[string]string)

	for resourceType, parser := range parsers {
		info[resourceType] = parser.GetConfig().PrimaryKind
	}

	return info
}
