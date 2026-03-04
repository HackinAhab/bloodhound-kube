package edges

// TODO: Port LM_POD_MOUNT_SERVICEACCOUNT edge rule from rego.
import "bloodhound-kube/internal/model"

type lateralMovementPodMountServiceAccountEdgeRule struct{}

// func init() {
// 	RegisterEdgeRule(lateralMovementPodMountServiceAccountEdgeRule{})
// }

func (r lateralMovementPodMountServiceAccountEdgeRule) Name() string {
	return "lateral_movement_pod_mount_service_account"
}

var edgePropertiesLateralMovementPodMountServiceAccount = map[string]any{
	"Description": "Pod has a host mount of a ServiceAccount token, which can allow for lateral movement and potential privilege escalation",
	"Reference":   "https://kubehound.io/reference/attacks/TOKEN_STEAL/",
}

func (r lateralMovementPodMountServiceAccountEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podMountServiceAccountNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Pod w/ host mount of service account -> Service Account that can be impersonated
func podMountServiceAccountNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	// for i := range space.Pods {
	// 	pod := &space.Pods[i]
	// 	// TODO: Check if pod mounts service account token, if so create edge to
	// }
	return edges
}
