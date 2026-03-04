package edges

// TODO: Port LM_HOST_MOUNT_READ edge rule from rego.
import "bloodhound-kube/internal/model"

type lateralMovementHostMountReadEdgeRule struct{}

// func init() {
// 	RegisterEdgeRule(lateralMovementHostMountReadEdgeRule{})
// }

func (r lateralMovementHostMountReadEdgeRule) Name() string {
	return "lateral_movement_host_mount_read"
}

var edgePropertiesLateralMovementHostMountRead = map[string]any{
	"Description": "Pod has a host mount of sensitive directories, which can allow for lateral movement and potential information disclosure",
	"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_HOST_READ/",
}

func (r lateralMovementHostMountReadEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podHostMountReadNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Pod w/ host mount of sensitive Directories -> Nodes
func podHostMountReadNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	// for i := range space.Pods {
	// 	pod := &space.Pods[i]
	// 	// TODO: Check if pod has host mount of sensitive directories, if so create edge to node
	// }
	return edges
}
