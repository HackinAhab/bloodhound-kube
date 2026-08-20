package model

import (
	"bloodhound-kube/internal/nodes/addons/calico"
	"bloodhound-kube/internal/nodes/addons/certmanager"
	"bloodhound-kube/internal/nodes/addons/cilium"
	"bloodhound-kube/internal/nodes/addons/externalsecrets"
	"bloodhound-kube/internal/nodes/addons/istio"
	"bloodhound-kube/internal/nodes/addons/route"
	"bloodhound-kube/internal/nodes/addons/scc"
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
	CiliumNetworkPolicies  []cilium.CiliumNetworkPolicy
	Ingresses              []networking.Ingress
	Routes                 []route.Route
	Gateways               []networking.Gateway
	HTTPRoutes             []networking.HTTPRoute
	GRPCRoutes             []networking.GRPCRoute
	TCPRoutes              []networking.TCPRoute
	TLSRoutes              []networking.TLSRoute
	PersistentVolumeClaims []mounts.PersistentVolumeClaim
	Roles                  []rbac.Role
	RoleBindings           []rbac.RoleBinding
	ExternalSecrets        []externalsecrets.ExternalSecret
	SecretStores           []externalsecrets.SecretStore
	Certificates           []certmanager.Certificate
	Issuers                []certmanager.Issuer
	IstioGateways          []istio.IstioGateway
	VirtualServices        []istio.VirtualService
	PeerAuthentications    []istio.PeerAuthentication
	AuthorizationPolicies  []istio.AuthorizationPolicy
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
	ClusterSecretStores        []externalsecrets.ClusterSecretStore
	ClusterIssuers             []certmanager.ClusterIssuer
	SecurityContextConstraints []scc.SecurityContextConstraints
	GlobalNetworkPolicies      []calico.GlobalNetworkPolicy
	HostEndpoints              []calico.HostEndpoint
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
