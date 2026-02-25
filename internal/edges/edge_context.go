package edges

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type EdgeContext struct {
	Core  *model.CoreFacts
	Index EdgeIndex
}

type EdgeIndex struct {
	NodesByName                  map[string]*nodes.Node
	ClusterRolesByName           map[string]*nodes.ClusterRole
	ClusterRoleBindingsByName    map[string]*nodes.ClusterRoleBinding
	PersistentVolumesByName      map[string]*nodes.PersistentVolume
	ClusterSecretStoresByName    map[string]*nodes.ClusterSecretStore
	SecurityContextConstraintsBy map[string]*nodes.SecurityContextConstraints
	External                     *nodes.External

	PodsByNamespace            map[string]map[string]*nodes.Pod
	ServiceAccountsByNamespace map[string]map[string]*nodes.ServiceAccount
	SecretsByNamespace         map[string]map[string]*nodes.Secret
	ConfigMapsByNamespace      map[string]map[string]*nodes.ConfigMap
	ServicesByNamespace        map[string]map[string]*nodes.Service
	DeploymentsByNamespace     map[string]map[string]*nodes.Deployment
	NetworkPoliciesByNamespace map[string]map[string]*nodes.NetworkPolicy
	IngressesByNamespace       map[string]map[string]*nodes.Ingress
	HTTPRoutesByNamespace      map[string]map[string]*nodes.HTTPRoute
	PersistentVolumeClaimsByNS map[string]map[string]*nodes.PersistentVolumeClaim
	RolesByNamespace           map[string]map[string]*nodes.Role
	RoleBindingsByNamespace    map[string]map[string]*nodes.RoleBinding
	ExternalSecretsByNamespace map[string]map[string]*nodes.ExternalSecret
	SecretStoresByNamespace    map[string]map[string]*nodes.SecretStore
}

func NewEdgeContext(core *model.CoreFacts) *EdgeContext {
	ctx := &EdgeContext{Core: core}
	ctx.Index = buildEdgeIndex(core)
	return ctx
}

func buildEdgeIndex(core *model.CoreFacts) EdgeIndex {
	index := EdgeIndex{
		NodesByName:                  map[string]*nodes.Node{},
		ClusterRolesByName:           map[string]*nodes.ClusterRole{},
		ClusterRoleBindingsByName:    map[string]*nodes.ClusterRoleBinding{},
		PersistentVolumesByName:      map[string]*nodes.PersistentVolume{},
		ClusterSecretStoresByName:    map[string]*nodes.ClusterSecretStore{},
		SecurityContextConstraintsBy: map[string]*nodes.SecurityContextConstraints{},
		PodsByNamespace:              map[string]map[string]*nodes.Pod{},
		ServiceAccountsByNamespace:   map[string]map[string]*nodes.ServiceAccount{},
		SecretsByNamespace:           map[string]map[string]*nodes.Secret{},
		ConfigMapsByNamespace:        map[string]map[string]*nodes.ConfigMap{},
		ServicesByNamespace:          map[string]map[string]*nodes.Service{},
		DeploymentsByNamespace:       map[string]map[string]*nodes.Deployment{},
		NetworkPoliciesByNamespace:   map[string]map[string]*nodes.NetworkPolicy{},
		IngressesByNamespace:         map[string]map[string]*nodes.Ingress{},
		HTTPRoutesByNamespace:        map[string]map[string]*nodes.HTTPRoute{},
		PersistentVolumeClaimsByNS:   map[string]map[string]*nodes.PersistentVolumeClaim{},
		RolesByNamespace:             map[string]map[string]*nodes.Role{},
		RoleBindingsByNamespace:      map[string]map[string]*nodes.RoleBinding{},
		ExternalSecretsByNamespace:   map[string]map[string]*nodes.ExternalSecret{},
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
	for i := range core.Cluster.PersistentVolumes {
		pv := &core.Cluster.PersistentVolumes[i]
		if pv.Name != "" {
			index.PersistentVolumesByName[pv.Name] = pv
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
		index.PodsByNamespace[ns] = map[string]*nodes.Pod{}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.Name != "" {
				index.PodsByNamespace[ns][pod.Name] = pod
			}
		}
		index.ServiceAccountsByNamespace[ns] = map[string]*nodes.ServiceAccount{}
		for i := range space.ServiceAccounts {
			sa := &space.ServiceAccounts[i]
			if sa.Name != "" {
				index.ServiceAccountsByNamespace[ns][sa.Name] = sa
			}
		}
		index.SecretsByNamespace[ns] = map[string]*nodes.Secret{}
		for i := range space.Secrets {
			secret := &space.Secrets[i]
			if secret.Name != "" {
				index.SecretsByNamespace[ns][secret.Name] = secret
			}
		}
		index.ConfigMapsByNamespace[ns] = map[string]*nodes.ConfigMap{}
		for i := range space.ConfigMaps {
			cm := &space.ConfigMaps[i]
			if cm.Name != "" {
				index.ConfigMapsByNamespace[ns][cm.Name] = cm
			}
		}
		index.ServicesByNamespace[ns] = map[string]*nodes.Service{}
		for i := range space.Services {
			svc := &space.Services[i]
			if svc.Name != "" {
				index.ServicesByNamespace[ns][svc.Name] = svc
			}
		}
		index.DeploymentsByNamespace[ns] = map[string]*nodes.Deployment{}
		for i := range space.Deployments {
			deploy := &space.Deployments[i]
			if deploy.Name != "" {
				index.DeploymentsByNamespace[ns][deploy.Name] = deploy
			}
		}
		index.NetworkPoliciesByNamespace[ns] = map[string]*nodes.NetworkPolicy{}
		for i := range space.NetworkPolicies {
			netpol := &space.NetworkPolicies[i]
			if netpol.Name != "" {
				index.NetworkPoliciesByNamespace[ns][netpol.Name] = netpol
			}
		}
		index.IngressesByNamespace[ns] = map[string]*nodes.Ingress{}
		for i := range space.Ingresses {
			ing := &space.Ingresses[i]
			if ing.Name != "" {
				index.IngressesByNamespace[ns][ing.Name] = ing
			}
		}
		index.HTTPRoutesByNamespace[ns] = map[string]*nodes.HTTPRoute{}
		for i := range space.HTTPRoutes {
			route := &space.HTTPRoutes[i]
			if route.Name != "" {
				index.HTTPRoutesByNamespace[ns][route.Name] = route
			}
		}
		index.PersistentVolumeClaimsByNS[ns] = map[string]*nodes.PersistentVolumeClaim{}
		for i := range space.PersistentVolumeClaims {
			pvc := &space.PersistentVolumeClaims[i]
			if pvc.Name != "" {
				index.PersistentVolumeClaimsByNS[ns][pvc.Name] = pvc
			}
		}
		index.RolesByNamespace[ns] = map[string]*nodes.Role{}
		for i := range space.Roles {
			role := &space.Roles[i]
			if role.Name != "" {
				index.RolesByNamespace[ns][role.Name] = role
			}
		}
		index.RoleBindingsByNamespace[ns] = map[string]*nodes.RoleBinding{}
		for i := range space.RoleBindings {
			binding := &space.RoleBindings[i]
			if binding.Name != "" {
				index.RoleBindingsByNamespace[ns][binding.Name] = binding
			}
		}
		index.ExternalSecretsByNamespace[ns] = map[string]*nodes.ExternalSecret{}
		for i := range space.ExternalSecrets {
			es := &space.ExternalSecrets[i]
			if es.Name != "" {
				index.ExternalSecretsByNamespace[ns][es.Name] = es
			}
		}
		index.SecretStoresByNamespace[ns] = map[string]*nodes.SecretStore{}
		for i := range space.SecretStores {
			store := &space.SecretStores[i]
			if store.Name != "" {
				index.SecretStoresByNamespace[ns][store.Name] = store
			}
		}
	}

	return index
}
