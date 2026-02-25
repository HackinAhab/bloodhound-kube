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
	NetworkPolicies        []nodes.NetworkPolicy
	Ingresses              []nodes.Ingress
	HTTPRoutes             []nodes.HTTPRoute
	PersistentVolumeClaims []nodes.PersistentVolumeClaim
	Roles                  []nodes.Role
	RoleBindings           []nodes.RoleBinding
	ExternalSecrets        []nodes.ExternalSecret
	SecretStores           []nodes.SecretStore
}

type Cluster struct {
	Nodes                      []nodes.Node
	ClusterRoles               []nodes.ClusterRole
	ClusterRoleBindings        []nodes.ClusterRoleBinding
	PersistentVolumes          []nodes.PersistentVolume
	ClusterSecretStores        []nodes.ClusterSecretStore
	SecurityContextConstraints []nodes.SecurityContextConstraints
	External                   []nodes.External
}
