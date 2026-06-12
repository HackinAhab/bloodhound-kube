package addons

import (
	. "bloodhound-kube/internal/nodes/framework"

	securityv1 "github.com/openshift/api/security/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type SecurityContextConstraints struct {
	GraphNodeBase
}

func BuildSecurityContextConstraintsNode(obj runtime.Object) (BuildResult, bool) {
	scc, ok := obj.(*securityv1.SecurityContextConstraints)
	if !ok || scc == nil {
		return BuildResult{}, false
	}
	name := scc.Name
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := StringMapToAnyMap(scc.Labels)
	annotationsMap := StringMapToAnyMap(scc.Annotations)

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Cluster: true,
		Data: SecurityContextConstraints{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("SecurityContextConstraints", "", name),
				Kinds:          []string{"SecurityContextConstraints"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("SecurityContextConstraints", "", name),
			Kinds:      []string{"SecurityContextConstraints"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
