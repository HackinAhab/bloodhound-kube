package nodes

// no additional imports

func init() {
	Register("Secret", BuildSecretNode)
}

func BuildSecretNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	data := GetMap(resource, "data")
	keys := MapKeysSorted(data)
	entries := MapEntriesSorted(data)

	secretType := GetString(resource, "type")
	// Helm release secrets have the "release" key in their data, and it is huge/unnecessary for this.
	if secretType == "helm.sh/release.v1" {
		entries[0] = "release={{Removed for clarity}}"
	}

	properties := map[string]any{
		"name":                  name,
		"namespace":             namespace,
		"labels":                MapToSortedList(labelsMap),
		"annotations":           MapToSortedList(annotationsMap),
		"secretType":            secretType,
		"dataKeys":              keys,
		"dataEntries":           entries,
		"isServiceAccountToken": secretType == "kubernetes.io/service-account-token",
		"isTlsSecret":           secretType == "kubernetes.io/tls",
		"isOpaque":              secretType == "Opaque" || secretType == "",
	}

	core := CoreEntry{
		Key:       "secrets",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("Secret", namespace, name),
			"kinds":           []string{"Secret"},
			"name":            name,
			"namespace":       namespace,
			"type":            secretType,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"data":            data,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Secret", namespace, name),
			Kinds:      []string{"Secret"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
