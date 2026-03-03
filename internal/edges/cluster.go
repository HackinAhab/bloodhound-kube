package edges

import "bloodhound-kube/internal/model"

type clusterScopeEdgesRule struct{}

func (r clusterScopeEdgesRule) Name() string {
	return "cluster"
}

func init() {
	RegisterEdgeRule(clusterScopeEdgesRule{})
}

func (r clusterScopeEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, pvcMountedByPod(ctx, ns, space)...)
	}
	edges = append(edges, pvBoundToPVC(ctx)...)
	return edges
}

func pvcMountedByPod(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
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
	return edges
}

func pvBoundToPVC(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
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
