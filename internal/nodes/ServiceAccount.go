package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceAccount struct {
	GraphNodeBase
	Secrets []string
}

func BuildServiceAccountNode(obj runtime.Object) (BuildResult, bool) {
	sa, ok := obj.(*corev1.ServiceAccount)
	if !ok || sa == nil {
		return BuildResult{}, false
	}
	name := sa.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := sa.Namespace
	labelsMap := StringMapToAnyMap(sa.Labels)
	annotationsMap := StringMapToAnyMap(sa.Annotations)

	secrets := make([]string, 0, len(sa.Secrets))
	for _, secret := range sa.Secrets {
		if secret.Name != "" {
			secrets = append(secrets, secret.Name)
		}
	}

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"secrets":     secrets,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ServiceAccount{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ServiceAccount", namespace, name),
				Kinds:          []string{"ServiceAccount"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Secrets: secrets,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ServiceAccount", namespace, name),
			Kinds:      []string{"ServiceAccount"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
