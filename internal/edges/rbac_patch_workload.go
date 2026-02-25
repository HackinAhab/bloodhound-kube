package edges

import "bloodhound-kube/internal/model"

// SA w/ patch on workload -> Workloads that can be patched by the SA
func workloadPatchNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	verbs := []string{"patch", "update"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		var perms []string
		switch binding.RoleKind {
		case "Role":
			roleIndex := ctx.Index.RolesByNamespace[namespace]
			if roleIndex == nil {
				continue
			}
			role := roleIndex[binding.RoleName]
			if role == nil {
				continue
			}
			perms = role.PermsDisplay
		case "ClusterRole":
			clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
			if clusterRole == nil {
				continue
			}
			perms = clusterRole.PermsDisplay
		default:
			continue
		}

		allPods, podNames := accessForResource(perms, []string{"pods"}, verbs)
		allDeployments, deploymentNames := accessForResource(perms, []string{"deployments", "apps/deployments"}, verbs)
		allDaemonSets, daemonSetNames := accessForResource(perms, []string{"daemonsets", "apps/daemonsets"}, verbs)
		allStatefulSets, statefulSetNames := accessForResource(perms, []string{"statefulsets", "apps/statefulsets"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 {
			continue
		}

		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" {
				continue
			}
			subjectNS := subject.Namespace
			if subjectNS == "" {
				subjectNS = binding.Namespace
			}
			if subjectNS != namespace {
				continue
			}
			saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}

			if allPods || len(podNames) > 0 {
				for i := range space.Pods {
					pod := &space.Pods[i]
					if allPods {
						edges = append(edges, CreateEdge(sa, pod, "WorkloadPatch"))
						continue
					}
					if _, ok := podNames[pod.Name]; ok {
						edges = append(edges, CreateEdge(sa, pod, "WorkloadPatch"))
					}
				}
			}

			if allDeployments || len(deploymentNames) > 0 {
				for i := range space.Deployments {
					deployment := &space.Deployments[i]
					if allDeployments {
						edges = append(edges, CreateEdge(sa, deployment, "WorkloadPatch"))
						continue
					}
					if _, ok := deploymentNames[deployment.Name]; ok {
						edges = append(edges, CreateEdge(sa, deployment, "WorkloadPatch"))
					}
				}
			}

			if allDaemonSets || len(daemonSetNames) > 0 {
				for i := range space.DaemonSets {
					daemonSet := &space.DaemonSets[i]
					if allDaemonSets {
						edges = append(edges, CreateEdge(sa, daemonSet, "WorkloadPatch"))
						continue
					}
					if _, ok := daemonSetNames[daemonSet.Name]; ok {
						edges = append(edges, CreateEdge(sa, daemonSet, "WorkloadPatch"))
					}
				}
			}

			if allStatefulSets || len(statefulSetNames) > 0 {
				for i := range space.StatefulSets {
					statefulSet := &space.StatefulSets[i]
					if allStatefulSets {
						edges = append(edges, CreateEdge(sa, statefulSet, "WorkloadPatch"))
						continue
					}
					if _, ok := statefulSetNames[statefulSet.Name]; ok {
						edges = append(edges, CreateEdge(sa, statefulSet, "WorkloadPatch"))
					}
				}
			}
		}
	}
	return edges
}

func workloadPatchCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	verbs := []string{"patch", "update"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}

		allPods, podNames := accessForResource(clusterRole.PermsDisplay, []string{"pods"}, verbs)
		allDeployments, deploymentNames := accessForResource(clusterRole.PermsDisplay, []string{"deployments", "apps/deployments"}, verbs)
		allDaemonSets, daemonSetNames := accessForResource(clusterRole.PermsDisplay, []string{"daemonsets", "apps/daemonsets"}, verbs)
		allStatefulSets, statefulSetNames := accessForResource(clusterRole.PermsDisplay, []string{"statefulsets", "apps/statefulsets"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 {
			continue
		}

		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" || subject.Namespace == "" {
				continue
			}
			saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				if allPods || len(podNames) > 0 {
					for i := range space.Pods {
						pod := &space.Pods[i]
						if allPods {
							edges = append(edges, CreateEdge(sa, pod, "WorkloadPatch"))
							continue
						}
						if _, ok := podNames[pod.Name]; ok {
							edges = append(edges, CreateEdge(sa, pod, "WorkloadPatch"))
						}
					}
				}

				if allDeployments || len(deploymentNames) > 0 {
					for i := range space.Deployments {
						deployment := &space.Deployments[i]
						if allDeployments {
							edges = append(edges, CreateEdge(sa, deployment, "WorkloadPatch"))
							continue
						}
						if _, ok := deploymentNames[deployment.Name]; ok {
							edges = append(edges, CreateEdge(sa, deployment, "WorkloadPatch"))
						}
					}
				}

				if allDaemonSets || len(daemonSetNames) > 0 {
					for i := range space.DaemonSets {
						daemonSet := &space.DaemonSets[i]
						if allDaemonSets {
							edges = append(edges, CreateEdge(sa, daemonSet, "WorkloadPatch"))
							continue
						}
						if _, ok := daemonSetNames[daemonSet.Name]; ok {
							edges = append(edges, CreateEdge(sa, daemonSet, "WorkloadPatch"))
						}
					}
				}

				if allStatefulSets || len(statefulSetNames) > 0 {
					for i := range space.StatefulSets {
						statefulSet := &space.StatefulSets[i]
						if allStatefulSets {
							edges = append(edges, CreateEdge(sa, statefulSet, "WorkloadPatch"))
							continue
						}
						if _, ok := statefulSetNames[statefulSet.Name]; ok {
							edges = append(edges, CreateEdge(sa, statefulSet, "WorkloadPatch"))
						}
					}
				}
			}
		}
	}
	return edges
}
