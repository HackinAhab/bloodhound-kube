package platform

import fw "bloodhound-kube/internal/nodes/framework"

type External struct {
	fw.GraphNodeBase
}

func ExternalNode() fw.NodeResult {
	return fw.NodeResult{
		ID:    fw.BuildID("BHK_External", "", "external"),
		Kinds: []string{"BHK_External"},
		Properties: map[string]any{
			"name":          "external",
			"namespace":     "",
			"resource_type": "external",
		},
	}
}

func ExternalCoreEntry() External {
	return External{
		GraphNodeBase: fw.GraphNodeBase{
			ID:             fw.BuildID("BHK_External", "", "external"),
			Kinds:          []string{"BHK_External"},
			Name:           "external",
			Namespace:      "",
			LabelsMap:      map[string]any{},
			AnnotationsMap: map[string]any{},
		},
	}
}
