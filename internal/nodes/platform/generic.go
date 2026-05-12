package platform

import (
	fw "bloodhound-kube/internal/nodes/framework"
	"strings"
)

type GenericNode struct {
	fw.GraphNodeBase
}

func BuildGenericNode(resource map[string]any) (fw.BuildResult, bool) {
	metadata := fw.GetMap(resource, "metadata")
	name := fw.GetString(metadata, "name")
	if name == "" {
		return fw.BuildResult{}, false
	}

	namespace := fw.GetString(metadata, "namespace")
	labelsMap := fw.GetMap(metadata, "labels")
	annotationsMap := fw.GetMap(metadata, "annotations")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      fw.MapToSortedList(labelsMap),
		"annotations": fw.MapToSortedList(annotationsMap),
	}

	kind := fw.GetString(resource, "kind")
	apiVersion := fw.GetString(resource, "apiVersion")
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
		return fw.BuildResult{}, false
	}
	properties["resource_type"] = "generic"
	if group != "" {
		properties["group"] = group
	}
	if apiVersion != "" {
		properties["apiVersion"] = apiVersion
	}

	id := fw.BuildID(kindKey, namespace, name)

	core := fw.CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: GenericNode{
			GraphNodeBase: fw.GraphNodeBase{
				ID:             id,
				Kinds:          []string{kindKey},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         id,
			Kinds:      []string{kindKey},
			Properties: properties,
		},
		Core: []fw.CoreEntry{core},
	}, true
}
