package model

import "bloodhound-kube/internal/nodes"

type CoreFacts struct {
	Namespaces map[string]*Namespace
	Cluster    *Cluster
}

func NewCoreFacts() *CoreFacts {
	return &CoreFacts{
		Namespaces: map[string]*Namespace{},
		Cluster:    &Cluster{},
	}
}

func (c *CoreFacts) Add(entry nodes.CoreEntry) {
	if entry.Data == nil {
		return
	}
	if entry.Cluster {
		switch v := entry.Data.(type) {
		case nodes.Node:
			c.Cluster.Nodes = append(c.Cluster.Nodes, v)
		case nodes.ClusterRole:
			c.Cluster.ClusterRoles = append(c.Cluster.ClusterRoles, v)
		case nodes.ClusterRoleBinding:
			c.Cluster.ClusterRoleBindings = append(c.Cluster.ClusterRoleBindings, v)
		case nodes.PersistentVolume:
			c.Cluster.PersistentVolumes = append(c.Cluster.PersistentVolumes, v)
		case nodes.ClusterSecretStore:
			c.Cluster.ClusterSecretStores = append(c.Cluster.ClusterSecretStores, v)
		case nodes.SecurityContextConstraints:
			c.Cluster.SecurityContextConstraints = append(c.Cluster.SecurityContextConstraints, v)
		case nodes.External:
			c.Cluster.External = append(c.Cluster.External, v)
		case nodes.AllNodes:
			c.Cluster.AllNodes = append(c.Cluster.AllNodes, v)
		case nodes.AllPods:
			c.Cluster.AllPods = append(c.Cluster.AllPods, v)
		case nodes.AllSecrets:
			c.Cluster.AllSecrets = append(c.Cluster.AllSecrets, v)
		case nodes.AllServiceAccounts:
			c.Cluster.AllServiceAccounts = append(c.Cluster.AllServiceAccounts, v)
		case nodes.AllDeployments:
			c.Cluster.AllDeployments = append(c.Cluster.AllDeployments, v)
		case nodes.AllDaemonSets:
			c.Cluster.AllDaemonSets = append(c.Cluster.AllDaemonSets, v)
		case nodes.AllStatefulSets:
			c.Cluster.AllStatefulSets = append(c.Cluster.AllStatefulSets, v)
		}
		return
	}

	ns := entry.Namespace
	if c.Namespaces[ns] == nil {
		c.Namespaces[ns] = &Namespace{}
	}
	space := c.Namespaces[ns]
	switch v := entry.Data.(type) {
	case nodes.Pod:
		space.Pods = append(space.Pods, v)
	case nodes.ServiceAccount:
		space.ServiceAccounts = append(space.ServiceAccounts, v)
	case nodes.Secret:
		space.Secrets = append(space.Secrets, v)
	case nodes.ConfigMap:
		space.ConfigMaps = append(space.ConfigMaps, v)
	case nodes.Service:
		space.Services = append(space.Services, v)
	case nodes.Deployment:
		space.Deployments = append(space.Deployments, v)
	case nodes.DaemonSetCore:
		space.DaemonSets = append(space.DaemonSets, v)
	case nodes.StatefulSetCore:
		space.StatefulSets = append(space.StatefulSets, v)
	case nodes.NetworkPolicy:
		space.NetworkPolicies = append(space.NetworkPolicies, v)
	case nodes.Ingress:
		space.Ingresses = append(space.Ingresses, v)
	case nodes.HTTPRoute:
		space.HTTPRoutes = append(space.HTTPRoutes, v)
	case nodes.TCPRoute:
		space.TCPRoutes = append(space.TCPRoutes, v)
	case nodes.TLSRoute:
		space.TLSRoutes = append(space.TLSRoutes, v)
	case nodes.PersistentVolumeClaim:
		space.PersistentVolumeClaims = append(space.PersistentVolumeClaims, v)
	case nodes.Role:
		space.Roles = append(space.Roles, v)
	case nodes.RoleBinding:
		space.RoleBindings = append(space.RoleBindings, v)
	case nodes.ExternalSecret:
		space.ExternalSecrets = append(space.ExternalSecrets, v)
	case nodes.SecretStore:
		space.SecretStores = append(space.SecretStores, v)
	}
}
