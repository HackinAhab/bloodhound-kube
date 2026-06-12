package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacPatchWorkloadEdgesRule struct{}

func (r rbacPatchWorkloadEdgesRule) Name() string { return "rbac_patch_workload" }

var edgePropertiesRBACWorkloadPatch = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to patch workloads, which can allow for modification of running workloads.",
	"Reference":   "https://kubehound.io/reference/attacks/POD_PATCH/",
}

func (r rbacPatchWorkloadEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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

func workloadPatchNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	verbs := []string{"patch", "update"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		parsed := parseRBACPerms(perms)

		allPods, podNames := accessForParsedResource(parsed, []string{"pods"}, verbs)
		allDeployments, deploymentNames := accessForParsedResource(parsed, []string{"deployments", "apps/deployments"}, verbs)
		allDaemonSets, daemonSetNames := accessForParsedResource(parsed, []string{"daemonsets", "apps/daemonsets"}, verbs)
		allStatefulSets, statefulSetNames := accessForParsedResource(parsed, []string{"statefulsets", "apps/statefulsets"}, verbs)
		allJobs, jobNames := accessForParsedResource(parsed, []string{"jobs", "batch/jobs"}, verbs)
		allCronJobs, cronJobNames := accessForParsedResource(parsed, []string{"cronjobs", "batch/cronjobs"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 && !allJobs && len(jobNames) == 0 && !allCronJobs && len(cronJobNames) == 0 {
			continue
		}

		for _, subject := range binding.Subjects {
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}

			if allPods || len(podNames) > 0 {
				for i := range space.Pods {
					pod := &space.Pods[i]
					if allPods || hasName(podNames, pod.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, pod, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
			if allDeployments || len(deploymentNames) > 0 {
				for i := range space.Deployments {
					deployment := &space.Deployments[i]
					if allDeployments || hasName(deploymentNames, deployment.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, deployment, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
			if allDaemonSets || len(daemonSetNames) > 0 {
				for i := range space.DaemonSets {
					daemonSet := &space.DaemonSets[i]
					if allDaemonSets || hasName(daemonSetNames, daemonSet.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, daemonSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
			if allStatefulSets || len(statefulSetNames) > 0 {
				for i := range space.StatefulSets {
					statefulSet := &space.StatefulSets[i]
					if allStatefulSets || hasName(statefulSetNames, statefulSet.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, statefulSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
			if allJobs || len(jobNames) > 0 {
				for i := range space.Jobs {
					job := &space.Jobs[i]
					if allJobs || hasName(jobNames, job.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, job, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
			if allCronJobs || len(cronJobNames) > 0 {
				for i := range space.CronJobs {
					cronJob := &space.CronJobs[i]
					if allCronJobs || hasName(cronJobNames, cronJob.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, cronJob, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
					}
				}
			}
		}
	}
	return edges
}

func workloadPatchCluster(ctx *framework.Context) []model.BloodHoundEdge {
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
		parsed := parseRBACPerms(clusterRole.PermsDisplay)

		allPods, podNames := accessForParsedResource(parsed, []string{"pods"}, verbs)
		allDeployments, deploymentNames := accessForParsedResource(parsed, []string{"deployments", "apps/deployments"}, verbs)
		allDaemonSets, daemonSetNames := accessForParsedResource(parsed, []string{"daemonsets", "apps/daemonsets"}, verbs)
		allStatefulSets, statefulSetNames := accessForParsedResource(parsed, []string{"statefulsets", "apps/statefulsets"}, verbs)
		allJobs, jobNames := accessForParsedResource(parsed, []string{"jobs", "batch/jobs"}, verbs)
		allCronJobs, cronJobNames := accessForParsedResource(parsed, []string{"cronjobs", "batch/cronjobs"}, verbs)
		if !allPods && len(podNames) == 0 && !allDeployments && len(deploymentNames) == 0 && !allDaemonSets && len(daemonSetNames) == 0 && !allStatefulSets && len(statefulSetNames) == 0 && !allJobs && len(jobNames) == 0 && !allCronJobs && len(cronJobNames) == 0 {
			continue
		}

		for _, subject := range binding.Subjects {
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if allPods {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedPods(ctx, sa, podNames)...)
			}
			if allDeployments {
				if len(ctx.Core.Cluster.AllDeployments) > 0 {
					agg := &ctx.Core.Cluster.AllDeployments[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedDeployments(ctx, sa, deploymentNames)...)
			}
			if allDaemonSets {
				if len(ctx.Core.Cluster.AllDaemonSets) > 0 {
					agg := &ctx.Core.Cluster.AllDaemonSets[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedDaemonSets(ctx, sa, daemonSetNames)...)
			}
			if allStatefulSets {
				if len(ctx.Core.Cluster.AllStatefulSets) > 0 {
					agg := &ctx.Core.Cluster.AllStatefulSets[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedStatefulSets(ctx, sa, statefulSetNames)...)
			}
			if allJobs {
				if len(ctx.Core.Cluster.AllJobs) > 0 {
					agg := &ctx.Core.Cluster.AllJobs[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedJobs(ctx, sa, jobNames)...)
			}
			if allCronJobs {
				if len(ctx.Core.Cluster.AllCronJobs) > 0 {
					agg := &ctx.Core.Cluster.AllCronJobs[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
				}
			} else {
				edges = append(edges, appendNamedCronJobs(ctx, sa, cronJobNames)...)
			}
		}
	}
	return edges
}
