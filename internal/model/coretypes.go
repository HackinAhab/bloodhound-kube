package model

import "bloodhound-kube/internal/nodes"

type Namespace struct {
	Pods                   []nodes.Pod
	ServiceAccounts        []nodes.ServiceAccount
	Secrets                []nodes.Secret
	ConfigMaps             []nodes.ConfigMap
	Services               []nodes.Service
	Deployments            []nodes.Deployment
	DaemonSets             []nodes.DaemonSetCore
	StatefulSets           []nodes.StatefulSetCore
	Jobs                   []nodes.Job
	CronJobs               []nodes.CronJob
	NetworkPolicies        []nodes.NetworkPolicy
	Ingresses              []nodes.Ingress
	Gateways               []nodes.Gateway
	HTTPRoutes             []nodes.HTTPRoute
	GRPCRoutes             []nodes.GRPCRoute
	TCPRoutes              []nodes.TCPRoute
	TLSRoutes              []nodes.TLSRoute
	PersistentVolumeClaims []nodes.PersistentVolumeClaim
	Roles                  []nodes.Role
	RoleBindings           []nodes.RoleBinding
	ExternalSecrets        []nodes.ExternalSecret
	SecretStores           []nodes.SecretStore
}

type Cluster struct {
	Nodes                      []nodes.Node
	AllNodes                   []nodes.AllNodes
	ClusterRoles               []nodes.ClusterRole
	ClusterRoleBindings        []nodes.ClusterRoleBinding
	PersistentVolumes          []nodes.PersistentVolume
	ClusterSecretStores        []nodes.ClusterSecretStore
	SecurityContextConstraints []nodes.SecurityContextConstraints
	External                   []nodes.External
	AllPods                    []nodes.AllPods
	AllSecrets                 []nodes.AllSecrets
	AllConfigMaps              []nodes.AllConfigMaps
	AllServiceAccounts         []nodes.AllServiceAccounts
	AllDeployments             []nodes.AllDeployments
	AllDaemonSets              []nodes.AllDaemonSets
	AllStatefulSets            []nodes.AllStatefulSets
	AllJobs                    []nodes.AllJobs
	AllCronJobs                []nodes.AllCronJobs
}
