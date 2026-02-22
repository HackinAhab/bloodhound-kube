package model

import "bloodhound-kube/internal/nodes"

type NamespaceCore struct {
	Pods                   []nodes.PodCore
	ServiceAccounts        []nodes.ServiceAccountCore
	Secrets                []nodes.SecretCore
	ConfigMaps             []nodes.ConfigMapCore
	Services               []nodes.ServiceCore
	Deployments            []nodes.DeploymentCore
	DaemonSets             []nodes.DaemonSetCore
	StatefulSets           []nodes.StatefulSetCore
	NetworkPolicies        []nodes.NetworkPolicyCore
	Ingresses              []nodes.IngressCore
	HTTPRoutes             []nodes.HTTPRouteCore
	PersistentVolumeClaims []nodes.PersistentVolumeClaimCore
	Roles                  []nodes.RoleCore
	RoleBindings           []nodes.RoleBindingCore
	ExternalSecrets        []nodes.ExternalSecretCore
	SecretStores           []nodes.SecretStoreCore
}

type ClusterCore struct {
	Nodes                      []nodes.NodeCore
	ClusterRoles               []nodes.ClusterRoleCore
	ClusterRoleBindings        []nodes.ClusterRoleBindingCore
	PersistentVolumes          []nodes.PersistentVolumeCore
	ClusterSecretStores        []nodes.ClusterSecretStoreCore
	SecurityContextConstraints []nodes.SecurityContextConstraintsCore
	External                   []nodes.ExternalCore
}
