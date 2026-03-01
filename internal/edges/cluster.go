package edges

import "bloodhound-kube/internal/model"

type clusterEdgesRule struct{}

func (r clusterEdgesRule) Name() string {
	return "cluster"
}

func (r clusterEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.PersistentVolumeClaims {
			pvc := &space.PersistentVolumeClaims[i]
			for j := range space.Pods {
				pod := &space.Pods[j]
				for _, volume := range pod.Volumes {
					if volume.PVCName == pvc.Name {
						edges = append(edges, CreateEdge(pvc, pod, "MountedBy"))
					}
				}
			}
		}
	}
	for i := range ctx.Core.Cluster.PersistentVolumes {
		pv := &ctx.Core.Cluster.PersistentVolumes[i]
		claimRef := pv.ClaimRef
		if claimRef == nil {
			continue
		}
		claimName := claimRef.Name
		claimNamespace := claimRef.Namespace
		if claimName == "" || claimNamespace == "" {
			continue
		}
		space := ctx.Core.Namespaces[claimNamespace]
		if space == nil {
			continue
		}
		for j := range space.PersistentVolumeClaims {
			pvc := &space.PersistentVolumeClaims[j]
			if pvc.Name == claimName {
				edges = append(edges, CreateEdge(pv, pvc, "BoundTo"))
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(clusterEdgesRule{})
}
