package edges

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type podEdgesRule struct{}

func (r podEdgesRule) Name() string {
	return "pods"
}

func (r podEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		pods := space.Pods
		serviceAccounts := space.ServiceAccounts
		secrets := space.Secrets
		for i := range pods {
			pod := &pods[i]
			if pod.NodeName != "" {
				if node := ctx.Index.NodesByName[pod.NodeName]; node != nil {
					edges = append(edges, CreateEdge(pod, node, "ScheduledOn"))
				}
			}

			saName := pod.ServiceAccount
			if saName == "" {
				saName = "default"
			}
			if saName != "" && saName != "default" {
				for j := range serviceAccounts {
					sa := &serviceAccounts[j]
					if sa.Name == saName {
						edges = append(edges, CreateEdge(pod, sa, "Uses"))
						break
					}
				}
			}

			volumes := pod.Volumes
			if len(volumes) > 0 {
				for _, secret := range secrets {
					for _, volume := range volumes {
						if MapString(volume, "type") != "secret" {
							continue
						}
						if MapString(volume, "secretName") == secret.Name {
							edges = append(edges, CreateEdge(&secret, pod, "MountedBy"))
						}
					}
				}
			}

			containers := pod.Containers
			if len(containers) > 0 {
				for _, secret := range secrets {
					for _, container := range containers {
						for _, envSource := range MapSlice(container, "envFrom") {
							entry, ok := envSource.(map[string]any)
							if !ok {
								continue
							}
							secretRef := MapMap(entry, "secretRef")
							if MapString(secretRef, "name") == secret.Name {
								edges = append(edges, CreateEdge(&secret, pod, "EnvVars"))
							}
						}
					}
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(podEdgesRule{})
}

var _ nodes.EdgeNode = nodes.PodCore{}
