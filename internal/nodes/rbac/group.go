package rbac

import fw "bloodhound-kube/internal/nodes/framework"

type Group struct {
	fw.GraphNodeBase
}

func BuildGroupNode(name string) (fw.BuildResult, bool) {
	base, properties, ok := buildIdentityBase("BHK_Group", name)
	if !ok {
		return fw.BuildResult{}, false
	}

	core := fw.CoreEntry{
		Cluster: true,
		Data:    Group{GraphNodeBase: base},
	}

	return fw.BuildResult{
		Node: fw.NodeResult{
			ID:         base.ID,
			Kinds:      base.Kinds,
			Properties: properties,
		},
		Core: []fw.CoreEntry{core},
	}, true
}
