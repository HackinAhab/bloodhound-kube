package framework

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type Context struct {
	Core  *model.CoreFacts
	Index EdgeIndex
}

type EdgeIndex struct {
	NodesByName                  map[string]*nodes.Node
	ClusterRolesByName           map[string]*nodes.ClusterRole
	ClusterRoleBindingsByName    map[string]*nodes.ClusterRoleBinding
	ClusterSecretStoresByName    map[string]*nodes.ClusterSecretStore
	SecurityContextConstraintsBy map[string]*nodes.SecurityContextConstraints
	External                     *nodes.External

	PodsByNamespace            map[string]map[string]*nodes.Pod
	ServiceAccountsByNamespace map[string]map[string]*nodes.ServiceAccount
	SecretsByNamespace         map[string]map[string]*nodes.Secret
	ConfigMapsByNamespace      map[string]map[string]*nodes.ConfigMap
	ServicesByNamespace        map[string]map[string]*nodes.Service
	DeploymentsByNamespace     map[string]map[string]*nodes.Deployment
	DaemonSetsByNamespace      map[string]map[string]*nodes.DaemonSetCore
	StatefulSetsByNamespace    map[string]map[string]*nodes.StatefulSetCore
	JobsByNamespace            map[string]map[string]*nodes.Job
	CronJobsByNamespace        map[string]map[string]*nodes.CronJob
	RolesByNamespace           map[string]map[string]*nodes.Role
	RoleBindingsByNamespace    map[string]map[string]*nodes.RoleBinding
	SecretStoresByNamespace    map[string]map[string]*nodes.SecretStore
}

func NewContext(core *model.CoreFacts) *Context {
	ctx := &Context{Core: core}
	ctx.Index = buildEdgeIndex(core)
	return ctx
}

func buildEdgeIndex(core *model.CoreFacts) EdgeIndex {
	index := EdgeIndex{
		NodesByName:                  map[string]*nodes.Node{},
		ClusterRolesByName:           map[string]*nodes.ClusterRole{},
		ClusterRoleBindingsByName:    map[string]*nodes.ClusterRoleBinding{},
		ClusterSecretStoresByName:    map[string]*nodes.ClusterSecretStore{},
		SecurityContextConstraintsBy: map[string]*nodes.SecurityContextConstraints{},
		PodsByNamespace:              map[string]map[string]*nodes.Pod{},
		ServiceAccountsByNamespace:   map[string]map[string]*nodes.ServiceAccount{},
		SecretsByNamespace:           map[string]map[string]*nodes.Secret{},
		ConfigMapsByNamespace:        map[string]map[string]*nodes.ConfigMap{},
		ServicesByNamespace:          map[string]map[string]*nodes.Service{},
		DeploymentsByNamespace:       map[string]map[string]*nodes.Deployment{},
		DaemonSetsByNamespace:        map[string]map[string]*nodes.DaemonSetCore{},
		StatefulSetsByNamespace:      map[string]map[string]*nodes.StatefulSetCore{},
		JobsByNamespace:              map[string]map[string]*nodes.Job{},
		CronJobsByNamespace:          map[string]map[string]*nodes.CronJob{},
		RolesByNamespace:             map[string]map[string]*nodes.Role{},
		RoleBindingsByNamespace:      map[string]map[string]*nodes.RoleBinding{},
		SecretStoresByNamespace:      map[string]map[string]*nodes.SecretStore{},
	}

	if core == nil {
		return index
	}

	for i := range core.Cluster.Nodes {
		node := &core.Cluster.Nodes[i]
		if node.Name != "" {
			index.NodesByName[node.Name] = node
		}
	}
	for i := range core.Cluster.ClusterRoles {
		role := &core.Cluster.ClusterRoles[i]
		if role.Name != "" {
			index.ClusterRolesByName[role.Name] = role
		}
	}
	for i := range core.Cluster.ClusterRoleBindings {
		binding := &core.Cluster.ClusterRoleBindings[i]
		if binding.Name != "" {
			index.ClusterRoleBindingsByName[binding.Name] = binding
		}
	}
	for i := range core.Cluster.ClusterSecretStores {
		store := &core.Cluster.ClusterSecretStores[i]
		if store.Name != "" {
			index.ClusterSecretStoresByName[store.Name] = store
		}
	}
	for i := range core.Cluster.SecurityContextConstraints {
		scc := &core.Cluster.SecurityContextConstraints[i]
		if scc.Name != "" {
			index.SecurityContextConstraintsBy[scc.Name] = scc
		}
	}
	if len(core.Cluster.External) > 0 {
		index.External = &core.Cluster.External[0]
	}

	for ns, space := range core.Namespaces {
		if space == nil {
			continue
		}
		index.PodsByNamespace[ns] = indexByName(space.Pods)
		index.ServiceAccountsByNamespace[ns] = indexByName(space.ServiceAccounts)
		index.SecretsByNamespace[ns] = indexByName(space.Secrets)
		index.ConfigMapsByNamespace[ns] = indexByName(space.ConfigMaps)
		index.ServicesByNamespace[ns] = indexByName(space.Services)
		index.DeploymentsByNamespace[ns] = indexByName(space.Deployments)
		index.DaemonSetsByNamespace[ns] = indexByName(space.DaemonSets)
		index.StatefulSetsByNamespace[ns] = indexByName(space.StatefulSets)
		index.JobsByNamespace[ns] = indexByName(space.Jobs)
		index.CronJobsByNamespace[ns] = indexByName(space.CronJobs)
		index.RolesByNamespace[ns] = indexByName(space.Roles)
		index.RoleBindingsByNamespace[ns] = indexByName(space.RoleBindings)
		index.SecretStoresByNamespace[ns] = indexByName(space.SecretStores)
	}

	return index
}

type namedNode interface {
	EdgeName() string
}

func indexByName[T namedNode](items []T) map[string]*T {
	idx := map[string]*T{}
	for i := range items {
		item := items[i]
		if item.EdgeName() != "" {
			idx[item.EdgeName()] = &items[i]
		}
	}
	return idx
}
