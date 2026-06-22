package parser

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
	"fmt"
	"sort"
)

func enrichPodNodesWithControllerEnv(nodeList []model.BloodHoundNode, coreFacts *model.CoreFacts) {
	if coreFacts == nil {
		return
	}

	podNodesByID := map[string]*model.BloodHoundNode{}
	for i := range nodeList {
		node := &nodeList[i]
		if len(node.Kinds) == 0 || node.Kinds[0] != "BHK_Pod" {
			continue
		}
		podNodesByID[node.ID] = node
	}

	for ns, space := range coreFacts.Namespaces {
		if space == nil {
			continue
		}

		for i := range space.Pods {
			pod := &space.Pods[i]
			podID := nodefw.BuildID("BHK_Pod", ns, pod.Name)
			node := podNodesByID[podID]
			if node == nil || node.Properties == nil {
				continue
			}

			controllerDefs := collectControllerEnvDefinitions(pod, space)
			combined := make([]workload.EnvDefinition, 0, len(pod.EnvDefinitions)+len(controllerDefs))
			combined = append(combined, pod.EnvDefinitions...)
			combined = append(combined, controllerDefs...)
			envProps := buildEnvProperties(combined)
			for k, v := range envProps {
				node.Properties[k] = v
			}
		}
	}
}

func buildEnvProperties(defs []workload.EnvDefinition) map[string]any {
	properties := map[string]any{}
	if len(defs) == 0 {
		return properties
	}

	defs = dedupeEnvDefinitions(defs)

	sorted := make([]workload.EnvDefinition, 0, len(defs))
	sorted = append(sorted, defs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := envDefinitionSortKey(sorted[i])
		right := envDefinitionSortKey(sorted[j])
		return left < right
	})

	for i, def := range sorted {
		key := fmt.Sprintf("Env%02d", i)
		properties[key] = envDefinitionTokens(def)
	}

	return properties
}

func dedupeEnvDefinitions(defs []workload.EnvDefinition) []workload.EnvDefinition {
	if len(defs) == 0 {
		return defs
	}
	bestByKey := map[string]workload.EnvDefinition{}
	for _, def := range defs {
		key := fmt.Sprintf("%s|%s|%s|%s|%s|%s", def.Container, def.ValueSourceType, def.EnvName, def.RefName, def.RefKey, def.Value)
		existing, exists := bestByKey[key]
		if !exists || preferSource(def.SourceKind, existing.SourceKind) {
			bestByKey[key] = def
		}
	}

	result := make([]workload.EnvDefinition, 0, len(bestByKey))
	for _, def := range bestByKey {
		result = append(result, def)
	}
	return result
}

func preferSource(candidate, current string) bool {
	if candidate == current {
		return false
	}
	if candidate == "Pod" {
		return false
	}
	if current == "Pod" {
		return true
	}
	return false
}

func envDefinitionSortKey(def workload.EnvDefinition) string {
	return fmt.Sprintf(
		"%s|%s|%s|%t|%s|%s|%s|%s|%s|%s",
		def.SourceKind,
		def.SourceName,
		def.Container,
		def.InitContainer,
		def.ValueSourceType,
		def.EnvName,
		def.RefName,
		def.RefKey,
		def.Value,
		def.SourcePath,
	)
}

func envDefinitionTokens(def workload.EnvDefinition) []string {
	tokens := []string{
		fmt.Sprintf("src=%s/%s", def.SourceKind, def.SourceName),
		fmt.Sprintf("container=%s", def.Container),
		fmt.Sprintf("type=%s", def.ValueSourceType),
	}
	if def.EnvName != "" {
		tokens = append(tokens, fmt.Sprintf("env=%s", def.EnvName))
	}
	if def.RefName != "" {
		tokens = append(tokens, fmt.Sprintf("ref=%s", def.RefName))
	}
	if def.RefKey != "" {
		tokens = append(tokens, fmt.Sprintf("key=%s", def.RefKey))
	}
	if def.ValueSourceType == "literal" {
		tokens = append(tokens, fmt.Sprintf("value=%s", def.Value))
	}
	return tokens
}

func collectControllerEnvDefinitions(pod *workload.Pod, space *model.Namespace) []workload.EnvDefinition {
	if pod == nil || space == nil {
		return []workload.EnvDefinition{}
	}

	defs := make([]workload.EnvDefinition, 0)
	for i := range space.Deployments {
		deploy := &space.Deployments[i]
		if framework.LabelsMatchOnly(pod.LabelsMap, deploy.SelectorLabels) {
			defs = append(defs, deploy.EnvDefinitions...)
		}
	}
	for i := range space.DaemonSets {
		daemonSet := &space.DaemonSets[i]
		if framework.LabelsMatchOnly(pod.LabelsMap, daemonSet.SelectorLabels) {
			defs = append(defs, daemonSet.EnvDefinitions...)
		}
	}
	for i := range space.StatefulSets {
		statefulSet := &space.StatefulSets[i]
		if framework.LabelsMatchOnly(pod.LabelsMap, statefulSet.SelectorLabels) {
			defs = append(defs, statefulSet.EnvDefinitions...)
		}
	}
	for i := range space.Jobs {
		job := &space.Jobs[i]
		if len(job.SelectorLabels) == 0 {
			continue
		}
		if framework.LabelsMatchOnly(pod.LabelsMap, job.SelectorLabels) {
			defs = append(defs, job.EnvDefinitions...)
		}
	}
	for i := range space.CronJobs {
		cronJob := &space.CronJobs[i]
		if len(cronJob.SelectorLabels) == 0 {
			continue
		}
		if framework.LabelsMatchOnly(pod.LabelsMap, cronJob.SelectorLabels) {
			defs = append(defs, cronJob.EnvDefinitions...)
		}
	}

	return defs
}
