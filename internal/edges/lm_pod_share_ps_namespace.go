package edges

// TODO: Port SHARE_PS_NAMESPACE edge rule from rego.
import "bloodhound-kube/internal/model"

type lateralMovementPodSharePSNamespaceEdgeRule struct{}

// func init() {
// 	RegisterEdgeRule(lateralMovementPodSharePSNamespaceEdgeRule{})
// }

func (r lateralMovementPodSharePSNamespaceEdgeRule) Name() string {
	return "lateral_movement_pod_share_ps_namespace"
}

var edgePropertiesLateralMovementPodSharePSNamespace = map[string]any{
	"Description": "Containers in a pod share a process namespace, which can allow for lateral movement and potential privilege escalation",
	"Reference":   "https://kubehound.io/reference/attacks/SHARE_PS_NAMESPACE/",
}

func (r lateralMovementPodSharePSNamespaceEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podSharePSNamespaceNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Containers in pod share process namespace.
func podSharePSNamespaceNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	// for i := range space.Pods {
	// 	pod := &space.Pods[i]
	// 	// TODO:
	// }
	return edges
}
