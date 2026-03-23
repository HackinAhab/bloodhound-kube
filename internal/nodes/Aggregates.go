package nodes

type AllPods struct {
	GraphNodeBase
}

type AllSecrets struct {
	GraphNodeBase
}

type AllNodes struct {
	GraphNodeBase
}

type AllServiceAccounts struct {
	GraphNodeBase
}

type AllDeployments struct {
	GraphNodeBase
}

type AllDaemonSets struct {
	GraphNodeBase
}

type AllStatefulSets struct {
	GraphNodeBase
}

type AllJobs struct {
	GraphNodeBase
}

type AllCronJobs struct {
	GraphNodeBase
}

func BuildAllPods() BuildResult {
	return buildAggregate("AllPods", func(base GraphNodeBase) any {
		return AllPods{GraphNodeBase: base}
	})
}

func BuildAllSecrets() BuildResult {
	return buildAggregate("AllSecrets", func(base GraphNodeBase) any {
		return AllSecrets{GraphNodeBase: base}
	})
}

func BuildAllNodes() BuildResult {
	return buildAggregate("AllNodes", func(base GraphNodeBase) any {
		return AllNodes{GraphNodeBase: base}
	})
}

func BuildAllServiceAccounts() BuildResult {
	return buildAggregate("AllServiceAccounts", func(base GraphNodeBase) any {
		return AllServiceAccounts{GraphNodeBase: base}
	})
}

func BuildAllDeployments() BuildResult {
	return buildAggregate("AllDeployments", func(base GraphNodeBase) any {
		return AllDeployments{GraphNodeBase: base}
	})
}

func BuildAllDaemonSets() BuildResult {
	return buildAggregate("AllDaemonSets", func(base GraphNodeBase) any {
		return AllDaemonSets{GraphNodeBase: base}
	})
}

func BuildAllStatefulSets() BuildResult {
	return buildAggregate("AllStatefulSets", func(base GraphNodeBase) any {
		return AllStatefulSets{GraphNodeBase: base}
	})
}

func BuildAllJobs() BuildResult {
	return buildAggregate("AllJobs", func(base GraphNodeBase) any {
		return AllJobs{GraphNodeBase: base}
	})
}

func BuildAllCronJobs() BuildResult {
	return buildAggregate("AllCronJobs", func(base GraphNodeBase) any {
		return AllCronJobs{GraphNodeBase: base}
	})
}

func buildAggregate(kind string, data func(GraphNodeBase) any) BuildResult {
	base := GraphNodeBase{
		ID:        BuildID(kind, "", kind),
		Kinds:     []string{kind},
		Name:      kind,
		Namespace: "",
	}
	properties := map[string]any{
		"name":      kind,
		"namespace": "",
	}
	core := CoreEntry{
		Cluster: true,
		Data:    data(base),
	}
	return BuildResult{
		Node: NodeResult{
			ID:         base.ID,
			Kinds:      base.Kinds,
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}
}
