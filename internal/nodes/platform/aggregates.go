package platform

import fw "bloodhound-kube/internal/nodes/framework"

type AllPods struct {
	fw.GraphNodeBase
}

type AllSecrets struct {
	fw.GraphNodeBase
}

type AllConfigMaps struct {
	fw.GraphNodeBase
}

type AllNodes struct {
	fw.GraphNodeBase
}

type AllServiceAccounts struct {
	fw.GraphNodeBase
}

type AllDeployments struct {
	fw.GraphNodeBase
}

type AllDaemonSets struct {
	fw.GraphNodeBase
}

type AllStatefulSets struct {
	fw.GraphNodeBase
}

type AllJobs struct {
	fw.GraphNodeBase
}

type AllCronJobs struct {
	fw.GraphNodeBase
}

type AllClusterRoles struct {
	fw.GraphNodeBase
}

type AllRoles struct {
	fw.GraphNodeBase
}

func BuildAllPods() fw.BuildResult {
	return buildAggregate("AllPods", func(base fw.GraphNodeBase) any {
		return AllPods{GraphNodeBase: base}
	})
}

func BuildAllSecrets() fw.BuildResult {
	return buildAggregate("AllSecrets", func(base fw.GraphNodeBase) any {
		return AllSecrets{GraphNodeBase: base}
	})
}

func BuildAllConfigMaps() fw.BuildResult {
	return buildAggregate("AllConfigMaps", func(base fw.GraphNodeBase) any {
		return AllConfigMaps{GraphNodeBase: base}
	})
}

func BuildAllNodes() fw.BuildResult {
	return buildAggregate("AllNodes", func(base fw.GraphNodeBase) any {
		return AllNodes{GraphNodeBase: base}
	})
}

func BuildAllServiceAccounts() fw.BuildResult {
	return buildAggregate("AllServiceAccounts", func(base fw.GraphNodeBase) any {
		return AllServiceAccounts{GraphNodeBase: base}
	})
}

func BuildAllDeployments() fw.BuildResult {
	return buildAggregate("AllDeployments", func(base fw.GraphNodeBase) any {
		return AllDeployments{GraphNodeBase: base}
	})
}

func BuildAllDaemonSets() fw.BuildResult {
	return buildAggregate("AllDaemonSets", func(base fw.GraphNodeBase) any {
		return AllDaemonSets{GraphNodeBase: base}
	})
}

func BuildAllStatefulSets() fw.BuildResult {
	return buildAggregate("AllStatefulSets", func(base fw.GraphNodeBase) any {
		return AllStatefulSets{GraphNodeBase: base}
	})
}

func BuildAllJobs() fw.BuildResult {
	return buildAggregate("AllJobs", func(base fw.GraphNodeBase) any {
		return AllJobs{GraphNodeBase: base}
	})
}

func BuildAllCronJobs() fw.BuildResult {
	return buildAggregate("AllCronJobs", func(base fw.GraphNodeBase) any {
		return AllCronJobs{GraphNodeBase: base}
	})
}

func BuildAllClusterRoles() fw.BuildResult {
	return buildAggregate("AllClusterRoles", func(base fw.GraphNodeBase) any {
		return AllClusterRoles{GraphNodeBase: base}
	})
}

func buildAggregate(kind string, data func(fw.GraphNodeBase) any) fw.BuildResult {
	base := fw.GraphNodeBase{
		ID:        fw.BuildID(kind, "", kind),
		Kinds:     []string{kind},
		Name:      kind,
		Namespace: "",
	}
	properties := map[string]any{
		"name":      kind,
		"namespace": "",
	}
	core := fw.CoreEntry{
		Cluster: true,
		Data:    data(base),
	}
	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         base.ID,
			Kinds:      base.Kinds,
			Properties: properties,
		},
		Core: []fw.CoreEntry{core},
	}
}

// buildNamespaceAggregate builds a per-namespace aggregate node. The kind label
// matches the cluster aggregate (e.g. "AllSecrets") so a single BloodHound
// query like MATCH (n:AllSecrets) works
func buildNamespaceAggregate(kind, namespace string, data func(fw.GraphNodeBase) any) fw.BuildResult {
	base := fw.GraphNodeBase{
		ID:        fw.BuildID(kind, namespace, kind),
		Kinds:     []string{kind},
		Name:      kind,
		Namespace: namespace,
	}
	properties := map[string]any{
		"name":      kind,
		"namespace": namespace,
	}
	core := fw.CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data:      data(base),
	}
	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         base.ID,
			Kinds:      base.Kinds,
			Properties: properties,
		},
		Core: []fw.CoreEntry{core},
	}
}

func BuildAllPodsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllPods", namespace, func(base fw.GraphNodeBase) any {
		return AllPods{GraphNodeBase: base}
	})
}

func BuildAllSecretsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllSecrets", namespace, func(base fw.GraphNodeBase) any {
		return AllSecrets{GraphNodeBase: base}
	})
}

func BuildAllConfigMapsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllConfigMaps", namespace, func(base fw.GraphNodeBase) any {
		return AllConfigMaps{GraphNodeBase: base}
	})
}

func BuildAllServiceAccountsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllServiceAccounts", namespace, func(base fw.GraphNodeBase) any {
		return AllServiceAccounts{GraphNodeBase: base}
	})
}

func BuildAllDeploymentsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllDeployments", namespace, func(base fw.GraphNodeBase) any {
		return AllDeployments{GraphNodeBase: base}
	})
}

func BuildAllDaemonSetsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllDaemonSets", namespace, func(base fw.GraphNodeBase) any {
		return AllDaemonSets{GraphNodeBase: base}
	})
}

func BuildAllStatefulSetsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllStatefulSets", namespace, func(base fw.GraphNodeBase) any {
		return AllStatefulSets{GraphNodeBase: base}
	})
}

func BuildAllJobsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllJobs", namespace, func(base fw.GraphNodeBase) any {
		return AllJobs{GraphNodeBase: base}
	})
}

func BuildAllCronJobsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllCronJobs", namespace, func(base fw.GraphNodeBase) any {
		return AllCronJobs{GraphNodeBase: base}
	})
}

func BuildAllRolesNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("AllRoles", namespace, func(base fw.GraphNodeBase) any {
		return AllRoles{GraphNodeBase: base}
	})
}
