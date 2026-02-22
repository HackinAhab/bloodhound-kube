package nodes

import "strings"

func init() {
	Register("", BuildGenericNode)
}

func BuildGenericNode(resource map[string]any) (BuildResult, bool) {
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

	kind := GetString(resource, "kind")
	apiVersion := GetString(resource, "apiVersion")
	group := ""
	if apiVersion != "" {
		parts := strings.SplitN(apiVersion, "/", 2)
		if len(parts) == 2 {
			group = parts[0]
		}
	}

	kindKey := kind
	if group != "" {
		kindKey = kind + ":" + group
	}
	if kindKey == "" {
		return BuildResult{}, false
	}
	properties["resource_type"] = "generic"
	if group != "" {
		properties["group"] = group
	}
	if apiVersion != "" {
		properties["apiVersion"] = apiVersion
	}

	id := BuildID(kindKey, namespace, name)

	return BuildResult{
		Node: NodeResult{
			ID:         id,
			Kinds:      []string{kindKey},
			Properties: properties,
		},
	}, true
}
