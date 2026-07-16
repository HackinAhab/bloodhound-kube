package model

import (
	"bloodhound-kube/internal/nodes/addons"
	"bloodhound-kube/internal/nodes/mounts"
	"bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

type Namespace struct {
	Pods                   []workload.Pod
	ServiceAccounts        []rbac.ServiceAccount
	Secrets                []workload.Secret
	ConfigMaps             []workload.ConfigMap
	Services               []networking.Service
	Deployments            []workload.Deployment
	DaemonSets             []workload.DaemonSetCore
	StatefulSets           []workload.StatefulSetCore
	Jobs                   []workload.Job
	CronJobs               []workload.CronJob
	NetworkPolicies        []networking.NetworkPolicy
	CiliumNetworkPolicies  []addons.CiliumNetworkPolicy
	Ingresses              []networking.Ingress
	Gateways               []networking.Gateway
	HTTPRoutes             []networking.HTTPRoute
	GRPCRoutes             []networking.GRPCRoute
	TCPRoutes              []networking.TCPRoute
	TLSRoutes              []networking.TLSRoute
	PersistentVolumeClaims []mounts.PersistentVolumeClaim
	Roles                  []rbac.Role
	RoleBindings           []rbac.RoleBinding
	ExternalSecrets        []addons.ExternalSecret
	SecretStores           []addons.SecretStore
	AllPods                []platform.AllPods
	AllSecrets             []platform.AllSecrets
	AllConfigMaps          []platform.AllConfigMaps
	AllServiceAccounts     []platform.AllServiceAccounts
	AllDeployments         []platform.AllDeployments
	AllDaemonSets          []platform.AllDaemonSets
	AllStatefulSets        []platform.AllStatefulSets
	AllJobs                []platform.AllJobs
	AllCronJobs            []platform.AllCronJobs
	AllRoles               []platform.AllRoles
}

type Cluster struct {
	Nodes                      []platform.Node
	AllNodes                   []platform.AllNodes
	ClusterRoles               []rbac.ClusterRole
	ClusterRoleBindings        []rbac.ClusterRoleBinding
	Users                      []rbac.User
	Groups                     []rbac.Group
	PersistentVolumes          []mounts.PersistentVolume
	ClusterSecretStores        []addons.ClusterSecretStore
	SecurityContextConstraints []addons.SecurityContextConstraints
	GlobalNetworkPolicies      []addons.GlobalNetworkPolicy
	External                   []platform.External
	AllPods                    []platform.AllPods
	AllSecrets                 []platform.AllSecrets
	AllConfigMaps              []platform.AllConfigMaps
	AllServiceAccounts         []platform.AllServiceAccounts
	AllUsers                   []platform.AllUsers
	AllGroups                  []platform.AllGroups
	AllDeployments             []platform.AllDeployments
	AllDaemonSets              []platform.AllDaemonSets
	AllStatefulSets            []platform.AllStatefulSets
	AllJobs                    []platform.AllJobs
	AllCronJobs                []platform.AllCronJobs
	AllClusterRoles            []platform.AllClusterRoles
}
