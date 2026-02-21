package nodes

func ExternalNode() NodeResult {
	return NodeResult{
		ID:    BuildID("External", "", "external"),
		Kinds: []string{"External"},
		Properties: map[string]any{
			"name":          "external",
			"namespace":     "",
			"resource_type": "external",
		},
	}
}
