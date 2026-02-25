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

func ExternalCoreEntry() External {
	return External{
		GraphNodeBase: GraphNodeBase{
			ID:             BuildID("External", "", "external"),
			Kinds:          []string{"External"},
			Name:           "external",
			Namespace:      "",
			LabelsMap:      map[string]any{},
			AnnotationsMap: map[string]any{},
		},
	}
}
