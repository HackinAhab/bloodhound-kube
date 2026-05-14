package mounts

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/workload"
)

type PodMountServiceAccountEdgeRule struct{}

func (r PodMountServiceAccountEdgeRule) Name() string {
	return "lateral_movement_pod_mount_service_account"
}

var edgePropertiesPodMountServiceAccount = map[string]any{
	"Description": "Pod mounts a ServiceAccount token, which may have additional privileges.",
	"Reference":   "https://kubehound.io/reference/attacks/TOKEN_STEAL/",
}

func (r PodMountServiceAccountEdgeRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space != nil {
			edges = append(edges, podMountServiceAccountNamespaced(ctx, ns, space)...)
		}
	}
	return edges
}

func podMountServiceAccountNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
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
		edges = append(edges, framework.CreateEdgeWithProperties(pod, serviceAccount, "mountedSA", edgePropertiesPodMountServiceAccount))
	}
	return edges
}

func podMountsServiceAccountToken(pod *workload.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.ServiceAccount != "default" {
		return pod.AutomountSAToken == nil || *pod.AutomountSAToken
	}
	return pod.AutomountSAToken != nil && *pod.AutomountSAToken
}
