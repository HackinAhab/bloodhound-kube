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

type AllUsers struct {
	fw.GraphNodeBase
}

type AllGroups struct {
	fw.GraphNodeBase
}

func BuildAllPods() fw.BuildResult {
	return buildAggregate("BHK_AllPods", func(base fw.GraphNodeBase) any {
		return AllPods{GraphNodeBase: base}
	})
}

func BuildAllSecrets() fw.BuildResult {
	return buildAggregate("BHK_AllSecrets", func(base fw.GraphNodeBase) any {
		return AllSecrets{GraphNodeBase: base}
	})
}

func BuildAllConfigMaps() fw.BuildResult {
	return buildAggregate("BHK_AllConfigMaps", func(base fw.GraphNodeBase) any {
		return AllConfigMaps{GraphNodeBase: base}
	})
}

func BuildAllNodes() fw.BuildResult {
	return buildAggregate("BHK_AllNodes", func(base fw.GraphNodeBase) any {
		return AllNodes{GraphNodeBase: base}
	})
}

func BuildAllServiceAccounts() fw.BuildResult {
	return buildAggregate("BHK_AllServiceAccounts", func(base fw.GraphNodeBase) any {
		return AllServiceAccounts{GraphNodeBase: base}
	})
}

func BuildAllDeployments() fw.BuildResult {
	return buildAggregate("BHK_AllDeployments", func(base fw.GraphNodeBase) any {
		return AllDeployments{GraphNodeBase: base}
	})
}

func BuildAllDaemonSets() fw.BuildResult {
	return buildAggregate("BHK_AllDaemonSets", func(base fw.GraphNodeBase) any {
		return AllDaemonSets{GraphNodeBase: base}
	})
}

func BuildAllStatefulSets() fw.BuildResult {
	return buildAggregate("BHK_AllStatefulSets", func(base fw.GraphNodeBase) any {
		return AllStatefulSets{GraphNodeBase: base}
	})
}

func BuildAllJobs() fw.BuildResult {
	return buildAggregate("BHK_AllJobs", func(base fw.GraphNodeBase) any {
		return AllJobs{GraphNodeBase: base}
	})
}

func BuildAllCronJobs() fw.BuildResult {
	return buildAggregate("BHK_AllCronJobs", func(base fw.GraphNodeBase) any {
		return AllCronJobs{GraphNodeBase: base}
	})
}

func BuildAllClusterRoles() fw.BuildResult {
	return buildAggregate("BHK_AllClusterRoles", func(base fw.GraphNodeBase) any {
		return AllClusterRoles{GraphNodeBase: base}
	})
}

func BuildAllUsers() fw.BuildResult {
	return buildAggregate("BHK_AllUsers", func(base fw.GraphNodeBase) any {
		return AllUsers{GraphNodeBase: base}
	})
}

func BuildAllGroups() fw.BuildResult {
	return buildAggregate("BHK_AllGroups", func(base fw.GraphNodeBase) any {
		return AllGroups{GraphNodeBase: base}
	})
}

func buildAggregate(kind string, data func(fw.GraphNodeBase) any) fw.BuildResult {
	base := fw.GraphNodeBase{
		ID:        fw.BuildID(kind, "", kind),
		Kinds:     []string{kind, "BHK_Aggregate"},
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
	displayName := kind + "[" + namespace + "]"
	base := fw.GraphNodeBase{
		ID:        fw.BuildID(kind, namespace, kind),
		Kinds:     []string{kind, "BHK_Aggregate"},
		Name:      displayName,
		Namespace: namespace,
	}
	properties := map[string]any{
		"name":      displayName,
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
	return buildNamespaceAggregate("BHK_AllPods", namespace, func(base fw.GraphNodeBase) any {
		return AllPods{GraphNodeBase: base}
	})
}

func BuildAllSecretsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllSecrets", namespace, func(base fw.GraphNodeBase) any {
		return AllSecrets{GraphNodeBase: base}
	})
}

func BuildAllConfigMapsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllConfigMaps", namespace, func(base fw.GraphNodeBase) any {
		return AllConfigMaps{GraphNodeBase: base}
	})
}

func BuildAllServiceAccountsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllServiceAccounts", namespace, func(base fw.GraphNodeBase) any {
		return AllServiceAccounts{GraphNodeBase: base}
	})
}

func BuildAllDeploymentsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllDeployments", namespace, func(base fw.GraphNodeBase) any {
		return AllDeployments{GraphNodeBase: base}
	})
}

func BuildAllDaemonSetsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllDaemonSets", namespace, func(base fw.GraphNodeBase) any {
		return AllDaemonSets{GraphNodeBase: base}
	})
}

func BuildAllStatefulSetsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllStatefulSets", namespace, func(base fw.GraphNodeBase) any {
		return AllStatefulSets{GraphNodeBase: base}
	})
}

func BuildAllJobsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllJobs", namespace, func(base fw.GraphNodeBase) any {
		return AllJobs{GraphNodeBase: base}
	})
}

func BuildAllCronJobsNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllCronJobs", namespace, func(base fw.GraphNodeBase) any {
		return AllCronJobs{GraphNodeBase: base}
	})
}

func BuildAllRolesNS(namespace string) fw.BuildResult {
	return buildNamespaceAggregate("BHK_AllRoles", namespace, func(base fw.GraphNodeBase) any {
		return AllRoles{GraphNodeBase: base}
	})
}
