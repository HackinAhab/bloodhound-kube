package nodes

func init() {
	Register("PersistentVolume", BuildPVNode)
}

func BuildPVNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	claimRef := GetMap(spec, "claimRef")

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Cluster: true,
		Data: PersistentVolumeCore{
			CoreNode: CoreNode{
				ID:             BuildID("PersistentVolume", "", name),
				Kinds:          []string{"PersistentVolume"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			ClaimRef: claimRef,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("PersistentVolume", "", name),
			Kinds:      []string{"PersistentVolume"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
