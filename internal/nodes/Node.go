package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("Node"), BuildNodeNode)
}

type Node struct {
	GraphNodeBase
}

func BuildNodeNode(obj runtime.Object) (BuildResult, bool) {
	node, ok := obj.(*corev1.Node)
	if !ok || node == nil {
		return BuildResult{}, false
	}
	name := node.Name
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := StringMapToAnyMap(node.Labels)
	annotationsMap := StringMapToAnyMap(node.Annotations)
	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Cluster: true,
		Data: Node{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Node", "", name),
				Kinds:          []string{"Node"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Node", "", name),
			Kinds:      []string{"K8s_Node"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
