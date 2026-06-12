package mounts

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	mountnodes "bloodhound-kube/internal/nodes/mounts"
	"bloodhound-kube/internal/nodes/platform"
	rbacnodes "bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

func base(kind, namespace, name string) nodefw.GraphNodeBase {
	return nodefw.NewGraphNodeBase(kind, namespace, name, nil, nil)
}

func newCore() *model.CoreFacts {
	return &model.CoreFacts{
		Namespaces: map[string]*model.Namespace{},
		Cluster:    &model.Cluster{},
	}
}

func ensureNamespace(core *model.CoreFacts, namespace string) *model.Namespace {
	if core.Namespaces[namespace] == nil {
		core.Namespaces[namespace] = &model.Namespace{}
	}
	return core.Namespaces[namespace]
}

func hasEdge(edges []model.BloodHoundEdge, startID, endID, kind string) bool {
	for _, edge := range edges {
		if edge.Kind == kind && edge.Start.Value == startID && edge.End.Value == endID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// HostMountReadEdgeRule
// ---------------------------------------------------------------------------

func TestHostMountReadEdgeRuleSensitivePath(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "my-pod"),
			Kinds:     []string{"Pod"},
			Name:      "my-pod",
			Namespace: "ns1",
		},
		NodeName: "node-1",
		Volumes: []workload.VolumeDetail{
			{Name: "etc-vol", HostPath: "/etc"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "etc-vol", MountPath: "/host-etc"}}},
		},
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("Node", "", "node-1"),
			Kinds: []string{"Node"},
			Name:  "node-1",
		}},
	)

	ctx := framework.NewContext(core)
	edges := HostMountReadEdgeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Pod", "ns1", "my-pod"), nodefw.BuildID("Node", "", "node-1"), "hostMountSensitive") {
		t.Fatalf("missing hostMountSensitive edge from pod to node")
	}
}

func TestHostMountReadEdgeRulePodWithoutNode(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "orphan-pod"),
			Kinds:     []string{"Pod"},
			Name:      "orphan-pod",
			Namespace: "ns1",
		},
		NodeName: "",
		Volumes: []workload.VolumeDetail{
			{Name: "etc-vol", HostPath: "/etc"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "etc-vol", MountPath: "/host-etc"}}},
		},
	})

	ctx := framework.NewContext(core)
	edges := HostMountReadEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for pod without node, got %d", len(edges))
	}
}

func TestHostMountReadEdgeRuleNonSensitivePath(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "my-pod"),
			Kinds:     []string{"Pod"},
			Name:      "my-pod",
			Namespace: "ns1",
		},
		NodeName: "node-1",
		Volumes: []workload.VolumeDetail{
			{Name: "data-vol", HostPath: "/data/app"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "data-vol", MountPath: "/host-data"}}},
		},
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("Node", "", "node-1"),
			Kinds: []string{"Node"},
			Name:  "node-1",
		}},
	)

	ctx := framework.NewContext(core)
	edges := HostMountReadEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for non-sensitive path, got %d", len(edges))
	}
}

func TestHostMountReadEdgeRuleSkipsSocketPath(t *testing.T) {
	// A .sock path under a sensitive dir must not emit hostMountSensitive —
	// that is exclusively MOUNT_CONTAINER_SOCKET's domain.
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "sock-pod"),
			Kinds:     []string{"Pod"},
			Name:      "sock-pod",
			Namespace: "ns1",
		},
		NodeName: "node-1",
		Volumes: []workload.VolumeDetail{
			{Name: "sock-vol", HostPath: "/var/run/containerd/containerd.sock"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "sock-vol", MountPath: "/run/containerd/containerd.sock"}}},
		},
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("Node", "", "node-1"),
			Kinds: []string{"Node"},
			Name:  "node-1",
		}},
	)

	ctx := framework.NewContext(core)
	edges := HostMountReadEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no hostMountSensitive edges for socket path, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// HostMountKubeletEdgeRule
// ---------------------------------------------------------------------------

func TestHostMountKubeletEdgeRuleKubeletPath(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "my-pod"),
			Kinds:     []string{"Pod"},
			Name:      "my-pod",
			Namespace: "ns1",
		},
		NodeName: "node-1",
		Volumes: []workload.VolumeDetail{
			{Name: "kub-vol", HostPath: "/var/lib/kubelet"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "kub-vol", MountPath: "/host-kub"}}},
		},
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("Node", "", "node-1"),
			Kinds: []string{"Node"},
			Name:  "node-1",
		}},
	)

	ctx := framework.NewContext(core)
	edges := HostMountKubeletEdgeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Pod", "ns1", "my-pod"), nodefw.BuildID("Node", "", "node-1"), "mountedKubelet") {
		t.Fatalf("missing mountedKubelet edge from pod to node")
	}
}

func TestHostMountKubeletEdgeRulePodWithoutNode(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("Pod", "ns1", "orphan-pod"),
			Kinds:     []string{"Pod"},
			Name:      "orphan-pod",
			Namespace: "ns1",
		},
		NodeName: "",
		Volumes: []workload.VolumeDetail{
			{Name: "kub-vol", HostPath: "/var/lib/kubelet"},
		},
		Containers: []workload.Container{
			{VolumeMounts: []workload.VolumeMount{{Name: "kub-vol", MountPath: "/host"}}},
		},
	})

	ctx := framework.NewContext(core)
	edges := HostMountKubeletEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for pod without node, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// PodMountServiceAccountEdgeRule
// ---------------------------------------------------------------------------

func TestPodMountServiceAccountNonDefaultMountsToken(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbacnodes.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "myapp-sa")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase:    base("Pod", "ns1", "my-pod"),
		ServiceAccount:   "myapp-sa",
		AutomountSAToken: nil, // non-default SA: nil → mounted
	})

	ctx := framework.NewContext(core)
	edges := PodMountServiceAccountEdgeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Pod", "ns1", "my-pod"), nodefw.BuildID("ServiceAccount", "ns1", "myapp-sa"), "mountSA") {
		t.Fatalf("missing mountSA edge for non-default SA")
	}
}

func TestPodMountServiceAccountDefaultNoAutomount(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbacnodes.ServiceAccount{GraphNodeBase: base("ServiceAccount", "ns1", "default")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase:    base("Pod", "ns1", "my-pod"),
		ServiceAccount:   "default",
		AutomountSAToken: nil, // default SA: nil → NOT mounted
	})

	ctx := framework.NewContext(core)
	edges := PodMountServiceAccountEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no mountSA edge for default SA without explicit automount, got %d", len(edges))
	}
}

func TestPodMountServiceAccountSANotInIndex(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	// No SA registered for "missing-sa"
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase:  base("Pod", "ns1", "my-pod"),
		ServiceAccount: "missing-sa",
	})

	ctx := framework.NewContext(core)
	edges := PodMountServiceAccountEdgeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edge when SA not in index, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// PersistentVolumesEdgesRule
// ---------------------------------------------------------------------------

func TestPersistentVolumesEdgesRulePVCMountedByPod(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.PersistentVolumeClaims = append(ns.PersistentVolumeClaims,
		mountnodes.PersistentVolumeClaim{GraphNodeBase: base("PersistentVolumeClaim", "ns1", "my-pvc")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Volumes: []workload.VolumeDetail{
			{Name: "pvc-vol", PVCName: "my-pvc"},
		},
	})

	ctx := framework.NewContext(core)
	edges := PersistentVolumesEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("PersistentVolumeClaim", "ns1", "my-pvc"), nodefw.BuildID("Pod", "ns1", "my-pod"), "MountedBy") {
		t.Fatalf("missing MountedBy edge from PVC to pod")
	}
}

func TestPersistentVolumesEdgesRulePVBoundToPVC(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.PersistentVolumeClaims = append(ns.PersistentVolumeClaims,
		mountnodes.PersistentVolumeClaim{GraphNodeBase: base("PersistentVolumeClaim", "ns1", "my-pvc")},
	)
	core.Cluster.PersistentVolumes = append(core.Cluster.PersistentVolumes,
		mountnodes.PersistentVolume{
			GraphNodeBase: base("PersistentVolume", "", "my-pv"),
			ClaimRef:      &mountnodes.ClaimRef{Name: "my-pvc", Namespace: "ns1"},
		},
	)

	ctx := framework.NewContext(core)
	edges := PersistentVolumesEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("PersistentVolume", "", "my-pv"), nodefw.BuildID("PersistentVolumeClaim", "ns1", "my-pvc"), "BoundTo") {
		t.Fatalf("missing BoundTo edge from PV to PVC")
	}
}

func TestPersistentVolumesEdgesRulePVWithNoClaimRefSkipped(t *testing.T) {
	core := newCore()
	core.Cluster.PersistentVolumes = append(core.Cluster.PersistentVolumes,
		mountnodes.PersistentVolume{
			GraphNodeBase: base("PersistentVolume", "", "unbound-pv"),
			ClaimRef:      nil,
		},
	)

	ctx := framework.NewContext(core)
	edges := PersistentVolumesEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for PV with nil ClaimRef, got %d", len(edges))
	}
}
