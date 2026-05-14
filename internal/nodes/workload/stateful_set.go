package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type StatefulSetCore struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
}

func BuildStatefulSetNode(obj runtime.Object) (BuildResult, bool) {
	set, ok := obj.(*appsv1.StatefulSet)
	if !ok || set == nil {
		return BuildResult{}, false
	}
	name := set.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := set.Namespace
	labelsMap := StringMapToAnyMap(set.Labels)
	annotationsMap := StringMapToAnyMap(set.Annotations)

	selectorLabels := Labels(set.Spec.Selector.MatchLabels)
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceName := set.Spec.ServiceName
	serviceAccount := set.Spec.Template.Spec.ServiceAccountName

	replicas := I32(set.Spec.Replicas)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["replicas"] = replicas
	properties["selector"] = MapToSortedList(selectorMap)
	properties["serviceAccount"] = serviceAccount
	properties["serviceName"] = serviceName

	base := NewGraphNodeBase("StatefulSet", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: StatefulSetCore{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			ServiceAccount: serviceAccount,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
