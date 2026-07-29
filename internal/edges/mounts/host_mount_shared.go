package mounts

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/workload"
)

// hostMountRule flags pods that host-mount one of sensitivePaths into a
// container, optionally excluding paths matched by exclude (e.g. sockets).
// Shared by HostMountReadEdgeRule and HostMountKubeletEdgeRule, which differ
// only in path list, edge kind, exclusion predicate, and description.
type hostMountRule struct {
	name           string
	edgeKind       string
	sensitivePaths []string
	exclude        func(hostPath string) bool // nil = no exclusion
	props          map[string]any
}

func (r hostMountRule) Name() string { return r.name }

func (r hostMountRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space != nil {
			edges = append(edges, r.applyNamespaced(ctx, ns, space)...)
		}
	}
	return edges
}

func (r hostMountRule) applyNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range space.Pods {
		pod := &space.Pods[i]
		if pod.NodeName == "" || pod.ID == "" {
			continue
		}
		node := ctx.Index.NodesByName[pod.NodeName]
		if node == nil || node.ID == "" {
			continue
		}
		mountPath, ok := r.check(pod)
		if !ok {
			continue
		}
		description, _ := r.props["Description"].(string)
		reference := r.props["Reference"]
		edges = append(edges, framework.CreateEdgeWithProperties(pod, node, r.edgeKind, map[string]any{
			"Description": description + " Mount path: " + mountPath,
			"Reference":   reference,
		}))
	}
	return edges
}

func (r hostMountRule) check(pod *workload.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" || !framework.HostPathMatchesAny(hostPath, r.sensitivePaths) {
			continue
		}
		if r.exclude != nil && r.exclude(hostPath) {
			continue
		}
		if volume.Name != "" {
			volumeNames[volume.Name] = struct{}{}
		}
	}
	if len(volumeNames) == 0 {
		return "", false
	}
	for _, container := range pod.Containers {
		for _, mount := range container.VolumeMounts {
			if _, ok := volumeNames[mount.Name]; ok && mount.MountPath != "" {
				return mount.MountPath, true
			}
		}
	}
	return "", false
}
