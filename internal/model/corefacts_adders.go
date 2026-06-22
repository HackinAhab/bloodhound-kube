package model

import (
	"bloodhound-kube/internal/nodes/addons"
	"bloodhound-kube/internal/nodes/mounts"
	"bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
	"reflect"
)

type clusterFactAdder func(*CoreFacts, any)
type namespacedFactAdder func(*Namespace, any)

var clusterFactAdders = map[reflect.Type]clusterFactAdder{}
var namespacedFactAdders = map[reflect.Type]namespacedFactAdder{}

func init() {
	registerClusterFactAdder(func(c *CoreFacts, v platform.Node) { c.Cluster.Nodes = append(c.Cluster.Nodes, v) })
	registerClusterFactAdder(func(c *CoreFacts, v rbac.ClusterRole) { c.Cluster.ClusterRoles = append(c.Cluster.ClusterRoles, v) })
	registerClusterFactAdder(func(c *CoreFacts, v rbac.ClusterRoleBinding) {
		c.Cluster.ClusterRoleBindings = append(c.Cluster.ClusterRoleBindings, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v mounts.PersistentVolume) {
		c.Cluster.PersistentVolumes = append(c.Cluster.PersistentVolumes, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v addons.ClusterSecretStore) {
		c.Cluster.ClusterSecretStores = append(c.Cluster.ClusterSecretStores, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v addons.SecurityContextConstraints) {
		c.Cluster.SecurityContextConstraints = append(c.Cluster.SecurityContextConstraints, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.External) { c.Cluster.External = append(c.Cluster.External, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllNodes) { c.Cluster.AllNodes = append(c.Cluster.AllNodes, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllPods) { c.Cluster.AllPods = append(c.Cluster.AllPods, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllSecrets) { c.Cluster.AllSecrets = append(c.Cluster.AllSecrets, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllConfigMaps) {
		c.Cluster.AllConfigMaps = append(c.Cluster.AllConfigMaps, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllServiceAccounts) {
		c.Cluster.AllServiceAccounts = append(c.Cluster.AllServiceAccounts, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllDeployments) {
		c.Cluster.AllDeployments = append(c.Cluster.AllDeployments, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllDaemonSets) {
		c.Cluster.AllDaemonSets = append(c.Cluster.AllDaemonSets, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllStatefulSets) {
		c.Cluster.AllStatefulSets = append(c.Cluster.AllStatefulSets, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllJobs) { c.Cluster.AllJobs = append(c.Cluster.AllJobs, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllCronJobs) { c.Cluster.AllCronJobs = append(c.Cluster.AllCronJobs, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllClusterRoles) {
		c.Cluster.AllClusterRoles = append(c.Cluster.AllClusterRoles, v)
	})
	registerClusterFactAdder(func(c *CoreFacts, v rbac.User) { c.Cluster.Users = append(c.Cluster.Users, v) })
	registerClusterFactAdder(func(c *CoreFacts, v rbac.Group) { c.Cluster.Groups = append(c.Cluster.Groups, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllUsers) { c.Cluster.AllUsers = append(c.Cluster.AllUsers, v) })
	registerClusterFactAdder(func(c *CoreFacts, v platform.AllGroups) { c.Cluster.AllGroups = append(c.Cluster.AllGroups, v) })

	registerNamespacedFactAdder(func(ns *Namespace, v workload.Pod) { ns.Pods = append(ns.Pods, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v rbac.ServiceAccount) {
		ns.ServiceAccounts = append(ns.ServiceAccounts, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v workload.Secret) { ns.Secrets = append(ns.Secrets, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v workload.ConfigMap) { ns.ConfigMaps = append(ns.ConfigMaps, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.Service) { ns.Services = append(ns.Services, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v workload.Deployment) { ns.Deployments = append(ns.Deployments, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v workload.DaemonSetCore) { ns.DaemonSets = append(ns.DaemonSets, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v workload.StatefulSetCore) {
		ns.StatefulSets = append(ns.StatefulSets, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v workload.Job) { ns.Jobs = append(ns.Jobs, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v workload.CronJob) { ns.CronJobs = append(ns.CronJobs, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.NetworkPolicy) {
		ns.NetworkPolicies = append(ns.NetworkPolicies, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v networking.Ingress) { ns.Ingresses = append(ns.Ingresses, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.Gateway) { ns.Gateways = append(ns.Gateways, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.HTTPRoute) { ns.HTTPRoutes = append(ns.HTTPRoutes, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.GRPCRoute) { ns.GRPCRoutes = append(ns.GRPCRoutes, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.TCPRoute) { ns.TCPRoutes = append(ns.TCPRoutes, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v networking.TLSRoute) { ns.TLSRoutes = append(ns.TLSRoutes, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v mounts.PersistentVolumeClaim) {
		ns.PersistentVolumeClaims = append(ns.PersistentVolumeClaims, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v rbac.Role) { ns.Roles = append(ns.Roles, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v rbac.RoleBinding) { ns.RoleBindings = append(ns.RoleBindings, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v addons.ExternalSecret) {
		ns.ExternalSecrets = append(ns.ExternalSecrets, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v addons.SecretStore) { ns.SecretStores = append(ns.SecretStores, v) })

	// Per-namespace aggregate adders. The same struct types are also registered
	// as cluster adders above; CoreFacts.Add selects the map via entry.Cluster
	// before performing the type lookup, so both registrations coexist safely.
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllPods) { ns.AllPods = append(ns.AllPods, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllSecrets) { ns.AllSecrets = append(ns.AllSecrets, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllConfigMaps) {
		ns.AllConfigMaps = append(ns.AllConfigMaps, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllServiceAccounts) {
		ns.AllServiceAccounts = append(ns.AllServiceAccounts, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllDeployments) {
		ns.AllDeployments = append(ns.AllDeployments, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllDaemonSets) {
		ns.AllDaemonSets = append(ns.AllDaemonSets, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllStatefulSets) {
		ns.AllStatefulSets = append(ns.AllStatefulSets, v)
	})
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllJobs) { ns.AllJobs = append(ns.AllJobs, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllCronJobs) { ns.AllCronJobs = append(ns.AllCronJobs, v) })
	registerNamespacedFactAdder(func(ns *Namespace, v platform.AllRoles) { ns.AllRoles = append(ns.AllRoles, v) })
}

func registerClusterFactAdder[T any](adder func(*CoreFacts, T)) {
	var zero T
	clusterFactAdders[reflect.TypeOf(zero)] = func(c *CoreFacts, data any) {
		adder(c, data.(T))
	}
}

func registerNamespacedFactAdder[T any](adder func(*Namespace, T)) {
	var zero T
	namespacedFactAdders[reflect.TypeOf(zero)] = func(ns *Namespace, data any) {
		adder(ns, data.(T))
	}
}
