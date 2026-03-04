package edges

import "bloodhound-kube/internal/model"

type lateralMovementHostMountKubeletEdgeRule struct{}

// func init() {
// 	RegisterEdgeRule(lateralMovementHostMountKubeletEdgeRule{})
// }

func (r lateralMovementHostMountKubeletEdgeRule) Name() string {
	return "lateral_movement_host_mount_kubelet"
}

var edgePropertiesLateralMovementHostMountKubelet = map[string]any{
	"Description": "Pod has a host mount of the kubelet, which can allow for lateral movement and potential RCE if the kubelet API is exposed",
	"Reference":   "",
}

func (r lateralMovementHostMountKubeletEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podHostMountKubeletNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Pod w/ host mount of kubelet -> Nodes
func podHostMountKubeletNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	// for i := range space.Pods {
	// 	pod := &space.Pods[i]
	// 	// TODO: Check if pod has host mount of kubelet, if so create edge to node
	// }
	return edges
}
