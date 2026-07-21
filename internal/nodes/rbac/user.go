package rbac

import fw "bloodhound-kube/internal/nodes/framework"

type User struct {
	fw.GraphNodeBase
}

// buildIdentityBase builds the shared GraphNodeBase and properties for a
// cluster-scoped identity node (User, Group). Returns ok=false for an empty name.
func buildIdentityBase(kind, name string) (fw.GraphNodeBase, map[string]any, bool) {
	if name == "" {
		return fw.GraphNodeBase{}, nil, false
	}
	base := fw.GraphNodeBase{
		ID:        fw.BuildID(kind, "", name),
		Kinds:     []string{kind, "BHK_Identity"},
		Name:      name,
		Namespace: "",
	}
	return base, map[string]any{"name": name}, true
}

func BuildUserNode(name string) (fw.BuildResult, bool) {
	base, properties, ok := buildIdentityBase("BHK_User", name)
	if !ok {
		return fw.BuildResult{}, false
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
