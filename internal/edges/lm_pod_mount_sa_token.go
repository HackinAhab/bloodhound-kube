package edges

// TODO: Port LM_POD_MOUNT_SERVICEACCOUNT edge rule from rego.
import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type lateralMovementPodMountServiceAccountEdgeRule struct{}

func init() {
	RegisterEdgeRule(lateralMovementPodMountServiceAccountEdgeRule{})
}

func (r lateralMovementPodMountServiceAccountEdgeRule) Name() string {
	return "lateral_movement_pod_mount_service_account"
}

var edgePropertiesLateralMovementPodMountServiceAccount = map[string]any{
	"Description": "Pod mounts a ServiceAccount token, which may have additioanl privileges.",
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
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	for i := range space.Pods {
		pod := &space.Pods[i]
		if !podMountsServiceAccountToken(pod) {
			continue
		}
		saName := pod.ServiceAccount
		if saName == "" {
			saName = "default"
		}
		if saName == "" || serviceAccounts == nil {
			continue
		}
		serviceAccount := serviceAccounts[saName]
		if serviceAccount == nil || serviceAccount.ID == "" {
			continue
		}
		edges = append(edges, CreateEdgeWithProperties(pod, serviceAccount, "LM_POD_MOUNT_SERVICEACCOUNT", edgePropertiesLateralMovementPodMountServiceAccount))
	}
	return edges
}

func podMountsServiceAccountToken(pod *nodes.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.ServiceAccount != "default" {
		return pod.AutomountSAToken == nil || *pod.AutomountSAToken
	}
	return pod.AutomountSAToken != nil && *pod.AutomountSAToken
}
