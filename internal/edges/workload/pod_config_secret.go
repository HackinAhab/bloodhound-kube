package workload

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

type podEdgesRule struct{}

func (r podEdgesRule) Name() string { return "pods" }

func (r podEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		pods := space.Pods
		secrets := space.Secrets
		for i := range pods {
			pod := &pods[i]
			if pod.NodeName != "" {
				if node := ctx.Index.NodesByName[pod.NodeName]; node != nil {
					edges = append(edges, framework.CreateEdge(pod, node, "ScheduledOn"))
				}
			}

			for _, secret := range secrets {
				for _, volume := range pod.Volumes {
					if volume.Type == "secret" && volume.SecretName == secret.Name {
						edges = append(edges, framework.CreateEdge(&secret, pod, "MountedBy"))
					}
				}
				for _, container := range pod.Containers {
					for _, envSource := range container.EnvFrom {
						if envSource.SecretRef != nil && envSource.SecretRef.Name == secret.Name {
							edges = append(edges, framework.CreateEdge(&secret, pod, "EnvVars"))
						}
					}
				}
			}
		}
	}
	return edges
}

var _ nodefw.EdgeNode = workload.Pod{}

type configMapEdgesRule struct{}

func (r configMapEdgesRule) Name() string { return "configmap" }

func (r configMapEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.ConfigMaps {
			cm := &space.ConfigMaps[i]
			for j := range space.Pods {
				pod := &space.Pods[j]
				for _, volume := range pod.Volumes {
					if volume.Type == "configmap" && volume.ConfigMapName == cm.Name {
						edges = append(edges, framework.CreateEdge(cm, pod, "MountedBy"))
					}
				}
				for _, container := range pod.Containers {
					for _, envSource := range container.EnvFrom {
						if envSource.ConfigMapRef != nil && envSource.ConfigMapRef.Name == cm.Name {
							edges = append(edges, framework.CreateEdge(cm, pod, "EnvVars"))
						}
					}
				}
			}
		}
	}
	return edges
}

type secretEdgesRule struct{}

func (r secretEdgesRule) Name() string { return "secret" }

var edgePropertiesServiceAccountToken = map[string]any{
	"Description": "Service account ",
}

func (r secretEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		serviceAccounts := ctx.Index.ServiceAccountsByNamespace[ns]
		for i := range space.Secrets {
			secret := &space.Secrets[i]
			if secret.SecretType != "kubernetes.io/service-account-token" {
				continue
			}
			for _, sa := range serviceAccounts {
				for _, secretName := range sa.Secrets {
					if secretName == secret.Name {
						edges = append(edges, framework.CreateEdgeWithProperties(secret, sa, "SAToken", edgePropertiesServiceAccountToken))
					}
				}
			}
		}
	}
	return edges
}
