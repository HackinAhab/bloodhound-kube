package networking

import (
	. "bloodhound-kube/internal/nodes/framework"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NetworkPolicy struct {
	GraphNodeBase
	PodSelectorLabels map[string]string
}

func BuildNetworkPolicyNode(obj runtime.Object) (BuildResult, bool) {
	policy, ok := obj.(*networkingv1.NetworkPolicy)
	if !ok || policy == nil {
		return BuildResult{}, false
	}
	name := policy.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := policy.Namespace
	labelsMap := StringMapToAnyMap(policy.Labels)
	annotationsMap := StringMapToAnyMap(policy.Annotations)
	selectorLabels := policy.Spec.PodSelector.MatchLabels

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: NetworkPolicy{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("NetworkPolicy", namespace, name),
				Kinds:          []string{"NetworkPolicy"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			PodSelectorLabels: selectorLabels,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("NetworkPolicy", namespace, name),
			Kinds:      []string{"NetworkPolicy"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
