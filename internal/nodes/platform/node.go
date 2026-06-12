package platform

import (
	fw "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)



type Node struct {
	fw.GraphNodeBase
}

func BuildNodeNode(obj runtime.Object) (fw.BuildResult, bool) {
	node, ok := obj.(*corev1.Node)
	if !ok || node == nil {
		return fw.BuildResult{}, false
	}
	name := node.Name
	if name == "" {
		return fw.BuildResult{}, false
	}
	labelsMap := fw.StringMapToAnyMap(node.Labels)
	annotationsMap := fw.StringMapToAnyMap(node.Annotations)
	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      fw.MapToSortedList(labelsMap),
		"annotations": fw.MapToSortedList(annotationsMap),
	}

	core := fw.CoreEntry{
		Cluster: true,
		Data: Node{
			GraphNodeBase: fw.GraphNodeBase{
				ID:             fw.BuildID("Node", "", name),
				Kinds:          []string{"Node"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         fw.BuildID("Node", "", name),
			Kinds:      []string{"K8s_Node"},
			Properties: properties,
		},
		Core: []fw.CoreEntry{core},
	}, true
}
