package nodes

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
		Key:       "deployments",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":                BuildID("Deployment", namespace, name),
			"kinds":             []string{"Deployment"},
			"name":              name,
			"namespace":         namespace,
			"selector_map":      selectorMap,
			"labels_map":        labelsMap,
			"annotations_map":   annotationsMap,
			"podTemplateLabels": templateLabels,
			"serviceAccount":    GetString(GetMap(template, "spec"), "serviceAccountName"),
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
