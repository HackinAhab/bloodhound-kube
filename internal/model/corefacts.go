package model

import "bloodhound-kube/internal/nodes"

type CoreFacts struct {
	Namespaces map[string]*NamespaceCore
	Cluster    *ClusterCore
}

func NewCoreFacts() *CoreFacts {
	return &CoreFacts{
		Namespaces: map[string]*NamespaceCore{},
		Cluster:    &ClusterCore{},
	}
}

func (c *CoreFacts) Add(entry nodes.CoreEntry) {
	if entry.Data == nil {
		return
	}
	if entry.Cluster {
		switch v := entry.Data.(type) {
		case nodes.NodeCore:
			c.Cluster.Nodes = append(c.Cluster.Nodes, v)
		case nodes.ClusterRoleCore:
			c.Cluster.ClusterRoles = append(c.Cluster.ClusterRoles, v)
		case nodes.ClusterRoleBindingCore:
			c.Cluster.ClusterRoleBindings = append(c.Cluster.ClusterRoleBindings, v)
		case nodes.PersistentVolumeCore:
			c.Cluster.PersistentVolumes = append(c.Cluster.PersistentVolumes, v)
		case nodes.ClusterSecretStoreCore:
			c.Cluster.ClusterSecretStores = append(c.Cluster.ClusterSecretStores, v)
		case nodes.SecurityContextConstraintsCore:
			c.Cluster.SecurityContextConstraints = append(c.Cluster.SecurityContextConstraints, v)
		case nodes.ExternalCore:
			c.Cluster.External = append(c.Cluster.External, v)
		}
		return
	}

	ns := entry.Namespace
	if c.Namespaces[ns] == nil {
		c.Namespaces[ns] = &NamespaceCore{}
	}
	space := c.Namespaces[ns]
	switch v := entry.Data.(type) {
	case nodes.PodCore:
		space.Pods = append(space.Pods, v)
	case nodes.ServiceAccountCore:
		space.ServiceAccounts = append(space.ServiceAccounts, v)
	case nodes.SecretCore:
		space.Secrets = append(space.Secrets, v)
	case nodes.ConfigMapCore:
		space.ConfigMaps = append(space.ConfigMaps, v)
	case nodes.ServiceCore:
		space.Services = append(space.Services, v)
	case nodes.DeploymentCore:
		space.Deployments = append(space.Deployments, v)
	case nodes.DaemonSetCore:
		space.DaemonSets = append(space.DaemonSets, v)
	case nodes.StatefulSetCore:
		space.StatefulSets = append(space.StatefulSets, v)
	case nodes.NetworkPolicyCore:
		space.NetworkPolicies = append(space.NetworkPolicies, v)
	case nodes.IngressCore:
		space.Ingresses = append(space.Ingresses, v)
	case nodes.HTTPRouteCore:
		space.HTTPRoutes = append(space.HTTPRoutes, v)
	case nodes.PersistentVolumeClaimCore:
		space.PersistentVolumeClaims = append(space.PersistentVolumeClaims, v)
	case nodes.RoleCore:
		space.Roles = append(space.Roles, v)
	case nodes.RoleBindingCore:
		space.RoleBindings = append(space.RoleBindings, v)
	case nodes.ExternalSecretCore:
		space.ExternalSecrets = append(space.ExternalSecrets, v)
	case nodes.SecretStoreCore:
		space.SecretStores = append(space.SecretStores, v)
	}
}
