package workload

import (
	"sort"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

type envVarsKey struct {
	sourceID   string
	podID      string
	sourceType string
}

type envVarsEntry struct {
	source     nodefw.EdgeNode
	containers map[string]struct{}
}

type podEdgesRule struct{}

func (r podEdgesRule) Name() string { return "pods" }

func (r podEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		secretIndex := ctx.Index.SecretsByNamespace[ns]
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.NodeName != "" {
				if node := ctx.Index.NodesByName[pod.NodeName]; node != nil {
					edges = append(edges, framework.CreateEdge(pod, node, "BHK_ScheduledOn"))
				}
			}

			for _, volume := range pod.Volumes {
				if volume.Type != "secret" || volume.SecretName == "" {
					continue
				}
				if secret := secretIndex[volume.SecretName]; secret != nil {
					edges = append(edges, framework.CreateEdge(secret, pod, "BHK_MountedBy"))
				}
			}

			// Accumulate EnvVars references: key=(secretID, podID, sourceType) → entry with source + container set
			envVarsAcc := map[envVarsKey]*envVarsEntry{}
			for _, container := range pod.Containers {
				for _, envSource := range container.EnvFrom {
					if envSource.SecretRef != nil && envSource.SecretRef.Name != "" {
						if secret := secretIndex[envSource.SecretRef.Name]; secret != nil {
							k := envVarsKey{sourceID: secret.EdgeID(), podID: pod.EdgeID(), sourceType: "envFrom"}
							if envVarsAcc[k] == nil {
								envVarsAcc[k] = &envVarsEntry{source: secret, containers: map[string]struct{}{}}
							}
							envVarsAcc[k].containers[container.Name] = struct{}{}
						}
					}
				}
				for _, env := range container.Env {
					if env.ValueRef == nil || env.ValueRef.SecretRef == nil {
						continue
					}
					if secret := secretIndex[env.ValueRef.SecretRef.Name]; secret != nil {
						k := envVarsKey{sourceID: secret.EdgeID(), podID: pod.EdgeID(), sourceType: "valueFrom"}
						if envVarsAcc[k] == nil {
							envVarsAcc[k] = &envVarsEntry{source: secret, containers: map[string]struct{}{}}
						}
						envVarsAcc[k].containers[container.Name] = struct{}{}
					}
				}
			}

			// Flush accumulator into edges with properties
			for k, entry := range envVarsAcc {
				containers := make([]string, 0, len(entry.containers))
				for name := range entry.containers {
					containers = append(containers, name)
				}
				sort.Strings(containers)
				edges = append(edges, framework.CreateEdgeWithProperties(entry.source, pod, "BHK_EnvVars", map[string]any{
					"SourceType": k.sourceType,
					"Containers": containers,
				}))
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
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		cmIndex := ctx.Index.ConfigMapsByNamespace[ns]
		for j := range space.Pods {
			pod := &space.Pods[j]

			for _, volume := range pod.Volumes {
				if volume.Type != "configmap" || volume.ConfigMapName == "" {
					continue
				}
				if cm := cmIndex[volume.ConfigMapName]; cm != nil {
					edges = append(edges, framework.CreateEdge(cm, pod, "BHK_MountedBy"))
				}
			}

			// Accumulate EnvVars references: key=(cmID, podID, sourceType) → entry with source + container set
			envVarsAcc := map[envVarsKey]*envVarsEntry{}
			for _, container := range pod.Containers {
				for _, envSource := range container.EnvFrom {
					if envSource.ConfigMapRef != nil && envSource.ConfigMapRef.Name != "" {
						if cm := cmIndex[envSource.ConfigMapRef.Name]; cm != nil {
							k := envVarsKey{sourceID: cm.EdgeID(), podID: pod.EdgeID(), sourceType: "envFrom"}
							if envVarsAcc[k] == nil {
								envVarsAcc[k] = &envVarsEntry{source: cm, containers: map[string]struct{}{}}
							}
							envVarsAcc[k].containers[container.Name] = struct{}{}
						}
					}
				}
				for _, env := range container.Env {
					if env.ValueRef == nil || env.ValueRef.ConfigMapRef == nil {
						continue
					}
					if cm := cmIndex[env.ValueRef.ConfigMapRef.Name]; cm != nil {
						k := envVarsKey{sourceID: cm.EdgeID(), podID: pod.EdgeID(), sourceType: "valueFrom"}
						if envVarsAcc[k] == nil {
							envVarsAcc[k] = &envVarsEntry{source: cm, containers: map[string]struct{}{}}
						}
						envVarsAcc[k].containers[container.Name] = struct{}{}
					}
				}
			}

			// Flush accumulator into edges with properties
			for k, entry := range envVarsAcc {
				containers := make([]string, 0, len(entry.containers))
				for name := range entry.containers {
					containers = append(containers, name)
				}
				sort.Strings(containers)
				edges = append(edges, framework.CreateEdgeWithProperties(entry.source, pod, "BHK_EnvVars", map[string]any{
					"SourceType": k.sourceType,
					"Containers": containers,
				}))
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
						edges = append(edges, framework.CreateEdgeWithProperties(secret, sa, "BHK_SAToken", edgePropertiesServiceAccountToken))
					}
				}
			}
		}
	}
	return edges
}
