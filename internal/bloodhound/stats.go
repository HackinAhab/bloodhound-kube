package bloodhound

import (
	"fmt"
	"sort"
	"strings"
)

type ParserStats struct {
	TotalParsers   int                       `json:"total_parsers"`
	SupportedTypes []string                  `json:"supported_types"`
	SupportedKinds []string                  `json:"supported_kinds"`
	ParserConfigs  map[string]ResourceConfig `json:"parser_configs"`
	KindsByType    map[string][]string       `json:"kinds_by_type"`
}

func GetParsingStats() ParserStats {
	parsers := DefaultRegistry.GetAllParsers()

	stats := ParserStats{
		TotalParsers:   len(parsers),
		SupportedTypes: make([]string, 0, len(parsers)),
		SupportedKinds: []string{},
		ParserConfigs:  make(map[string]ResourceConfig),
		KindsByType:    make(map[string][]string),
	}

	var allKinds []string
	kindSet := make(map[string]bool)

	for resourceType, parser := range parsers {
		stats.SupportedTypes = append(stats.SupportedTypes, resourceType)

		config := parser.GetConfig()
		stats.ParserConfigs[resourceType] = config

		kinds := parser.GetSupportedKinds()
		stats.KindsByType[resourceType] = kinds

		for _, kind := range kinds {
			if !kindSet[kind] {
				kindSet[kind] = true
				allKinds = append(allKinds, kind)
			}
		}
	}

	sort.Strings(stats.SupportedTypes)
	sort.Strings(allKinds)
	stats.SupportedKinds = allKinds

	return stats
}

func PrintParsingStats() {
	stats := GetParsingStats()

	fmt.Printf("BloodHound Parser Statistics:\n")
	fmt.Printf("============================\n")
	fmt.Printf("Total parsers: %d\n", stats.TotalParsers)
	fmt.Printf("Supported resource types: %s\n", strings.Join(stats.SupportedTypes, ", "))
	fmt.Printf("Supported BloodHound kinds: %s\n", strings.Join(stats.SupportedKinds, ", "))

	fmt.Printf("\nParser Configurations:\n")
	fmt.Printf("---------------------\n")
	for _, resourceType := range stats.SupportedTypes {
		config := stats.ParserConfigs[resourceType]
		fmt.Printf("• %s:\n", resourceType)
		fmt.Printf("  - Primary kind: %s\n", config.PrimaryKind)
		if len(config.SecondaryKinds) > 0 {
			fmt.Printf("  - Secondary kinds: %s\n", strings.Join(config.SecondaryKinds, ", "))
		}
		fmt.Printf("  - All kinds: %s\n", strings.Join(stats.KindsByType[resourceType], ", "))

		if config.PropertyMapper != nil {
			fmt.Printf("  - Custom property mapping: \n")
		} else {
			fmt.Printf("  - Custom property mapping:  (stub)\n")
		}
	}
}

func GetKindsByResourceType() map[string][]string {
	parsers := DefaultRegistry.GetAllParsers()
	kindsByType := make(map[string][]string)

	for resourceType, parser := range parsers {
		kindsByType[resourceType] = parser.GetSupportedKinds()
	}

	return kindsByType
}

func GetAllSupportedKinds() []string {
	parsers := DefaultRegistry.GetAllParsers()
	kindSet := make(map[string]bool)

	for _, parser := range parsers {
		for _, kind := range parser.GetSupportedKinds() {
			kindSet[kind] = true
		}
	}

	var kinds []string
	for kind := range kindSet {
		kinds = append(kinds, kind)
	}

	sort.Strings(kinds)
	return kinds
}

func GetParserByKind(kind string) (Parser, string, bool) {
	parsers := DefaultRegistry.GetAllParsers()

	for resourceType, parser := range parsers {
		for _, supportedKind := range parser.GetSupportedKinds() {
			if supportedKind == kind {
				return parser, resourceType, true
			}
		}
	}

	return nil, "", false
}

func ValidateKinds(kinds []string) error {
	supportedKinds := GetAllSupportedKinds()
	kindSet := make(map[string]bool)
	for _, kind := range supportedKinds {
		kindSet[kind] = true
	}

	var unsupported []string
	for _, kind := range kinds {
		if !kindSet[kind] {
			unsupported = append(unsupported, kind)
		}
	}

	if len(unsupported) > 0 {
		available := strings.Join(supportedKinds, ", ")
		return fmt.Errorf("unsupported BloodHound kinds: %s (available: %s)",
			strings.Join(unsupported, ", "), available)
	}

	return nil
}
