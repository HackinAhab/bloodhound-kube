package security

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/addons/scc"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
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

// podWithNode builds a Pod with an ID+NodeName set so the escape rules look it up.
func podWithNode(namespace, podName, nodeName string, mutateFn func(*workload.Pod)) workload.Pod {
	pod := workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:        nodefw.BuildID("BHK_Pod", namespace, podName),
			Kinds:     []string{"BHK_Pod"},
			Name:      podName,
			Namespace: namespace,
		},
		NodeName: nodeName,
	}
	if mutateFn != nil {
		mutateFn(&pod)
	}
	return pod
}

func addNode(core *model.CoreFacts, name string) {
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Node", "", name),
			Kinds: []string{"BHK_Node"},
			Name:  name,
		}},
	)
}

// ---------------------------------------------------------------------------
// containerEscapeRule – individual escape checks
// ---------------------------------------------------------------------------

func TestContainerEscapeRulePrivMount(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "priv-pod", "node-1", func(p *workload.Pod) {
		p.Containers = []workload.Container{{Privileged: true}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "priv-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_PRIV_MOUNT") {
		t.Fatalf("missing CE_PRIV_MOUNT edge")
	}
}

func TestContainerEscapeRuleNsEnter(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "nsenter-pod", "node-1", func(p *workload.Pod) {
		p.Containers = []workload.Container{{Privileged: true}}
		p.HostPID = true
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "nsenter-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_NSENTER") {
		t.Fatalf("missing CE_NSENTER edge")
	}
}

func TestContainerEscapeRuleSysPtrace(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "ptrace-pod", "node-1", func(p *workload.Pod) {
		p.Containers = []workload.Container{{Privileged: true}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "ptrace-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_SYS_PTRACE") {
		t.Fatalf("missing CE_SYS_PTRACE edge (privileged shortcut)")
	}
}

func TestContainerEscapeRuleUmhCorePattern(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "umh-pod", "node-1", func(p *workload.Pod) {
		p.Volumes = []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc"}}
		p.Containers = []workload.Container{{
			VolumeMounts: []workload.VolumeMount{{Name: "proc-vol", MountPath: "/host/proc", ReadOnly: false}},
		}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "umh-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_UMH_CORE_PATTERN") {
		t.Fatalf("missing CE_UMH_CORE_PATTERN edge")
	}
}

func TestContainerEscapeRuleMountContainerSocket(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "sock-pod", "node-1", func(p *workload.Pod) {
		p.Volumes = []workload.VolumeDetail{{Name: "sock-vol", HostPath: "/var/run/docker.sock"}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "sock-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_MOUNT_CONTAINER_SOCKET") {
		t.Fatalf("missing MOUNT_CONTAINER_SOCKET edge")
	}
}

func TestContainerEscapeRuleVarLogSymlink(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "varlog-pod", "node-1", func(p *workload.Pod) {
		p.Volumes = []workload.VolumeDetail{{Name: "log-vol", HostPath: "/var/log"}}
		p.Containers = []workload.Container{{Privileged: true, RunAsNonRoot: false}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "varlog-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_VAR_LOG_SYMLINK") {
		t.Fatalf("missing CE_VAR_LOG_SYMLINK edge")
	}
}

func TestContainerEscapeRuleHostIPC(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "ipc-pod", "node-1", func(p *workload.Pod) {
		p.HostIPC = true
		p.Containers = []workload.Container{{Privileged: true}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "ipc-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_HOST_IPC") {
		t.Fatalf("missing CE_HOST_IPC edge")
	}
}

func TestContainerEscapeRuleHostNetwork(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, podWithNode("ns1", "net-pod", "node-1", func(p *workload.Pod) {
		p.HostNetwork = true
		p.Containers = []workload.Container{{Privileged: true}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "net-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_HOST_NETWORK") {
		t.Fatalf("missing CE_HOST_NETWORK edge")
	}
}

func TestContainerEscapeRuleShareProcNs(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	shareTrue := true
	ns.Pods = append(ns.Pods, podWithNode("ns1", "procns-pod", "node-1", func(p *workload.Pod) {
		p.ShareProcNs = &shareTrue
		p.Containers = []workload.Container{{Privileged: true}}
	}))
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "procns-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_CE_SHARE_PROC_NS") {
		t.Fatalf("missing CE_SHARE_PROC_NS edge")
	}
}

func TestContainerEscapeRulePodWithoutNodeSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	// Pod has ID but no NodeName (and no node in cluster)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "orphan"),
			Kinds: []string{"BHK_Pod"},
			Name:  "orphan",
		},
		NodeName:   "",
		Containers: []workload.Container{{Privileged: true}},
	})

	ctx := framework.NewContext(core)
	edges := containerEscapeRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no escape edges for pod without node, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// capabilityEdgesRule
// ---------------------------------------------------------------------------

func TestCapabilityEdgesRuleKnownCaps(t *testing.T) {
	knownCaps := []string{"SYS_ADMIN", "NET_ADMIN", "SYS_MODULE", "SYS_PTRACE", "SYS_RAWIO"}
	expectedKinds := []string{"BHK_CAP_SYS_ADMIN", "BHK_CAP_NET_ADMIN", "BHK_CAP_SYS_MODULE", "BHK_CAP_SYS_PTRACE", "BHK_CAP_SYS_RAWIO"}

	for i, cap := range knownCaps {
		t.Run(cap, func(t *testing.T) {
			core := newCore()
			ns := ensureNamespace(core, "ns1")
			ns.Pods = append(ns.Pods, workload.Pod{
				GraphNodeBase: nodefw.GraphNodeBase{
					ID:    nodefw.BuildID("BHK_Pod", "ns1", "cap-pod"),
					Kinds: []string{"BHK_Pod"},
					Name:  "cap-pod",
				},
				NodeName:        "node-1",
				CapabilitiesAdd: []string{cap},
			})
			addNode(core, "node-1")

			ctx := framework.NewContext(core)
			edges := capabilityEdgesRule{}.Apply(ctx)
			if !hasEdge(edges, nodefw.BuildID("BHK_Pod", "ns1", "cap-pod"), nodefw.BuildID("BHK_Node", "", "node-1"), expectedKinds[i]) {
				t.Fatalf("missing %s edge", expectedKinds[i])
			}
		})
	}
}

func TestCapabilityEdgesRuleUnknownCapSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "cap-pod"),
			Kinds: []string{"BHK_Pod"},
			Name:  "cap-pod",
		},
		NodeName:        "node-1",
		CapabilitiesAdd: []string{"UNKNOWN_CAP"},
	})
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := capabilityEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for unknown cap, got %d", len(edges))
	}
}

func TestCapabilityEdgesRulePodWithoutNodeSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "cap-pod"),
			Kinds: []string{"BHK_Pod"},
			Name:  "cap-pod",
		},
		NodeName:        "", // no node
		CapabilitiesAdd: []string{"SYS_ADMIN"},
	})

	ctx := framework.NewContext(core)
	edges := capabilityEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for pod without node, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// sccEdgesRule
// ---------------------------------------------------------------------------

func TestSCCEdgesRuleEnforcedSCC(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.NewGraphNodeBase(
			"BHK_Pod", "ns1", "annotated-pod",
			nil,
			map[string]any{"openshift.io/scc": "restricted"},
		),
	})
	core.Cluster.SecurityContextConstraints = append(core.Cluster.SecurityContextConstraints,
		scc.SecurityContextConstraints{
			GraphNodeBase: nodefw.GraphNodeBase{
				ID:    nodefw.BuildID("BHK_SecurityContextConstraints", "", "restricted"),
				Kinds: []string{"BHK_SecurityContextConstraints"},
				Name:  "restricted",
			},
		},
	)

	ctx := framework.NewContext(core)
	edges := sccEdgesRule{}.Apply(ctx)
	if !hasEdge(edges,
		nodefw.BuildID("BHK_SecurityContextConstraints", "", "restricted"),
		nodefw.BuildID("BHK_Pod", "ns1", "annotated-pod"),
		"BHK_EnforcedSCC",
	) {
		t.Fatalf("missing EnforcedSCC edge from SCC to pod")
	}
}

func TestSCCEdgesRuleNoAnnotationSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns1", "unannotated-pod"),
	})
	core.Cluster.SecurityContextConstraints = append(core.Cluster.SecurityContextConstraints,
		scc.SecurityContextConstraints{
			GraphNodeBase: nodefw.GraphNodeBase{
				ID:    nodefw.BuildID("BHK_SecurityContextConstraints", "", "restricted"),
				Kinds: []string{"BHK_SecurityContextConstraints"},
				Name:  "restricted",
			},
		},
	)

	ctx := framework.NewContext(core)
	edges := sccEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no EnforcedSCC edges for pod without annotation, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// hostPortsEdgesRule
// ---------------------------------------------------------------------------

func TestHostPortsEdgesRuleHostPortEdge(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "hostport-pod"),
			Kinds: []string{"BHK_Pod"},
			Name:  "hostport-pod",
		},
		NodeName: "node-1",
		Containers: []workload.Container{
			{HostPorts: []workload.HostPort{{HostPort: 8080}}},
		},
	})
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := hostPortsEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("BHK_Node", "", "node-1"), nodefw.BuildID("BHK_Pod", "ns1", "hostport-pod"), "BHK_HostPort") {
		t.Fatalf("missing HostPort edge from node to pod")
	}
}

func TestHostPortsEdgesRuleExternalHostPort(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "hostport-pod"),
			Kinds: []string{"BHK_Pod"},
			Name:  "hostport-pod",
		},
		NodeName: "node-1",
		Containers: []workload.Container{
			{HostPorts: []workload.HostPort{{HostPort: 8080}}},
		},
	})
	addNode(core, "node-1")
	core.Cluster.External = append(core.Cluster.External,
		platform.ExternalCoreEntry(),
	)

	ctx := framework.NewContext(core)
	edges := hostPortsEdgesRule{}.Apply(ctx)
	externalID := nodefw.BuildID("BHK_External", "", "external")
	if !hasEdge(edges, externalID, nodefw.BuildID("BHK_Node", "", "node-1"), "BHK_ExternalHostPort") {
		t.Fatalf("missing ExternalHostPort edge from External to node")
	}
}

func TestHostPortsEdgesRuleZeroHostPortSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: nodefw.GraphNodeBase{
			ID:    nodefw.BuildID("BHK_Pod", "ns1", "no-hostport-pod"),
			Kinds: []string{"BHK_Pod"},
			Name:  "no-hostport-pod",
		},
		NodeName: "node-1",
		Containers: []workload.Container{
			{HostPorts: []workload.HostPort{{HostPort: 0}}},
		},
	})
	addNode(core, "node-1")

	ctx := framework.NewContext(core)
	edges := hostPortsEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for hostPort=0, got %d", len(edges))
	}
}

func TestHostPortsEdgesRulePodWithoutNodeSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("BHK_Pod", "ns1", "orphan-pod"),
		NodeName:      "",
		Containers: []workload.Container{
			{HostPorts: []workload.HostPort{{HostPort: 9090}}},
		},
	})

	ctx := framework.NewContext(core)
	edges := hostPortsEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for pod without node, got %d", len(edges))
	}
}
