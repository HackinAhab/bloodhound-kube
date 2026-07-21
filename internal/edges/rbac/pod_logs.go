package rbac

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

var rbacReadLogsEdgesRule = simpleAccessRule[workload.Pod, platform.AllPods]{
	name:         "rbac_read_logs",
	resourceKeys: []string{"pods/log"},
	verbs:        []string{"get", "list", "watch"},
	edgeKind:     "BHK_ReadLogs",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to read pod logs.",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Pod { return space.Pods },
	namespacedAll:     func(space *model.Namespace) []platform.AllPods { return space.AllPods },
	clusterAll:        func(c *model.Cluster) []platform.AllPods { return c.AllPods },
}
