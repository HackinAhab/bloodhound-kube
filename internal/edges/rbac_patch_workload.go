package edges

import "bloodhound-kube/internal/model"

type rbacPatchWorkloadEdgesRule struct{}

func init() {
	RegisterEdgeRule(rbacPatchWorkloadEdgesRule{})
}

func (r rbacPatchWorkloadEdgesRule) Name() string {
	return "rbac_patch_workload"
}

var edgePropertiesRBACWorkloadPatch = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to patch workloads, which can allow for modification of running workloads.",
	"Reference":   "https://kubehound.io/reference/attacks/POD_PATCH/",
}

func (r rbacPatchWorkloadEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, workloadPatchNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, workloadPatchCluster(ctx)...)
	return edges
}

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
		allJobs, jobNames := accessForResource(perms, []string{"jobs", "batch/jobs"}, verbs)
		allCronJobs, cronJobNames := accessForResource(perms, []string{"cronjobs", "batch/cronjobs"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 && !allJobs && len(jobNames) == 0 && !allCronJobs && len(cronJobNames) == 0 {
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
						edges = append(edges, CreateEdgeWithProperties(sa, pod, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := podNames[pod.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, pod, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}

			if allDeployments || len(deploymentNames) > 0 {
				for i := range space.Deployments {
					deployment := &space.Deployments[i]
					if allDeployments {
						edges = append(edges, CreateEdgeWithProperties(sa, deployment, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := deploymentNames[deployment.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, deployment, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}

			if allDaemonSets || len(daemonSetNames) > 0 {
				for i := range space.DaemonSets {
					daemonSet := &space.DaemonSets[i]
					if allDaemonSets {
						edges = append(edges, CreateEdgeWithProperties(sa, daemonSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := daemonSetNames[daemonSet.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, daemonSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}

			if allStatefulSets || len(statefulSetNames) > 0 {
				for i := range space.StatefulSets {
					statefulSet := &space.StatefulSets[i]
					if allStatefulSets {
						edges = append(edges, CreateEdgeWithProperties(sa, statefulSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := statefulSetNames[statefulSet.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, statefulSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}

			if allJobs || len(jobNames) > 0 {
				for i := range space.Jobs {
					job := &space.Jobs[i]
					if allJobs {
						edges = append(edges, CreateEdgeWithProperties(sa, job, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := jobNames[job.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, job, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}

			if allCronJobs || len(cronJobNames) > 0 {
				for i := range space.CronJobs {
					cronJob := &space.CronJobs[i]
					if allCronJobs {
						edges = append(edges, CreateEdgeWithProperties(sa, cronJob, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						continue
					}
					if _, ok := cronJobNames[cronJob.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, cronJob, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
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
		allJobs, jobNames := accessForResource(clusterRole.PermsDisplay, []string{"jobs", "batch/jobs"}, verbs)
		allCronJobs, cronJobNames := accessForResource(clusterRole.PermsDisplay, []string{"cronjobs", "batch/cronjobs"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 && !allJobs && len(jobNames) == 0 && !allCronJobs && len(cronJobNames) == 0 {
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
			if allPods {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(podNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Pods {
						pod := &space.Pods[i]
						if _, ok := podNames[pod.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, pod, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}

			if allDeployments {
				if len(ctx.Core.Cluster.AllDeployments) > 0 {
					agg := &ctx.Core.Cluster.AllDeployments[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(deploymentNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Deployments {
						deployment := &space.Deployments[i]
						if _, ok := deploymentNames[deployment.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, deployment, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}

			if allDaemonSets {
				if len(ctx.Core.Cluster.AllDaemonSets) > 0 {
					agg := &ctx.Core.Cluster.AllDaemonSets[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(daemonSetNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.DaemonSets {
						daemonSet := &space.DaemonSets[i]
						if _, ok := daemonSetNames[daemonSet.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, daemonSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}

			if allStatefulSets {
				if len(ctx.Core.Cluster.AllStatefulSets) > 0 {
					agg := &ctx.Core.Cluster.AllStatefulSets[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(statefulSetNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.StatefulSets {
						statefulSet := &space.StatefulSets[i]
						if _, ok := statefulSetNames[statefulSet.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, statefulSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}

			if allJobs {
				if len(ctx.Core.Cluster.AllJobs) > 0 {
					agg := &ctx.Core.Cluster.AllJobs[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(jobNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Jobs {
						job := &space.Jobs[i]
						if _, ok := jobNames[job.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, job, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}

			if allCronJobs {
				if len(ctx.Core.Cluster.AllCronJobs) > 0 {
					agg := &ctx.Core.Cluster.AllCronJobs[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else if len(cronJobNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.CronJobs {
						cronJob := &space.CronJobs[i]
						if _, ok := cronJobNames[cronJob.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, cronJob, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
						}
					}
				}
			}
		}
	}
	return edges
}
