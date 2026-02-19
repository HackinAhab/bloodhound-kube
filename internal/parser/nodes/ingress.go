package nodes

func init() {
	Register("Ingress", BuildIngressNode)
}

func BuildIngressNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	backendServices := extractIngressBackendServices(spec)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Key:       "ingresses",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("Ingress", namespace, name),
			"kinds":           []string{"Ingress"},
			"name":            name,
			"namespace":       namespace,
			"backendServices": backendServices,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"tls":             GetSlice(spec, "tls"),
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Ingress", namespace, name),
			Kinds:      []string{"Ingress"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func extractIngressBackendServices(spec map[string]any) []string {
	rules := GetSlice(spec, "rules")
	if len(rules) == 0 {
		return []string{}
	}
	services := map[string]struct{}{}
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		http := GetMap(ruleMap, "http")
		paths := GetSlice(http, "paths")
		for _, path := range paths {
			pathMap, ok := path.(map[string]any)
			if !ok {
				continue
			}
			backend := GetMap(pathMap, "backend")
			if svc := GetString(backend, "serviceName"); svc != "" {
				services[svc] = struct{}{}
				continue
			}
			service := GetMap(backend, "service")
			if svc := GetString(service, "name"); svc != "" {
				services[svc] = struct{}{}
			}
		}
	}
	return setToSortedList(services)
}
