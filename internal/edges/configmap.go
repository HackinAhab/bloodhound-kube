package edges

import "bloodhound-kube/internal/model"

type configMapEdgesRule struct{}

func (r configMapEdgesRule) Name() string {
	return "configmap"
}

func (r configMapEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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
					if MapString(volume, "type") == "configmap" && MapString(volume, "configMapName") == cm.Name {
						edges = append(edges, CreateEdge(cm, pod, "MountedBy"))
					}
				}
				for _, container := range pod.Containers {
					for _, envSource := range MapSlice(container, "envFrom") {
						entry, ok := envSource.(map[string]any)
						if !ok {
							continue
						}
						configMapRef := MapMap(entry, "configMapRef")
						if MapString(configMapRef, "name") == cm.Name {
							edges = append(edges, CreateEdge(cm, pod, "EnvVars"))
						}
					}
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(configMapEdgesRule{})
}
