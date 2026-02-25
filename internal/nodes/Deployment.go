package nodes

type Deployment struct {
	GraphNodeBase
	SelectorMap       map[string]any
	PodTemplateLabels map[string]any
	ServiceAccount    string
}

func init() {
	Register("Deployment", BuildDeploymentNode)
}

func BuildDeploymentNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorMap := GetMap(GetMap(spec, "selector"), "matchLabels")
	template := GetMap(spec, "template")
	templateLabels := GetMap(GetMap(template, "metadata"), "labels")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"replicas":    GetNumber(spec, "replicas"),
		"selector":    MapToSortedList(selectorMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Deployment{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Deployment", namespace, name),
				Kinds:          []string{"Deployment"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			SelectorMap:       selectorMap,
			PodTemplateLabels: templateLabels,
			ServiceAccount:    GetString(GetMap(template, "spec"), "serviceAccountName"),
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Deployment", namespace, name),
			Kinds:      []string{"Deployment"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
