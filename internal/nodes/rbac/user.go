package rbac

import fw "bloodhound-kube/internal/nodes/framework"

type User struct {
	fw.GraphNodeBase
}

func BuildUserNode(name string) (fw.BuildResult, bool) {
	if name == "" {
		return fw.BuildResult{}, false
	}

	base := fw.GraphNodeBase{
		ID:        fw.BuildID("BHK_User", "", name),
		Kinds:     []string{"BHK_User", "BHK_Identity"},
		Name:      name,
		Namespace: "",
	}

	properties := map[string]any{
		"name": name,
	}

	core := fw.CoreEntry{
		Cluster: true,
		Data:    User{GraphNodeBase: base},
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
