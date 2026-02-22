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
		Namespace: namespace,
		Cluster:   false,
		Data: PersistentVolumeClaimCore{
			CoreNode: CoreNode{
				ID:             BuildID("PersistentVolumeClaim", namespace, name),
				Kinds:          []string{"PersistentVolumeClaim"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
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
