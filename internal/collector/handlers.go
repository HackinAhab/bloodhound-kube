package collector

import (
	"bloodhound-kube/internal/utils"
)

// AllHandlers defines all available resource handlers with their metadata
// This is the single source of truth for all collectors in the system.
// To add a new collector:
// 1. Add the collection function to collectors.go
// 2. Add a metadata entry here
var AllHandlers = []HandlerMetadata{
	{
		Name:                  "nodes",
		ResourceType:          "node",
		Description:           "Collect Kubernetes nodes",
		ClusterScoped:         true,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectNodes,
	},
	{
		Name:                  "secrets",
		ResourceType:          "secret",
		Description:           "Collect Kubernetes secrets",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectSecrets,
	},
	{
		Name:                  "configmaps",
		ResourceType:          "configmap",
		Description:           "Collect Kubernetes configmaps",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectConfigMaps,
	},
	{
		Name:                  "deployments",
		ResourceType:          "deployment",
		Description:           "Collect Kubernetes deployments",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectDeployments,
	},
	{
		Name:                  "services",
		ResourceType:          "service",
		Description:           "Collect Kubernetes services",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectServices,
	},
	{
		Name:                  "ingresses",
		ResourceType:          "ingress",
		Description:           "Collect Kubernetes ingresses",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectIngresses,
	},
	{
		Name:                  "gateways",
		ResourceType:          "gateway",
		Description:           "Collect Kubernetes gateways",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectGateways,
	},
	{
		Name:                  "rbac",
		ResourceType:          "rbac",
		Description:           "Collect Kubernetes RBAC resources",
		ClusterScoped:         true,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectRBAC,
	},
	{
		Name:                  "networkpolicies",
		ResourceType:          "networkpolicy",
		Description:           "Collect Kubernetes network policies",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectNetworkPolicies,
	},
	{
		Name:                  "crds",
		ResourceType:          "crd",
		Description:           "Collect Kubernetes custom resource definitions",
		ClusterScoped:         true,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectCRDs,
	},
	{
		Name:                  "daemonsets",
		ResourceType:          "daemonset",
		Description:           "Collect Kubernetes daemonsets",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectDaemonSets,
	},
	{
		Name:                  "statefulsets",
		ResourceType:          "statefulset",
		Description:           "Collect Kubernetes statefulsets",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectStatefulSets,
	},
	// OpenShift-specific handlers
	{
		Name:                  "routes",
		ResourceType:          "route",
		Description:           "Collect OpenShift routes",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeOpenShift},
		CollectFunc:           collectRoutes,
	},
	{
		Name:                  "projects",
		ResourceType:          "project",
		Description:           "Collect OpenShift projects",
		ClusterScoped:         true,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeOpenShift},
		CollectFunc:           collectProjects,
	},
	{
		Name:                  "images",
		ResourceType:          "image",
		Description:           "Collect OpenShift images",
		ClusterScoped:         true,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeOpenShift},
		CollectFunc:           collectImages,
	},
	// Example: Adding a new collector - just add this metadata entry!
	{
		Name:                  "pods",
		ResourceType:          "pod",
		Description:           "Collect Kubernetes pods",
		ClusterScoped:         false,
		SupportedClusterTypes: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
		CollectFunc:           collectPods,
	},
}
