package platform

import fw "bloodhound-kube/internal/nodes/framework"



func BuildNamespaceNode(resource map[string]any) (fw.BuildResult, bool) {
	metadata := fw.GetMap(resource, "metadata")
	name := fw.GetString(metadata, "name")
	if name == "" {
		return fw.BuildResult{}, false
	}
	labelsMap := fw.GetMap(metadata, "labels")
	annotationsMap := fw.GetMap(metadata, "annotations")

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      fw.MapToSortedList(labelsMap),
		"annotations": fw.MapToSortedList(annotationsMap),
	}

	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         fw.BuildID("BHK_Namespace", "", name),
			Kinds:      []string{"BHK_Namespace"},
			Properties: properties,
		},
	}, true
}
