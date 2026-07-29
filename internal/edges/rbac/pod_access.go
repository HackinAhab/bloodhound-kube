package rbac

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

var rbacPodExecEdgesRule = simpleAccessRule[workload.Pod, platform.AllPods]{
	name:         "rbac_pod_exec",
	resourceKeys: []string{"pods/exec"},
	verbs:        []string{"create"},
	edgeKind:     "BHK_PodExec",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to exec into pods.",
		"Reference":   "https://kubehound.io/reference/attacks/POD_EXEC/",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Pod { return space.Pods },
	namespacedAll:     func(space *model.Namespace) []platform.AllPods { return space.AllPods },
	clusterAll:        func(c *model.Cluster) []platform.AllPods { return c.AllPods },
}

var rbacPodPortForwardEdgesRule = simpleAccessRule[workload.Pod, platform.AllPods]{
	name:         "rbac_pod_portforward",
	resourceKeys: []string{"pods/portforward"},
	verbs:        []string{"create"},
	edgeKind:     "BHK_PodPortForward",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to port-forward to pods, allowing TCP tunneling to any port on any pod in scope.",
		"Reference":   "https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Pod { return space.Pods },
	namespacedAll:     func(space *model.Namespace) []platform.AllPods { return space.AllPods },
	clusterAll:        func(c *model.Cluster) []platform.AllPods { return c.AllPods },
}

var rbacPodAttachEdgesRule = simpleAccessRule[workload.Pod, platform.AllPods]{
	name:         "rbac_pod_attach",
	resourceKeys: []string{"pods/attach"},
	verbs:        []string{"create"},
	edgeKind:     "BHK_PodAttach",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to attach to pods (kubectl attach), allowing interaction with running container stdin/stdout.",
		"Reference":   "https://kubehound.io/reference/attacks/POD_ATTACH/",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Pod { return space.Pods },
	namespacedAll:     func(space *model.Namespace) []platform.AllPods { return space.AllPods },
	clusterAll:        func(c *model.Cluster) []platform.AllPods { return c.AllPods },
}

var rbacPodDebugEdgesRule = simpleAccessRule[workload.Pod, platform.AllPods]{
	name:         "rbac_pod_debug",
	resourceKeys: []string{"pods/ephemeralcontainers"},
	verbs:        []string{"update"},
	edgeKind:     "BHK_PodDebug",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to debug pods.",
		"Reference":   "https://kubehound.io/reference/attacks/POD_ATTACH/",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Pod { return space.Pods },
	namespacedAll:     func(space *model.Namespace) []platform.AllPods { return space.AllPods },
	clusterAll:        func(c *model.Cluster) []platform.AllPods { return c.AllPods },
}
