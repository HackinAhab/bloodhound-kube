package nodes

func init() {
	Register("PersistentVolumeClaim", BuildPVCNode)
}

func BuildPVCNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Key:       "persistentvolumeclaims",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("PersistentVolumeClaim", namespace, name),
			"kinds":           []string{"PersistentVolumeClaim"},
			"name":            name,
			"namespace":       namespace,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("PersistentVolumeClaim", namespace, name),
			Kinds:      []string{"PersistentVolumeClaim"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
