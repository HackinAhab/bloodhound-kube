package model

import (
	"bloodhound-kube/internal/nodes/addons/externalsecrets"
	"bloodhound-kube/internal/nodes/addons/scc"
	"bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

type EdgeIndex struct {
	NodesByName                  map[string]*platform.Node
	ClusterRolesByName           map[string]*rbac.ClusterRole
	ClusterRoleBindingsByName    map[string]*rbac.ClusterRoleBinding
	ClusterSecretStoresByName    map[string]*externalsecrets.ClusterSecretStore
	SecurityContextConstraintsBy map[string]*scc.SecurityContextConstraints
	UsersByName                  map[string]*rbac.User
	GroupsByName                 map[string]*rbac.Group
	External                     *platform.External

	PodsByNamespace            map[string]map[string]*workload.Pod
	ServiceAccountsByNamespace map[string]map[string]*rbac.ServiceAccount
	SecretsByNamespace         map[string]map[string]*workload.Secret
	ConfigMapsByNamespace      map[string]map[string]*workload.ConfigMap
	ServicesByNamespace        map[string]map[string]*networking.Service
	RolesByNamespace           map[string]map[string]*rbac.Role
	RoleBindingsByNamespace    map[string]map[string]*rbac.RoleBinding
	SecretStoresByNamespace    map[string]map[string]*externalsecrets.SecretStore
}

func NewEdgeIndex(core *CoreFacts) EdgeIndex {
	index := EdgeIndex{
		NodesByName:                  map[string]*platform.Node{},
		ClusterRolesByName:           map[string]*rbac.ClusterRole{},
		ClusterRoleBindingsByName:    map[string]*rbac.ClusterRoleBinding{},
		ClusterSecretStoresByName:    map[string]*externalsecrets.ClusterSecretStore{},
		SecurityContextConstraintsBy: map[string]*scc.SecurityContextConstraints{},
		UsersByName:                  map[string]*rbac.User{},
		GroupsByName:                 map[string]*rbac.Group{},
		PodsByNamespace:              map[string]map[string]*workload.Pod{},
		ServiceAccountsByNamespace:   map[string]map[string]*rbac.ServiceAccount{},
		SecretsByNamespace:           map[string]map[string]*workload.Secret{},
		ConfigMapsByNamespace:        map[string]map[string]*workload.ConfigMap{},
		ServicesByNamespace:          map[string]map[string]*networking.Service{},
		RolesByNamespace:             map[string]map[string]*rbac.Role{},
		RoleBindingsByNamespace:      map[string]map[string]*rbac.RoleBinding{},
		SecretStoresByNamespace:      map[string]map[string]*externalsecrets.SecretStore{},
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
	for i := range core.Cluster.Users {
		user := &core.Cluster.Users[i]
		if user.Name != "" {
			index.UsersByName[user.Name] = user
		}
	}
	for i := range core.Cluster.Groups {
		group := &core.Cluster.Groups[i]
		if group.Name != "" {
			index.GroupsByName[group.Name] = group
		}
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
