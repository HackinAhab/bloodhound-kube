package nodes

func init() {
	Register("SecretStore", BuildSecretStoreNode)
	Register("ClusterSecretStore", BuildClusterSecretStoreNode)
	Register("ExternalSecret", BuildExternalSecretNode)
}

func BuildSecretStoreNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	providerType := providerTypeFromSpec(spec)

	properties := map[string]any{
		"name":         name,
		"namespace":    namespace,
		"labels":       MapToSortedList(labelsMap),
		"annotations":  MapToSortedList(annotationsMap),
		"providerType": providerType,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: SecretStoreCore{
			CoreNode: CoreNode{
				ID:             BuildID("SecretStore", namespace, name),
				Kinds:          []string{"SecretStore"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			ProviderType: providerType,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("SecretStore", namespace, name),
			Kinds:      []string{"SecretStore"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func BuildClusterSecretStoreNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	providerType := providerTypeFromSpec(spec)

	properties := map[string]any{
		"name":         name,
		"namespace":    "",
		"labels":       MapToSortedList(labelsMap),
		"annotations":  MapToSortedList(annotationsMap),
		"providerType": providerType,
	}

	core := CoreEntry{
		Cluster: true,
		Data: ClusterSecretStoreCore{
			CoreNode: CoreNode{
				ID:             BuildID("ClusterSecretStore", "", name),
				Kinds:          []string{"ClusterSecretStore"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			ProviderType: providerType,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ClusterSecretStore", "", name),
			Kinds:      []string{"ClusterSecretStore"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func BuildExternalSecretNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	storeRef := GetMap(spec, "secretStoreRef")
	target := GetMap(spec, "target")
	storeName := GetString(storeRef, "name")
	storeKind := GetStringDefault(storeRef, "kind", "SecretStore")

	dataKeys := extractExternalSecretDataKeys(GetSlice(spec, "data"))
	dataFromTypes := extractExternalSecretDataFromTypes(GetSlice(spec, "dataFrom"))

	properties := map[string]any{
		"name":            name,
		"namespace":       namespace,
		"labels":          MapToSortedList(labelsMap),
		"annotations":     MapToSortedList(annotationsMap),
		"storeName":       storeName,
		"storeKind":       storeKind,
		"targetName":      GetString(target, "name"),
		"refreshInterval": GetString(spec, "refreshInterval"),
		"creationPolicy":  GetString(target, "creationPolicy"),
		"dataKeys":        dataKeys,
		"dataFromTypes":   dataFromTypes,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ExternalSecretCore{
			CoreNode: CoreNode{
				ID:             BuildID("ExternalSecret", namespace, name),
				Kinds:          []string{"ExternalSecret"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			StoreName:     storeName,
			StoreKind:     storeKind,
			TargetName:    GetString(target, "name"),
			DataKeys:      dataKeys,
			DataFromTypes: dataFromTypes,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ExternalSecret", namespace, name),
			Kinds:      []string{"ExternalSecret"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func providerTypeFromSpec(spec map[string]any) string {
	provider := GetMap(spec, "provider")
	if len(provider) == 0 {
		return ""
	}
	keys := MapKeysSorted(provider)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func extractExternalSecretDataKeys(items []any) []string {
	if len(items) == 0 {
		return []string{}
	}
	keys := map[string]struct{}{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := GetString(entry, "secretKey")
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return SortedSetKeys(keys)
}

func extractExternalSecretDataFromTypes(items []any) []string {
	if len(items) == 0 {
		return []string{}
	}
	types := map[string]struct{}{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		keys := MapKeysSorted(entry)
		if len(keys) == 0 {
			continue
		}
		types[keys[0]] = struct{}{}
	}
	return SortedSetKeys(types)
}
