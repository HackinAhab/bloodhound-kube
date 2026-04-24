package mounts

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type PersistentVolumesEdgesRule struct{}

func (r PersistentVolumesEdgesRule) Name() string { return "cluster" }

func (r PersistentVolumesEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, pvcMountedByPod(ns, space)...)
	}
	edges = append(edges, pvBoundToPVC(ctx)...)
	return edges
}

func pvcMountedByPod(namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if namespace == "" || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range space.PersistentVolumeClaims {
		pvc := &space.PersistentVolumeClaims[i]
		for j := range space.Pods {
			pod := &space.Pods[j]
			for _, volume := range pod.Volumes {
				if volume.PVCName == pvc.Name {
					edges = append(edges, framework.CreateEdge(pvc, pod, "MountedBy"))
				}
			}
		}
	}
	return edges
}

func pvBoundToPVC(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range ctx.Core.Cluster.PersistentVolumes {
		pv := &ctx.Core.Cluster.PersistentVolumes[i]
		claimRef := pv.ClaimRef
		if claimRef == nil || claimRef.Name == "" || claimRef.Namespace == "" {
			continue
		}
		space := ctx.Core.Namespaces[claimRef.Namespace]
		if space == nil {
			continue
		}
		for j := range space.PersistentVolumeClaims {
			pvc := &space.PersistentVolumeClaims[j]
			if pvc.Name == claimRef.Name {
				edges = append(edges, framework.CreateEdge(pv, pvc, "BoundTo"))
			}
		}
	}
	return edges
}
