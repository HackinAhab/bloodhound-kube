package mounts

import (
	. "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PersistentVolume struct {
	GraphNodeBase
	ClaimRef *ClaimRef
}

type ClaimRef struct {
	APIVersion      string
	Kind            string
	Name            string
	Namespace       string
	UID             string
	ResourceVersion string
	FieldPath       string
}

func BuildPVNode(obj runtime.Object) (BuildResult, bool) {
	pv, ok := obj.(*corev1.PersistentVolume)
	if !ok || pv == nil {
		return BuildResult{}, false
	}
	name := pv.Name
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := StringMapToAnyMap(pv.Labels)
	annotationsMap := StringMapToAnyMap(pv.Annotations)

	claimRef := objectRefToClaimRef(pv.Spec.ClaimRef)

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	base := NewGraphNodeBase("PersistentVolume", "", name, labelsMap, annotationsMap)

	core := CoreEntry{
		Cluster: true,
		Data: PersistentVolume{
			GraphNodeBase: base,
			ClaimRef:      claimRef,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func objectRefToClaimRef(ref *corev1.ObjectReference) *ClaimRef {
	if ref == nil {
		return nil
	}
	return &ClaimRef{
		APIVersion:      ref.APIVersion,
		Kind:            ref.Kind,
		Name:            ref.Name,
		Namespace:       ref.Namespace,
		UID:             string(ref.UID),
		ResourceVersion: ref.ResourceVersion,
		FieldPath:       ref.FieldPath,
	}
}
